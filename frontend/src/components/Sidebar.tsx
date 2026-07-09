import { Session, UserInfo } from '../types';

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffSec < 60) {
    return 'now';
  } else if (diffMin < 60) {
    return `${diffMin}m`;
  } else if (diffHr < 24) {
    return `${diffHr}h`;
  } else {
    return `${diffDay}d`;
  }
}

interface SidebarProps {
  sessions: Session[];
  currentSessionID: string;
  isStreaming: boolean;
  pendingInterrupt: any;
  userInfo: UserInfo | null;
  userID: string;
  isDarkMode: boolean;
  onSelectSession: (id: string) => void;
  onCreateNewSession: () => void;
  onToggleTheme: () => void;
  onDeleteSession: (id: string, e: React.MouseEvent) => void;
}

export default function Sidebar({
  sessions,
  currentSessionID,
  isStreaming,
  pendingInterrupt,
  userInfo,
  userID,
  isDarkMode,
  onSelectSession,
  onCreateNewSession,
  onToggleTheme,
  onDeleteSession,
}: SidebarProps) {
  return (
    <aside className="sidebar">
      {/* 桌面端 macOS 窗口三色圆点控制区 */}
      <div className="window-controls">
        <span className="control-dot red"></span>
        <span className="control-dot yellow"></span>
        <span className="control-dot green"></span>
      </div>

      <div className="sidebar-header">
        <div className="logo-container">
          {/* 符合 Vine (葡萄藤蔓) 科技拓扑网格风格的 LOGO */}
          <svg
            viewBox="0 0 24 24"
            className="logo-svg"
            style={{
              fill: 'none',
              stroke: 'var(--primary-color)',
              strokeWidth: 1.5,
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              flexShrink: 0,
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
            <circle cx="8" cy="8" r="1.8" fill="var(--primary-color)" />
            <circle cx="12" cy="7" r="1.8" fill="var(--primary-color)" />
            <circle cx="16" cy="8" r="1.8" fill="var(--primary-color)" />
            <circle cx="10" cy="12" r="1.8" fill="var(--primary-color)" />
            <circle cx="14" cy="12" r="1.8" fill="var(--primary-color)" />
            <circle cx="12" cy="16" r="1.8" fill="var(--primary-color)" />
            <path d="M12 4.5V2.5c0-.5.4-.8.8-.8h1.2" />
          </svg>
          <h1>Vine-Agent</h1>
        </div>
      </div>

      {/* 极简无界 New chat 按钮 */}
      <button className="new-chat-btn" onClick={onCreateNewSession}>
        <svg
          viewBox="0 0 24 24"
          width="13.5"
          height="13.5"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.2"
          strokeLinecap="round"
          strokeLinejoin="round"
          style={{ marginRight: '8.5px', flexShrink: 0 }}
        >
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
            onClick={() => onSelectSession(s.id)}
          >
            <div className="session-name" style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>{s.id}</span>
            </div>
            <div className="session-meta">
              <span className="session-time">{formatRelativeTime(s.updated_at)}</span>
              <button
                className="session-delete-btn"
                title="Delete session"
                onClick={(e) => onDeleteSession(s.id, e)}
              >
                <svg
                  viewBox="0 0 24 24"
                  width="13"
                  height="13"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                  <line x1="10" y1="11" x2="10" y2="17"></line>
                  <line x1="14" y1="11" x2="14" y2="17"></line>
                </svg>
              </button>
              {s.id === currentSessionID && isStreaming && (
                <svg
                  className="spin-svg"
                  viewBox="0 0 24 24"
                  style={{
                    width: '10px',
                    height: '10px',
                    stroke: 'var(--primary-color)',
                    strokeWidth: 3,
                    fill: 'none',
                    strokeLinecap: 'round',
                    marginLeft: '6px',
                    animation: 'spin 1.2s linear infinite',
                    flexShrink: 0,
                  }}
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <circle cx="12" cy="12" r="10" strokeDasharray="30 12" />
                </svg>
              )}
              {(s.status === 'pending_confirmation' ||
                (s.id === currentSessionID && pendingInterrupt)) && (
                  <span
                    className="status-badge pending"
                    style={{
                      marginLeft: '6px',
                      fontSize: '9px',
                      padding: '1.5px 5px',
                      lineHeight: 1,
                    }}
                  >
                    PENDING
                  </span>
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
          <div style={{ fontSize: '10px', color: 'var(--text-secondary)' }}>
            Status: Active
          </div>
        </div>
        {/* 夜间模式切换按钮 */}
        <button
          onClick={onToggleTheme}
          className="theme-toggle-btn"
          title={isDarkMode ? '切换至明亮模式' : '切换至暗色模式'}
        >
          {isDarkMode ? (
            <svg
              viewBox="0 0 24 24"
              width="15"
              height="15"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
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
            <svg
              viewBox="0 0 24 24"
              width="15"
              height="15"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          )}
        </button>
      </div>
    </aside>
  );
}
