import React, { useState, useEffect, useRef } from 'react';
import { Message, PendingTool } from '../types';
import MessageList from './chat/MessageList';
import InterruptCard from './chat/InterruptCard';
import InputBar from './chat/InputBar';

interface ChatAreaProps {
  messages: Message[];
  currentSessionID: string;
  currentSessionName?: string;
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
  username?: string;
  onShowTooltip: (text: string, e: React.MouseEvent) => void;
  onMoveTooltip: (e: React.MouseEvent) => void;
  onHideTooltip: () => void;
}

export default function ChatArea({
  messages,
  currentSessionID,
  currentSessionName,
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
  username,
  onShowTooltip,
  onMoveTooltip,
  onHideTooltip,
}: ChatAreaProps) {
  const [inputValue, setInputValue] = useState<string>('');
  const messageEndRef = useRef<HTMLDivElement | null>(null);
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const inputBarRef = useRef<HTMLDivElement | null>(null);
  const [showScrollTop, setShowScrollTop] = useState(false);
  const [showScrollBottom, setShowScrollBottom] = useState(false);
  const [inputBarHeight, setInputBarHeight] = useState<number>(106);

  const handleScroll = () => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    setShowScrollTop(scrollTop > 200);
    setShowScrollBottom(scrollHeight - (scrollTop + clientHeight) > 200);
  };
  const scrollToTop = () => {
    scrollContainerRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const scrollToBottom = () => {
    const container = scrollContainerRef.current;
    if (container) {
      container.scrollTo({ top: container.scrollHeight, behavior: 'smooth' });
    }
  };
  // 消息更新后滚动到底部
  useEffect(() => {
    messageEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    setTimeout(handleScroll, 100);
  }, [messages, isStreaming, pendingInterrupt]);

  // 监听输入区高度变化以防在小屏幕或多行输入时发生组件重叠
  useEffect(() => {
    const el = inputBarRef.current;
    if (!el) return;
    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setInputBarHeight(entry.target.clientHeight);
      }
    });
    resizeObserver.observe(el);
    return () => resizeObserver.disconnect();
  }, []);

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

  // 检查当前消息流中是否有任何 reasoning 思考块处于展开状态
  const hasAnyExpanded = messages.some((m, idx) => {
    const hasThinking = (m.timeline && m.timeline.some(t => t.kind === 'reasoning')) || m.reasoning_content;
    return hasThinking && expandedReasoning[idx] !== false;
  });

  // 一键折叠或展开所有 AI 思考过程
  const handleToggleAllThinking = () => {
    const newExpanded: Record<number, boolean> = {};
    messages.forEach((_, idx) => {
      newExpanded[idx] = !hasAnyExpanded;
    });
    setExpandedReasoning(newExpanded);
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
        <div className="chat-title" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {/* 精致的淡色会话图标 */}
          <svg
            viewBox="0 0 24 24"
            style={{
              width: '15px',
              height: '15px',
              fill: 'none',
              stroke: 'var(--text-muted)',
              strokeWidth: 2,
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              opacity: 0.8,
              flexShrink: 0,
            }}
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
          </svg>

          <div style={{ display: 'flex', alignItems: 'baseline', gap: '6px', minWidth: 0 }}>
            {/* 会话 Name 标题 */}
            <h2
              style={{
                fontSize: '13.5px',
                fontWeight: 600,
                color: 'var(--text-main)',
                margin: 0,
                lineHeight: 1.2,
                letterSpacing: '-0.1px',
                fontFamily: 'var(--font-sans)',
                display: 'inline-block',
                textOverflow: 'ellipsis',
                overflow: 'hidden',
                whiteSpace: 'nowrap',
                maxWidth: '180px',
                verticalAlign: 'bottom',
                minWidth: 0
              }}
              onMouseEnter={(e) => {
                const isTruncated = e.currentTarget.scrollWidth > e.currentTarget.clientWidth;
                if (isTruncated) {
                  onShowTooltip(currentSessionName || currentSessionID || 'No active session', e);
                }
              }}
              onMouseMove={onMoveTooltip}
              onMouseLeave={onHideTooltip}
            >
              {currentSessionName || currentSessionID || 'No active session'}
            </h2>

            {/* 如果有 Name 且与 ID 不同，我们不突出地展示 ID */}
            {currentSessionName && currentSessionName !== currentSessionID && (
              <span
                style={{
                  fontSize: '10.5px',
                  color: 'var(--text-muted)',
                  fontWeight: 400,
                  fontFamily: 'var(--font-mono, monospace)',
                  opacity: 0.6,
                  display: 'inline-block',
                  textOverflow: 'ellipsis',
                  overflow: 'hidden',
                  whiteSpace: 'nowrap',
                  maxWidth: '100px',
                  verticalAlign: 'bottom',
                  minWidth: 0
                }}
                onMouseEnter={(e) => {
                  const isTruncated = e.currentTarget.scrollWidth > e.currentTarget.clientWidth;
                  if (isTruncated) {
                    onShowTooltip(currentSessionID, e);
                  }
                }}
                onMouseMove={onMoveTooltip}
                onMouseLeave={onHideTooltip}
              >
                {currentSessionID}
              </span>
            )}
          </div>

          {/* 面包屑斜杠分隔符 */}
          <span style={{ fontSize: '11px', color: 'var(--border-color)', margin: '0 2px', userSelect: 'none' }}>/</span>

          {/* 精致扁平状态徽标 */}
          {renderStatusBadge()}
        </div>
        <div className="chat-header-actions">
          {/* 全局折叠所有思考的智能按钮 */}
          {messages.length > 0 && messages.some(m => (m.timeline && m.timeline.some(t => t.kind === 'reasoning')) || m.reasoning_content) && (
            <button
              className="toggle-thinking-btn"
              onClick={handleToggleAllThinking}
              title={hasAnyExpanded ? '收起所有思考过程' : '展开所有思考过程'}
            >
              {hasAnyExpanded ? (
                /* 展开中，显示向上收起的双折线 Chevron */
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="17 11 12 6 7 11" />
                  <polyline points="17 18 12 13 7 18" />
                </svg>
              ) : (
                /* 折叠中，显示向下展开的双折线 Chevron */
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="7 13 12 18 17 13" />
                  <polyline points="7 6 12 11 17 6" />
                </svg>
              )}
            </button>
          )}

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
        </div>
      </header>

      <div className="message-stream" ref={scrollContainerRef} onScroll={handleScroll}>
        <MessageList
          messages={messages}
          isStreaming={isStreaming}
          expandedReasoning={expandedReasoning}
          setExpandedReasoning={setExpandedReasoning}
          onQuickAction={handleQuickAction}
          username={username}
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

      {/* 滚动浮动控制组：高度自适应计算定位，防组件重叠 */}
      <div className="scroll-float-group" style={{ bottom: `${inputBarHeight + 12}px` }}>
        <button
          className={`scroll-float-btn ${showScrollTop ? 'visible' : ''}`}
          onClick={scrollToTop}
          title="回到顶部"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="18 15 12 9 6 15" />
          </svg>
        </button>
        <button
          className={`scroll-float-btn ${showScrollBottom ? 'visible' : ''}`}
          onClick={scrollToBottom}
          title="回到底部"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </button>
      </div>

      {/* 使用普通的 div 包裹 InputBar，用以被 ResizeObserver 实时监控其高度 */}
      <div ref={inputBarRef} style={{ width: '100%' }}>
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
      </div>
    </main>
  );
}
