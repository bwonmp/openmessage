package app

import (
	"testing"
)

func TestRealGMClient(t *testing.T) {
	// A nil libgm.Client will cause panics when its methods are called.
	// We can use this to verify the wrapper simply delegates to it.
	client := newRealGMClient(nil)

	tests := []struct {
		name string
		call func()
	}{
		{
			name: "ListConversationsWithCursor",
			call: func() { client.ListConversationsWithCursor(0, 0, nil) },
		},
		{
			name: "FetchMessages",
			call: func() { client.FetchMessages("", 0, nil) },
		},
		{
			name: "GetOrCreateConversation",
			call: func() { client.GetOrCreateConversation(nil) },
		},
		{
			name: "ListContacts",
			call: func() { client.ListContacts() },
		},
		{
			name: "GetParticipantThumbnail",
			call: func() { client.GetParticipantThumbnail("a") },
		},
		{
			name: "GetContactThumbnail",
			call: func() { client.GetContactThumbnail("a") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()
			tt.call()
		})
	}
}
