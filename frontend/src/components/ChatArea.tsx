import React, { useState, useEffect, useRef } from 'react';
import { marked } from 'marked';
import { Message, PendingTool } from '../types';

// 配置 marked，使其支持单换行解析为 <br> 标签
marked.setOptions({
  breaks: true,
});

interface ChatAreaProps {
  messages: Message[];
  currentSessionID: string;
  isStreaming: boolean;
  pendingInterrupt: {
    session_id: string;
    pending_tools: PendingTool[];
  } | null;
  selectedModel: string;
  isDarkMode: boolean;
  expandedReasoning: Record<number, boolean>;
  setExpandedReasoning: React.Dispatch<React.SetStateAction<Record<number, boolean>>>;
  onSendMessage: (text: string) => void;
  onApproveInterrupt: () => void;
  onRejectInterrupt: () => void;
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
  isDarkMode,
  expandedReasoning,
  setExpandedReasoning,
  onSendMessage,
  onApproveInterrupt,
  onRejectInterrupt,
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
            /* 折叠状态：显示“面板隐藏”标识 — 两条竖线 + 左展开箭头 */
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
            /* 展开状态：显示“面板可见”标识 — 两条竖线 + 右折叠箭头 */
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
        {messages.length === 0 ? (
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
                onClick={() => handleQuickAction('分析我近期的偏好有哪些新的进化？')}
              >
                <div className="action-card-title">🔍 分析近期偏好</div>
                <div className="action-card-desc">探索在近期对话流中，系统自动归纳的最新行为特征。</div>
              </div>
              <div
                className="action-card"
                onClick={() => handleQuickAction('检查当前会话中是否有挂起的敏感工具调用？')}
              >
                <div className="action-card-title">🛡 安全审查拦截</div>
                <div className="action-card-desc">检测是否存在挂起、需要人工干预审批的敏感工具执行。</div>
              </div>
              <div
                className="action-card"
                onClick={() => handleQuickAction('梳理一下你目前记录关于我的客观事实有哪些？')}
              >
                <div className="action-card-title">📂 整理长期记忆</div>
                <div className="action-card-desc">打印已存入数据库的客观事实事实画像列表。</div>
              </div>
              <div
                className="action-card"
                onClick={() => handleQuickAction('开始一个新的系统测试，帮我调用几个工具试试。')}
              >
                <div className="action-card-title">⚙️ 发起工具测试</div>
                <div className="action-card-desc">输入测试命令，让智能体尝试调用底层预置的服务。</div>
              </div>
            </div>
          </div>
        ) : (
          messages.map((m, idx) => {
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
            // 是否展开推理日志，默认展开。若在该 map 节点被手动置为 false，则折叠。
            const isReasoningExpanded = expandedReasoning[idx] !== false;

            return (
              <div key={idx} className={`message-wrapper ${isUser ? 'user' : 'assistant'}`}>
                {/* 对话头像：USER 和 AI 均在左侧完美对齐呈现 */}
                <div
                  className="message-avatar"
                  style={{
                    background: isUser ? (isDarkMode ? '#334155' : '#e2e8f0') : 'var(--ai-accent)',
                    color: isUser ? (isDarkMode ? '#cbd5e1' : '#475569') : '#ffffff',
                    borderColor: isUser ? 'var(--border-color)' : 'transparent',
                    fontSize: '12px',
                    fontWeight: 600,
                  }}
                >
                  {isUser ? (
                    'U'
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
                        {m.timeline && m.timeline.filter((i) => i.kind === 'tool_call').length > 0 && (
                          <span className="tool-steps-badge">
                            {m.timeline.filter((i) => i.kind === 'tool_call').length} tool call(s)
                          </span>
                        )}
                      </div>
                      {isReasoningExpanded && (
                        <div className="reasoning-content">
                          {/* 按 timeline 顺序渲染：推理文本 + 工具步骤卡片 */}
                          {m.timeline && m.timeline.length > 0 ? (
                            m.timeline.map((item, tIdx) => {
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
                              } catch {}
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
                                    <span className="tool-step-name">
                                      {item.toolName || 'tool'}
                                    </span>
                                    <span className={`tool-step-status ${statusClass}`}>
                                      {statusText}
                                    </span>
                                  </div>
                                  {item.toolArgs && (
                                    <div className="tool-step-section">
                                      <div className="tool-step-section-label">Input</div>
                                      <pre className="tool-step-code">
                                        {parsedArgs
                                          ? JSON.stringify(parsedArgs, null, 2)
                                          : item.toolArgs}
                                      </pre>
                                    </div>
                                  )}
                                  {hasResult && (
                                    <div
                                      className={`tool-step-section ${
                                        item.error ? 'tool-step-error' : 'tool-step-result'
                                      }`}
                                    >
                                      <div className="tool-step-section-label">
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
                      )}
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
                    ) : !isUser && isStreaming && idx === messages.length - 1 ? (
                      <span className="typing-cursor">正在思考...</span>
                    ) : (
                      ''
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
              <button className="btn-approve" onClick={onApproveInterrupt}>
                授权执行
              </button>
              <button className="btn-reject" onClick={onRejectInterrupt}>
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
      <form className="input-area" onSubmit={handleSubmit}>
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
                margin: 0,
              }}
            >
              <option
                value="deepseek-v4-flash"
                style={{ color: 'var(--text-main)', background: 'var(--bg-card)' }}
              >
                deepseek-v4-flash
              </option>
              <option
                value="deepseek-v4-pro"
                style={{ color: 'var(--text-main)', background: 'var(--bg-card)' }}
              >
                deepseek-v4-pro
              </option>
            </select>
          </div>
          <input
            type="text"
            className="chat-input-box"
            placeholder={
              pendingInterrupt
                ? '当前会话已挂起，请审查上面的敏感操作安全卡片。'
                : '给 Vine-Agent 发送消息...'
            }
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
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <line x1="12" y1="19" x2="12" y2="5"></line>
              <polyline points="5 12 12 5 19 12"></polyline>
            </svg>
          </button>
        </div>
      </form>
    </main>
  );
}
