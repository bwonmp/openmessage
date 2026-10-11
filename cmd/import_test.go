package cmd

import (
	"strings"
	"testing"

	"github.com/maxghenis/openmessage/internal/importer"
	"github.com/rs/zerolog"
)

func TestRunImport(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("OPENMESSAGES_DATA_DIR", dataDir)
	t.Setenv("OPENMESSAGES_DEMO", "0")
	t.Setenv("OPENMESSAGES_APP_SANDBOX", "1")
	t.Setenv("OPENMESSAGES_V2_PRIMARY", "0")
	t.Setenv("OPENMESSAGES_V2_SEND", "0")
	t.Setenv("OPENMESSAGES_V2_INGEST", "0")

	logger := zerolog.Nop()

	tests := []struct {
		name      string
		source    string
		args      []string
		wantError string
	}{
		{
			name:      "unknown source",
			source:    "unknown-source",
			args:      []string{},
			wantError: "unknown import source: unknown-source\nSupported: gchat, gchat-conversation, imessage, whatsapp, signal",
		},
		{
			name:      "gchat missing args",
			source:    "gchat",
			args:      []string{},
			wantError: "usage: openmessage import gchat <path-to-groups-dir> [--email your@email.com]",
		},
		{
			name:      "gchat-conversation missing args",
			source:    "gchat-conversation",
			args:      []string{},
			wantError: "usage: openmessage import gchat-conversation <path-to-messages.json> [--email your@email.com]",
		},
		{
			name:      "gchat-conversation missing file",
			source:    "gchat-conversation",
			args:      []string{"non-existent-file.json"},
			wantError: "open file: open non-existent-file.json: no such file or directory",
		},
		{
			name:      "whatsapp text export missing file",
			source:    "whatsapp",
			args:      []string{"non-existent-file.txt"},
			wantError: "open file: open non-existent-file.txt: no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunImport(logger, tt.source, tt.args)
			if tt.wantError == "" {
				if err != nil {
					t.Errorf("RunImport() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("RunImport() error = nil, want %v", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("RunImport() error = %v, want substring %v", err, tt.wantError)
				}
			}
		})
	}
}

func TestPrintResult(t *testing.T) {
	// Simple invocation to ensure it doesn't panic
	printResult("test", &importer.ImportResult{
		ConversationsCreated: 1,
		MessagesImported:     2,
		MessagesDuplicate:    3,
		Errors:               []string{"test error"},
	})
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{"empty args", []string{}, "--test", false},
		{"flag not present", []string{"--foo", "--bar"}, "--test", false},
		{"flag present alone", []string{"--test"}, "--test", true},
		{"flag present among others", []string{"--foo", "--test", "--bar"}, "--test", true},
		{"flag is prefix of another", []string{"--test-foo"}, "--test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasFlag(tt.args, tt.flag); got != tt.want {
				t.Errorf("hasFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlagValue(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want string
	}{
		{"empty args", []string{}, "--test", ""},
		{"flag not present", []string{"--foo", "bar"}, "--test", ""},
		{"flag present without value", []string{"--test"}, "--test", ""},
		{"flag present with space value", []string{"--test", "value"}, "--test", "value"},
		{"flag present with equal value", []string{"--test=value"}, "--test", "value"},
		{"flag present multiple times takes first with space", []string{"--test", "val1", "--test=val2"}, "--test", "val1"},
		{"flag present multiple times takes first with equal", []string{"--test=val1", "--test", "val2"}, "--test", "val1"},
		{"flag is prefix of another", []string{"--test-foo=value"}, "--test", ""},
		{"flag equal value with equal sign", []string{"--test=val=ue"}, "--test", "val=ue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flagValue(tt.args, tt.flag); got != tt.want {
				t.Errorf("flagValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
