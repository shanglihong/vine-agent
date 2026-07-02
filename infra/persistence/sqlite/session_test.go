package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
)

// newTestStore 返回基于共享内存 SQLite 的 SessionStore 实例，并清空数据以保证测试隔离
func newTestStore(t *testing.T) *SessionStore {
	t.Helper()
	if _, err := testDB.Exec("DELETE FROM sessions"); err != nil {
		t.Fatalf("clear database: %v", err)
	}
	return newSessionStoreWithDB(testDB)
}

// buildSession 构造一个测试用 Session
func buildSession(id, userID string) *session.Session {
	now := time.Now().Truncate(time.Second) // SQLite DATETIME 精度到秒
	return &session.Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  map[string]string{"env": "test"},
		Messages: []message.Message{
			{Role: message.RoleUser, Content: "hello"},
			{Role: message.RoleAssistant, Content: "world"},
		},
	}
}

// ==================== Save ====================

func TestSessionStore_Save_Create(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sess := buildSession("sess-1", "user-1")

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get after Save: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("ID: want %q, got %q", sess.ID, got.ID)
	}
	if got.UserID != sess.UserID {
		t.Errorf("UserID: want %q, got %q", sess.UserID, got.UserID)
	}
	if got.Metadata["env"] != "test" {
		t.Errorf("Metadata[env]: want %q, got %q", "test", got.Metadata["env"])
	}
	if len(got.Messages) != 2 {
		t.Errorf("Messages len: want 2, got %d", len(got.Messages))
	}
}

func TestSessionStore_Save_Update(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sess := buildSession("sess-2", "user-1")

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	// 更新消息与元数据
	sess.Messages = append(sess.Messages, message.Message{Role: message.RoleUser, Content: "update"})
	sess.Metadata["status"] = "updated"
	sess.UpdatedAt = sess.UpdatedAt.Add(time.Second)

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("update Save: %v", err)
	}

	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if len(got.Messages) != 3 {
		t.Errorf("Messages len after update: want 3, got %d", len(got.Messages))
	}
	if got.Metadata["status"] != "updated" {
		t.Errorf("Metadata[status]: want %q, got %q", "updated", got.Metadata["status"])
	}
}

func TestSessionStore_Save_NilSession(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.Save(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil session, got nil")
	}
}

// ==================== Get ====================

func TestSessionStore_Get_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("want ErrSessionNotFound, got %v", err)
	}
}

func TestSessionStore_Get_MessagesRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sess := buildSession("sess-3", "user-2")
	sess.Messages = []message.Message{
		{Role: message.RoleUser, Content: "ping"},
		{Role: message.RoleAssistant, Content: "pong", ToolCalls: []message.ToolCall{
			{ID: "tc-1", Type: "function", Function: message.FunctionCall{Name: "search", Arguments: `{"q":"go"}`}},
		}},
	}

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if len(got.Messages) != 2 {
		t.Fatalf("Messages len: want 2, got %d", len(got.Messages))
	}
	toolMsg := got.Messages[1]
	if len(toolMsg.ToolCalls) != 1 || toolMsg.ToolCalls[0].ID != "tc-1" {
		t.Errorf("ToolCall not preserved: %+v", toolMsg.ToolCalls)
	}
}

// ==================== Delete ====================

func TestSessionStore_Delete_Success(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sess := buildSession("sess-4", "user-3")

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get(ctx, sess.ID)
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("after Delete: want ErrSessionNotFound, got %v", err)
	}
}

func TestSessionStore_Delete_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.Delete(ctx, "ghost")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Errorf("want ErrSessionNotFound, got %v", err)
	}
}

// ==================== List ====================

func TestSessionStore_List_MultipleUsers(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// user-A 有 2 个会话，user-B 有 1 个
	for _, id := range []string{"a1", "a2"} {
		if err := store.Save(ctx, buildSession(id, "user-A")); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}
	if err := store.Save(ctx, buildSession("b1", "user-B")); err != nil {
		t.Fatalf("Save b1: %v", err)
	}

	listA, err := store.List(ctx, "user-A")
	if err != nil {
		t.Fatalf("List user-A: %v", err)
	}
	if len(listA) != 2 {
		t.Errorf("user-A session count: want 2, got %d", len(listA))
	}

	listB, err := store.List(ctx, "user-B")
	if err != nil {
		t.Fatalf("List user-B: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("user-B session count: want 1, got %d", len(listB))
	}
}

func TestSessionStore_List_NoMessages(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sess := buildSession("sess-5", "user-C")

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	list, err := store.List(ctx, "user-C")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 session, got %d", len(list))
	}
	// List 接口约定：不携带历史消息
	if list[0].Messages != nil {
		t.Errorf("List should not return messages, got %v", list[0].Messages)
	}
}

func TestSessionStore_List_Empty(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	list, err := store.List(ctx, "nobody")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("want empty list, got %d", len(list))
	}
}
