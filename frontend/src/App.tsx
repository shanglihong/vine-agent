import React, { useState, useEffect, useRef } from 'react';
import { marked } from 'marked';

// 配置 marked，使其支持单换行解析为 <br> 标签
marked.setOptions({
  breaks: true,
});

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
  const [userInfo, setUserInfo] = useState<{ id: string; username: string; email: string } | null>(null);
  const [userID, setUserID] = useState<string>('');
  const [userProfile, setUserProfile] = useState<Profile>({
    user_id: '',
    preferences: [],
    facts: [],
  });
  const [inputValue, setInputValue] = useState<string>('');
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



  // 强视觉会话状态指示 Badge 标签 (带 LED 脉冲呼吸指示点)
  const renderStatusBadge = () => {
    if (isStreaming) {
      return (
        <span className="status-badge-header thinking">
          <span className="status-dot"></span>
          Responding
        </span>
      );
    }
    if (pendingInterrupt) {
      return (
        <span className="status-badge-header pending">
          <span className="status-dot"></span>
          Pending Approval
        </span>
      );
    }
    return (
      <span className="status-badge-header ready">
        <span className="status-dot"></span>
        Ready
      </span>
    );
  };

  // 用于在流式对话中动态更新的消息缓冲区
  const streamingMsgRef = useRef<Message | null>(null);
  const messageEndRef = useRef<HTMLDivElement | null>(null);

  // 初始化加载及主题检测
  useEffect(() => {
    const fetchUser = async (retries = 5, delay = 1500) => {
      try {
        const res = await fetch('/api/user');
        if (res.ok) {
          const u = await res.json();
          if (u && u.id) {
            setUserInfo(u);
            setUserID(u.id);
            return;
          }
        }
        throw new Error(`status: ${res.status}`);
      } catch (err) {
        if (retries > 0) {
          console.warn(`加载用户信息失败，将在 ${delay}ms 后重试... 剩余重试次数: ${retries}`);
          setTimeout(() => fetchUser(retries - 1, delay), delay);
        } else {
          console.error('加载用户信息失败，已达到最大重试次数:', err);
        }
      }
    };

    fetchUser();

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

  // 消息更新后滚动到底部
  useEffect(() => {
    messageEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isStreaming, pendingInterrupt]);

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
    setExpandedReasoning({}); // 重置折叠状态
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
    if (!userID) return;
    try {
      const res = await fetch(`/api/users/${userID}/profile`);
      if (res.ok) {
        const data = await res.json();
        setUserProfile({
          user_id: data.user_id,
          preferences: data.preferences || [],
          facts: data.facts || [],
        });
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
        setUserProfile({
          user_id: data.user_id,
          preferences: data.preferences || [],
          facts: data.facts || [],
        });
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
        body: JSON.stringify({ user_id: userID, message: userText, model: selectedModel }),
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
          msg.reasoning_content = (msg.reasoning_content || '') + rText;
        });
        break;

      case 'tool_call':
        try {
          const toolCall = JSON.parse(data);
          updateLastAiMessage((msg) => {
            msg.reasoning_content = (msg.reasoning_content || '') + `\n[系统] 正在调用工具: ${toolCall.function.name}...`;
          });
        } catch { }
        break;

      case 'tool_result':
        try {
          const toolResult = JSON.parse(data);
          updateLastAiMessage((msg) => {
            msg.reasoning_content = (msg.reasoning_content || '') + `\n[工具回显] 返回值: ${toolResult.content}`;
          });
        } catch { }
        break;

      case 'interrupt':
        try {
          const interruptData = JSON.parse(data);
          setPendingInterrupt(interruptData);
          setIsStreaming(false);
          loadSessions(); // 刷新 sidebar 中会话的 pending 状态
        } catch { }
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
        } catch { }
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

  // 快捷卡片填充并聚焦
  const handleQuickAction = (text: string) => {
    if (pendingInterrupt || isStreaming || !currentSessionID) return;
    setInputValue(text);
  };

  return (
    <div className="portal-container">
      {/* 1. 左栏：会话历史列表 (Sidebar) */}
      <aside className="sidebar">
        <div className="sidebar-header">
          <div className="logo-container">
            {/* 符合 Vine (葡萄藤蔓) 科技拓扑网格风格的 LOGO */}
            <svg viewBox="0 0 24 24" className="logo-svg" style={{ fill: 'none', stroke: 'var(--primary-color)', strokeWidth: 1.8, strokeLinecap: 'round', strokeLinejoin: 'round' }} xmlns="http://www.w3.org/2000/svg">
              <line x1="8" y1="8" x2="12" y2="7" />
              <line x1="12" y1="7" x2="16" y2="8" />
              <line x1="8" y1="8" x2="10" y2="12" />
              <line x1="12" y1="7" x2="10" y2="12" />
              <line x1="12" y1="7" x2="14" y2="12" />
              <line x1="16" y1="8" x2="14" y2="12" />
              <line x1="10" y1="12" x2="14" y2="12" />
              <line x1="10" y1="12" x2="12" y2="16" />
              <line x1="14" y1="12" x2="12" y2="16" />
              <circle cx="8" cy="8" r="2" fill="var(--primary-color)" />
              <circle cx="12" cy="7" r="2" fill="var(--primary-color)" />
              <circle cx="16" cy="8" r="2" fill="var(--primary-color)" />
              <circle cx="10" cy="12" r="2" fill="var(--primary-color)" />
              <circle cx="14" cy="12" r="2" fill="var(--primary-color)" />
              <circle cx="12" cy="16" r="2" fill="var(--primary-color)" />
              <path d="M12 4.5V2.5c0-.5.4-.8.8-.8h1.2" />
            </svg>
            <h1>Vine-Agent</h1>
          </div>
        </div>

        {/* 新对话按钮 */}
        <button className="new-chat-btn" onClick={createNewSession}>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          New chat
        </button>

        <div className="session-list">
          {sessions.map((s) => (
            <button
              key={s.id}
              className={`session-item ${currentSessionID === s.id ? 'active' : ''}`}
              onClick={() => selectSession(s.id)}
            >
              <div className="session-name">{s.id}</div>
              <div className="session-meta">
                <span>{new Date(s.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                {s.id === currentSessionID && isStreaming && (
                  <svg className="spin-svg" viewBox="0 0 24 24" style={{ width: '10px', height: '10px', stroke: 'var(--primary-color)', strokeWidth: 3, fill: 'none', strokeLinecap: 'round', marginLeft: '6px', animation: 'spin 1.2s linear infinite', flexShrink: 0 }} xmlns="http://www.w3.org/2000/svg">
                    <circle cx="12" cy="12" r="10" strokeDasharray="30 12" />
                  </svg>
                )}
                {(s.status === 'pending_confirmation' || (s.id === currentSessionID && pendingInterrupt)) && (
                  <span className="status-badge pending" style={{ marginLeft: '6px', fontSize: '9px', padding: '1.5px 5px', lineHeight: 1 }}>PENDING</span>
                )}
              </div>
            </button>
          ))}
        </div>

        <div className="user-footer">
          <div className="user-avatar">
            {userInfo?.username ? userInfo.username[0].toUpperCase() : 'U'}
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 600, fontSize: '12px' }}>
              {userInfo?.username || userID || 'Loading...'}
            </div>
            <div style={{ fontSize: '10px', color: 'var(--text-secondary)' }}>Status: Active</div>
          </div>
          {/* 夜间模式切换按钮 */}
          <button
            onClick={toggleTheme}
            className="theme-toggle-btn"
            title={isDarkMode ? '切换至明亮模式' : '切换至暗色模式'}
          >
            {isDarkMode ? (
              <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="5"></circle>
                <line x1="12" y1="1" x2="12" y2="3"></line>
                <line x1="12" y1="21" x2="12" y2="23"></line>
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
                <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
                <line x1="1" y1="12" x2="3" y2="12"></line>
                <line x1="21" y1="12" x2="23" y2="12"></line>
                <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
                <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
              </svg>
            )}
          </button>
        </div>
      </aside>

      {/* 2. 中栏：对话核心区 */}
      <main className="chat-area">
        <header className="chat-header">
          <div className="chat-title" style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            {/* 顶栏左侧会话 (Session) 含义的对话气泡图标 */}
            <svg viewBox="0 0 24 24" style={{ width: '18px', height: '18px', fill: 'none', stroke: 'var(--primary-color)', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round', flexShrink: 0 }} xmlns="http://www.w3.org/2000/svg">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
            </svg>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
              <h2 style={{ fontSize: '14.5px', fontWeight: 600, color: 'var(--text-main)', margin: 0, lineHeight: 1 }}>
                {currentSessionID || 'No active session'}
              </h2>
              {renderStatusBadge()}
            </div>
          </div>
          <button 
            className="toggle-memory-btn" 
            onClick={() => setIsMemoryCollapsed(!isMemoryCollapsed)}
            title={isMemoryCollapsed ? "展开记忆面板" : "折叠记忆面板"}
          >
            {isMemoryCollapsed ? (
              /* 折叠状态：显示“面板隐藏”标识 — 两条竖线 + 左展开箭头 */
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="3" width="18" height="18" rx="2" />
                <line x1="15" y1="3" x2="15" y2="21" />
                <polyline points="11 9 8 12 11 15" />
              </svg>
            ) : (
              /* 展开状态：显示“面板可见”标识 — 两条竖线 + 右折叠箭头 */
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="3" width="18" height="18" rx="2" />
                <line x1="15" y1="3" x2="15" y2="21" />
                <polyline points="13 9 16 12 13 15" />
              </svg>
            )}
          </button>
        </header>

        <div className="message-stream">
          {messages.length === 0 ? (
            <div className="empty-state-thesis">
              <div className="empty-logo-container" style={{ border: 'none', background: 'transparent', boxShadow: 'none', width: 'auto', height: 'auto', marginBottom: '16px' }}>
                {/* 首页大图标：符合 Vine 的科技风格葡萄图标 */}
                <svg viewBox="0 0 24 24" style={{ width: '42px', height: '42px', fill: 'none', stroke: 'var(--primary-color)', strokeWidth: 1.8, strokeLinecap: 'round', strokeLinejoin: 'round' }} xmlns="http://www.w3.org/2000/svg">
                  <line x1="8" y1="8" x2="12" y2="7" />
                  <line x1="12" y1="7" x2="16" y2="8" />
                  <line x1="8" y1="8" x2="10" y2="12" />
                  <line x1="12" y1="7" x2="10" y2="12" />
                  <line x1="12" y1="7" x2="14" y2="12" />
                  <line x1="16" y1="8" x2="14" y2="12" />
                  <line x1="10" y1="12" x2="14" y2="12" />
                  <line x1="10" y1="12" x2="12" y2="16" />
                  <line x1="14" y1="12" x2="12" y2="16" />
                  <circle cx="8" cy="8" r="2" fill="var(--primary-color)" />
                  <circle cx="12" cy="7" r="2" fill="var(--primary-color)" />
                  <circle cx="16" cy="8" r="2" fill="var(--primary-color)" />
                  <circle cx="10" cy="12" r="2" fill="var(--primary-color)" />
                  <circle cx="14" cy="12" r="2" fill="var(--primary-color)" />
                  <circle cx="12" cy="16" r="2" fill="var(--primary-color)" />
                  <path d="M12 4.5V2.5c0-.5.4-.8.8-.8h1.2" />
                </svg>
              </div>
              <h3>How can I help you today?</h3>

              {/* ChatGPT 风格快捷动作网格 */}
              <div className="quick-action-cards">
                <div className="action-card" onClick={() => handleQuickAction('分析我近期的偏好有哪些新的进化？')}>
                  <div className="action-card-title">🔍 分析近期偏好</div>
                  <div className="action-card-desc">探索在近期对话流中，系统自动归纳的最新行为特征。</div>
                </div>
                <div className="action-card" onClick={() => handleQuickAction('检查当前会话中是否有挂起的敏感工具调用？')}>
                  <div className="action-card-title">🛡 安全审查拦截</div>
                  <div className="action-card-desc">检测是否存在挂起、需要人工干预审批的敏感工具执行。</div>
                </div>
                <div className="action-card" onClick={() => handleQuickAction('梳理一下你目前记录关于我的客观事实有哪些？')}>
                  <div className="action-card-title">📂 整理长期记忆</div>
                  <div className="action-card-desc">打印已存入数据库的客观事实事实画像列表。</div>
                </div>
                <div className="action-card" onClick={() => handleQuickAction('开始一个新的系统测试，帮我调用几个工具试试。')}>
                  <div className="action-card-title">⚙️ 发起工具测试</div>
                  <div className="action-card-desc">输入测试命令，让智能体尝试调用底层预置的服务。</div>
                </div>
              </div>
            </div>
          ) : (
            messages.map((m, idx) => {
              if (m.role === 'system') {
                return (
                  <div key={idx} style={{ textAlign: 'center', margin: '12px 0', fontSize: '12px', color: 'var(--text-secondary)' }}>
                    <span style={{ padding: '4px 12px', borderRadius: '12px', background: '#f4f4f5', border: '1px solid var(--border-color)', fontSize: '11px', display: 'inline-block' }}>
                      System Message: {m.content}
                    </span>
                  </div>
                );
              }
              const isUser = m.role === 'user';
              // 是否展开推理日志，默认展开。若在该 map 节点被手动置为 false，则折叠。
              const isReasoningExpanded = expandedReasoning[idx] !== false;

              return (
                <div key={idx} className={`message-wrapper ${isUser ? 'user' : 'assistant'}`}>
                  {/* 对话头像：USER 和 AI 均在左侧完美对齐呈现 */}
                  <div className="message-avatar" style={{
                    background: isUser ? (isDarkMode ? '#334155' : '#e2e8f0') : 'var(--ai-accent)',
                    color: isUser ? (isDarkMode ? '#cbd5e1' : '#475569') : '#ffffff',
                    borderColor: isUser ? 'var(--border-color)' : 'transparent',
                    fontSize: '12px',
                    fontWeight: 600
                  }}>
                    {isUser ? (
                      'U'
                    ) : (
                      // AI 专用 DeepSeek 星体神经网络蓝标
                      <svg viewBox="0 0 24 24" style={{ fill: '#ffffff', stroke: 'none' }} xmlns="http://www.w3.org/2000/svg">
                        <path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8 8 0 0 1-8 8z" />
                        <circle cx="12" cy="12" r="2.5" />
                        <circle cx="12" cy="7" r="1.5" />
                        <circle cx="12" cy="17" r="1.5" />
                        <circle cx="7" cy="12" r="1.5" />
                        <circle cx="17" cy="12" r="1.5" />
                      </svg>
                    )}
                  </div>

                  <div className="chat-bubble">
                    {/* 推理思考展示块 (DeepSeek Accordion 风格) */}
                    {!isUser && m.reasoning_content && (
                      <div className="reasoning-accordion">
                        <div
                          className={`reasoning-toggle ${isReasoningExpanded ? 'open' : ''}`}
                          onClick={() => {
                            setExpandedReasoning(prev => ({
                              ...prev,
                              [idx]: !isReasoningExpanded
                            }));
                          }}
                        >
                          {/* 旋转的小箭头 */}
                          <svg className="reasoning-toggle-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                            <polyline points="9 18 15 12 9 6"></polyline>
                          </svg>
                          <span>思考过程</span>
                        </div>
                        {isReasoningExpanded && (
                          <div className="reasoning-content">
                            {m.reasoning_content}
                          </div>
                        )}
                      </div>
                    )}
                    {/* 主答复文本 */}
                    <div className={isUser ? "message-content-user" : "markdown-body"}>
                      {m.content ? (
                        isUser ? (
                          <div style={{ whiteSpace: 'pre-wrap' }}>{m.content}</div>
                        ) : (
                          <div dangerouslySetInnerHTML={{ __html: marked.parse(m.content) as string }} />
                        )
                      ) : (
                        !isUser && isStreaming && idx === messages.length - 1 ? (
                          <span className="typing-cursor">正在思考...</span>
                        ) : ''
                      )}
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
                <svg className="warning-icon-svg" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" />
                </svg>
                <span className="interrupt-title">安全授权确认: 检测到敏感工具调用</span>
              </div>
              <p className="interrupt-desc">
                系统检测到智能体发出包含敏感修改或删除动作的工具调用请求。该执行链已自动拦截并挂起，请审查工具参数以授权批准：
              </p>
              <div className="tools-requested-list">
                {pendingInterrupt.pending_tools.map((tool) => (
                  <div key={tool.id} className="tool-req-item">
                    <div className="tool-req-name">🔧 拟调用工具: {tool.function.name}</div>
                    <div className="tool-req-args">
                      <strong>参数明细 (JSON):</strong>
                      <br />
                      {tool.function.arguments}
                    </div>
                  </div>
                ))}
              </div>
              <div className="approval-btn-group">
                <button className="btn-approve" onClick={handleApproveInterrupt}>
                  授权执行
                </button>
                <button className="btn-reject" onClick={handleRejectInterrupt}>
                  拒绝操作
                </button>
              </div>
            </div>
          )}

          {isStreaming && messages[messages.length - 1]?.role === 'user' && (
            <div className="message-wrapper assistant">
              <div className="message-avatar" style={{ background: 'var(--ai-accent)' }}>
                <svg viewBox="0 0 24 24" style={{ fill: '#ffffff' }} xmlns="http://www.w3.org/2000/svg">
                  <path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8 8 0 0 1-8 8z" />
                  <circle cx="12" cy="12" r="2.5" />
                  <circle cx="12" cy="7" r="1.5" />
                  <circle cx="12" cy="17" r="1.5" />
                  <circle cx="7" cy="12" r="1.5" />
                  <circle cx="17" cy="12" r="1.5" />
                </svg>
              </div>
              <div className="chat-bubble">
                <div className="tool-call-indicator">
                  <div className="pulse-dot"></div>
                  <span>正在提炼和计算智能体响应...</span>
                </div>
              </div>
            </div>
          )}

          <div ref={messageEndRef} />
        </div>

        {/* 底部输入框区域 */}
        <form className="input-area" onSubmit={handleSendMessage}>
          <div className="input-container">
            <div className="model-selector-pill">
              <select
                value={selectedModel}
                onChange={(e) => setSelectedModel(e.target.value)}
                style={{
                  background: 'transparent',
                  border: 'none',
                  outline: 'none',
                  fontSize: 'inherit',
                  fontWeight: 'inherit',
                  color: 'inherit',
                  cursor: 'pointer',
                  padding: 0,
                  margin: 0
                }}
              >
                <option value="deepseek-v4-flash" style={{ color: 'var(--text-main)', background: 'var(--bg-card)' }}>deepseek-v4-flash</option>
                <option value="deepseek-v4-pro" style={{ color: 'var(--text-main)', background: 'var(--bg-card)' }}>deepseek-v4-pro</option>
              </select>
            </div>
            <input
              type="text"
              className="chat-input-box"
              placeholder={pendingInterrupt ? '当前会话已挂起，请审查上面的敏感操作安全卡片。' : '给 Vine-Agent 发送消息...'}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              disabled={isStreaming || !!pendingInterrupt || !currentSessionID}
            />
            <button
              type="submit"
              className="send-btn"
              style={{ background: inputValue.trim() ? 'var(--ai-accent)' : '#e5e7eb' }}
              disabled={!inputValue.trim() || isStreaming || !!pendingInterrupt || !currentSessionID}
            >
              {/* 向上发送箭头 */}
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <line x1="12" y1="19" x2="12" y2="5"></line>
                <polyline points="5 12 12 5 19 12"></polyline>
              </svg>
            </button>
          </div>
        </form>
      </main>

      {/* 3. 右栏：画像长期记忆面板 */}
      <aside className={`memory-panel ${isMemoryCollapsed ? 'collapsed' : ''}`}>
        <header className="memory-header">
          <div className="memory-header-title">
            {/* Memory Vineyard 的 Header 图标 - 采用 Vine 科技葡萄图标 */}
            <svg viewBox="0 0 24 24" style={{ width: '16px', height: '16px', fill: 'none', stroke: 'var(--primary-color)', strokeWidth: 1.8, strokeLinecap: 'round', strokeLinejoin: 'round', marginRight: '8px' }} xmlns="http://www.w3.org/2000/svg">
              <path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96-.44 2.5 2.5 0 0 1 0-3.12 3 3 0 0 1 0-3.88 2.5 2.5 0 0 1 0-3.12A2.5 2.5 0 0 1 9.5 2zM14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96-.44 2.5 2.5 0 0 0 0-3.12 3 3 0 0 0 0-3.88 2.5 2.5 0 0 0 0-3.12A2.5 2.5 0 0 0 14.5 2z" />
              <path d="M12 5h1M12 9h2M12 13h1M12 17h2M12 7h-1M12 11h-2M12 15h-1M12 19h-2" />
            </svg>
            <h3>Memory Vineyard</h3>
          </div>
          <button
            className={`evolve-btn ${isEvolving ? 'spinning' : ''}`}
            onClick={evolveProfile}
            disabled={isEvolving || !currentSessionID}
            title="手动归纳并提炼对话中的长期画像"
          >
            {isEvolving ? 'Distilling...' : 'Distill'}
          </button>
        </header>

        <div className="memory-content">
          {/* A. 个人偏好 */}
          <div className="memory-sec preferences">
            <div className="memory-sec-title">
              <svg className="memory-sec-title-svg" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
              </svg>
              User Preferences
            </div>
            {userProfile.preferences.length === 0 ? (
              <div className="empty-state">No preferences distilled yet.</div>
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

          {/* B. 客观事实 */}
          <div className="memory-sec facts">
            <div className="memory-sec-title">
              <svg className="memory-sec-title-svg" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M16 12V4h1V2H7v2h1v8l-2 2v2h5.2v6h1.6v-6H18v-2l-2-2z" />
              </svg>
              Objective Facts
            </div>
            {userProfile.facts.length === 0 ? (
              <div className="empty-state">No factual memory nodes recorded.</div>
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
