import { useState, useRef } from 'react';
import { TimelineItem, Message, PendingTool } from '../types';
import {
  fetchSessionMessages,
  sendChatMessage,
  confirmInterrupt,
  cancelSessionChat,
} from '../api';

interface UseChatOptions {
  userID: string;
  selectedModel: string;
  loadSessions: () => void;
  evolveProfile: (sessionID: string) => void;
}

export function useChat({ userID, selectedModel, loadSessions, evolveProfile }: UseChatOptions) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isStreaming, setIsStreaming] = useState<boolean>(false);
  const [webSearchEnabled, setWebSearchEnabled] = useState<boolean>(true);
  const [pendingInterrupt, setPendingInterrupt] = useState<{
    session_id: string;
    pending_tools: PendingTool[];
  } | null>(null);
  const [expandedReasoning, setExpandedReasoning] = useState<Record<number, boolean>>({});

  // 用于在流式对话中动态更新的消息缓冲区
  const streamingMsgRef = useRef<Message | null>(null);
  const aiMsgIdxRef = useRef<number>(-1);

  // 2a. 从服务器加载消息，并将 assistant → tool → assistant 链合并为单条消息
  //     使历史消息结构与流式时的单气泡完全一致
  const rebuildAndSetMessages = async (id: string) => {
    try {
      const data = await fetchSessionMessages(id);
      const raw: any[] = data.messages || [];

      const merged: Message[] = [];
      let i = 0;

      while (i < raw.length) {
        const msg = raw[i];

        // ── 遇到带 tool_calls 的 assistant 消息 → 开始合并链 ──
        if (msg.role === 'assistant' && msg.tool_calls?.length > 0) {
          const timeline: TimelineItem[] = [];
          let finalContent = '';

          // 可能有多轮 tool-call，循环消费直到遇到无 tool_calls 的 assistant
          while (i < raw.length) {
            const cur = raw[i];

            if (cur.role === 'assistant' && cur.tool_calls?.length > 0) {
              // 1. 推理文本
              if (cur.reasoning_content) {
                timeline.push({ kind: 'reasoning', text: cur.reasoning_content });
              }
              // 2. 收集本轮 tool_call 条目（暂无 output）
              const pending: TimelineItem[] = cur.tool_calls.map((tc: any) => ({
                kind: 'tool_call' as const,
                toolCallId: tc.id,
                toolName: tc.function?.name,
                toolArgs: tc.function?.arguments,
              }));
              i++;

              // 3. 紧跟的 tool 消息 → 填充 output
              while (i < raw.length && raw[i].role === 'tool') {
                const toolMsg = raw[i];
                const item = pending.find((p) => (p as any).toolCallId === toolMsg.tool_call_id);
                if (item) (item as any).output = toolMsg.content;
                i++;
              }
              timeline.push(...pending);

            } else if (cur.role === 'assistant') {
              // 最终 assistant 回答（无 tool_calls）
              if (cur.reasoning_content) {
                timeline.push({ kind: 'reasoning', text: cur.reasoning_content });
              }
              finalContent = cur.content;
              i++;
              break;
            } else {
              break; // 非 assistant 消息，链结束
            }
          }

          merged.push({ role: 'assistant', content: finalContent, timeline });

        } else if (msg.role === 'tool') {
          // 孤立的 tool 消息（不应出现，跳过）
          i++;
        } else {
          merged.push(msg as Message);
          i++;
        }
      }

      setMessages(merged);
      return data;
    } catch (err) {
      console.error('加载消息失败:', err);
      return null;
    }
  };

  // 6. 发送对话请求并处理流式响应
  const handleSendMessage = async (text: string, currentSessionID: string) => {
    if (!text.trim() || isStreaming || !currentSessionID) return;

    setPendingInterrupt(null);
    setIsStreaming(true);

    // 追加用户消息到列表
    const userMsg: Message = { role: 'user', content: text };
    const initialAiMsg: Message = { role: 'assistant', content: '', reasoning_content: '' };
    setMessages((prev) => {
      const next = [...prev, userMsg, initialAiMsg];
      aiMsgIdxRef.current = next.length - 1;
      return next;
    });
    streamingMsgRef.current = initialAiMsg;

    const activeTools: string[] = [];
    if (webSearchEnabled) {
      activeTools.push('web_search', 'fetch_webpage');
    }

    try {
      const res = await sendChatMessage(currentSessionID, userID, text, selectedModel, activeTools);
      await parseSSEResponse(res, currentSessionID);
    } catch (err: any) {
      console.error('流处理异常:', err);
      setMessages((prev) => {
        const copy = [...prev];
        if (copy.length > 0 && copy[copy.length - 1].role === 'assistant') {
          copy[copy.length - 1].content = `【连接异常】无法连接到后端服务: ${err.message}`;
        }
        return copy;
      });
      setIsStreaming(false);
    }
  };

  // 7. 处理工具人工审批 (Approve)
  const handleApproveInterrupt = async (currentSessionID: string) => {
    if (!pendingInterrupt || isStreaming) return;
    const confirmedIDs = pendingInterrupt.pending_tools.map((t) => t.id);
    setPendingInterrupt(null);
    setIsStreaming(true);

    // 在页面上模拟插入一条系统提示
    // 重新追加 AI 占位符
    const initialAiMsg: Message = { role: 'assistant', content: '', reasoning_content: '' };
    setMessages((prev) => {
      const next = [...prev, { role: 'system' as const, content: '✓ 人工确认：已同意执行敏感工具操作。正在恢复执行...' }, initialAiMsg];
      aiMsgIdxRef.current = next.length - 1;
      return next;
    });
    streamingMsgRef.current = initialAiMsg;

    try {
      const res = await confirmInterrupt(currentSessionID, userID, confirmedIDs);
      await parseSSEResponse(res, currentSessionID);
    } catch (err: any) {
      console.error('恢复流处理异常:', err);
      setMessages((prev) => {
        const copy = [...prev];
        if (copy.length > 0 && copy[copy.length - 1].role === 'assistant') {
          copy[copy.length - 1].content = `【恢复异常】操作失败: ${err.message}`;
        }
        return copy;
      });
      setIsStreaming(false);
    }
  };

  // 7. 处理工具人工审批 (Reject)
  const handleRejectInterrupt = () => {
    setPendingInterrupt(null);
    setMessages((prev) => [
      ...prev,
      { role: 'system', content: '✗ 人工确认：已拒绝执行敏感操作。当前智能体流程安全中止。' },
    ]);
    // 强制把后端的 Session 状态刷新
    loadSessions();
  };

  const handleCancelChat = async (currentSessionID: string) => {
    if (!currentSessionID || !isStreaming) return;
    setIsStreaming(false); // 立即恢复交互状态
    setMessages((prev) => [...prev, { role: 'interrupted', content: '对话已中断' }]);
    try {
      await cancelSessionChat(currentSessionID);
    } catch (err: any) {
      console.error('停止生成请求失败:', err);
    } finally {
      loadSessions(); // 刷新会话状态以保持同步
    }
  };

  // 8. 核心 SSE 响应读取解析器
  const parseSSEResponse = async (res: Response, currentSessionID: string) => {
    const reader = res.body?.getReader();
    if (!reader) return;

    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      buffer = buffer.replace(/\r\n/g, '\n');

      const blocks = buffer.split('\n\n');
      buffer = blocks.pop() || '';

      for (const block of blocks) {
        const trimmedBlock = block.trim();
        if (!trimmedBlock) continue;

        let currentEvent = '';
        let currentData = '';
        const lines = trimmedBlock.split('\n');

        for (const line of lines) {
          const trimmedLine = line.trim();
          if (trimmedLine.startsWith('event:')) {
            currentEvent = trimmedLine.slice(6).trim();
          } else if (trimmedLine.startsWith('data:')) {
            // 如果有多行 data，拼接它们
            const dataValue = trimmedLine.slice(5).trim();
            if (currentData) {
              currentData += dataValue;
            } else {
              currentData = dataValue;
            }
          }
        }

        handleSSEChunk(currentEvent, currentData, currentSessionID);
      }
    }
  };

  const handleSSEChunk = (event: string, data: string, currentSessionID: string) => {
    switch (event) {
      case 'text_delta': {
        // 优化：尝试多种解析方式
        let text = data;

        // 如果 data 为空，跳过
        if (!data || data === '') {
          return;
        }

        try {
          // 尝试解析 JSON
          const parsed = JSON.parse(data);
          // 如果是对象，提取 content 字段
          if (typeof parsed === 'object' && parsed !== null) {
            text = parsed.content || parsed.text || JSON.stringify(parsed);
          } else if (typeof parsed === 'string') {
            text = parsed;
          }
        } catch {
          // 不是 JSON，直接使用原始数据
          text = data;
        }

        // 确保是字符串
        text = String(text);

        updateLastAiMessage((msg) => {
          msg.content = (msg.content || '') + text;
        });
        break;
      }

      case 'reasoning_delta': {
        let rText = data;
        if (!data || data === '') {
          return;
        }

        try {
          const parsed = JSON.parse(data);
          if (typeof parsed === 'object' && parsed !== null) {
            rText = parsed.content || parsed.text || JSON.stringify(parsed);
          } else if (typeof parsed === 'string') {
            rText = parsed;
          }
        } catch {
          rText = data;
        }

        rText = String(rText);

        updateLastAiMessage((msg) => {
          const tl = [...(msg.timeline || [])];
          const last = tl[tl.length - 1];
          if (last && last.kind === 'reasoning') {
            tl[tl.length - 1] = { kind: 'reasoning', text: (last.text || '') + rText };
          } else {
            tl.push({ kind: 'reasoning', text: rText });
          }
          msg.timeline = tl;
        });
        break;
      }

      case 'tool_call': {
        if (!data) break;
        try {
          const toolCall = JSON.parse(data);
          updateLastAiMessage((msg) => {
            const item: TimelineItem = {
              kind: 'tool_call',
              toolCallId: toolCall.id || toolCall.tool_call_id,
              toolName: toolCall.function?.name || toolCall.name,
              toolArgs: toolCall.function?.arguments || toolCall.arguments,
            };
            msg.timeline = [...(msg.timeline || []), item];
          });
        } catch (error) {
          console.warn('[SSE] tool_call 解析失败:', error, data);
        }
        break;
      }

      case 'tool_result': {
        if (!data) break;
        try {
          const toolResult = JSON.parse(data);
          updateLastAiMessage((msg) => {
            const tl = [...(msg.timeline || [])];
            let callIdx = -1;
            for (let k = tl.length - 1; k >= 0; k--) {
              if (tl[k].kind === 'tool_call' && tl[k].toolCallId === toolResult.tool_call_id) {
                callIdx = k;
                break;
              }
            }
            if (callIdx !== -1) {
              tl[callIdx] = {
                ...tl[callIdx],
                output: toolResult.output,
                error: toolResult.error
              } as TimelineItem;
            } else {
              tl.push({
                kind: 'tool_call',
                toolCallId: toolResult.tool_call_id,
                output: toolResult.output,
                error: toolResult.error
              });
            }
            msg.timeline = tl;
          });
        } catch (error) {
          console.warn('[SSE] tool_result 解析失败:', error, data);
        }
        break;
      }

      case 'interrupt': {
        if (!data) break;
        try {
          const interruptData = JSON.parse(data);
          setPendingInterrupt(interruptData);
          setIsStreaming(false);
          loadSessions();
        } catch (error) {
          console.warn('[SSE] interrupt 解析失败:', error, data);
        }
        break;
      }

      case 'done': {
        setIsStreaming(false);
        loadSessions();
        setTimeout(() => {
          evolveProfile(currentSessionID);
        }, 1200);
        break;
      }

      case 'error': {
        let errorMsg = data;
        try {
          const errObj = JSON.parse(data);
          errorMsg = errObj.message || errObj.error || data;
        } catch {
          errorMsg = data;
        }
        updateLastAiMessage((msg) => {
          msg.content = (msg.content || '') + `\n【系统错误】${errorMsg}`;
        });
        setIsStreaming(false);
        break;
      }

      default: {
        // 🔥 处理未知事件，打印日志便于调试
        if (event) {
          console.log('[SSE] 未知事件:', event, data);
        }
        break;
      }
    }
  };

  const updateLastAiMessage = (updateFn: (msg: Message) => void) => {
    setMessages((prev) => {
      if (prev.length === 0) return prev;
      const copy = [...prev];
      const last = { ...copy[copy.length - 1] };
      if (last.role === 'assistant') {
        updateFn(last);
        copy[copy.length - 1] = last;
      }
      return copy;
    });
  };

  return {
    messages,
    setMessages,
    isStreaming,
    pendingInterrupt,
    setPendingInterrupt,
    expandedReasoning,
    setExpandedReasoning,
    rebuildAndSetMessages,
    handleSendMessage,
    handleApproveInterrupt,
    handleRejectInterrupt,
    handleCancelChat,
    webSearchEnabled,
    setWebSearchEnabled,
  };
}
