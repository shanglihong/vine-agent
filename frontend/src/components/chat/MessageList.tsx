import React, { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Message, TimelineItem } from '../../types';

interface CodeBlockProps {
  className?: string;
  inline?: boolean;
  children: React.ReactNode;
}

function CodeBlock({ className, inline, children, ...props }: CodeBlockProps) {
  const [copied, setCopied] = useState(false);
  const match = /language-(\w+)/.exec(className || '');
  const lang = match ? match[1] : '';
  const rawCode = String(children).replace(/\n$/, '');

  const handleCopy = () => {
    navigator.clipboard.writeText(rawCode).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  if (!inline && match) {
    return (
      <div className="code-block-wrapper">
        <div className="code-block-header">
          <span className="code-block-lang">{lang}</span>
          <button className="code-block-copy-btn" onClick={handleCopy}>
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>
        <pre className={className} {...props}>
          <code>{children}</code>
        </pre>
      </div>
    );
  }

  return (
    <code className={className} {...props}>
      {children}
    </code>
  );
}

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

interface SearchItem {
  title: string;
  url: string;
  snippet: string;
}

interface MessageListProps {
  messages: Message[];
  isStreaming: boolean;
  expandedReasoning: Record<number, boolean>;
  setExpandedReasoning: React.Dispatch<React.SetStateAction<Record<number, boolean>>>;
  onQuickAction: (text: string) => void;
  username?: string;
  onShowTooltip: (text: string, e: React.MouseEvent) => void;
  onMoveTooltip: (e: React.MouseEvent) => void;
  onHideTooltip: () => void;
  onOpenSearchResults: (items: { title: string; url: string; snippet: string }[], fetchedUrls: Set<string>) => void;
  isSearchPanelOpen: boolean;
  searchResults: { title: string; url: string; snippet: string }[];
  highlightedMessageIdx?: number | null;
}

export default function MessageList({
  messages,
  isStreaming,
  expandedReasoning,
  setExpandedReasoning,
  onQuickAction,
  username,
  onShowTooltip,
  onMoveTooltip,
  onHideTooltip,
  onOpenSearchResults,
  isSearchPanelOpen,
  searchResults,
  highlightedMessageIdx,
}: MessageListProps) {
  if (messages.length === 0) {
    return (
      <div className="empty-state-thesis">
        <div
          className="empty-logo-container"
        >
          {/* 首页大图标：符合 Vine 的科技风格葡萄图标 */}
          <svg
            viewBox="0 0 24 24"
            className="logo-vine"
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
            onClick={() => onQuickAction('How has my user preference profile evolved recently?')}
          >
            <div className="action-card-header">
              <span className="action-card-icon-wrapper pref">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M3 3v18h18" />
                  <path d="M18.7 8l-5.1 5.2-2.8-2.7L7 14.3" />
                  <path d="M19 11.5l3-3-3-3" />
                  <path d="M22 8.5h-6" />
                </svg>
              </span>
              <div className="action-card-title">Analyze Preferences</div>
            </div>
            <div className="action-card-desc">Explore the latest behavioral profiles auto-extracted from recent sessions.</div>
          </div>

          <div
            className="action-card"
            onClick={() => onQuickAction('Are there any pending sensitive tool calls in the current session?')}
          >
            <div className="action-card-header">
              <span className="action-card-icon-wrapper security">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                  <path d="m9 11 2 2 4-4" />
                </svg>
              </span>
              <div className="action-card-title">Security & Guardrails</div>
            </div>
            <div className="action-card-desc">Check if there are any pending sensitive tool executions awaiting manual approval.</div>
          </div>

          <div
            className="action-card"
            onClick={() => onQuickAction('Summarize all the objective facts about me stored in your long-term memory.')}
          >
            <div className="action-card-header">
              <span className="action-card-icon-wrapper memory">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <ellipse cx="12" cy="5" rx="9" ry="3" />
                  <path d="M3 5V19c0 1.66 4 3 9 3s9-1.34 9-3V5" />
                  <path d="M3 12c0 1.66 4 3 9 3s9-1.34 9-3" />
                </svg>
              </span>
              <div className="action-card-title">Manage Core Memory</div>
            </div>
            <div className="action-card-desc">Retrieve the list of objective facts and core persona stored in the memory database.</div>
          </div>

          <div
            className="action-card"
            onClick={() => onQuickAction('Initiate a tool call test session to try executing some basic tools.')}
          >
            <div className="action-card-header">
              <span className="action-card-icon-wrapper test">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="4 17 10 11 4 5" />
                  <line x1="12" y1="19" x2="20" y2="19" />
                </svg>
              </span>
              <div className="action-card-title">Trigger Tool Testing</div>
            </div>
            <div className="action-card-desc">Provide test inputs to verify and run execution of underlying agent tools.</div>
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
          <div
            key={idx}
            id={`msg-${idx}`}
            className={`message-wrapper ${isUser ? 'user' : 'assistant'} ${isStreaming && idx === messages.length - 1 ? 'streaming' : ''} ${highlightedMessageIdx === idx ? 'highlight' : ''}`}
          >
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
              {!isUser && ((m.timeline && m.timeline.length > 0) || m.reasoning_content) && (() => {
                const searchItem = m.timeline?.find(i => i.kind === 'tool_call' && i.toolName === 'web_search');
                let searchItems: SearchItem[] = [];
                if (searchItem && searchItem.output) {
                  try {
                    searchItems = JSON.parse(searchItem.output);
                  } catch {
                    // 容错
                  }
                }
                // 扫描整个会话所有消息，收集所有 fetch_webpage/web_crawl 精读的 URL
                const fetchedUrls = new Set<string>();
                messages.forEach((msg, mIndex) => {
                  console.log(`[MsgList] Message #${mIndex} role=${msg.role} timeline=`, msg.timeline, "tool_calls=", msg.tool_calls);
                  (msg.timeline ?? []).forEach(i => {
                    if (i.kind === 'tool_call' && (i.toolName === 'fetch_webpage' || i.toolName === 'web_crawl')) {
                      try {
                        const parsedArgs = JSON.parse(i.toolArgs || '{}');
                        const url = parsedArgs.url as string;
                        console.log(`[MsgList] Found fetch tool call with URL: ${url}`);
                        if (url) fetchedUrls.add(url);
                      } catch (e) {
                        console.error(`[MsgList] Failed to parse toolArgs for fetch:`, i.toolArgs, e);
                      }
                    }
                  });
                });
                console.log("[MsgList] Final extracted fetchedUrls set:", Array.from(fetchedUrls));

                return (
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
                      {(() => {
                        const isActive = isStreaming && !m.content && idx === messages.length - 1;
                        const toolCalls = m.timeline?.filter(i => i.kind === 'tool_call') ?? [];
                        const hasSearch = searchItems.length > 0;
                        if (isActive) {
                          return toolCalls.length > 0
                            ? <span>Using tools</span>
                            : <span>Thinking</span>;
                        }
                        if (hasSearch) return <span>Searched</span>;
                        if (toolCalls.length > 0) return <span>Used tools</span>;
                        return <span>Thought</span>;
                      })()}
                      {isStreaming && !m.content && idx === messages.length - 1 && (
                        <span className="thinking-spinner-container">
                          <svg className="thinking-spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round">
                            <circle cx="12" cy="12" r="10" strokeDasharray="30 15" />
                          </svg>
                        </span>
                      )}
                      {m.timeline && m.timeline.filter((i) => i.kind === 'tool_call').length > 0 && (
                        <span className="tool-steps-badge">
                          <svg className="tool-badge-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                          </svg>
                          {(() => {
                            const count = m.timeline.filter((i) => i.kind === 'tool_call').length;
                            return count === 1 ? 'Used 1 tool' : `Used ${count} tools`;
                          })()}
                        </span>
                      )}
                      {searchItems.length > 0 && (() => {
                        const isActive = isSearchPanelOpen && 
                          searchResults.length === searchItems.length &&
                          searchResults.every((val, index) => val.url === searchItems[index]?.url);
                        return (
                          <span
                            className={`search-summary-bubble ${isActive ? 'active' : ''}`}
                            onClick={(e) => {
                              e.stopPropagation();
                              onOpenSearchResults(searchItems, fetchedUrls);
                            }}
                            onMouseEnter={(e) => onShowTooltip('View search sources', e)}
                            onMouseMove={(e) => onMoveTooltip(e)}
                            onMouseLeave={() => onHideTooltip()}
                          >
                            <svg className="search-bubble-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                              <circle cx="12" cy="12" r="10" />
                              <line x1="2" y1="12" x2="22" y2="12" />
                              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
                            </svg>
                            <span>Searched {searchItems.length} sources</span>
                            <span className="search-chevron-wrapper">
                              <svg className="search-chevron-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.8" strokeLinecap="round" strokeLinejoin="round" style={{ width: '8px', height: '8px' }}>
                                <polyline points="9 18 15 12 9 6" />
                              </svg>
                            </span>
                          </span>
                        );
                      })()}
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
                );
              })()}
              {/* 主答复文本 */}
              <div className={isUser ? 'message-content-user' : 'markdown-body'}>
                {m.content ? (
                  isUser ? (
                    <div style={{ whiteSpace: 'pre-wrap' }}>{m.content}</div>
                  ) : (
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm]}
                      components={{
                        code({ inline, className, children, ...props }) {
                          const isInline = !!inline;
                          return (
                            <CodeBlock className={className} inline={isInline} {...props}>
                              {children}
                            </CodeBlock>
                          );
                        }
                      }}
                    >
                      {m.content}
                    </ReactMarkdown>
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
