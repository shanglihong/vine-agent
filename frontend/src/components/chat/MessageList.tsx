import React from 'react';
import { marked } from 'marked';
import { Message, TimelineItem } from '../../types';

// 配置 marked，使其支持单换行解析为 <br> 标签
marked.setOptions({
  breaks: true,
});

interface MessageListProps {
  messages: Message[];
  isStreaming: boolean;
  expandedReasoning: Record<number, boolean>;
  setExpandedReasoning: React.Dispatch<React.SetStateAction<Record<number, boolean>>>;
  onQuickAction: (text: string) => void;
}

export default function MessageList({
  messages,
  isStreaming,
  expandedReasoning,
  setExpandedReasoning,
  onQuickAction,
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
          return (
            <div key={idx} className="interrupted-divider">
              <span className="interrupted-divider-label">
                {m.content || '对话已中断'}
              </span>
            </div>
          );
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
        // 是否展开推理日志，默认展开。若在该 map 节点被手动置为 false，则折叠。
        const isReasoningExpanded = expandedReasoning[idx] !== false;

        return (
          <div key={idx} className={`message-wrapper ${isUser ? 'user' : 'assistant'}`}>
            {/* 对话头像：USER 和 AI 均在左侧完美对齐呈现 */}
            <div className="message-avatar">
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
      })}
    </>
  );
}
