package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.json")
	data := `{"system": "test system", "demos": [{"comment": "list files", "command": "ls"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	opt, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if opt.System != "test system" {
		t.Errorf("system = %q", opt.System)
	}
	if len(opt.Demos) != 1 || opt.Demos[0].Command != "ls" {
		t.Errorf("demos = %+v", opt.Demos)
	}
}

func TestLoadEmptySystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.json")
	data := `{"system": "", "demos": []}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for empty system")
	}
}

func TestLoadEmptyDemoField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.json")
	data := `{"system": "ok", "demos": [{"comment": "", "command": "ls"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for empty demo comment")
	}

	data = `{"system": "ok", "demos": [{"comment": "list", "command": ""}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for empty demo command")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	if _, err := Load("/nonexistent/path.json"); err == nil {
		t.Error("expected error for missing file")
	}
}
