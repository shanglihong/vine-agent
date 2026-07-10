import { Profile } from '../types';
import RightDrawer from './RightDrawer';

interface MemoryPanelProps {
  userProfile: Profile;
  isMemoryCollapsed: boolean;
  onClose: () => void;
}

export default function MemoryPanel({
  userProfile,
  isMemoryCollapsed,
  onClose,
}: MemoryPanelProps) {
  // 范式一：对属性(短标签)与细节描述(长卡片)进行分类分区
  const partitionItems = (items: string[]) => {
    const shorts = items.filter(item => item.length <= 16);
    const longs = items.filter(item => item.length > 16);
    return { shorts, longs };
  };

  const prefPart = partitionItems(userProfile.preferences || []);
  const factPart = partitionItems(userProfile.facts || []);

  const icon = (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{ width: 15, height: 15, flexShrink: 0 }}
      xmlns="http://www.w3.org/2000/svg"
    >
      <path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96-.44 2.5 2.5 0 0 1 0-3.12 3 3 0 0 1 0-3.88 2.5 2.5 0 0 1 0-3.12A2.5 2.5 0 0 1 9.5 2z" />
      <path d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96-.44 2.5 2.5 0 0 0 0-3.12 3 3 0 0 0 0-3.88 2.5 2.5 0 0 0 0-3.12A2.5 2.5 0 0 0 14.5 2z" />
    </svg>
  );

  return (
    <RightDrawer
      isOpen={!isMemoryCollapsed}
      onClose={onClose}
      title="Memory Vineyard"
      icon={icon}
      className="memory-panel"
    >
      <div className="memory-content">
        {/* A. 个人偏好 */}
        <div className="memory-divider-wrapper preferences">
          <svg
            className="memory-divider-icon"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z" />
          </svg>
          <span className="memory-divider">User Preferences</span>
          <span className="divider-count-badge other-count">{userProfile.preferences.length}</span>
        </div>
        {userProfile.preferences.length === 0 ? (
          <div className="empty-state">No preferences distilled yet.</div>
        ) : (
          <div className="memory-section-body">
            {prefPart.shorts.length > 0 && (
              <div className="memory-tags-group">
                {prefPart.shorts.map((p, i) => (
                  <span key={`short-${i}`} className="memory-tag-item preferences-tag">
                    {p}
                  </span>
                ))}
              </div>
            )}
            {prefPart.longs.length > 0 && (
              <div className="memory-cards-group">
                {prefPart.longs.map((p, i) => (
                  <div
                    key={`long-${i}`}
                    className="memory-card-item preferences-tag"
                  >
                    {p}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* B. 客观事实 */}
        <div className="memory-divider-wrapper facts">
          <svg
            className="memory-divider-icon"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M19 3H5c-1.1 0-2 .9-2 2v16l9-4 9 4V5c0-1.1-.9-2-2-2z" />
          </svg>
          <span className="memory-divider">Objective Facts</span>
          <span className="divider-count-badge other-count">{userProfile.facts.length}</span>
        </div>
        {userProfile.facts.length === 0 ? (
          <div className="empty-state">No factual memory nodes recorded.</div>
        ) : (
          <div className="memory-section-body">
            {factPart.shorts.length > 0 && (
              <div className="memory-tags-group">
                {factPart.shorts.map((f, i) => (
                  <span key={`short-${i}`} className="memory-tag-item facts-tag">
                    {f}
                  </span>
                ))}
              </div>
            )}
            {factPart.longs.length > 0 && (
              <div className="memory-cards-group">
                {factPart.longs.map((f, i) => (
                  <div
                    key={`long-${i}`}
                    className="memory-card-item facts-tag"
                  >
                    {f}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </RightDrawer>
  );
}
