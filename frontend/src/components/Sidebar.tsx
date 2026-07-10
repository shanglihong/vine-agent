import { useState } from 'react';
import { Session, UserInfo, Project } from '../types';

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
  isMemoryCollapsed: boolean;
  setIsMemoryCollapsed: (v: boolean) => void;
  onSelectSession: (id: string) => void;
  onCreateNewSession: (projectId?: string) => void;
  onToggleTheme: () => void;
  onDeleteSession: (id: string, e: React.MouseEvent) => void;
  onRenameSession: (id: string, newName: string) => Promise<void>;
  onShowTooltip: (text: string, e: React.MouseEvent) => void;
  onMoveTooltip: (e: React.MouseEvent) => void;
  onHideTooltip: () => void;
  projects: Project[];
  onCreateProject: (name: string) => Promise<string>;
  onRenameProject: (projectId: string, newName: string) => Promise<void>;
  onDeleteProject: (projectId: string) => Promise<void>;
}

export default function Sidebar({
  sessions,
  currentSessionID,
  isStreaming,
  pendingInterrupt,
  userInfo,
  userID,
  isDarkMode,
  isMemoryCollapsed,
  setIsMemoryCollapsed,
  onSelectSession,
  onCreateNewSession,
  onToggleTheme,
  onDeleteSession,
  onRenameSession,
  onShowTooltip,
  onMoveTooltip,
  onHideTooltip,
  projects,
  onCreateProject,
  onRenameProject,
  onDeleteProject,
}: SidebarProps) {
  const [editingSessionID, setEditingSessionID] = useState<string | null>(null);
  const [editingName, setEditingName] = useState<string>('');

  // 折叠项目的状态管理
  const [collapsedProjects, setCollapsedProjects] = useState<Record<string, boolean>>({});

  const [editingProjectID, setEditingProjectID] = useState<string | null>(null);
  const [editingProjectName, setEditingProjectName] = useState('');

  const toggleProjectCollapse = (projectId: string) => {
    setCollapsedProjects((prev) => ({
      ...prev,
      [projectId]: !prev[projectId],
    }));
  };

  const handleSaveProjectRename = async (id: string) => {
    const trimmed = editingProjectName.trim();
    if (!trimmed) {
      setEditingProjectID(null);
      return;
    }
    try {
      await onRenameProject(id, trimmed);
    } finally {
      setEditingProjectID(null);
    }
  };

  const [isCreatingProjectOutside, setIsCreatingProjectOutside] = useState(false);
  const [newProjectNameOutside, setNewProjectNameOutside] = useState('');

  const handleCreateProjectOutsideSubmit = async () => {
    const trimmed = newProjectNameOutside.trim();
    if (!trimmed) {
      setIsCreatingProjectOutside(false);
      return;
    }
    try {
      await onCreateProject(trimmed);
      setNewProjectNameOutside('');
      setIsCreatingProjectOutside(false);
    } catch (err) {
      setIsCreatingProjectOutside(false);
    }
  };

  const handleCreateProjectOutsideKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // 中文输入法正在选词/组合文字时，直接放行，不拦截回车
    if (e.nativeEvent.isComposing) return;
    if (e.key === 'Enter') {
      handleCreateProjectOutsideSubmit();
    } else if (e.key === 'Escape') {
      setIsCreatingProjectOutside(false);
      setNewProjectNameOutside('');
    }
  };

  const handleStartRename = (id: string, currentName: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setEditingSessionID(id);
    setEditingName(currentName);
  };

  const handleSaveRename = async (id: string) => {
    const trimmed = editingName.trim();
    if (!trimmed) {
      setEditingSessionID(null);
      return;
    }
    try {
      await onRenameSession(id, trimmed);
    } finally {
      setEditingSessionID(null);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>, id: string) => {
    // 中文输入法正在选词/组合文字时，直接放行，不拦截回车
    if (e.nativeEvent.isComposing) return;
    if (e.key === 'Enter') {
      handleSaveRename(id);
    } else if (e.key === 'Escape') {
      setEditingSessionID(null);
    }
  };

  const renderSessionItem = (s: Session) => {
    return (
      <div
        key={s.id}
        className={`session-item ${currentSessionID === s.id ? 'active' : ''}`}
        onClick={() => onSelectSession(s.id)}
      >
        {editingSessionID === s.id ? (
          <input
            type="text"
            className="session-rename-input"
            value={editingName}
            onChange={(e) => setEditingName(e.target.value)}
            onBlur={() => handleSaveRename(s.id)}
            onKeyDown={(e) => handleKeyDown(e, s.id)}
            autoFocus
            onClick={(e) => e.stopPropagation()}
          />
        ) : (
          <div
            className="session-name"
            style={{ display: 'flex', alignItems: 'center' }}
            onDoubleClick={(e) => handleStartRename(s.id, s.name || s.id, e)}
          >
            <span
              style={{ textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}
              onMouseEnter={(e) => {
                const isTruncated = e.currentTarget.scrollWidth > e.currentTarget.clientWidth;
                if (isTruncated) {
                  onShowTooltip(s.name || s.id, e);
                }
              }}
              onMouseMove={onMoveTooltip}
              onMouseLeave={onHideTooltip}
            >
              {s.name || s.id}
            </span>
          </div>
        )}
        <div className="session-meta">
          <span className="session-time">{formatRelativeTime(s.updated_at)}</span>
          {editingSessionID !== s.id && (
            <button
              className="session-rename-btn"
              title="Rename session"
              onClick={(e) => handleStartRename(s.id, s.name || s.id, e)}
            >
              <svg
                viewBox="0 0 24 24"
                width="12"
                height="12"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M12 20h9"></path>
                <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path>
              </svg>
            </button>
          )}
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
              <span className="status-badge pending">
                PENDING
              </span>
            )}
        </div>
      </div>
    );
  };

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

      {/* 极简无界 New Project 按钮 */}
      {isCreatingProjectOutside ? (
        <div className="new-project-input-container">
          <input
            type="text"
            className="new-project-input"
            placeholder="Enter project name..."
            value={newProjectNameOutside}
            onChange={(e) => setNewProjectNameOutside(e.target.value)}
            onKeyDown={handleCreateProjectOutsideKeyDown}
            onBlur={handleCreateProjectOutsideSubmit}
            autoFocus
          />
        </div>
      ) : (
        <button className="new-project-btn" onClick={() => setIsCreatingProjectOutside(true)}>
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
          New Project
        </button>
      )}

      <div className="session-list">
        {/* 1. 未分类会话平铺展示，不再使用 Unclassified 分组包裹 */}
        {(sessions || [])
          .filter((s) => !s.project_id)
          .map((s) => renderSessionItem(s))}

        {/* 2. 自定义项目平铺 */}
        {(projects || []).map((proj) => {
          const projectSessions = (sessions || []).filter((s) => s.project_id === proj.id);
          const isCollapsed = !!collapsedProjects[proj.id];
          return (
            <div key={proj.id} className="project-group">
              <div
                className="project-group-header"
                onClick={() => toggleProjectCollapse(proj.id)}
              >
                <svg
                  className="project-group-icon"
                  viewBox="0 0 24 24"
                  width="12"
                  height="12"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.2"
                >
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                </svg>
                {editingProjectID === proj.id ? (
                  <input
                    type="text"
                    className="project-group-rename-input"
                    value={editingProjectName}
                    onChange={(e) => setEditingProjectName(e.target.value)}
                    onBlur={() => handleSaveProjectRename(proj.id)}
                    onKeyDown={(e) => {
                      // 中文输入法正在选词/组合文字时，直接放行，不拦截回车
                      if (e.nativeEvent.isComposing) return;
                      if (e.key === 'Enter') handleSaveProjectRename(proj.id);
                      if (e.key === 'Escape') setEditingProjectID(null);
                    }}
                    autoFocus
                    onClick={(e) => e.stopPropagation()}
                  />
                ) : (
                  <span className="project-group-name">{proj.name}</span>
                )}

                <div className="project-group-actions" onClick={(e) => e.stopPropagation()}>
                  <button
                    className="project-group-action-btn add"
                    title="New Chat"
                    onClick={() => onCreateNewSession(proj.id)}
                  >
                    <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" strokeWidth="2.5">
                      <line x1="12" y1="5" x2="12" y2="19"></line>
                      <line x1="5" y1="12" x2="19" y2="12"></line>
                    </svg>
                  </button>
                  <button
                    className="project-group-action-btn edit"
                    title="Rename"
                    onClick={() => {
                      setEditingProjectID(proj.id);
                      setEditingProjectName(proj.name);
                    }}
                  >
                    <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" strokeWidth="2.5">
                      <path d="M12 20h9"></path>
                      <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path>
                    </svg>
                  </button>
                  <button
                    className="project-group-action-btn delete"
                    title="Delete"
                    onClick={() => onDeleteProject(proj.id)}
                  >
                    <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" strokeWidth="2.5">
                      <polyline points="3 6 5 6 21 6"></polyline>
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                    </svg>
                  </button>
                </div>
              </div>

              {!isCollapsed && projectSessions.length > 0 && (
                <div className="project-sessions-container">
                  {projectSessions.map((s) => renderSessionItem(s))}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="user-footer">
        <div className="user-avatar">
          {userInfo?.username ? userInfo.username[0].toUpperCase() : 'U'}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontWeight: 500, fontSize: '12.5px', color: 'var(--text-main)', letterSpacing: '-0.1px', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>
            {userInfo?.username || userID || 'Loading...'}
          </div>
          <div style={{ fontSize: '10.5px', color: 'var(--text-muted)' }}>
            Status: Active
          </div>
        </div>
        {/* Memory Vineyard 按钮 */}
        <button
          className="toggle-memory-btn"
          onClick={() => setIsMemoryCollapsed(!isMemoryCollapsed)}
        >
          <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96-.44 2.5 2.5 0 0 1 0-3.12 3 3 0 0 1 0-3.88 2.5 2.5 0 0 1 0-3.12A2.5 2.5 0 0 1 9.5 2z" />
            <path d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96-.44 2.5 2.5 0 0 0 0-3.12 3 3 0 0 0 0-3.88 2.5 2.5 0 0 0 0-3.12A2.5 2.5 0 0 0 14.5 2z" />
          </svg>
        </button>
        {/* 夜间模式切换按钮 */}
        <button
          onClick={onToggleTheme}
          className="theme-toggle-btn"
          title={isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'}
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
