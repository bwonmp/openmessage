package cmd

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRunSendGroup(t *testing.T) {
	t.Setenv("OPENMESSAGES_DEMO", "1")
	t.Setenv("OPENMESSAGES_DATA_DIR", t.TempDir())
	t.Setenv("OPENMESSAGES_APP_SANDBOX", "1")

	logger := zerolog.Nop()

	tests := []struct {
		name    string
		phones  []string
		message string
		wantErr string
	}{
		{
			name:    "legacy send - sandbox environment",
			phones:  []string{"+15551234567", "+15557654321"},
			message: "hello group",
			// With sandbox environment, `a.LoadAndConnect()` in runLegacySendGroup will fail due to no paired session.
			wantErr: "connect: load session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunSendGroup(logger, tt.phones, tt.message)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("RunSendGroup() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("RunSendGroup() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("RunSendGroup() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}
