package app

import (
	"os"
	"testing"
	"time"

	"github.com/maxghenis/openmessage/internal/db"
	"github.com/rs/zerolog"
)

func TestApp_QueueGoogleAvatarCandidates(t *testing.T) {
	tests := []struct {
		name       string
		appFactory func() *App
		candidates []db.ContactAvatarCandidate
		envValue   string
		setup      func(*App)
		verify     func(t *testing.T, a *App)
	}{
		{
			name:       "nil app",
			appFactory: func() *App { return nil },
			candidates: []db.ContactAvatarCandidate{{ParticipantID: "123"}},
			envValue:   "1",
			verify: func(t *testing.T, a *App) {
				// Should not panic
			},
		},
		{
			name:       "empty candidates",
			appFactory: func() *App { return &App{Logger: zerolog.Nop()} },
			candidates: []db.ContactAvatarCandidate{},
			envValue:   "1",
			verify: func(t *testing.T, a *App) {
				a.avatarSyncMu.Lock()
				defer a.avatarSyncMu.Unlock()
				if a.avatarSyncQueue != nil {
					t.Error("expected queue to be nil")
				}
			},
		},
		{
			name:       "disabled via env",
			appFactory: func() *App { return &App{Logger: zerolog.Nop()} },
			candidates: []db.ContactAvatarCandidate{{ParticipantID: "123"}},
			envValue:   "0",
			verify: func(t *testing.T, a *App) {
				a.avatarSyncMu.Lock()
				defer a.avatarSyncMu.Unlock()
				if a.avatarSyncQueue != nil {
					t.Error("expected queue to be nil")
				}
			},
		},
		{
			name:       "already closed",
			appFactory: func() *App { return &App{Logger: zerolog.Nop()} },
			candidates: []db.ContactAvatarCandidate{{ParticipantID: "123"}},
			envValue:   "1",
			setup: func(a *App) {
				a.avatarSyncClosed = true
			},
			verify: func(t *testing.T, a *App) {
				a.avatarSyncMu.Lock()
				defer a.avatarSyncMu.Unlock()
				if a.avatarSyncQueue != nil {
					t.Error("expected queue to be nil")
				}
			},
		},
		{
			name:       "valid queueing",
			appFactory: func() *App { return &App{Logger: zerolog.Nop()} },
			candidates: []db.ContactAvatarCandidate{
				{ParticipantID: "123", SourcePlatform: "google"}, // Needs SourcePlatform etc so ContactAvatarID is not empty
			},
			envValue:   "1",
			verify: func(t *testing.T, a *App) {
				a.avatarSyncMu.Lock()
				queue := a.avatarSyncQueue
				a.avatarSyncMu.Unlock()
				if queue == nil {
					t.Error("expected queue to be created")
					return
				}
				// Give the goroutine a moment to start
				time.Sleep(10 * time.Millisecond)

				select {
				case <-queue:
					// wait, queue might be read by the loop
				default:
				}
				a.StopGoogleAvatarSync()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("OPENMESSAGE_GOOGLE_AVATAR_SYNC", tt.envValue)
			} else {
				os.Unsetenv("OPENMESSAGE_GOOGLE_AVATAR_SYNC")
				os.Unsetenv("OPENMESSAGES_GOOGLE_AVATAR_SYNC")
			}

			a := tt.appFactory()
			if tt.setup != nil {
				tt.setup(a)
			}

			a.QueueGoogleAvatarCandidates(tt.candidates)

			if tt.verify != nil {
				tt.verify(t, a)
			}
			if a != nil && a.avatarSyncQueue != nil {
				a.StopGoogleAvatarSync()
			}
		})
	}
}

func TestApp_StopGoogleAvatarSync(t *testing.T) {
	tests := []struct {
		name       string
		appFactory func() *App
		setup      func(*App)
		verify     func(t *testing.T, a *App)
	}{
		{
			name:       "nil app",
			appFactory: func() *App { return nil },
			verify: func(t *testing.T, a *App) {
				// Should not panic
			},
		},
		{
			name:       "stop without queueing",
			appFactory: func() *App { return &App{Logger: zerolog.Nop()} },
			verify: func(t *testing.T, a *App) {
				if !a.avatarSyncClosed {
					t.Error("expected avatarSyncClosed to be true")
				}
			},
		},
		{
			name:       "stop after queueing",
			appFactory: func() *App { return &App{Logger: zerolog.Nop()} },
			setup: func(a *App) {
				t.Setenv("OPENMESSAGE_GOOGLE_AVATAR_SYNC", "1")
				a.QueueGoogleAvatarCandidates([]db.ContactAvatarCandidate{
					{ParticipantID: "123", SourcePlatform: "google"},
				})
			},
			verify: func(t *testing.T, a *App) {
				if !a.avatarSyncClosed {
					t.Error("expected avatarSyncClosed to be true")
				}
				select {
				case <-a.avatarSyncStop:
					// Expected to be closed
				default:
					t.Error("expected avatarSyncStop to be closed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.appFactory()
			if tt.setup != nil {
				tt.setup(a)
			}

			a.StopGoogleAvatarSync()

			if tt.verify != nil {
				tt.verify(t, a)
			}
		})
	}
}
