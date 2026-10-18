package app

import (
	"errors"
	"testing"
	"time"

	"github.com/maxghenis/openmessage/internal/client"
	"github.com/maxghenis/openmessage/internal/db"
	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
)

type mockReadCursorMarker struct {
	err error
	convID string
	msgID string
}

func (m *mockReadCursorMarker) MarkRead(conversationID, messageID string) error {
	m.convID = conversationID
	m.msgID = messageID
	return m.err
}

func testAppMarkRead(t *testing.T) (*App, *db.Store) {
	t.Helper()
	store, err := db.New(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	a := &App{
		Store:  store,
		Logger: zerolog.Nop(),
	}
	return a, store
}

func TestSyncConversationReadWith(t *testing.T) {
	_, store := testAppMarkRead(t)

	// Seed data
	err := store.UpsertConversation(&db.Conversation{
		ConversationID: "conv-rcs",
		SourcePlatform: "sms",
		UnreadCount: 1,
	})
	if err != nil { t.Fatalf("seed conv: %v", err) }

	err = store.UpsertConversation(&db.Conversation{
		ConversationID: "conv-sms",
		SourcePlatform: "sms",
		UnreadCount: 1,
	})
	if err != nil { t.Fatalf("seed conv: %v", err) }

	err = store.UpsertConversation(&db.Conversation{
		ConversationID: "conv-whatsapp",
		SourcePlatform: "whatsapp",
		UnreadCount: 1,
	})
	if err != nil { t.Fatalf("seed conv: %v", err) }

	err = store.UpsertMessage(&db.Message{
		ConversationID: "conv-rcs",
		MessageID: "msg-1",
		IsFromMe: false,
		TimestampMS: time.Now().UnixMilli(),
	})
	if err != nil { t.Fatalf("seed msg: %v", err) }

	err = store.UpsertMessage(&db.Message{
		ConversationID: "conv-rcs",
		MessageID: "msg-2",
		IsFromMe: true, // Should be skipped
		TimestampMS: time.Now().UnixMilli() + 1000,
	})
	if err != nil { t.Fatalf("seed msg: %v", err) }

	tests := []struct {
		name           string
		convID         string
		msgID          string
		remote         *mockReadCursorMarker
		wantMsgID      string
		wantSynced     bool
		wantRemoteSkip string
		wantErrMsg     string
		wantRemoteErr  error
	}{
		{
			name:       "empty conversation id",
			convID:     "",
			wantErrMsg: "conversation_id is required",
		},
		{
			name:           "whatsapp conversation skipped remote",
			convID:         "conv-whatsapp",
			wantRemoteSkip: "marking read on the source platform is not supported for whatsapp yet; cleared locally only",
		},
		{
			name:           "sms conversation no remote",
			convID:         "conv-sms",
			remote:         nil,
			wantRemoteSkip: ErrNotConnected + "; cleared locally only",
		},
		{
			name:           "rcs conversation custom msg id",
			convID:         "conv-rcs",
			msgID:          "custom-msg",
			remote:         &mockReadCursorMarker{},
			wantMsgID:      "custom-msg",
			wantSynced:     true,
		},
		{
			name:           "rcs conversation derived msg id",
			convID:         "conv-rcs",
			msgID:          "",
			remote:         &mockReadCursorMarker{},
			wantMsgID:      "msg-1", // Should skip msg-2 because IsFromMe=true
			wantSynced:     true,
		},
		{
			name:           "rcs conversation remote error",
			convID:         "conv-rcs",
			msgID:          "msg-1",
			remote:         &mockReadCursorMarker{err: errors.New("remote error")},
			wantMsgID:      "msg-1",
			wantSynced:     false,
			wantRemoteErr:  errors.New("remote error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var remoteMarker ReadCursorMarker
			if tt.remote != nil {
				remoteMarker = tt.remote
			}

			res, err := SyncConversationReadWith(store, remoteMarker, tt.convID, tt.msgID)

			if tt.wantErrMsg != "" {
				if err == nil || err.Error() != tt.wantErrMsg {
					t.Errorf("SyncConversationReadWith() error = %v, want %v", err, tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("SyncConversationReadWith() unexpected error = %v", err)
			}

			if res.MessageID != tt.wantMsgID {
				t.Errorf("MessageID = %v, want %v", res.MessageID, tt.wantMsgID)
			}
			if res.RemoteSynced != tt.wantSynced {
				t.Errorf("RemoteSynced = %v, want %v", res.RemoteSynced, tt.wantSynced)
			}
			if res.RemoteSkipped != tt.wantRemoteSkip {
				t.Errorf("RemoteSkipped = %v, want %v", res.RemoteSkipped, tt.wantRemoteSkip)
			}
			if tt.wantRemoteErr != nil && res.RemoteErr.Error() != tt.wantRemoteErr.Error() {
				t.Errorf("RemoteErr = %v, want %v", res.RemoteErr, tt.wantRemoteErr)
			}
			if tt.wantRemoteErr == nil && res.RemoteErr != nil {
				t.Errorf("RemoteErr = %v, want nil", res.RemoteErr)
			}

			if tt.remote != nil && res.RemoteErr == nil && tt.wantRemoteSkip == "" {
				if tt.remote.convID != tt.convID {
					t.Errorf("Mock remote called with ConvID = %v, want %v", tt.remote.convID, tt.convID)
				}
				if tt.remote.msgID != res.MessageID {
					t.Errorf("Mock remote called with MsgID = %v, want %v", tt.remote.msgID, res.MessageID)
				}
			}
		})
	}

	t.Run("nil store", func(t *testing.T) {
		_, err := SyncConversationReadWith(nil, nil, "conv-rcs", "")
		if err == nil || err.Error() != "no store available" {
			t.Errorf("SyncConversationReadWith(nil store) error = %v, want 'no store available'", err)
		}
	})
}

func TestSyncConversationRead(t *testing.T) {
	app, store := testAppMarkRead(t)

	err := store.UpsertConversation(&db.Conversation{
		ConversationID: "conv-1",
		SourcePlatform: "sms",
	})
	if err != nil { t.Fatalf("seed conv: %v", err) }

	tests := []struct {
		name          string
		remoteErr     error
		wantDisconn   bool
	}{
		{
			name:        "success",
			remoteErr:   nil,
			wantDisconn: false,
		},
		{
			name:        "other error",
			remoteErr:   errors.New("other error"),
			wantDisconn: false,
		},
		{
			name:        "auth expired error",
			remoteErr:   errors.New("session cookie expired"),
			wantDisconn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.Connected.Store(true)
			app.googleAuthExpired.Store(false)

			marker := &mockReadCursorMarker{err: tt.remoteErr}
			restore := SwapReadCursorForTest(marker)
			defer restore()

			res, err := app.SyncConversationRead("conv-1", "msg-1")
			if err != nil {
				t.Fatalf("SyncConversationRead() unexpected error = %v", err)
			}

			if tt.remoteErr != nil {
				if res.RemoteErr == nil || res.RemoteErr.Error() != tt.remoteErr.Error() {
					t.Errorf("RemoteErr = %v, want %v", res.RemoteErr, tt.remoteErr)
				}
			} else {
				if res.RemoteErr != nil {
					t.Errorf("RemoteErr = %v, want nil", res.RemoteErr)
				}
			}

			disconnected := !app.Connected.Load()
			if disconnected != tt.wantDisconn {
				t.Errorf("app disconnected = %v, want %v", disconnected, tt.wantDisconn)
			}
		})
	}
}

func TestSwapReadCursorForTest(t *testing.T) {
	marker := &mockReadCursorMarker{}
	restore := SwapReadCursorForTest(marker)

	res := resolveRemoteReadCursor(nil)
	if res != marker {
		t.Errorf("resolveRemoteReadCursor() = %v, want %v", res, marker)
	}

	restore()

	// It's hard to test function equality for the original, but we can verify it doesn't crash
	// when we give it a nil app if we call it now (it should return nil from RemoteReadCursor(nil))
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("resolveRemoteReadCursor panicked after restore: %v", r)
		}
	}()
	app := &App{}
	resolveRemoteReadCursor(app)
}

func TestRemoteReadCursor(t *testing.T) {
	tests := []struct {
		name string
		cli  *client.Client
		want ReadCursorMarker
	}{
		{
			name: "nil client",
			cli:  nil,
			want: nil,
		},
		{
			name: "client with nil gm",
			cli:  &client.Client{GM: nil},
			want: nil,
		},
		{
			name: "client with valid gm",
			cli:  &client.Client{GM: &libgm.Client{}},
			want: &libgm.Client{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoteReadCursor(tt.cli)

			if tt.want == nil {
				if got != nil {
					t.Errorf("RemoteReadCursor() = %v, want nil", got)
				}
			} else {
				// We can't do deep equal on unexported libgm.Client, but we can check it's not nil
				// and has the right type.
				if got == nil {
					t.Errorf("RemoteReadCursor() = nil, want non-nil")
				}
				if _, ok := got.(*libgm.Client); !ok {
					t.Errorf("RemoteReadCursor() = %T, want *libgm.Client", got)
				}
			}
		})
	}
}
