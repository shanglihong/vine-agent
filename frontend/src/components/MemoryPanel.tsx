import { Profile } from '../types';

interface MemoryPanelProps {
  userProfile: Profile;
  isMemoryCollapsed: boolean;
  isEvolving: boolean;
  currentSessionID: string;
  onEvolveProfile: () => void;
}

export default function MemoryPanel({
  userProfile,
  isMemoryCollapsed,
  isEvolving,
  currentSessionID,
  onEvolveProfile,
}: MemoryPanelProps) {
  return (
    <aside className={`memory-panel ${isMemoryCollapsed ? 'collapsed' : ''}`}>
      <header className="memory-header">
        <div className="memory-header-title">
          {/* Memory Vineyard 的 Header 图标 - 采用 Vine 科技葡萄图标 */}
          <svg
            viewBox="0 0 24 24"
            style={{
              width: '16px',
              height: '16px',
              fill: 'none',
              stroke: 'var(--primary-color)',
              strokeWidth: 1.8,
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              marginRight: '8px',
            }}
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96-.44 2.5 2.5 0 0 1 0-3.12 3 3 0 0 1 0-3.88 2.5 2.5 0 0 1 0-3.12A2.5 2.5 0 0 1 9.5 2zM14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96-.44 2.5 2.5 0 0 0 0-3.12 3 3 0 0 0 0-3.88 2.5 2.5 0 0 0 0-3.12A2.5 2.5 0 0 0 14.5 2z" />
            <path d="M12 5h1M12 9h2M12 13h1M12 17h2M12 7h-1M12 11h-2M12 15h-1M12 19h-2" />
          </svg>
          <h3>Memory Vineyard</h3>
        </div>
        <button
          className={`evolve-btn ${isEvolving ? 'spinning' : ''}`}
          onClick={onEvolveProfile}
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
            <svg
              className="memory-sec-title-svg"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
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
            <svg
              className="memory-sec-title-svg"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
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
  );
}
