package app

import (
	"errors"
	"testing"
	"os"

	"github.com/maxghenis/openmessage/internal/client"
	"github.com/rs/zerolog"
)

func setupTestApp(t *testing.T) *App {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("OPENMESSAGES_DATA_DIR", tempDir)
	t.Setenv("OPENMESSAGES_DEMO", "1") // Ensure demo mode is on for fast setup
	t.Setenv("OPENMESSAGES_GOOGLE_AVATAR_SYNC", "0")

	a, err := New(zerolog.Nop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Create a dummy session file so GooglePaired() returns true
	if err := os.WriteFile(a.SessionPath, []byte("test"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Cleanup(func() {
		// Just clear the Client so it doesn't crash app.Close
		a.clientMu.Lock()
		a.Client = nil
		a.clientMu.Unlock()
		a.Close()
	})
	return a
}

func mockClient() *client.Client {
	return &client.Client{}
}

func TestApp_BeginGoogleGeneration(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "success"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()

			gen := a.BeginGoogleGeneration(cli)

			if gen == nil {
				t.Fatal("BeginGoogleGeneration returned nil")
			}
			if gen.app != a {
				t.Errorf("expected gen.app == a")
			}
			if gen.Client != cli {
				t.Errorf("expected gen.Client == cli")
			}
			if gen.Handler == nil {
				t.Errorf("expected gen.Handler != nil")
			}

			a.clientMu.RLock()
			defer a.clientMu.RUnlock()
			if a.Client != cli {
				t.Errorf("expected a.Client == cli")
			}
			if a.EventHandler != gen.Handler {
				t.Errorf("expected a.EventHandler == gen.Handler")
			}
			if a.googleGeneration != gen {
				t.Errorf("expected a.googleGeneration == gen")
			}
			if a.Connected.Load() {
				t.Errorf("expected Connected to be false")
			}
		})
	}
}

func TestGoogleGeneration_Ready(t *testing.T) {
	tests := []struct {
		name      string
		stale     bool
		wantReady bool
	}{
		{
			name:      "current generation",
			stale:     false,
			wantReady: true,
		},
		{
			name:      "stale generation",
			stale:     true,
			wantReady: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()
			gen := a.BeginGoogleGeneration(cli)

			if tt.stale {
				a.BeginGoogleGeneration(mockClient())
			}

			got := gen.Ready()
			if got != tt.wantReady {
				t.Errorf("Ready() = %v, want %v", got, tt.wantReady)
			}

			if tt.wantReady {
				if !a.Connected.Load() {
					t.Errorf("expected Connected to be true")
				}
				if a.GoogleStatus().LastError != "" {
					t.Errorf("expected empty LastError, got %q", a.GoogleStatus().LastError)
				}
			}
		})
	}
}

func TestGoogleGeneration_PhoneResponding(t *testing.T) {
	tests := []struct {
		name       string
		stale      bool
		responding bool
		wantError  bool
	}{
		{
			name:       "current responding",
			stale:      false,
			responding: true,
			wantError:  false,
		},
		{
			name:       "current not responding",
			stale:      false,
			responding: false,
			wantError:  true,
		},
		{
			name:       "stale not responding",
			stale:      true,
			responding: false,
			wantError:  false, // State should not change
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()
			gen := a.BeginGoogleGeneration(cli)

			if tt.stale {
				a.BeginGoogleGeneration(mockClient())
			}

			gen.PhoneResponding(tt.responding)

			status := a.GoogleStatus()
			if !tt.stale {
				if status.PhoneResponding != tt.responding {
					t.Errorf("PhoneResponding = %v, want %v", status.PhoneResponding, tt.responding)
				}
			}

			hasError := status.LastError != ""
			if !tt.stale && hasError != tt.wantError {
				t.Errorf("hasError = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

func TestGoogleGeneration_ConnectionLost(t *testing.T) {
	tests := []struct {
		name  string
		stale bool
	}{
		{name: "current generation", stale: false},
		{name: "stale generation", stale: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()
			gen := a.BeginGoogleGeneration(cli)

			// Simulate ready state to test connection loss
			gen.Ready()

			if tt.stale {
				a.BeginGoogleGeneration(mockClient())
				a.Connected.Store(true) // reset for the new generation
			}

			gen.ConnectionLost()

			if !tt.stale {
				if a.Connected.Load() {
					t.Errorf("expected Connected to be false")
				}
				if a.GoogleStatus().LastError == "" {
					t.Errorf("expected LastError to be set")
				}
			} else {
				// If stale, the state should remain as set above
				if !a.Connected.Load() {
					t.Errorf("expected Connected to remain true for stale generation")
				}
			}
		})
	}
}

func TestGoogleGeneration_AuthExpired(t *testing.T) {
	tests := []struct {
		name        string
		stale       bool
		err         error
		wantExpired bool
	}{
		{
			name:        "current with auth error",
			stale:       false,
			err:         errors.New("http 401 unauthenticated"),
			wantExpired: true,
		},
		{
			name:        "current with other error",
			stale:       false,
			err:         errors.New("network timeout"),
			wantExpired: false,
		},
		{
			name:        "stale with auth error",
			stale:       true,
			err:         errors.New("http 401"),
			wantExpired: false, // stale should ignore
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()
			gen := a.BeginGoogleGeneration(cli)

			gen.Ready()

			if tt.stale {
				a.BeginGoogleGeneration(mockClient())
			}

			gen.AuthExpired(tt.err)

			status := a.GoogleStatus()
			if !tt.stale && tt.wantExpired {
				if a.Connected.Load() {
					t.Errorf("expected Connected to be false")
				}
				if !status.AuthExpired {
					t.Errorf("expected AuthExpired to be true")
				}
				if status.LastError == "" {
					t.Errorf("expected LastError to be set")
				}
			} else if !tt.stale {
				// Shouldn't have expired
				if status.AuthExpired {
					t.Errorf("expected AuthExpired to be false")
				}
			}
		})
	}
}

func TestGoogleGeneration_SessionInvalid(t *testing.T) {
	tests := []struct {
		name  string
		stale bool
	}{
		{name: "current generation", stale: false},
		{name: "stale generation", stale: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()
			gen := a.BeginGoogleGeneration(cli)

			gen.Ready()

			if tt.stale {
				a.BeginGoogleGeneration(mockClient())
				a.Connected.Store(true)
			}

			gen.SessionInvalid()

			if !tt.stale {
				if a.Connected.Load() {
					t.Errorf("expected Connected to be false")
				}
				a.clientMu.RLock()
				if a.Client != nil {
					t.Errorf("expected a.Client to be nil")
				}
				if a.EventHandler != nil {
					t.Errorf("expected a.EventHandler to be nil")
				}
				a.clientMu.RUnlock()
				if a.GoogleStatus().LastError == "" {
					t.Errorf("expected LastError to be set")
				}
			} else {
				a.clientMu.RLock()
				if a.Client == nil {
					t.Errorf("expected a.Client to not be nil for stale generation")
				}
				a.clientMu.RUnlock()
			}
		})
	}
}

func TestGoogleGeneration_SyncInterrupted(t *testing.T) {
	tests := []struct {
		name  string
		stale bool
	}{
		{name: "current generation", stale: false},
		{name: "stale generation", stale: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()
			gen := a.BeginGoogleGeneration(cli)

			gen.Ready()

			if tt.stale {
				a.BeginGoogleGeneration(mockClient())
				a.Connected.Store(true)
			}

			gen.SyncInterrupted()

			if !tt.stale {
				if a.Connected.Load() {
					t.Errorf("expected Connected to be false")
				}
				if a.GoogleStatus().LastError == "" {
					t.Errorf("expected LastError to be set")
				}
			} else {
				if !a.Connected.Load() {
					t.Errorf("expected Connected to remain true for stale generation")
				}
			}
		})
	}
}

func TestGoogleGeneration_Release(t *testing.T) {
	tests := []struct {
		name  string
		stale bool
	}{
		{name: "current generation", stale: false},
		{name: "stale generation", stale: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := setupTestApp(t)
			cli := mockClient()
			gen := a.BeginGoogleGeneration(cli)

			gen.Ready() // Connected = true

			var newCli *client.Client
			if tt.stale {
				newCli = mockClient()
				a.BeginGoogleGeneration(newCli)
				a.Connected.Store(true)
			}

			gen.Release()

			a.clientMu.RLock()
			defer a.clientMu.RUnlock()

			if !tt.stale {
				if a.googleGeneration != nil {
					t.Errorf("expected a.googleGeneration to be nil")
				}
				if a.Client != nil {
					t.Errorf("expected a.Client to be nil")
				}
				if a.EventHandler != nil {
					t.Errorf("expected a.EventHandler to be nil")
				}
				if a.Connected.Load() {
					t.Errorf("expected Connected to be false")
				}
			} else {
				if a.googleGeneration == nil {
					t.Errorf("expected a.googleGeneration to not be nil")
				}
				if a.Client != newCli {
					t.Errorf("expected a.Client to be newCli")
				}
				if !a.Connected.Load() {
					t.Errorf("expected Connected to remain true")
				}
			}
		})
	}
}
