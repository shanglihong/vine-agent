package sqlite

import (
	"context"
	"testing"

	"vine-agent/domain/message"
)

func newTestRetrievalStore(t *testing.T) *RetrievalStore {
	t.Helper()
	return newRetrievalStoreWithDB(testDB)
}

func TestRetrievalStore_SaveAndSearch(t *testing.T) {
	store := newTestRetrievalStore(t)
	ctx := context.Background()

	msgs := []struct {
		sessionID string
		userID    string
		msg       message.Message
	}{
		{
			sessionID: "session-1",
			userID:    "user-1",
			msg:       message.Message{Role: message.RoleUser, Content: "Go concurrency model is very powerful"},
		},
		{
			sessionID: "session-1",
			userID:    "user-1",
			msg:       message.Message{Role: message.RoleAssistant, Content: "Yes, it is based on CSP model and goroutines"},
		},
		{
			sessionID: "session-2",
			userID:    "user-1",
			msg:       message.Message{Role: message.RoleUser, Content: "How to write a full text search in Go?"},
		},
		{
			sessionID: "session-3",
			userID:    "user-2",
			msg:       message.Message{Role: message.RoleUser, Content: "Today is a sunny weather"},
		},
	}

	for _, tc := range msgs {
		err := store.Save(ctx, tc.sessionID, tc.userID, tc.msg)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	t.Run("SearchSession - 限制在单会话内检索", func(t *testing.T) {
		res, err := store.SearchSession(ctx, "session-1", "model", 10)
		if err != nil {
			t.Fatalf("SearchSession failed: %v", err)
		}
		if len(res) != 2 {
			t.Errorf("expected 2 matches, got %d", len(res))
		}
		
		for _, msg := range res {
			if msg.Content == "" {
				t.Error("expected non-empty content")
			}
		}

		res, err = store.SearchSession(ctx, "session-1", "search", 10)
		if err != nil {
			t.Fatalf("SearchSession failed: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("expected 0 matches, got %d", len(res))
		}
	})

	t.Run("SearchUser - 跨会话检索指定用户的所有消息", func(t *testing.T) {
		res, err := store.SearchUser(ctx, "user-1", "Go", 10)
		if err != nil {
			t.Fatalf("SearchUser failed: %v", err)
		}
		if len(res) != 2 {
			t.Errorf("expected 2 matches, got %d", len(res))
		}

		expectedSessions := map[string]bool{"session-1": true, "session-2": true}
		for _, item := range res {
			if !expectedSessions[item.SessionID] {
				t.Errorf("unexpected sessionID: %s", item.SessionID)
			}
		}
	})

	t.Run("DeleteBySession - 清除指定会话的所有检索数据", func(t *testing.T) {
		err := store.DeleteBySession(ctx, "session-1")
		if err != nil {
			t.Fatalf("DeleteBySession failed: %v", err)
		}

		res, err := store.SearchSession(ctx, "session-1", "model", 10)
		if err != nil {
			t.Fatalf("SearchSession failed: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("expected 0 matches after deletion, got %d", len(res))
		}

		res, err = store.SearchSession(ctx, "session-2", "search", 10)
		if err != nil {
			t.Fatalf("SearchSession failed: %v", err)
		}
		if len(res) != 1 {
			t.Errorf("expected session-2 match to still exist, got %d", len(res))
		}
	})
}
