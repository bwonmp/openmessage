package db

import (
	"reflect"
	"testing"
)

func TestListMessagedPeople(t *testing.T) {
	store := newTestStore(t)

	// Set up conversations and messages for testing
	conv1 := &Conversation{
		ConversationID: "c1",
		Name:           "Alice",
		IsGroup:        false,
		Participants:   `[{"number":"+15551234567","is_me":false}]`,
		LastMessageTS:  1000,
		SourcePlatform: "sms",
	}
	store.UpsertConversation(conv1)

	msg1 := &Message{
		MessageID:      "m1",
		ConversationID: "c1",
		Body:           "hello",
		TimestampMS:    1000,
	}
	store.UpsertMessage(msg1)

	conv2 := &Conversation{
		ConversationID: "c2",
		Name:           "Alice", // Same name, should be grouped
		IsGroup:        false,
		Participants:   `[{"number":"+15551234568","is_me":false}]`,
		LastMessageTS:  2000,
		SourcePlatform: "gchat",
	}
	store.UpsertConversation(conv2)

	msg2 := &Message{
		MessageID:      "m2",
		ConversationID: "c2",
		Body:           "hi",
		TimestampMS:    2000,
	}
	store.UpsertMessage(msg2)

	conv3 := &Conversation{
		ConversationID: "c3",
		Name:           "Group", // Group conversation, should be skipped
		IsGroup:        true,
		Participants:   `[{"number":"+15551234569","is_me":false}]`,
		LastMessageTS:  1500,
		SourcePlatform: "sms",
	}
	store.UpsertConversation(conv3)

	conv4 := &Conversation{
		ConversationID: "c4",
		Name:           "", // No name, should use number
		IsGroup:        false,
		Participants:   `[{"number":"+15551234560","is_me":false}]`,
		LastMessageTS:  3000,
		SourcePlatform: "imessage",
	}
	store.UpsertConversation(conv4)

	msg4 := &Message{
		MessageID:      "m4",
		ConversationID: "c4",
		Body:           "hey",
		TimestampMS:    3000,
	}
	store.UpsertMessage(msg4)


	tests := []struct {
		name    string
		want    []*Person
		wantErr bool
	}{
		{
			name: "list messaged people",
			want: []*Person{
				{
					Key:             "15551234560",
					Name:            "+15551234560",
					Numbers:         []string{"+15551234560"},
					Platforms:       []string{"imessage"},
					LastContactedTS: 3000,
					MessageCount:    1,
					ConversationIDs: []string{"c4"},
				},
				{
					Key:             "alice",
					Name:            "Alice",
					Numbers:         []string{"+15551234568", "+15551234567"},
					Platforms:       []string{"gchat", "sms"},
					LastContactedTS: 2000,
					MessageCount:    2, // From both c1 and c2
					ConversationIDs: []string{"c2", "c1"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.ListMessagedPeople()
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.ListMessagedPeople() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Store.ListMessagedPeople() returned %d people, want %d", len(got), len(tt.want))
			} else {
				for i, p := range got {
					if p.Key != tt.want[i].Key || p.Name != tt.want[i].Name ||
						p.LastContactedTS != tt.want[i].LastContactedTS || p.MessageCount != tt.want[i].MessageCount ||
						!reflect.DeepEqual(p.Numbers, tt.want[i].Numbers) || !reflect.DeepEqual(p.Platforms, tt.want[i].Platforms) {
						t.Errorf("Store.ListMessagedPeople() person %d = %+v, want %+v", i, p, tt.want[i])
					}
				}
			}
		})
	}
}

func TestPersonByKey(t *testing.T) {
	store := newTestStore(t)

	conv := &Conversation{
		ConversationID: "c1",
		Name:           "Bob",
		IsGroup:        false,
		Participants:   `[{"number":"+15551234567","is_me":false}]`,
		LastMessageTS:  1000,
	}
	store.UpsertConversation(conv)

	tests := []struct {
		name    string
		key     string
		wantKey string // empty if we expect nil
		wantErr bool
	}{
		{
			name:    "found",
			key:     "bob",
			wantKey: "bob",
			wantErr: false,
		},
		{
			name:    "not found",
			key:     "nonexistent",
			wantKey: "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.PersonByKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.PersonByKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantKey == "" {
				if got != nil {
					t.Errorf("Store.PersonByKey() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Fatalf("Store.PersonByKey() returned nil, want non-nil person")
				}
				if got.Key != tt.wantKey {
					t.Errorf("Store.PersonByKey() key = %v, want %v", got.Key, tt.wantKey)
				}
			}
		})
	}
}

func TestPersonMessages(t *testing.T) {
	store := newTestStore(t)

	conv1 := &Conversation{
		ConversationID: "c1",
	}
	store.UpsertConversation(conv1)

	conv2 := &Conversation{
		ConversationID: "c2",
	}
	store.UpsertConversation(conv2)

	// Msg 1 in c1
	msg1 := &Message{
		MessageID:      "m1",
		ConversationID: "c1",
		Body:           "hello",
		TimestampMS:    1000,
		IsFromMe:       false,
	}
	store.UpsertMessage(msg1)

	// Msg 2 in c2, exact duplicate of msg1, within 2s
	msg2 := &Message{
		MessageID:      "m2",
		ConversationID: "c2",
		Body:           "hello",
		TimestampMS:    2500, // +1500ms
		IsFromMe:       false,
	}
	store.UpsertMessage(msg2)

	// Msg 3 in c1, beyond 2s
	msg3 := &Message{
		MessageID:      "m3",
		ConversationID: "c1",
		Body:           "hello",
		TimestampMS:    3500, // +2500ms from m1, +1000 from m2
		IsFromMe:       false,
	}
	store.UpsertMessage(msg3)

	// Msg 4 in c2, completely different
	msg4 := &Message{
		MessageID:      "m4",
		ConversationID: "c2",
		Body:           "how are you",
		TimestampMS:    4000,
		IsFromMe:       true,
	}
	store.UpsertMessage(msg4)


	tests := []struct {
		name    string
		convIDs []string
		wantIDs []string
		wantErr bool
	}{
		{
			name:    "empty conversation IDs",
			convIDs: nil,
			wantIDs: nil,
			wantErr: false,
		},
		{
			name:    "messages from one conversation",
			convIDs: []string{"c2"},
			wantIDs: []string{"m2", "m4"}, // m2, m4
			wantErr: false,
		},
		{
			name:    "messages from multiple conversations with deduplication",
			convIDs: []string{"c1", "c2"},
			wantIDs: []string{"m1", "m3", "m4"}, // m2 should be deduplicated because it is a duplicate of m1 (same body, sender, <2s)
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.PersonMessages(tt.convIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.PersonMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.wantIDs) {
				t.Errorf("Store.PersonMessages() returned %d messages, want %d", len(got), len(tt.wantIDs))
				for i, m := range got {
					t.Logf("  got[%d]: id=%s ts=%d body=%q", i, m.MessageID, m.TimestampMS, m.Body)
				}
			} else {
				for i, m := range got {
					if m.MessageID != tt.wantIDs[i] {
						t.Errorf("Store.PersonMessages() message %d ID = %s, want %s", i, m.MessageID, tt.wantIDs[i])
					}
				}
			}
		})
	}
}
