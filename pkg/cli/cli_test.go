package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	tests := []struct {
		input, want string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative", "relative"},
	}
	for _, tt := range tests {
		if got := ExpandPath(tt.input); got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Test env var expansion.
	t.Setenv("POUNDAI_TEST_DIR", "/tmp/test")
	if got := ExpandPath("$POUNDAI_TEST_DIR/file"); got != "/tmp/test/file" {
		t.Errorf("ExpandPath with env = %q", got)
	}
}
