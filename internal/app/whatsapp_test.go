package app

import (
	"os"
	"strings"
	"testing"
	"path/filepath"

	"github.com/maxghenis/openmessage/internal/db"
	"github.com/rs/zerolog"
)

func TestWhatsAppMethods(t *testing.T) {
	t.Setenv("OPENMESSAGES_DATA_DIR", t.TempDir())
	t.Setenv("OPENMESSAGES_GOOGLE_AVATAR_SYNC", "0")
	logger := zerolog.Nop()
	app, err := New(logger)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	app.setClient(nil)
	defer app.Close()

	if _, err := app.InitializeWhatsApp(); err != nil {
		t.Fatalf("InitializeWhatsApp() error = %v", err)
	}

	tests := []struct {
		name string
		fn   func(t *testing.T, app *App)
	}{
		{
			name: "GetWhatsApp",
			fn: func(t *testing.T, app *App) {
				if wa := app.GetWhatsApp(); wa == nil {
					t.Errorf("GetWhatsApp() = nil, want non-nil after initialization")
				}
			},
		},
		{
			name: "WhatsAppStatus",
			fn: func(t *testing.T, app *App) {
				status := app.WhatsAppStatus()
				if status.LastError != "" {
					t.Errorf("WhatsAppStatus().LastError = %q, want empty", status.LastError)
				}
			},
		},
		{
			name: "UsesWhatsAppLiveBridge",
			fn: func(t *testing.T, app *App) {
				if uses := app.UsesWhatsAppLiveBridge(); uses {
					t.Errorf("UsesWhatsAppLiveBridge() = %v, want false for fresh bridge", uses)
				}
			},
		},
		{
			name: "WhatsAppQRCode",
			fn: func(t *testing.T, app *App) {
				_, err := app.WhatsAppQRCode()
				if err == nil || !strings.Contains(err.Error(), "no active WhatsApp QR code") {
					t.Errorf("WhatsAppQRCode() error = %v, want 'no active WhatsApp QR code'", err)
				}
			},
		},
		{
			name: "UnpairWhatsApp",
			fn: func(t *testing.T, app *App) {
				err := app.UnpairWhatsApp()
				if err != nil {
					t.Errorf("UnpairWhatsApp() error = %v, want nil", err)
				}
			},
		},
		{
			name: "PairWhatsAppPhone",
			fn: func(t *testing.T, app *App) {
				_, err := app.PairWhatsAppPhone("1234567890")
				if err == nil {
					// Should return error or not panic depending on state
					t.Errorf("PairWhatsAppPhone() error = nil, want error for fake setup")
				}
			},
		},
		{
			name: "SendWhatsAppText",
			fn: func(t *testing.T, app *App) {
				_, err := app.SendWhatsAppText("test-convo", "body", "")
				if err == nil {
					t.Errorf("SendWhatsAppText() error = nil, want error")
				}
			},
		},
		{
			name: "SendWhatsAppMedia",
			fn: func(t *testing.T, app *App) {
				_, err := app.SendWhatsAppMedia("test-convo", []byte("data"), "file.jpg", "image/jpeg", "cap", "")
				if err == nil {
					t.Errorf("SendWhatsAppMedia() error = nil, want error")
				}
			},
		},
		{
			name: "SendWhatsAppReaction",
			fn: func(t *testing.T, app *App) {
				err := app.SendWhatsAppReaction("test-convo", "msg-id", "👍", "add")
				if err == nil {
					t.Errorf("SendWhatsAppReaction() error = nil, want error")
				}
			},
		},
		{
			name: "LeaveWhatsAppGroup",
			fn: func(t *testing.T, app *App) {
				err := app.LeaveWhatsAppGroup("test-convo")
				if err == nil {
					t.Errorf("LeaveWhatsAppGroup() error = nil, want error")
				}
			},
		},
		{
			name: "DownloadWhatsAppMedia",
			fn: func(t *testing.T, app *App) {
				msg := &db.Message{}
				_, _, err := app.DownloadWhatsAppMedia(msg)
				if err == nil {
					t.Errorf("DownloadWhatsAppMedia() error = nil, want error")
				}
			},
		},
		{
			name: "WhatsAppAvatar",
			fn: func(t *testing.T, app *App) {
				_, _, err := app.WhatsAppAvatar("test-convo")
				if err == nil {
					t.Errorf("WhatsAppAvatar() error = nil, want error")
				}
			},
		},
		{
			name: "LoadAndConnectWhatsApp",
			fn: func(t *testing.T, app *App) {
				err := app.LoadAndConnectWhatsApp()
				if err != nil {
					t.Errorf("LoadAndConnectWhatsApp() error = %v, want nil", err)
				}
			},
		},
		{
			name: "StartWhatsAppConnect",
			fn: func(t *testing.T, app *App) {
				err := app.StartWhatsAppConnect()
				if err != nil {
					t.Errorf("StartWhatsAppConnect() error = %v, want nil", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t, app)
		})
	}
}

func TestWhatsAppMethods_InitError(t *testing.T) {
	t.Setenv("OPENMESSAGES_DATA_DIR", t.TempDir())
	t.Setenv("OPENMESSAGES_GOOGLE_AVATAR_SYNC", "0")
	logger := zerolog.Nop()
	app, err := New(logger)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	app.setClient(nil)
	defer app.Close()

	// Clear out any existing bridge and force an initialization error
	// by giving it a bad session path (e.g. an existing directory instead of a file)
	app.whatsAppMu.Lock()
	app.WhatsApp = nil
	app.WhatsAppSessionPath = filepath.Join(t.TempDir(), "not-a-db")
	// Make not-a-db a directory to trigger SQLite error
	os.Mkdir(app.WhatsAppSessionPath, 0755)
	app.whatsAppMu.Unlock()

	tests := []struct {
		name string
		fn   func(t *testing.T, app *App)
	}{
		{
			name: "InitializeWhatsApp",
			fn: func(t *testing.T, app *App) {
				_, err := app.InitializeWhatsApp()
				if err == nil {
					t.Errorf("InitializeWhatsApp() error = nil, want error")
				}
			},
		},
		{
			name: "WhatsAppStatus",
			fn: func(t *testing.T, app *App) {
				status := app.WhatsAppStatus()
				if status.LastError == "" {
					t.Errorf("WhatsAppStatus().LastError = empty, want error string")
				}
			},
		},
		{
			name: "UsesWhatsAppLiveBridge",
			fn: func(t *testing.T, app *App) {
				if uses := app.UsesWhatsAppLiveBridge(); uses {
					t.Errorf("UsesWhatsAppLiveBridge() = %v, want false for error", uses)
				}
			},
		},
		{
			name: "WhatsAppQRCode",
			fn: func(t *testing.T, app *App) {
				_, err := app.WhatsAppQRCode()
				if err == nil {
					t.Errorf("WhatsAppQRCode() error = nil, want error")
				}
			},
		},
		{
			name: "UnpairWhatsApp",
			fn: func(t *testing.T, app *App) {
				err := app.UnpairWhatsApp()
				if err == nil {
					t.Errorf("UnpairWhatsApp() error = nil, want error")
				}
			},
		},
		{
			name: "PairWhatsAppPhone",
			fn: func(t *testing.T, app *App) {
				_, err := app.PairWhatsAppPhone("1234567890")
				if err == nil {
					t.Errorf("PairWhatsAppPhone() error = nil, want error")
				}
			},
		},
		{
			name: "SendWhatsAppText",
			fn: func(t *testing.T, app *App) {
				_, err := app.SendWhatsAppText("test-convo", "body", "")
				if err == nil {
					t.Errorf("SendWhatsAppText() error = nil, want error")
				}
			},
		},
		{
			name: "SendWhatsAppMedia",
			fn: func(t *testing.T, app *App) {
				_, err := app.SendWhatsAppMedia("test-convo", []byte("data"), "file.jpg", "image/jpeg", "cap", "")
				if err == nil {
					t.Errorf("SendWhatsAppMedia() error = nil, want error")
				}
			},
		},
		{
			name: "SendWhatsAppReaction",
			fn: func(t *testing.T, app *App) {
				err := app.SendWhatsAppReaction("test-convo", "msg-id", "👍", "add")
				if err == nil {
					t.Errorf("SendWhatsAppReaction() error = nil, want error")
				}
			},
		},
		{
			name: "LeaveWhatsAppGroup",
			fn: func(t *testing.T, app *App) {
				err := app.LeaveWhatsAppGroup("test-convo")
				if err == nil {
					t.Errorf("LeaveWhatsAppGroup() error = nil, want error")
				}
			},
		},
		{
			name: "DownloadWhatsAppMedia",
			fn: func(t *testing.T, app *App) {
				msg := &db.Message{}
				_, _, err := app.DownloadWhatsAppMedia(msg)
				if err == nil {
					t.Errorf("DownloadWhatsAppMedia() error = nil, want error")
				}
			},
		},
		{
			name: "WhatsAppAvatar",
			fn: func(t *testing.T, app *App) {
				_, _, err := app.WhatsAppAvatar("test-convo")
				if err == nil {
					t.Errorf("WhatsAppAvatar() error = nil, want error")
				}
			},
		},
		{
			name: "LoadAndConnectWhatsApp",
			fn: func(t *testing.T, app *App) {
				err := app.LoadAndConnectWhatsApp()
				if err == nil {
					t.Errorf("LoadAndConnectWhatsApp() error = nil, want error")
				}
			},
		},
		{
			name: "StartWhatsAppConnect",
			fn: func(t *testing.T, app *App) {
				err := app.StartWhatsAppConnect()
				if err == nil {
					t.Errorf("StartWhatsAppConnect() error = nil, want error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t, app)
		})
	}
}
