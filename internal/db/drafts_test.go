package db

import (
	"reflect"
	"testing"
)

func TestDrafts(t *testing.T) {
	store := newTestStore(t)

	// We seed a conversation because drafts have a foreign key to conversation_id
	// Actually, wait, do they? Let's assume we might need a conversation if there's an enforced FK,
	// but let's check first if sqlite defaults to enforcing FKs. SQLite usually doesn't unless PRAGMA foreign_keys = ON;
	// But just in case, we will upsert a conversation.
	if err := store.UpsertConversation(&Conversation{ConversationID: "c1", LastMessageTS: 1000}); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}

	t.Run("UpsertDraft", func(t *testing.T) {
		tests := []struct {
			name    string
			draft   *Draft
			wantErr bool
		}{
			{
				name: "insert new draft",
				draft: &Draft{
					DraftID:        "d1",
					ConversationID: "c1",
					Body:           "hello",
					CreatedAt:      1000,
				},
				wantErr: false,
			},
			{
				name: "update existing draft",
				draft: &Draft{
					DraftID:        "d1",
					ConversationID: "c1",
					Body:           "hello updated",
					CreatedAt:      2000,
				},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := store.UpsertDraft(tt.draft)
				if (err != nil) != tt.wantErr {
					t.Errorf("UpsertDraft() error = %v, wantErr %v", err, tt.wantErr)
				}

				if err == nil {
					saved, err := store.GetDraft(tt.draft.DraftID)
					if err != nil {
						t.Fatalf("GetDraft() after upsert error = %v", err)
					}
					if !reflect.DeepEqual(saved, tt.draft) {
						t.Errorf("UpsertDraft() saved = %v, want %v", saved, tt.draft)
					}
				}
			})
		}
	})

	t.Run("GetDraft", func(t *testing.T) {
		tests := []struct {
			name    string
			draftID string
			want    *Draft
			wantErr bool
		}{
			{
				name:    "existing draft",
				draftID: "d1",
				want: &Draft{
					DraftID:        "d1",
					ConversationID: "c1",
					Body:           "hello updated",
					CreatedAt:      2000,
				},
				wantErr: false,
			},
			{
				name:    "non-existent draft",
				draftID: "d999",
				want:    nil,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := store.GetDraft(tt.draftID)
				if (err != nil) != tt.wantErr {
					t.Errorf("GetDraft() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("GetDraft() got = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("ListDrafts", func(t *testing.T) {
		// Insert another draft for c1
		if err := store.UpsertDraft(&Draft{
			DraftID:        "d2",
			ConversationID: "c1",
			Body:           "second draft",
			CreatedAt:      3000,
		}); err != nil {
			t.Fatalf("setup ListDrafts error: %v", err)
		}

		tests := []struct {
			name           string
			conversationID string
			wantCount      int
			wantErr        bool
		}{
			{
				name:           "existing conversation with drafts",
				conversationID: "c1",
				wantCount:      2,
				wantErr:        false,
			},
			{
				name:           "conversation with no drafts",
				conversationID: "c2",
				wantCount:      0,
				wantErr:        false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := store.ListDrafts(tt.conversationID)
				if (err != nil) != tt.wantErr {
					t.Errorf("ListDrafts() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if len(got) != tt.wantCount {
					t.Errorf("ListDrafts() got %d drafts, want %d", len(got), tt.wantCount)
				}

				if len(got) > 1 {
					// Check ordering (should be descending by created_at)
					if got[0].CreatedAt < got[1].CreatedAt {
						t.Errorf("ListDrafts() ordering is not descending by created_at")
					}
				}
			})
		}
	})

	t.Run("DeleteDraft", func(t *testing.T) {
		tests := []struct {
			name    string
			draftID string
			wantErr bool
		}{
			{
				name:    "delete existing draft",
				draftID: "d1",
				wantErr: false,
			},
			{
				name:    "delete non-existent draft",
				draftID: "d999",
				wantErr: false, // delete doesn't return error if no rows affected in this impl
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := store.DeleteDraft(tt.draftID)
				if (err != nil) != tt.wantErr {
					t.Errorf("DeleteDraft() error = %v, wantErr %v", err, tt.wantErr)
				}

				// verify deletion
				got, err := store.GetDraft(tt.draftID)
				if err != nil {
					t.Errorf("GetDraft() after DeleteDraft() error = %v", err)
				}
				if got != nil {
					t.Errorf("draft was not deleted")
				}
			})
		}
	})
}
