package poundai

import (
	_ "embed"
	"fmt"
)

//go:embed poundai.plugin.zsh
var zshPlugin string

//go:embed poundai.plugin.bash
var bashPlugin string

// Plugin returns the shell plugin source bundled into the binary.
func Plugin(shell string) (string, error) {
	switch shell {
	case "zsh":
		return zshPlugin, nil
	case "bash":
		return bashPlugin, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (want zsh or bash)", shell)
	}
}
