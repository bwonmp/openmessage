package client

import (
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
)

func TestNewFromSession(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name        string
		sessionData *SessionData
		wantErr     bool
	}{
		{
			name: "valid auth only",
			sessionData: &SessionData{
				AuthDataJSON: json.RawMessage(`{"browser_id":"test-browser"}`),
			},
			wantErr: false,
		},
		{
			name: "valid auth and push keys",
			sessionData: &SessionData{
				AuthDataJSON: json.RawMessage(`{"browser_id":"test-browser"}`),
				PushKeysJSON: json.RawMessage(`{"private_key":"test-key"}`),
			},
			wantErr: false,
		},
		{
			name: "invalid auth data",
			sessionData: &SessionData{
				AuthDataJSON: json.RawMessage(`{invalid`),
			},
			wantErr: true,
		},
		{
			name: "invalid push keys",
			sessionData: &SessionData{
				AuthDataJSON: json.RawMessage(`{}`),
				PushKeysJSON: json.RawMessage(`{invalid`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := NewFromSession(tt.sessionData, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cli == nil {
				t.Errorf("NewFromSession() returned nil client without error")
			}
			if !tt.wantErr && cli != nil && cli.GM == nil {
				t.Errorf("NewFromSession() returned client with nil GM")
			}
		})
	}
}

func TestNewForPairing(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name string
	}{
		{
			name: "creates pairing client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewForPairing(logger)
			if cli == nil {
				t.Fatal("NewForPairing() returned nil")
			}
			if cli.GM == nil {
				t.Error("NewForPairing() returned client with nil GM")
			}
			if cli.GM.AuthData == nil {
				t.Error("NewForPairing() returned client with nil AuthData")
			}
			if cli.GM.PushKeys != nil {
				t.Error("NewForPairing() returned client with non-nil PushKeys")
			}
		})
	}
}

func TestSessionData(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		setup   func() *Client
		wantErr bool
	}{
		{
			name: "client without push keys",
			setup: func() *Client {
				return NewForPairing(logger)
			},
			wantErr: false,
		},
		{
			name: "client with push keys",
			setup: func() *Client {
				cli, _ := NewFromSession(&SessionData{
					AuthDataJSON: json.RawMessage(`{}`),
					PushKeysJSON: json.RawMessage(`{}`),
				}, logger)
				return cli
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := tt.setup()
			sd, err := cli.SessionData()
			if (err != nil) != tt.wantErr {
				t.Errorf("SessionData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && sd == nil {
				t.Error("SessionData() returned nil data without error")
			}
			if !tt.wantErr && sd != nil {
				if len(sd.AuthDataJSON) == 0 {
					t.Error("SessionData() returned empty AuthDataJSON")
				}
				// Verify push keys presence matches client
				if cli.GM.PushKeys != nil && len(sd.PushKeysJSON) == 0 {
					t.Error("SessionData() returned empty PushKeysJSON when client has push keys")
				}
				if cli.GM.PushKeys == nil && len(sd.PushKeysJSON) != 0 {
					t.Error("SessionData() returned non-empty PushKeysJSON when client has no push keys")
				}
			}
		})
	}
}
