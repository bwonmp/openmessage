//go:build darwin

package googlecookies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeSupported(t *testing.T) {
	// NativeSupported uses DefaultChromeProfile().
	// To test it without altering the system, we can set OPENMESSAGE_CHROME_PROFILE
	// to a temporary directory that we control.

	tests := []struct {
		name       string
		setup      func(t *testing.T, profileDir string)
		want       bool
	}{
		{
			name:       "no profile dir",
			setup:      func(t *testing.T, profileDir string) {},
			want:       false,
		},
		{
			name: "Network/Cookies exists",
			setup: func(t *testing.T, profileDir string) {
				networkDir := filepath.Join(profileDir, "Network")
				if err := os.MkdirAll(networkDir, 0755); err != nil {
					t.Fatalf("failed to create Network dir: %v", err)
				}
				cookiesFile := filepath.Join(networkDir, "Cookies")
				if err := os.WriteFile(cookiesFile, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create Cookies file: %v", err)
				}
			},
			want: true,
		},
		{
			name: "Cookies exists in root",
			setup: func(t *testing.T, profileDir string) {
				cookiesFile := filepath.Join(profileDir, "Cookies")
				if err := os.WriteFile(cookiesFile, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create Cookies file: %v", err)
				}
			},
			want: true,
		},
		{
			name: "profile dir exists but no Cookies file",
			setup: func(t *testing.T, profileDir string) {
				// Just the profile dir exists, which is already created by t.TempDir()
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp dir to act as our fake Chrome profile
			profileDir := t.TempDir()
			tt.setup(t, profileDir)

			// Set the environment variable so DefaultChromeProfile returns our temp dir
			t.Setenv("OPENMESSAGE_CHROME_PROFILE", profileDir)

			got := NativeSupported()
			if got != tt.want {
				t.Errorf("NativeSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}
