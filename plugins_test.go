package poundai

import (
	"os"
	"strings"
	"testing"
)

func TestPlugin(t *testing.T) {
	tests := []struct {
		shell string
		path  string
	}{
		{shell: "zsh", path: "poundai.plugin.zsh"},
		{shell: "bash", path: "poundai.plugin.bash"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			want, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Plugin(tt.shell)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("embedded %s plugin differs from %s", tt.shell, tt.path)
			}
			if !strings.HasSuffix(got, "\n") {
				t.Errorf("embedded %s plugin has no trailing newline", tt.shell)
			}
		})
	}
}

func TestPluginRejectsUnknownShell(t *testing.T) {
	if _, err := Plugin("fish"); err == nil {
		t.Fatal("Plugin(fish) returned no error")
	}
}
