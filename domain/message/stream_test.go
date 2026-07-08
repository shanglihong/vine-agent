package message_test

import (
	"errors"
	"io"
	"testing"

	"vine-agent/domain/message"

	"github.com/stretchr/testify/assert"
)

// mockStreamReader 模拟 StreamMessageReader
type mockStreamReader struct {
	messages []*message.StreamMessage
	err      error
	idx      int
}

func (m *mockStreamReader) Recv() (*message.StreamMessage, error) {
	if m.idx >= len(m.messages) {
		if m.err != nil {
			return nil, m.err
		}
		return nil, io.EOF
	}
	msg := m.messages[m.idx]
	m.idx++
	return msg, nil
}

func (m *mockStreamReader) Close() error {
	return nil
}

func (m *mockStreamReader) Interrupt() error {
	return m.Close()
}

func TestReadAndAssembleMessage_WithError(t *testing.T) {
	t.Run("returns partial message on stream error", func(t *testing.T) {
		stream := &mockStreamReader{
			messages: []*message.StreamMessage{
				{Type: message.StreamMessageTextDelta, Content: "Hello "},
				{Type: message.StreamMessageTextDelta, Content: "world"},
			},
			err: errors.New("stream closed unexpectedly"),
		}

		msg, err := message.ReadAndAssembleMessage(stream, func(m *message.StreamMessage) {})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stream closed unexpectedly")
		assert.NotNil(t, msg)
		assert.Equal(t, message.RoleAssistant, msg.Role)
		assert.Equal(t, "Hello world", msg.Content)
	})

	t.Run("returns completed message on EOF", func(t *testing.T) {
		stream := &mockStreamReader{
			messages: []*message.StreamMessage{
				{Type: message.StreamMessageTextDelta, Content: "Hello "},
				{Type: message.StreamMessageTextDelta, Content: "world"},
			},
			err: nil,
		}

		msg, err := message.ReadAndAssembleMessage(stream, func(m *message.StreamMessage) {})
		assert.NoError(t, err)
		assert.NotNil(t, msg)
		assert.Equal(t, "Hello world", msg.Content)
	})
}
