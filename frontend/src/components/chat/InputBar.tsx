import React, { useState, useRef, useEffect } from 'react';

interface InputBarProps {
  inputValue: string;
  setInputValue: (value: string) => void;
  selectedModel: string;
  setSelectedModel: (model: string) => void;
  isStreaming: boolean;
  pendingInterrupt: object | null;
  currentSessionID: string;
  onSubmit: (e: React.FormEvent) => void;
  onCancelChat: () => void;
  webSearchEnabled: boolean;
  setWebSearchEnabled: (enabled: boolean) => void;
}

export default function InputBar({
  inputValue,
  setInputValue,
  selectedModel,
  setSelectedModel,
  isStreaming,
  pendingInterrupt,
  currentSessionID,
  onSubmit,
  onCancelChat,
  webSearchEnabled,
  setWebSearchEnabled,
}: InputBarProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 监听点击下拉菜单外部自动关闭
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 自动根据打字内容调整 textarea 高度
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
    }
  }, [inputValue]);

  // 智能拦截键盘回车发送，Shift+Enter 则换行
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // 中文输入法正在选词/组合文字时，直接放行，不拦截回车
    if (e.nativeEvent.isComposing) return;
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (inputValue.trim() && !isStreaming && !pendingInterrupt && currentSessionID) {
        onSubmit(e as unknown as React.FormEvent);
      }
    }
  };

  const models = [
    { id: 'deepseek-v4-flash', label: 'deepseek-v4-flash' },
    { id: 'deepseek-v4-pro', label: 'deepseek-v4-pro' }
  ];

  const selectedLabel = models.find((m) => m.id === selectedModel)?.label || selectedModel;

  return (
    <form className="input-area" onSubmit={onSubmit}>
      <div className="input-container">
        {/* 1. 上层：打字输入区域 */}
        <div className="input-body-row">
          <textarea
            ref={textareaRef}
            className="chat-input-box"
            placeholder={
              pendingInterrupt
                ? 'The current session is suspended. Please review the sensitive action card above.'
                : 'Message Vine-Agent...'
            }
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={isStreaming || !!pendingInterrupt || !currentSessionID}
            rows={1}
          />
        </div>

        {/* 2. 下层：控制栏（左侧模型选择器，右侧发送/停止按钮） */}
        <div className="input-control-row">
          <div className="input-control-left">
            {/* 左侧：芯片状模型选择药丸 */}
            <div className="model-selector-container" ref={dropdownRef}>
              <button
                type="button"
                className={`model-selector-pill ${isOpen ? 'active' : ''}`}
                onClick={() => setIsOpen(!isOpen)}
                disabled={isStreaming || !!pendingInterrupt || !currentSessionID}
              >
                {/* 极简 AI 星光 SVG 图标，代表智能模型 */}
                <svg className="model-selector-chip-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 2C12 2 12 12 2 12C12 12 12 22 12 22C12 22 12 12 22 12C12 12 22 12 12 2z" />
                </svg>
                <span className="model-selector-label">{selectedLabel}</span>
                <svg
                  className={`model-selector-arrow ${isOpen ? 'open' : ''}`}
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polyline points="6 9 12 15 18 9" />
                </svg>
              </button>

              {isOpen && (
                <div className="model-dropdown-menu">
                  {models.map((m) => (
                    <div
                      key={m.id}
                      className={`model-dropdown-item ${selectedModel === m.id ? 'selected' : ''}`}
                      onClick={() => {
                        setSelectedModel(m.id);
                        setIsOpen(false);
                      }}
                    >
                      <span className="item-check-icon">
                        {selectedModel === m.id && (
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.2" strokeLinecap="round" strokeLinejoin="round">
                            <polyline points="20 6 9 17 4 12" />
                          </svg>
                        )}
                      </span>
                      <span className="item-label">{m.label}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* 联网搜索开关 */}
            <button
              type="button"
              className={`web-search-toggle-pill ${webSearchEnabled ? 'active' : ''}`}
              onClick={() => setWebSearchEnabled(!webSearchEnabled)}
              disabled={isStreaming || !!pendingInterrupt || !currentSessionID}
              title={webSearchEnabled ? 'Search enabled' : 'Search the web'}
            >
              <svg className="web-search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10" />
                <line x1="2" y1="12" x2="22" y2="12" />
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
              </svg>
              <span className="web-search-label">Web Search</span>
            </button>
          </div>

          {/* 右侧：发送/停止按钮 */}
          <div className="action-buttons-group">
            {isStreaming ? (
              <button
                type="button"
                className="send-btn stop-btn"
                style={{ background: '#ef4444', color: '#ffffff' }}
                onClick={onCancelChat}
                title="Stop generating"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="15"
                  height="15"
                  fill="currentColor"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <rect x="4" y="4" width="16" height="16" rx="2" ry="2" />
                </svg>
              </button>
            ) : (
              <button
                type="submit"
                className="send-btn"
                style={{ background: inputValue.trim() ? 'var(--ai-accent)' : '#e5e7eb' }}
                disabled={!inputValue.trim() || !!pendingInterrupt || !currentSessionID}
              >
                <svg
                  viewBox="0 0 24 24"
                  width="15"
                  height="15"
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
            )}
          </div>
        </div>
      </div>
    </form>
  );
}


