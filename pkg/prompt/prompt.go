// Package prompt loads prompt artifacts: JSON files carrying a system
// prompt and optional few-shot demos (see prompt.example.json).
package prompt

import (
	"encoding/json"
	"fmt"
	"os"
)

// Demo is a single few-shot example selected by the optimizer.
type Demo struct {
	Comment string `json:"comment"`
	Command string `json:"command"`
}

// Optimized is the prompt artifact schema.
type Optimized struct {
	System string `json:"system"`
	Demos  []Demo `json:"demos"`
}

// Load reads an optimized prompt artifact from path.
func Load(path string) (*Optimized, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading optimized prompt: %w", err)
	}
	var opt Optimized
	if err := json.Unmarshal(data, &opt); err != nil {
		return nil, fmt.Errorf("parsing optimized prompt %s: %w", path, err)
	}
	if opt.System == "" {
		return nil, fmt.Errorf("optimized prompt %s: empty 'system' field", path)
	}
	for i, d := range opt.Demos {
		if d.Comment == "" || d.Command == "" {
			return nil, fmt.Errorf("optimized prompt %s: demo[%d] has empty comment or command", path, i)
		}
	}
	return &opt, nil
}
