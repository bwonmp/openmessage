package app

import (
	"errors"
	"testing"

	"github.com/maxghenis/openmessage/internal/db"
	"github.com/maxghenis/openmessage/internal/signallive"
	"github.com/rs/zerolog"
)

func TestSignal(t *testing.T) {
	store, err := db.New(":memory:")
	if err != nil {
		t.Fatalf("db.New(): %v", err)
	}

	live, err := signallive.New(t.TempDir(), store, zerolog.Nop(), signallive.Callbacks{})
	if err != nil {
		t.Fatalf("signallive.New(): %v", err)
	}
	t.Cleanup(func() { _ = live.Close() })

	a := &App{Signal: live, Logger: zerolog.Nop(), Store: store}

	tests := []struct {
		name    string
		f       func() error
		wantErr bool
	}{
		{
			name: "EnsureSignal",
			f: func() error {
				_, err := a.EnsureSignal()
				return err
			},
			wantErr: false,
		},
		{
			name: "GetSignal",
			f: func() error {
				if a.GetSignal() == nil {
					return errors.New("GetSignal returned nil")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "SignalStatus",
			f: func() error {
				_ = a.SignalStatus()
				return nil
			},
			wantErr: false,
		},
		{
			name: "SignalQRCode",
			f: func() error {
				_, err := a.SignalQRCode()
				return err
			},
			wantErr: true,
		},
		{
			name: "ReplaySignalRecoveryQueue",
			f: func() error {
				return a.ReplaySignalRecoveryQueue()
			},
			wantErr: true, // expect error because signal is not paired
		},
		{
			name: "SendSignalText",
			f: func() error {
				_, err := a.SendSignalText("conv", "body", "")
				return err
			},
			wantErr: true, // expect error since it's not fully mocked to send
		},
		{
			name: "SendSignalMedia",
			f: func() error {
				_, err := a.SendSignalMedia("conv", nil, "file", "mime", "cap", "")
				return err
			},
			wantErr: true,
		},
		{
			name: "SendSignalReaction",
			f: func() error {
				return a.SendSignalReaction("conv", "msg", "em", "add")
			},
			wantErr: true,
		},
		{
			name: "DownloadSignalMedia",
			f: func() error {
				_, _, err := a.DownloadSignalMedia(&db.Message{})
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.f()
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
