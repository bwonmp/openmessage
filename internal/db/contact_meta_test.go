package db

import (
	"reflect"
	"testing"
)

func TestPersonKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"simple", "Alice", "alice"},
		{"mixed case", "bOb", "bob"},
		{"spaces", " Charlie  Brown ", "charlie brown"},
		{"special chars", "David (Work) 123", "david work 123"},
		{"punctuation", "Eve, O'Connor!", "eve o connor"},
		{"multiple spaces internal", "Frank   Sinatra", "frank sinatra"},
		{"only special chars", "!!??()", ""},
		{"numbers only", "123 456", "123 456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PersonKey(tt.input); got != tt.expected {
				t.Errorf("PersonKey(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContactMetaStore(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	// Initial Get
	t.Run("Get empty", func(t *testing.T) {
		meta, err := store.GetContactMeta("alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.PersonKey != "alice" {
			t.Errorf("expected key 'alice', got %q", meta.PersonKey)
		}
		if len(meta.Tags) != 0 {
			t.Errorf("expected empty tags, got %v", meta.Tags)
		}
	})

	// Set and Get Tags
	t.Run("Set and Get Tags", func(t *testing.T) {
		tags := []string{"friend", "work", "Friend", ""}
		err := store.SetContactTags("alice", "Alice Smith", tags)
		if err != nil {
			t.Fatalf("SetContactTags failed: %v", err)
		}

		meta, err := store.GetContactMeta("alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.DisplayName != "Alice Smith" {
			t.Errorf("expected DisplayName 'Alice Smith', got %q", meta.DisplayName)
		}
		expectedTags := []string{"friend", "work"}
		if !reflect.DeepEqual(meta.Tags, expectedTags) {
			t.Errorf("expected tags %v, got %v", expectedTags, meta.Tags)
		}
	})

	// Set ReachOut
	t.Run("Set ReachOut", func(t *testing.T) {
		err := store.SetContactReachOut("alice", "", 30)
		if err != nil {
			t.Fatalf("SetContactReachOut failed: %v", err)
		}

		meta, err := store.GetContactMeta("alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.ReachOutDays != 30 {
			t.Errorf("expected ReachOutDays 30, got %d", meta.ReachOutDays)
		}
		// ensure display name wasn't overwritten by empty string
		if meta.DisplayName != "Alice Smith" {
			t.Errorf("expected DisplayName 'Alice Smith', got %q", meta.DisplayName)
		}

		// Set negative days
		err = store.SetContactReachOut("bob", "Bob Jones", -5)
		if err != nil {
			t.Fatalf("SetContactReachOut failed: %v", err)
		}
		metaBob, _ := store.GetContactMeta("bob")
		if metaBob.ReachOutDays != 0 {
			t.Errorf("expected ReachOutDays 0 for negative input, got %d", metaBob.ReachOutDays)
		}
	})

	// Set Summary
	t.Run("Set Summary", func(t *testing.T) {
		summaryTime := int64(1234567890)
		err := store.SetContactSummary("alice", "Alice Smith New", "Met at a conference.", summaryTime)
		if err != nil {
			t.Fatalf("SetContactSummary failed: %v", err)
		}

		meta, err := store.GetContactMeta("alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.Summary != "Met at a conference." {
			t.Errorf("expected Summary 'Met at a conference.', got %q", meta.Summary)
		}
		if meta.SummaryAt != summaryTime {
			t.Errorf("expected SummaryAt %d, got %d", summaryTime, meta.SummaryAt)
		}
		if meta.DisplayName != "Alice Smith New" {
			t.Errorf("expected DisplayName 'Alice Smith New', got %q", meta.DisplayName)
		}
	})

	// GetContactMetaMap
	t.Run("GetContactMetaMap", func(t *testing.T) {
		m, err := store.GetContactMetaMap()
		if err != nil {
			t.Fatalf("GetContactMetaMap failed: %v", err)
		}

		if len(m) != 2 {
			t.Errorf("expected map length 2, got %d", len(m))
		}

		aliceMeta, ok := m["alice"]
		if !ok {
			t.Fatalf("expected to find 'alice' in map")
		}
		if aliceMeta.DisplayName != "Alice Smith New" || aliceMeta.Summary != "Met at a conference." {
			t.Errorf("unexpected data for alice in map: %+v", aliceMeta)
		}
		expectedTags := []string{"friend", "work"}
		if !reflect.DeepEqual(aliceMeta.Tags, expectedTags) {
			t.Errorf("expected tags %v, got %v", expectedTags, aliceMeta.Tags)
		}

		bobMeta, ok := m["bob"]
		if !ok {
			t.Fatalf("expected to find 'bob' in map")
		}
		if bobMeta.DisplayName != "Bob Jones" {
			t.Errorf("unexpected data for bob in map: %+v", bobMeta)
		}
	})
}
