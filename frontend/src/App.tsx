import React, { useState, useEffect, useRef } from 'react';

// 定义数据契约
interface Message {
  role: 'user' | 'assistant' | 'tool' | 'system';
  content: string;
  reasoning_content?: string;
  tool_calls?: any[];
}

interface Session {
  id: string;
  user_id: string;
  updated_at: string;
  status?: string;
}

interface Profile {
  user_id: string;
  preferences: string[];
  facts: string[];
}

interface PendingTool {
  id: string;
  function: {
    name: string;
    arguments: string;
  };
}

export default function App() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [currentSessionID, setCurrentSessionID] = useState<string>('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [userProfile, setUserProfile] = useState<Profile>({
    user_id: 'user_test_999',
    preferences: [],
    facts: [],
  });
  const [inputValue, setInputValue] = useState<string>('');
  const [isStreaming, setIsStreaming] = useState<boolean>(false);
  const [isEvolving, setIsEvolving] = useState<boolean>(false);
  const [pendingInterrupt, setPendingInterrupt] = useState<{
    session_id: string;
    pending_tools: PendingTool[];
  } | null>(null);

  // 本地默认用户 ID
  const userID = 'user_test_999';

  // 用于在流式对话中动态更新的消息缓冲区
  const streamingMsgRef = useRef<Message | null>(null);
  const messageEndRef = useRef<HTMLDivElement | null>(null);

  // 初始化加载
  useEffect(() => {
    loadSessions();
    loadProfile();
  }, []);

  // 消息更新后滚动到底部
  useEffect(() => {
    messageEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isStreaming, pendingInterrupt]);

  // 1. 获取会话历史列表
  const loadSessions = async () => {
    try {
      const res = await fetch(`/api/sessions?user_id=${userID}`);
      if (res.ok) {
        const data = await res.json();
        setSessions(data);
        // 如果有会话，默认选中第一个
        if (data.length > 0 && !currentSessionID) {
          selectSession(data[0].id);
        }
      } else {
        console.error('加载会话失败，状态码:', res.status);
      }
    } catch (err: any) {
      alert('加载会话失败，网络或后端连接异常: ' + err.message);
      console.error('加载会话失败:', err);
    }
  };

  // 2. 加载指定会话的消息
  const selectSession = async (id: string) => {
    if (isStreaming) return;
    setCurrentSessionID(id);
    setPendingInterrupt(null);
    try {
      const res = await fetch(`/api/sessions/${id}/messages`);
      if (res.ok) {
        const data = await res.json();
        setMessages(data.messages || []);
        if (data.status === 'pending_confirmation') {
          // 如果会话在后端是挂起状态，通过解析最后一条消息提取待确认项以在前端渲染
          const lastMsg = data.messages?.[data.messages.length - 1];
          if (lastMsg && lastMsg.tool_calls) {
            setPendingInterrupt({
              session_id: id,
              pending_tools: lastMsg.tool_calls.map((tc: any) => ({
                id: tc.id,
                function: {
                  name: tc.function.name,
                  arguments: tc.function.arguments,
                },
              })),
            });
          }
        }
      } else {
        alert('读取历史消息失败，状态码: ' + res.status);
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
      const res = await fetch('/api/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: newSessionID, user_id: userID }),
      });
      if (res.ok) {
        await loadSessions();
        selectSession(newSessionID);
      } else {
        alert('创建会话失败，状态码: ' + res.status);
      }
    } catch (err: any) {
      alert('创建会话失败，网络或后端连接异常: ' + err.message);
      console.error('创建会话失败:', err);
    }
  };

  // 4. 获取用户长期记忆画像
  const loadProfile = async () => {
    try {
      const res = await fetch(`/api/users/${userID}/profile`);
      if (res.ok) {
        const data = await res.json();
        setUserProfile(data);
      }
    } catch (err) {
      console.error('加载画像失败:', err);
    }
  };

  // 5. 触发画像演化
  const evolveProfile = async () => {
    if (!currentSessionID || isEvolving) return;
    setIsEvolving(true);
    try {
      const res = await fetch(`/api/users/${userID}/evolve?session_id=${currentSessionID}`, {
        method: 'POST',
      });
      if (res.ok) {
        const data = await res.json();
        setUserProfile(data);
      }
    } catch (err) {
      console.error('触发记忆演化失败:', err);
    } finally {
      setIsEvolving(false);
    }
  };

  // 6. 发送对话请求并处理流式响应
  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || isStreaming || !currentSessionID) return;

    const userText = inputValue;
    setInputValue('');
    setPendingInterrupt(null);
    setIsStreaming(true);

    // 追加用户消息到列表
    const userMsg: Message = { role: 'user', content: userText };
    setMessages((prev) => [...prev, userMsg]);

    // 初始化占位 AI 消息
    const initialAiMsg: Message = { role: 'assistant', content: '', reasoning_content: '' };
    setMessages((prev) => [...prev, initialAiMsg]);
    streamingMsgRef.current = initialAiMsg;

    try {
      const res = await fetch(`/api/sessions/${currentSessionID}/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userID, message: userText }),
      });

      if (!res.ok) {
        throw new Error('网络请求异常');
      }

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
      const res = await fetch(`/api/sessions/${currentSessionID}/confirm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userID, confirmed_tool_call_ids: confirmedIDs }),
      });

      if (!res.ok) {
        throw new Error('确认请求失败');
      }

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

  // 8. 核心 SSE 响应读取解析器 (零第三方依赖，解析 POST 流)
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
        updateLastAiMessage((msg) => {
          msg.content += data;
        });
        break;

      case 'tool_call':
        // 可以根据需要把工具调用添加到思考流中渲染展示
        try {
          const toolCall = JSON.parse(data);
          updateLastAiMessage((msg) => {
            msg.reasoning_content = (msg.reasoning_content || '') + `\n[系统] 正在调用工具: ${toolCall.function.name}...`;
          });
        } catch {}
        break;

      case 'tool_result':
        try {
          const toolResult = JSON.parse(data);
          updateLastAiMessage((msg) => {
            msg.reasoning_content = (msg.reasoning_content || '') + `\n[工具回显] 返回值: ${toolResult.content}`;
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
      {/* 1. 左栏：会话列表 */}
      <aside className="sidebar">
        <div className="sidebar-header">
          <div className="logo-icon">🍇</div>
          <h1>Vine-Agent Portal</h1>
        </div>
        <button className="new-chat-btn" onClick={createNewSession}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          新建会话
        </button>
        <div className="session-list">
          {sessions.map((s) => (
            <div
              key={s.id}
              className={`session-item ${currentSessionID === s.id ? 'active' : ''}`}
              onClick={() => selectSession(s.id)}
            >
              <div className="session-name">{s.id}</div>
              <div className="session-meta">
                <span>{new Date(s.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                {s.Status === 'pending_confirmation' && (
                  <span className="status-badge pending">待审批</span>
                )}
              </div>
            </div>
          ))}
        </div>
        <div className="user-footer">
          <div className="user-avatar">U</div>
          <div>
            <div style={{ fontWeight: 600 }}>Test User</div>
            <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>ID: {userID}</div>
          </div>
        </div>
      </aside>

      {/* 2. 中栏：对话核心区 */}
      <main className="chat-area">
        <header className="chat-header">
          <div className="chat-title">
            <h2>
              <span style={{ color: 'var(--accent-purple)' }}>●</span> {currentSessionID || '未选择会话'}
            </h2>
            <div className="chat-desc">
              {isStreaming ? 'AI 正在推理中...' : pendingInterrupt ? '等待敏感操作授权确认' : '就绪'}
            </div>
          </div>
        </header>

        <div className="message-stream">
          {messages.length === 0 ? (
            <div style={{ margin: 'auto', textAlign: 'center', color: 'var(--text-muted)' }}>
              <div style={{ fontSize: '48px', marginBottom: '16px' }}>🍇</div>
              <p style={{ fontSize: '15px', fontWeight: 500, color: 'var(--text-secondary)' }}>开始与智能体进行对话</p>
              <p style={{ fontSize: '12px', marginTop: '6px' }}>系统会在多轮交互后自动提取您的长期偏好与客观事实画像</p>
            </div>
          ) : (
            messages.map((m, idx) => {
              if (m.role === 'system') {
                return (
                  <div key={idx} style={{ textAlign: 'center', margin: '8px 0', fontSize: '12px', color: 'var(--text-secondary)' }}>
                    <span style={{ background: 'rgba(255,255,255,0.03)', padding: '6px 12px', borderRadius: '20px', border: '1px solid rgba(255,255,255,0.05)' }}>
                      {m.content}
                    </span>
                  </div>
                );
              }
              const isUser = m.role === 'user';
              return (
                <div key={idx} className={`message-wrapper ${isUser ? 'user' : 'assistant'}`}>
                  <div className="chat-bubble">
                    {/* 推理思考展示块 */}
                    {!isUser && m.reasoning_content && (
                      <div className="reasoning-container">
                        <div className="reasoning-header">
                          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" style={{ animation: 'spin 4s linear infinite' }}>
                            <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"></path>
                          </svg>
                          思考推理链
                        </div>
                        {m.reasoning_content}
                      </div>
                    )}
                    {/* 主答复文本 */}
                    <div style={{ whiteSpace: 'pre-wrap' }}>
                      {m.content || (!isUser && isStreaming && idx === messages.length - 1 ? (
                        <span className="typing-cursor">思考中...</span>
                      ) : '')}
                    </div>
                  </div>
                </div>
              );
            })
          )}

          {/* 敏感工具确认审批卡片 */}
          {pendingInterrupt && (
            <div className="interrupt-approval-card">
              <div className="interrupt-header">
                <span className="warning-icon">⚠</span>
                <span className="interrupt-title">高危敏感工具审批申请</span>
              </div>
              <p className="interrupt-desc">
                智能体请求执行以下具有修改/清除性质的高危接口动作，根据系统安全规则已被拦截。请对调用请求做出审核决策：
              </p>
              <div className="tools-requested-list">
                {pendingInterrupt.pending_tools.map((tool) => (
                  <div key={tool.id} className="tool-req-item">
                    <div className="tool-req-name">⚙ 工具名称: {tool.function.name}</div>
                    <div className="tool-req-args">
                      <strong>参数明细:</strong>
                      <br />
                      {tool.function.arguments}
                    </div>
                  </div>
                ))}
              </div>
              <div className="approval-btn-group">
                <button className="btn-approve" onClick={handleApproveInterrupt}>
                  同意并授权执行
                </button>
                <button className="btn-reject" onClick={handleRejectInterrupt}>
                  安全拒绝
                </button>
              </div>
            </div>
          )}

          {isStreaming && messages[messages.length - 1]?.role === 'user' && (
            <div className="tool-call-indicator">
              <div className="pulse-dot"></div>
              <span>智能体正在生成分析中...</span>
            </div>
          )}

          <div ref={messageEndRef} />
        </div>

        <form className="input-area" onSubmit={handleSendMessage}>
          <div className="input-container">
            <input
              type="text"
              className="chat-input-box"
              placeholder={pendingInterrupt ? '当前处于等待授权拦截状态，请做出决策' : '输入消息以交互...'}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              disabled={isStreaming || !!pendingInterrupt || !currentSessionID}
            />
            <button
              type="submit"
              className="send-btn"
              disabled={!inputValue.trim() || isStreaming || !!pendingInterrupt || !currentSessionID}
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <line x1="22" y1="2" x2="11" y2="13"></line>
                <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
              </svg>
            </button>
          </div>
        </form>
      </main>

      {/* 3. 右栏：画像长期记忆面板 */}
      <aside className="memory-panel">
        <header className="memory-header">
          <div className="memory-header-title">
            <span style={{ fontSize: '18px' }}>🧠</span>
            <h3>用户长期记忆画像</h3>
          </div>
          <button
            className={`evolve-btn ${isEvolving ? 'spinning' : ''}`}
            onClick={evolveProfile}
            disabled={isEvolving || !currentSessionID}
            title="强制触发增量会话记忆演化"
          >
            <span className="icon">🔄</span>
            {isEvolving ? '演化提炼中...' : 'Sync & Evolve'}
          </button>
        </header>

        <div className="memory-content">
          {/* A. 个人偏好 */}
          <div className="memory-sec preferences">
            <div className="memory-sec-title">
              <span className="icon">⭐</span>
              个人偏好 (Preferences)
            </div>
            {userProfile.preferences.length === 0 ? (
              <div className="empty-state">暂未提炼出明确的用户偏好</div>
            ) : (
              <ul className="memory-list">
                {userProfile.preferences.map((p, i) => (
                  <li key={i} className="memory-item">
                    {p}
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* B. 静态事实 */}
          <div className="memory-sec facts">
            <div className="memory-sec-title">
              <span className="icon">📌</span>
              客观事实 (Facts)
            </div>
            {userProfile.facts.length === 0 ? (
              <div className="empty-state">暂未捕捉到相关的静态事实信息</div>
            ) : (
              <ul className="memory-list">
                {userProfile.facts.map((f, i) => (
                  <li key={i} className="memory-item">
                    {f}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </aside>
    </div>
  );
}
