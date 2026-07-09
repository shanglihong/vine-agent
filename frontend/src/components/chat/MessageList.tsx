import React from 'react';
import { marked } from 'marked';
import { Message, TimelineItem } from '../../types';

// 配置 marked，使其支持单换行解析为 <br> 标签
marked.setOptions({
  breaks: true,
});

// 专属工具图标映射
const getToolIcon = (toolName: string) => {
  const name = toolName.toLowerCase();
  if (name.includes('weather')) {
    return (
      <svg className="tool-icon tool-icon-weather" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: '13.5px', height: '13.5px', stroke: 'var(--warning-color)', flexShrink: 0 }}>
        <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
        <circle cx="12" cy="12" r="4" />
      </svg>
    );
  }
  if (name.includes('city') || name.includes('location') || name.includes('geo')) {
    return (
      <svg className="tool-icon tool-icon-location" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: '13.5px', height: '13.5px', stroke: 'var(--primary-color)', flexShrink: 0 }}>
        <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z" />
        <circle cx="12" cy="10" r="3" />
      </svg>
    );
  }
  if (name.includes('memory') || name.includes('profile') || name.includes('fact')) {
    return (
      <svg className="tool-icon tool-icon-memory" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: '13.5px', height: '13.5px', stroke: 'var(--success-color)', flexShrink: 0 }}>
        <path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96-.44 2.5 2.5 0 0 1 0-3.12 3 3 0 0 1 0-3.88 2.5 2.5 0 0 1 0-3.12A2.5 2.5 0 0 1 9.5 2zM14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96-.44 2.5 2.5 0 0 0 0-3.12 3 3 0 0 0 0-3.88 2.5 2.5 0 0 0 0-3.12A2.5 2.5 0 0 0 14.5 2z" />
      </svg>
    );
  }
  if (name.includes('search')) {
    return (
      <svg className="tool-icon tool-icon-search" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: '13.5px', height: '13.5px', stroke: 'var(--primary-color)', flexShrink: 0 }}>
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
      </svg>
    );
  }
  if (name.includes('crawl') || name.includes('webpage') || name.includes('fetch')) {
    return (
      <svg className="tool-icon tool-icon-webpage" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: '13.5px', height: '13.5px', stroke: '#a855f7', flexShrink: 0 }}>
        <circle cx="12" cy="12" r="10" />
        <line x1="2" y1="12" x2="22" y2="12" />
        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
      </svg>
    );
  }
  return (
    <svg className="tool-icon tool-icon-default" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: '13.5px', height: '13.5px', stroke: 'var(--text-secondary)', flexShrink: 0 }}>
      <polyline points="16 18 22 12 16 6" />
      <polyline points="8 6 2 12 8 18" />
      <line x1="12" y1="4" x2="12" y2="20" />
    </svg>
  );
};

interface MessageListProps {
  messages: Message[];
  isStreaming: boolean;
  expandedReasoning: Record<number, boolean>;
  setExpandedReasoning: React.Dispatch<React.SetStateAction<Record<number, boolean>>>;
  onQuickAction: (text: string) => void;
  username?: string;
}

export default function MessageList({
  messages,
  isStreaming,
  expandedReasoning,
  setExpandedReasoning,
  onQuickAction,
  username,
}: MessageListProps) {
  if (messages.length === 0) {
    return (
      <div className="empty-state-thesis">
        <div
          className="empty-logo-container"
          style={{
            border: 'none',
            background: 'transparent',
            boxShadow: 'none',
            width: 'auto',
            height: 'auto',
            marginBottom: '16px',
          }}
        >
          {/* 首页大图标：符合 Vine 的科技风格葡萄图标 */}
          <svg
            viewBox="0 0 24 24"
            style={{
              width: '42px',
              height: '42px',
              fill: 'none',
              stroke: 'var(--primary-color)',
              strokeWidth: 1.8,
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
            }}
            xmlns="http://www.w3.org/2000/svg"
          >
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
          <div
            className="action-card"
            onClick={() => onQuickAction('分析我近期的偏好有哪些新的进化？')}
          >
            <div className="action-card-title">🔍 分析近期偏好</div>
            <div className="action-card-desc">探索在近期对话流中，系统自动归纳的最新行为特征。</div>
          </div>
          <div
            className="action-card"
            onClick={() => onQuickAction('检查当前会话中是否有挂起的敏感工具调用？')}
          >
            <div className="action-card-title">🛡 安全审查拦截</div>
            <div className="action-card-desc">检测是否存在挂起、需要人工干预审批的敏感工具执行。</div>
          </div>
          <div
            className="action-card"
            onClick={() => onQuickAction('梳理一下你目前记录关于我的客观事实有哪些？')}
          >
            <div className="action-card-title">📂 整理长期记忆</div>
            <div className="action-card-desc">打印已存入数据库的客观事实事实画像列表。</div>
          </div>
          <div
            className="action-card"
            onClick={() => onQuickAction('开始一个新的系统测试，帮我调用几个工具试试。')}
          >
            <div className="action-card-title">⚙️ 发起工具测试</div>
            <div className="action-card-desc">输入测试命令，让智能体尝试调用底层预置的服务。</div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      {messages.map((m, idx) => {
        if (m.role === 'interrupted') {
          return null;
        }

        if (m.role === 'system') {
          return (
            <div
              key={idx}
              style={{
                textAlign: 'center',
                margin: '12px 0',
                fontSize: '12px',
                color: 'var(--text-secondary)',
              }}
            >
              <span
                style={{
                  padding: '4px 12px',
                  borderRadius: '12px',
                  background: '#f4f4f5',
                  border: '1px solid var(--border-color)',
                  fontSize: '11px',
                  display: 'inline-block',
                }}
              >
                System Message: {m.content}
              </span>
            </div>
          );
        }

        const isUser = m.role === 'user';
        const isNextInterrupted = !isUser && idx < messages.length - 1 && messages[idx + 1].role === 'interrupted';
        // 无论是正在流式还是历史消息，所有思维推理卡片默认皆保持收起折叠，唯有主动点击展开时才为 true
        const isReasoningExpanded = expandedReasoning[idx] === true;

        return (
          <div key={idx} className={`message-wrapper ${isUser ? 'user' : 'assistant'} ${isStreaming && idx === messages.length - 1 ? 'streaming' : ''}`}>
            {/* 对话头像：USER 和 AI 均在左侧完美对齐呈现 */}
            <div className="message-avatar">
              {isUser ? (
                username ? username[0].toUpperCase() : 'U'
              ) : (
                // AI 专用 DeepSeek 星体神经网络蓝标
                <svg
                  viewBox="0 0 24 24"
                  style={{ fill: '#ffffff', stroke: 'none' }}
                  xmlns="http://www.w3.org/2000/svg"
                >
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
              {/* Reasoning / Tool Steps Accordion */}
              {!isUser && ((m.timeline && m.timeline.length > 0) || m.reasoning_content) && (
                <div className="reasoning-accordion">
                  <div
                    className={`reasoning-toggle ${isReasoningExpanded ? 'open' : ''}`}
                    onClick={() => {
                      setExpandedReasoning((prev) => ({
                        ...prev,
                        [idx]: !isReasoningExpanded,
                      }));
                    }}
                  >
                    {/* 旋转的小箭头 */}
                    <svg
                      className="reasoning-toggle-arrow"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <polyline points="9 18 15 12 9 6"></polyline>
                    </svg>
                    <span>Thinking</span>
                    {isStreaming && !m.content && idx === messages.length - 1 && (
                      <span className="thinking-spinner-container">
                        <svg className="thinking-spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round">
                          <circle cx="12" cy="12" r="10" strokeDasharray="30 15" />
                        </svg>
                      </span>
                    )}
                    {m.timeline && m.timeline.filter((i) => i.kind === 'tool_call').length > 0 && (
                      <span className="tool-steps-badge">
                        {m.timeline.filter((i) => i.kind === 'tool_call').length} tool call(s)
                      </span>
                    )}
                  </div>
                  <div className={`reasoning-content-wrapper ${isReasoningExpanded ? 'expanded' : 'collapsed'}`}>
                    <div className="reasoning-content">
                      {/* 按 timeline 顺序渲染：推理文本 + 工具步骤卡片 */}
                      {m.timeline && m.timeline.length > 0 ? (
                        m.timeline.map((item: TimelineItem, tIdx) => {
                          if (item.kind === 'reasoning') {
                            return (
                              <div key={tIdx} className="reasoning-text">
                                {item.text}
                              </div>
                            );
                          }
                          // tool_call item
                          const hasResult =
                            item.output !== undefined || item.error !== undefined;
                          let parsedArgs: Record<string, unknown> | null = null;
                          try {
                            if (item.toolArgs) parsedArgs = JSON.parse(item.toolArgs);
                          } catch { }
                          const statusClass = hasResult
                            ? item.error
                              ? 'error'
                              : 'success'
                            : 'pending';
                          const statusText = hasResult
                            ? item.error
                              ? 'Failed'
                              : 'Done'
                            : 'Running';
                          return (
                            <div key={tIdx} className="tool-step-card">
                              <div className="tool-step-header">
                                <div className="tool-name-container">
                                  {getToolIcon(item.toolName || '')}
                                  <span className="tool-step-name">
                                    {item.toolName || 'tool'}
                                  </span>
                                </div>
                                <span className={`tool-step-status ${statusClass}`}>
                                  <span className="status-dot-indicator"></span>
                                  {statusText}
                                </span>
                              </div>
                              {item.toolArgs && (
                                <div className="tool-step-section tool-step-input">
                                  <div className="tool-step-section-label">
                                    <svg className="section-label-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                                      <polyline points="9 18 15 12 9 6" />
                                    </svg>
                                    Input
                                  </div>
                                  <pre className="tool-step-code">
                                    {parsedArgs
                                      ? JSON.stringify(parsedArgs, null, 2)
                                      : item.toolArgs}
                                  </pre>
                                </div>
                              )}
                              {hasResult && (
                                <div
                                  className={`tool-step-section ${item.error ? 'tool-step-error' : 'tool-step-result'
                                    }`}
                                >
                                  <div className="tool-step-section-label">
                                    <svg className="section-label-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                                      <polyline points="15 18 9 12 15 6" />
                                    </svg>
                                    {item.error ? 'Error' : 'Output'}
                                  </div>
                                  <pre className="tool-step-code">
                                    {item.error ? String(item.error) : item.output || '(empty)'}
                                  </pre>
                                </div>
                              )}
                            </div>
                          );
                        })
                      ) : (
                        /* 兼容历史消息：仅有 reasoning_content */
                        m.reasoning_content && (
                          <div className="reasoning-text">{m.reasoning_content}</div>
                        )
                      )}
                    </div>
                  </div>
                </div>
              )}
              {/* 主答复文本 */}
              <div className={isUser ? 'message-content-user' : 'markdown-body'}>
                {m.content ? (
                  isUser ? (
                    <div style={{ whiteSpace: 'pre-wrap' }}>{m.content}</div>
                  ) : (
                    <div
                      dangerouslySetInnerHTML={{
                        __html: marked.parse(m.content) as string,
                      }}
                    />
                  )
                ) : (
                  ''
                )}
              </div>
              {isNextInterrupted && (
                <div className="message-interrupted-footer">
                  <svg className="interrupted-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                    <circle cx="12" cy="12" r="10" />
                    <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
                  </svg>
                  <span>Generation stopped</span>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </>
  );
}
