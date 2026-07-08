import React, { useState, useEffect, useRef } from 'react';
import { Message, PendingTool } from '../types';
import MessageList from './chat/MessageList';
import InterruptCard from './chat/InterruptCard';
import InputBar from './chat/InputBar';

interface ChatAreaProps {
  messages: Message[];
  currentSessionID: string;
  isStreaming: boolean;
  pendingInterrupt: {
    session_id: string;
    pending_tools: PendingTool[];
  } | null;
  selectedModel: string;
  expandedReasoning: Record<number, boolean>;
  setExpandedReasoning: React.Dispatch<React.SetStateAction<Record<number, boolean>>>;
  onSendMessage: (text: string) => void;
  onApproveInterrupt: () => void;
  onRejectInterrupt: () => void;
  onCancelChat: () => void;
  setSelectedModel: (model: string) => void;
  isMemoryCollapsed: boolean;
  setIsMemoryCollapsed: (collapsed: boolean) => void;
}

export default function ChatArea({
  messages,
  currentSessionID,
  isStreaming,
  pendingInterrupt,
  selectedModel,
  expandedReasoning,
  setExpandedReasoning,
  onSendMessage,
  onApproveInterrupt,
  onRejectInterrupt,
  onCancelChat,
  setSelectedModel,
  isMemoryCollapsed,
  setIsMemoryCollapsed,
}: ChatAreaProps) {
  const [inputValue, setInputValue] = useState<string>('');
  const messageEndRef = useRef<HTMLDivElement | null>(null);

  // 消息更新后滚动到底部
  useEffect(() => {
    messageEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isStreaming, pendingInterrupt]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || isStreaming || !currentSessionID) return;
    onSendMessage(inputValue);
    setInputValue('');
  };

  const handleQuickAction = (text: string) => {
    if (pendingInterrupt || isStreaming || !currentSessionID) return;
    setInputValue(text);
  };

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

  return (
    <main className="chat-area">
      <header className="chat-header">
        <div className="chat-title" style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          {/* 顶栏左侧会话 (Session) 含义的对话气泡图标 */}
          <svg
            viewBox="0 0 24 24"
            style={{
              width: '18px',
              height: '18px',
              fill: 'none',
              stroke: 'var(--primary-color)',
              strokeWidth: 2,
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              flexShrink: 0,
            }}
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
          </svg>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
            <h2
              style={{
                fontSize: '14.5px',
                fontWeight: 600,
                color: 'var(--text-main)',
                margin: 0,
                lineHeight: 1,
              }}
            >
              {currentSessionID || 'No active session'}
            </h2>
            {renderStatusBadge()}
          </div>
        </div>
        <button
          className="toggle-memory-btn"
          onClick={() => setIsMemoryCollapsed(!isMemoryCollapsed)}
          title={isMemoryCollapsed ? '展开记忆面板' : '折叠记忆面板'}
        >
          {isMemoryCollapsed ? (
            /* 折叠状态：显示"面板隐藏"标识 — 两条竖线 + 左展开箭头 */
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <rect x="3" y="3" width="18" height="18" rx="2" />
              <line x1="15" y1="3" x2="15" y2="21" />
              <polyline points="11 9 8 12 11 15" />
            </svg>
          ) : (
            /* 展开状态：显示"面板可见"标识 — 两条竖线 + 右折叠箭头 */
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <rect x="3" y="3" width="18" height="18" rx="2" />
              <line x1="15" y1="3" x2="15" y2="21" />
              <polyline points="13 9 16 12 13 15" />
            </svg>
          )}
        </button>
      </header>

      <div className="message-stream">
        <MessageList
          messages={messages}
          isStreaming={isStreaming}
          expandedReasoning={expandedReasoning}
          setExpandedReasoning={setExpandedReasoning}
          onQuickAction={handleQuickAction}
        />

        {/* 敏感工具确认审批卡片 */}
        {pendingInterrupt && (
          <InterruptCard
            pendingInterrupt={pendingInterrupt}
            onApproveInterrupt={onApproveInterrupt}
            onRejectInterrupt={onRejectInterrupt}
          />
        )}

        {/* 等待 AI 响应的加载气泡（最后一条消息是用户消息时展示） */}
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

      <InputBar
        inputValue={inputValue}
        setInputValue={setInputValue}
        selectedModel={selectedModel}
        setSelectedModel={setSelectedModel}
        isStreaming={isStreaming}
        pendingInterrupt={pendingInterrupt}
        currentSessionID={currentSessionID}
        onSubmit={handleSubmit}
        onCancelChat={onCancelChat}
      />
    </main>
  );
}
