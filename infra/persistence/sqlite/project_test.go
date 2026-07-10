package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"vine-agent/domain/project"
)

func newTestProjectStore(t *testing.T) *ProjectStore {
	t.Helper()
	return newProjectStoreWithDB(testDB)
}

func clearProjectTestData(t *testing.T) {
	t.Helper()
	_, _ = testDB.Exec("DELETE FROM projects")
	_, _ = testDB.Exec("DELETE FROM project_sessions")
	_, _ = testDB.Exec("DELETE FROM sessions")
}

func buildTestProject(id, userID string) *project.Project {
	now := time.Now().Truncate(time.Second)
	return &project.Project{
		ID:          id,
		UserID:      userID,
		Name:        "project-name-" + id,
		Path:        "/path/to/" + id,
		Description: "description-" + id,
		Metadata:    map[string]string{"env": "test"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestProjectStore_SaveGet(t *testing.T) {
	clearProjectTestData(t)
	defer clearProjectTestData(t)

	store := newTestProjectStore(t)
	ctx := context.Background()
	proj := buildTestProject("proj-1", "user-1")

	// 1. 测试保存/新增
	if err := store.Save(ctx, proj); err != nil {
		t.Fatalf("Save project: %v", err)
	}

	// 2. 测试获取并对比
	got, err := store.Get(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Get project: %v", err)
	}
	if got.ID != proj.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, proj.ID)
	}
	if got.UserID != proj.UserID {
		t.Errorf("UserID mismatch: got %q, want %q", got.UserID, proj.UserID)
	}
	if got.Name != proj.Name {
		t.Errorf("Name mismatch: got %q, want %q", got.Name, proj.Name)
	}
	if got.Path != proj.Path {
		t.Errorf("Path mismatch: got %q, want %q", got.Path, proj.Path)
	}
	if got.Description != proj.Description {
		t.Errorf("Description mismatch: got %q, want %q", got.Description, proj.Description)
	}
	if got.Metadata["env"] != "test" {
		t.Errorf("Metadata mismatch: got %v", got.Metadata)
	}

	// 3. 测试更新
	proj.Name = "updated-name"
	proj.UpdatedAt = proj.UpdatedAt.Add(time.Second)
	if err := store.Save(ctx, proj); err != nil {
		t.Fatalf("Update project: %v", err)
	}

	gotUpdated, err := store.Get(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Get updated project: %v", err)
	}
	if gotUpdated.Name != "updated-name" {
		t.Errorf("Name update fail: got %q", gotUpdated.Name)
	}

	// 4. 测试不存在的 Project
	_, err = store.Get(ctx, "proj-not-exist")
	if !errors.Is(err, project.ErrProjectNotFound) {
		t.Errorf("expected ErrProjectNotFound, got: %v", err)
	}
}

func TestProjectStore_Delete(t *testing.T) {
	clearProjectTestData(t)
	defer clearProjectTestData(t)

	store := newTestProjectStore(t)
	ctx := context.Background()
	proj := buildTestProject("proj-delete", "user-1")

	if err := store.Save(ctx, proj); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 建立模拟绑定
	if err := store.BindSession(ctx, proj.ID, "sess-mock-1"); err != nil {
		t.Fatalf("BindSession: %v", err)
	}

	// 物理删除
	if err := store.Delete(ctx, proj.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// 检查项目本身已被删除
	_, err := store.Get(ctx, proj.ID)
	if !errors.Is(err, project.ErrProjectNotFound) {
		t.Errorf("expected ErrProjectNotFound, got: %v", err)
	}

	// 检查关联关系已被删除
	sessIDs, err := store.ListSessionsByProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListSessionsByProject: %v", err)
	}
	if len(sessIDs) != 0 {
		t.Errorf("expected 0 bound sessions, got %v", sessIDs)
	}
}

func TestProjectStore_List(t *testing.T) {
	clearProjectTestData(t)
	defer clearProjectTestData(t)

	store := newTestProjectStore(t)
	ctx := context.Background()

	p1 := buildTestProject("proj-list-1", "user-list")
	p2 := buildTestProject("proj-list-2", "user-list")
	p3 := buildTestProject("proj-list-3", "user-other")

	_ = store.Save(ctx, p1)
	_ = store.Save(ctx, p2)
	_ = store.Save(ctx, p3)

	list, err := store.List(ctx, "user-list")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 projects, got %d", len(list))
	}
}

func TestProjectStore_BindAndListSessions(t *testing.T) {
	clearProjectTestData(t)
	defer clearProjectTestData(t)

	store := newTestProjectStore(t)
	ctx := context.Background()

	// 插入会话数据以便支持 NOT IN (SELECT session_id FROM project_sessions)
	// 在内存库写入 sessions
	_, err := testDB.Exec(`
		INSERT INTO sessions (id, user_id, messages, metadata, created_at, updated_at) 
		VALUES ('sess-1', 'user-1', '[]', '{}', ?, ?),
		       ('sess-2', 'user-1', '[]', '{}', ?, ?),
		       ('sess-3', 'user-1', '[]', '{}', ?, ?)`,
		time.Now(), time.Now(), time.Now(), time.Now(), time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("insert test sessions: %v", err)
	}

	// 1. 绑定 sess-1 和 sess-2 到 proj-1
	if err := store.BindSession(ctx, "proj-1", "sess-1"); err != nil {
		t.Fatalf("BindSession: %v", err)
	}
	if err := store.BindSession(ctx, "proj-1", "sess-2"); err != nil {
		t.Fatalf("BindSession: %v", err)
	}

	// 2. 列出项目关联的 sessionIDs
	sessionIDs, err := store.ListSessionsByProject(ctx, "proj-1")
	if err != nil {
		t.Fatalf("ListSessionsByProject: %v", err)
	}
	if len(sessionIDs) != 2 {
		t.Errorf("expected 2 bound sessions, got %v", sessionIDs)
	}

	// 3. 再次绑定 sess-1 到 proj-2（原绑定应当被物理删除，保证每个 Session 只有单项目归属）
	if err := store.BindSession(ctx, "proj-2", "sess-1"); err != nil {
		t.Fatalf("Re-BindSession: %v", err)
	}

	sessionIDsProj1, _ := store.ListSessionsByProject(ctx, "proj-1")
	if len(sessionIDsProj1) != 1 || sessionIDsProj1[0] != "sess-2" {
		t.Errorf("expected only sess-2 in proj-1, got %v", sessionIDsProj1)
	}

	sessionIDsProj2, _ := store.ListSessionsByProject(ctx, "proj-2")
	if len(sessionIDsProj2) != 1 || sessionIDsProj2[0] != "sess-1" {
		t.Errorf("expected sess-1 in proj-2, got %v", sessionIDsProj2)
	}

	// 4. 列出未分类会话 ID (此时 sess-3 依然未绑定任何项目，sess-1, sess-2 已经绑定了)
	unclassified, err := store.ListUnclassifiedSessions(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListUnclassifiedSessions: %v", err)
	}
	if len(unclassified) != 1 || unclassified[0] != "sess-3" {
		t.Errorf("expected only sess-3 as unclassified, got %v", unclassified)
	}
}

func TestProjectStore_GetProjectBySession(t *testing.T) {
	clearProjectTestData(t)
	defer clearProjectTestData(t)

	store := newTestProjectStore(t)
	ctx := context.Background()

	// 1. 创建并保存项目
	proj := buildTestProject("proj-test-session", "user-1")
	if err := store.Save(ctx, proj); err != nil {
		t.Fatalf("Save project: %v", err)
	}

	// 2. 绑定会话到项目
	sessionID := "session-bound-1"
	if err := store.BindSession(ctx, proj.ID, sessionID); err != nil {
		t.Fatalf("BindSession: %v", err)
	}

	// 3. 测试通过 SessionID 获取项目
	got, err := store.GetProjectBySession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetProjectBySession: %v", err)
	}
	if got.ID != proj.ID {
		t.Errorf("expected project ID %q, got %q", proj.ID, got.ID)
	}
	if got.Path != proj.Path {
		t.Errorf("expected project Path %q, got %q", proj.Path, got.Path)
	}

	// 4. 测试未绑定的 SessionID
	_, err = store.GetProjectBySession(ctx, "session-unbound-999")
	if !errors.Is(err, project.ErrProjectNotFound) {
		t.Errorf("expected ErrProjectNotFound, got: %v", err)
	}
}
