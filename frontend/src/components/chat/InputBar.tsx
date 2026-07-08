import React from 'react';

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
}: InputBarProps) {
  return (
    <form className="input-area" onSubmit={onSubmit}>
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
        {isStreaming ? (
          <button
            type="button"
            className="send-btn stop-btn"
            style={{ background: '#ef4444', color: '#ffffff' }}
            onClick={onCancelChat}
            title="停止生成"
          >
            {/* 停止方块图标 */}
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
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
        )}
      </div>
    </form>
  );
}
