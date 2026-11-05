//go:build !darwin

package googlecookies

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNativeSupported(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "always false on non-darwin",
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NativeSupported(); got != tc.expected {
				t.Errorf("NativeSupported() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestDefaultChromeProfileDir(t *testing.T) {
	tests := []struct {
		name string
		home string
		want string
	}{
		{
			name: "basic home dir",
			home: "/home/user",
			want: filepath.Join("/home/user", ".config", "google-chrome", "Default"),
		},
		{
			name: "empty home dir",
			home: "",
			want: filepath.Join("", ".config", "google-chrome", "Default"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := defaultChromeProfileDir(tc.home); got != tc.want {
				t.Errorf("defaultChromeProfileDir(%q) = %q, want %q", tc.home, got, tc.want)
			}
		})
	}
}

func TestChromeSafeStorageSecret(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr string
	}{
		{
			name:    "always errors",
			ctx:     context.Background(),
			wantErr: "native Chrome cookie refresh is not supported on this platform",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secret, err := chromeSafeStorageSecret(tc.ctx)
			if secret != nil {
				t.Errorf("chromeSafeStorageSecret() secret = %v, want nil", secret)
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("chromeSafeStorageSecret() error = %v, want error with message %q", err, tc.wantErr)
			}
		})
	}
}
