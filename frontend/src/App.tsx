import { useState, useEffect, useRef } from 'react';
import Sidebar from './components/Sidebar';
import ChatArea from './components/ChatArea';
import MemoryPanel from './components/MemoryPanel';
import { TimelineItem, Message, Session, Profile, PendingTool, UserInfo } from './types';
import {
  fetchUserInfo,
  fetchSessions,
  fetchSessionMessages,
  createSession,
  fetchUserProfile,
  evolveUserProfile,
  sendChatMessage,
  confirmInterrupt,
} from './api';

export default function App() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [currentSessionID, setCurrentSessionID] = useState<string>('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [userID, setUserID] = useState<string>('');
  const [userProfile, setUserProfile] = useState<Profile>({
    user_id: '',
    preferences: [],
    facts: [],
  });
  const [selectedModel, setSelectedModel] = useState<string>('deepseek-v4-flash');
  const [isStreaming, setIsStreaming] = useState<boolean>(false);
  const [isEvolving, setIsEvolving] = useState<boolean>(false);
  const [pendingInterrupt, setPendingInterrupt] = useState<{
    session_id: string;
    pending_tools: PendingTool[];
  } | null>(null);

  // 控制推理思考流的折叠/展开状态，默认 key 对应 message 索引，值为 false 表示折叠
  const [expandedReasoning, setExpandedReasoning] = useState<Record<number, boolean>>({});

  // 状态：深色/明亮模式
  const [isDarkMode, setIsDarkMode] = useState<boolean>(false);

  // 状态：长期记忆画像面板折叠状态，默认折叠
  const [isMemoryCollapsed, setIsMemoryCollapsed] = useState<boolean>(true);

  // 用于在流式对话中动态更新的消息缓冲区
  const streamingMsgRef = useRef<Message | null>(null);

  // 初始化加载及主题检测
  useEffect(() => {
    const loadUser = async (retries = 5, delay = 1500) => {
      try {
        const u = await fetchUserInfo();
        if (u && u.id) {
          setUserInfo(u);
          setUserID(u.id);
          return;
        }
        throw new Error('User info has no valid ID');
      } catch (err) {
        if (retries > 0) {
          console.warn(`加载用户信息失败，将在 ${delay}ms 后重试... 剩余重试次数: ${retries}`);
          setTimeout(() => loadUser(retries - 1, delay), delay);
        } else {
          console.error('加载用户信息失败，已达到最大重试次数:', err);
        }
      }
    };

    loadUser();

    const savedTheme = localStorage.getItem('theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    if (savedTheme === 'dark' || (!savedTheme && prefersDark)) {
      setIsDarkMode(true);
      document.documentElement.classList.add('dark');
    } else {
      setIsDarkMode(false);
      document.documentElement.classList.remove('dark');
    }
  }, []);

  // 当 userID 就绪后，再加载会话历史和画像
  useEffect(() => {
    if (userID) {
      loadSessions();
      loadProfile();
    }
  }, [userID]);

  // 切换主题
  const toggleTheme = () => {
    if (isDarkMode) {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
      setIsDarkMode(false);
    } else {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
      setIsDarkMode(true);
    }
  };

  // 1. 获取会话历史列表
  const loadSessions = async () => {
    if (!userID) return;
    try {
      const data = await fetchSessions(userID);
      setSessions(data);
      // 如果有会话，默认选中第一个
      if (data.length > 0 && !currentSessionID) {
        selectSession(data[0].id);
      }
    } catch (err: any) {
      alert('加载会话失败，网络或后端连接异常: ' + err.message);
      console.error('加载会话失败:', err);
    }
  };

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

  // 2. 加载指定会话的消息
  const selectSession = async (id: string) => {
    if (isStreaming) return;
    setCurrentSessionID(id);
    setPendingInterrupt(null);
    setExpandedReasoning({}); // 重置折叠状态
    try {
      const data = await rebuildAndSetMessages(id);
      if (data?.status === 'pending_confirmation') {
        const lastMsg = data.messages?.[data.messages.length - 1];
        if (lastMsg && lastMsg.tool_calls) {
          setPendingInterrupt({
            session_id: id,
            pending_tools: lastMsg.tool_calls.map((tc: any) => ({
              id: tc.id,
              function: { name: tc.function.name, arguments: tc.function.arguments },
            })),
          });
        }
      }
    } catch (err: any) {
      alert('读取历史消息失败，网络或后端连接异常: ' + err.message);
      console.error('切换会话失败:', err);
    }
  };

  // 3. 创建全新会话
  const createNewSession = async () => {
    if (isStreaming) return;
    const newSessionID = 'sess_' + Math.random().toString(36).substr(2, 9);
    try {
      await createSession(newSessionID, userID);
      await loadSessions();
      selectSession(newSessionID);
    } catch (err: any) {
      alert('创建会话失败，网络或后端连接异常: ' + err.message);
      console.error('创建会话失败:', err);
    }
  };

  // 4. 获取用户长期记忆画像
  const loadProfile = async () => {
    if (!userID) return;
    try {
      const profile = await fetchUserProfile(userID);
      setUserProfile(profile);
    } catch (err) {
      console.error('加载画像失败:', err);
    }
  };

  // 5. 触发画像演化
  const evolveProfile = async () => {
    if (!currentSessionID || isEvolving) return;
    setIsEvolving(true);
    try {
      const profile = await evolveUserProfile(userID, currentSessionID);
      setUserProfile(profile);
    } catch (err) {
      console.error('触发记忆演化失败:', err);
    } finally {
      setIsEvolving(false);
    }
  };

  // 6. 发送对话请求并处理流式响应
  const handleSendMessage = async (text: string) => {
    if (!text.trim() || isStreaming || !currentSessionID) return;

    setPendingInterrupt(null);
    setIsStreaming(true);

    // 追加用户消息到列表
    const userMsg: Message = { role: 'user', content: text };
    setMessages((prev) => [...prev, userMsg]);

    // 初始化占位 AI 消息
    const initialAiMsg: Message = { role: 'assistant', content: '', reasoning_content: '' };
    setMessages((prev) => [...prev, initialAiMsg]);
    streamingMsgRef.current = initialAiMsg;

    try {
      const res = await sendChatMessage(currentSessionID, userID, text, selectedModel);
      await parseSSEResponse(res);
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

  // 7. 处理工具人工审批 (Approve / Reject)
  const handleApproveInterrupt = async () => {
    if (!pendingInterrupt || isStreaming) return;
    const confirmedIDs = pendingInterrupt.pending_tools.map((t) => t.id);
    setPendingInterrupt(null);
    setIsStreaming(true);

    // 在页面上模拟插入一条系统提示
    setMessages((prev) => [...prev, { role: 'system', content: '✓ 人工确认：已同意执行敏感工具操作。正在恢复执行...' }]);

    // 重新追加 AI 占位符
    const initialAiMsg: Message = { role: 'assistant', content: '', reasoning_content: '' };
    setMessages((prev) => [...prev, initialAiMsg]);
    streamingMsgRef.current = initialAiMsg;

    try {
      const res = await confirmInterrupt(currentSessionID, userID, confirmedIDs);
      await parseSSEResponse(res);
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

  const handleRejectInterrupt = () => {
    setPendingInterrupt(null);
    setMessages((prev) => [
      ...prev,
      { role: 'system', content: '✗ 人工确认：已拒绝执行敏感操作。当前智能体流程安全中止。' },
    ]);
    // 强制把后端的 Session 状态刷新
    loadSessions();
  };

  // 8. 核心 SSE 响应读取解析器
  const parseSSEResponse = async (res: Response) => {
    const reader = res.body?.getReader();
    if (!reader) return;

    const decoder = new TextDecoder();
    let buffer = '';
    let currentEvent = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || ''; // 尾部未满行退回 buffer

      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;

        if (trimmed.startsWith('event:')) {
          currentEvent = trimmed.slice(6).trim();
        } else if (trimmed.startsWith('data:')) {
          const payload = trimmed.slice(5).trim();
          handleSSEChunk(currentEvent, payload);
        }
      }
    }
  };

  const handleSSEChunk = (event: string, data: string) => {
    switch (event) {
      case 'text_delta':
        let text = data;
        try {
          text = JSON.parse(data);
        } catch {
          // 降级使用 raw data
        }
        updateLastAiMessage((msg) => {
          msg.content += text;
        });
        break;

      case 'reasoning_delta':
        let rText = data;
        try {
          rText = JSON.parse(data);
        } catch {
          // 降级使用 raw data
        }
        updateLastAiMessage((msg) => {
          const tl = [...(msg.timeline || [])];
          const last = tl[tl.length - 1];
          if (last && last.kind === 'reasoning') {
            tl[tl.length - 1] = { kind: 'reasoning', text: last.text + rText };
          } else {
            tl.push({ kind: 'reasoning', text: rText });
          }
          msg.timeline = tl;
        });
        break;

      case 'tool_call':
        try {
          const toolCall = JSON.parse(data);
          updateLastAiMessage((msg) => {
            const item: TimelineItem = {
              kind: 'tool_call',
              toolCallId: toolCall.id,
              toolName: toolCall.function?.name,
              toolArgs: toolCall.function?.arguments,
            };
            msg.timeline = [...(msg.timeline || []), item];
          });
        } catch {}
        break;

      case 'tool_result':
        try {
          const toolResult = JSON.parse(data);
          updateLastAiMessage((msg) => {
            const tl = [...(msg.timeline || [])];
            // 找到最近的对应 tool_call_id 条目，填充 output/error
            let callIdx = -1;
            for (let k = tl.length - 1; k >= 0; k--) {
              if (tl[k].kind === 'tool_call' && tl[k].toolCallId === toolResult.tool_call_id) {
                callIdx = k;
                break;
              }
            }
            if (callIdx !== -1) {
              tl[callIdx] = { ...tl[callIdx], output: toolResult.output, error: toolResult.error } as TimelineItem;
            } else {
              // 找不到对应 call，追加一个无名工具结果
              tl.push({ kind: 'tool_call', toolCallId: toolResult.tool_call_id, output: toolResult.output, error: toolResult.error });
            }
            msg.timeline = tl;
          });
        } catch {}
        break;

      case 'interrupt':
        try {
          const interruptData = JSON.parse(data);
          setPendingInterrupt(interruptData);
          setIsStreaming(false);
          loadSessions(); // 刷新 sidebar 中会话的 pending 状态
        } catch {}
        break;

      case 'done':
        setIsStreaming(false);
        loadSessions(); // 刷新会话的更新时间
        // 流式已在内存中构建了与历史加载相同的 timeline 结构，无需再次请求
        // 对话结束后，触发一次偏好事实进化以保画像同步
        setTimeout(() => {
          evolveProfile();
        }, 1200);
        break;

      case 'error':
        try {
          const errObj = JSON.parse(data);
          updateLastAiMessage((msg) => {
            msg.content += `\n【系统错误】${errObj.message}`;
          });
        } catch {}
        setIsStreaming(false);
        break;

      default:
        break;
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

  return (
    <div className="portal-container">
      <Sidebar
        sessions={sessions}
        currentSessionID={currentSessionID}
        isStreaming={isStreaming}
        pendingInterrupt={pendingInterrupt}
        userInfo={userInfo}
        userID={userID}
        isDarkMode={isDarkMode}
        onSelectSession={selectSession}
        onCreateNewSession={createNewSession}
        onToggleTheme={toggleTheme}
      />
      <ChatArea
        messages={messages}
        currentSessionID={currentSessionID}
        isStreaming={isStreaming}
        pendingInterrupt={pendingInterrupt}
        selectedModel={selectedModel}
        isDarkMode={isDarkMode}
        expandedReasoning={expandedReasoning}
        setExpandedReasoning={setExpandedReasoning}
        onSendMessage={handleSendMessage}
        onApproveInterrupt={handleApproveInterrupt}
        onRejectInterrupt={handleRejectInterrupt}
        setSelectedModel={setSelectedModel}
        isMemoryCollapsed={isMemoryCollapsed}
        setIsMemoryCollapsed={setIsMemoryCollapsed}
      />
      <MemoryPanel
        userProfile={userProfile}
        isMemoryCollapsed={isMemoryCollapsed}
        isEvolving={isEvolving}
        currentSessionID={currentSessionID}
        onEvolveProfile={evolveProfile}
      />
    </div>
  );
}
