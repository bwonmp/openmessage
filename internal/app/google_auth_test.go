package app

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

func TestIsGoogleAuthExpiredError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "http 401 error",
			err:  errors.New("some error: http 401 unauthorized"),
			want: true,
		},
		{
			name: "session_cookie_invalid error",
			err:  errors.New("SESSION_COOKIE_INVALID"),
			want: true,
		},
		{
			name: "session cookie expired error",
			err:  errors.New("Session Cookie Expired"),
			want: true,
		},
		{
			name: "invalid authentication credentials error",
			err:  errors.New("Invalid Authentication Credentials error"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("connection reset by peer"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGoogleAuthExpiredError(tt.err); got != tt.want {
				t.Errorf("IsGoogleAuthExpiredError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleGoogleAuthExpiredError(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		initialConnected  bool
		initialExpired    bool
		wantHandled       bool
		wantConnected     bool
		wantExpired       bool
		wantReported      bool
		wantStatusEmitted bool
	}{
		{
			name:              "not an auth expired error",
			err:               errors.New("temporary failure in name resolution"),
			initialConnected:  true,
			initialExpired:    false,
			wantHandled:       false,
			wantConnected:     true,
			wantExpired:       false,
			wantReported:      false,
			wantStatusEmitted: false,
		},
		{
			name:              "auth expired error first notification",
			err:               errors.New("send message: HTTP 401: 16: Request had invalid authentication credentials"),
			initialConnected:  true,
			initialExpired:    false,
			wantHandled:       true,
			wantConnected:     false,
			wantExpired:       true,
			wantReported:      true,
			wantStatusEmitted: true,
		},
		{
			name:              "auth expired error already notified",
			err:               errors.New("send message: HTTP 401: 16: Request had invalid authentication credentials"),
			initialConnected:  true,
			initialExpired:    true,
			wantHandled:       true,
			wantConnected:     false,
			wantExpired:       true,
			wantReported:      false,
			wantStatusEmitted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{Logger: zerolog.Nop()}
			a.Connected.Store(tt.initialConnected)
			a.googleAuthExpired.Store(tt.initialExpired)

			statusEmitted := false
			a.OnStatusChange = func(connected bool) {
				statusEmitted = true
				if connected {
					t.Errorf("expected status event to report disconnected (false), got %v", connected)
				}
			}

			notifier := &recordingGoogleLifecycleNotifier{}
			a.SetGoogleLifecycleNotifier(notifier)

			handled := a.HandleGoogleAuthExpiredError(tt.err)

			if handled != tt.wantHandled {
				t.Errorf("HandleGoogleAuthExpiredError() = %v, want %v", handled, tt.wantHandled)
			}
			if a.Connected.Load() != tt.wantConnected {
				t.Errorf("a.Connected = %v, want %v", a.Connected.Load(), tt.wantConnected)
			}
			if a.googleAuthExpired.Load() != tt.wantExpired {
				t.Errorf("a.googleAuthExpired = %v, want %v", a.googleAuthExpired.Load(), tt.wantExpired)
			}

			reported := len(notifier.reported) > 0
			if reported != tt.wantReported {
				t.Errorf("reported to lifecycle notifier = %v, want %v", reported, tt.wantReported)
			}

			if statusEmitted != tt.wantStatusEmitted {
				t.Errorf("statusEmitted = %v, want %v", statusEmitted, tt.wantStatusEmitted)
			}

			if tt.wantHandled {
				if got := a.GoogleStatus().LastError; got != googleAuthExpiredStatusMessage {
					t.Errorf("last error = %q, want %q", got, googleAuthExpiredStatusMessage)
				}
			}
		})
	}
}
