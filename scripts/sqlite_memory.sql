CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT DEFAULT '',
    messages TEXT NOT NULL,
    metadata TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    session_id,
    user_id,
    role,
    content,
    tokenize="unicode61"
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT DEFAULT '',
    description TEXT DEFAULT '',
    metadata TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);

CREATE TABLE IF NOT EXISTS project_sessions (
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    PRIMARY KEY (project_id, session_id)
);
CREATE INDEX IF NOT EXISTS idx_project_sessions_session_id ON project_sessions(session_id);

