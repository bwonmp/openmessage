package cmd

import (
	"testing"
	"github.com/rs/zerolog"
)

func TestRunDebugMedia(t *testing.T) {
	tests := []struct {
		name    string
		convID  string
		wantErr bool
	}{
		{
			name:    "empty convID",
			convID:  "",
			wantErr: true,
		},
		{
			name:    "non-empty convID",
			convID:  "conv123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			err := RunDebugMedia(logger, tt.convID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunDebugMedia() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
