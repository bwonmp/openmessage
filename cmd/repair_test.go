package cmd

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRunRepair(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "no args",
			args:    []string{},
			wantErr: "repair: unknown target",
		},
		{
			name:    "unknown target",
			args:    []string{"unknown-target"},
			wantErr: "repair: unknown target",
		},
		{
			name:    "google-idspace missing since",
			args:    []string{"google-idspace"},
			wantErr: "--since is required (RFC3339 or unix milliseconds)",
		},
		{
			name:    "google-idspace invalid since",
			args:    []string{"google-idspace", "--since", "invalid"},
			wantErr: "is neither unix milliseconds nor RFC3339",
		},
		{
			name:    "google-idspace negative since",
			args:    []string{"google-idspace", "--since", "-1"},
			wantErr: "--since must be positive",
		},
		{
			name:    "help",
			args:    []string{"google-idspace", "--help"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunRepair(logger, tt.args...)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("RunRepair() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("RunRepair() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("RunRepair() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}
