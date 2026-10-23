package db

import (
	"testing"
)

func TestCreateTab(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	tests := []struct {
		name        string
		tabName     string
		wantErr     bool
		wantName    string
		wantPos     int
	}{
		{
			name:     "valid name",
			tabName:  "Work",
			wantErr:  false,
			wantName: "Work",
			wantPos:  0,
		},
		{
			name:     "trim whitespace",
			tabName:  "  Personal  ",
			wantErr:  false,
			wantName: "Personal",
			wantPos:  1,
		},
		{
			name:    "empty name",
			tabName: "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tab, err := store.CreateTab(tt.tabName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CreateTab() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if tab.Name != tt.wantName {
					t.Errorf("CreateTab() name = %v, want %v", tab.Name, tt.wantName)
				}
				if tab.Position != tt.wantPos {
					t.Errorf("CreateTab() position = %v, want %v", tab.Position, tt.wantPos)
				}
				if tab.TabID == "" {
					t.Errorf("CreateTab() TabID is empty")
				}
				if tab.CreatedAt == 0 {
					t.Errorf("CreateTab() CreatedAt is 0")
				}
			}
		})
	}
}

func TestListTabs(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	// Initially empty
	tabs, err := store.ListTabs()
	if err != nil {
		t.Fatalf("ListTabs() error = %v", err)
	}
	if len(tabs) != 0 {
		t.Errorf("ListTabs() initial len = %v, want 0", len(tabs))
	}

	tab1, err := store.CreateTab("Tab1")
	if err != nil {
		t.Fatal(err)
	}
	tab2, err := store.CreateTab("Tab2")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		want []string
	}{
		{
			name: "list all tabs",
			want: []string{tab1.Name, tab2.Name},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tabs, err := store.ListTabs()
			if err != nil {
				t.Fatalf("ListTabs() error = %v", err)
			}
			if len(tabs) != len(tt.want) {
				t.Fatalf("ListTabs() len = %v, want %v", len(tabs), len(tt.want))
			}
			for i, w := range tt.want {
				if tabs[i].Name != w {
					t.Errorf("ListTabs()[%d].Name = %v, want %v", i, tabs[i].Name, w)
				}
			}
		})
	}
}

func TestRenameTab(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	tab, err := store.CreateTab("OldName")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		tabID   string
		newName string
		wantErr bool
		verify  string
	}{
		{
			name:    "valid rename",
			tabID:   tab.TabID,
			newName: "NewName",
			wantErr: false,
			verify:  "NewName",
		},
		{
			name:    "trim whitespace",
			tabID:   tab.TabID,
			newName: "  TrimmedName  ",
			wantErr: false,
			verify:  "TrimmedName",
		},
		{
			name:    "empty name error",
			tabID:   tab.TabID,
			newName: "   ",
			wantErr: true,
			verify:  "TrimmedName", // name shouldn't change
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.RenameTab(tt.tabID, tt.newName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RenameTab() error = %v, wantErr %v", err, tt.wantErr)
			}

			tabs, err := store.ListTabs()
			if err != nil {
				t.Fatalf("ListTabs() error = %v", err)
			}
			if len(tabs) == 0 {
				t.Fatalf("ListTabs() returned empty")
			}
			if tabs[0].Name != tt.verify {
				t.Errorf("After RenameTab(), name = %v, want %v", tabs[0].Name, tt.verify)
			}
		})
	}
}

func TestDeleteTab(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	tab, err := store.CreateTab("ToDelete")
	if err != nil {
		t.Fatal(err)
	}

	// Insert a conversation with this tab
	convID := "conv1"
	if _, err := store.db.Exec(`INSERT INTO conversations (conversation_id, name, unread_count, notification_mode, tab) VALUES (?, ?, ?, ?, ?)`,
		convID, "Chat", 0, "all", tab.TabID); err != nil {
		t.Fatalf("failed to insert conversation: %v", err)
	}

	tests := []struct {
		name       string
		tabID      string
		wantErr    bool
		convWantTab string
	}{
		{
			name:       "delete tab",
			tabID:      tab.TabID,
			wantErr:    false,
			convWantTab: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.DeleteTab(tt.tabID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DeleteTab() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify tab is gone
			tabs, err := store.ListTabs()
			if err != nil {
				t.Fatalf("ListTabs() error = %v", err)
			}
			for _, t2 := range tabs {
				if t2.TabID == tt.tabID {
					t.Errorf("Tab %v was not deleted", tt.tabID)
				}
			}

			// Verify conversation is updated
			var currentTab string
			if err := store.db.QueryRow(`SELECT tab FROM conversations WHERE conversation_id = ?`, convID).Scan(&currentTab); err != nil {
				t.Fatalf("failed to query conversation: %v", err)
			}
			if currentTab != tt.convWantTab {
				t.Errorf("Conversation tab = %v, want %v", currentTab, tt.convWantTab)
			}
		})
	}
}
