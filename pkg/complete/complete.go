// Package complete builds prompts from a shell editing buffer and cleans up model
// output so it can be spliced back into the buffer at the cursor.
package complete

import (
	"strings"

	"github.com/elee1766/zsh_poundai/pkg/prompt"
	"github.com/elee1766/zsh_poundai/pkg/provider"
)

const zshShebang = "#!/bin/zsh\n\n"

// Input is a parsed completion request.
type Input struct {
	Buffer string // full shell editing buffer
	Cursor int    // cursor offset in bytes
	Shell  string // shell dialect; defaults to zsh
}

func (in Input) shebang() string {
	if in.Shell == "bash" {
		return "#!/bin/bash\n\n"
	}
	return zshShebang
}

// Prefix returns the buffer text before the cursor.
func (in Input) Prefix() string {
	c := clamp(in.Cursor, 0, len(in.Buffer))
	return in.Buffer[:c]
}

// Suffix returns the buffer text after the cursor.
func (in Input) Suffix() string {
	c := clamp(in.Cursor, 0, len(in.Buffer))
	return in.Buffer[c:]
}

// LinePrefix returns the current line's text before the cursor.
func (in Input) LinePrefix() string {
	p := in.Prefix()
	if i := strings.LastIndex(p, "\n"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// UserMessage builds the user-role message sent to the model: a shell shebang
// followed by the full buffer.
func (in Input) UserMessage() string {
	return in.shebang() + in.Buffer
}

// DefaultSystemPrompt is the built-in system prompt, used when neither
// prompt.system nor prompt.optimized_file is configured. Deliberately
// minimal: evals show strong models (gpt-oss-20b+) score best with exactly
// this prompt and are hurt by rule-heavy additions. For small local models
// (<= ~14B), point prompt.optimized_file at a rich artifact instead (see
// prompt.example.json).
const DefaultSystemPrompt = "Convert a natural language comment describing a shell task into the correct shell command.\n\n" +
	"Output only the shell command itself: no explanation, no markdown code block, no leading $ prompt. " +
	"The command may span multiple lines when the task calls for it (loops, heredocs, conditionals), " +
	"and must quote filenames safely (prefer find -print0 | xargs -0 when names may contain spaces)."

// SystemMessage assembles the system prompt from the base prompt (from a
// config override or optimized file), an optional extra suffix, and a rendered
// context block.
func SystemMessage(base, extra, contextBlock string) string {
	system := base
	if extra != "" {
		system += "\n\n" + extra
	}
	if contextBlock != "" {
		system += "\n\n" + contextBlock
	}
	return system
}

// Messages assembles the full conversation: system prompt, optional few-shot
// demos from an optimized prompt artifact (formatted exactly as the real
// request will be), and the actual buffer.
func Messages(system string, demos []prompt.Demo, in Input) []provider.Message {
	msgs := []provider.Message{{Role: "system", Content: system}}
	for _, d := range demos {
		comment := d.Comment
		if !strings.HasPrefix(strings.TrimSpace(comment), "#") {
			comment = "# " + comment
		}
		msgs = append(msgs,
			provider.Message{Role: "user", Content: in.shebang() + comment},
			provider.Message{Role: "assistant", Content: d.Command},
		)
	}
	return append(msgs, provider.Message{Role: "user", Content: in.UserMessage()})
}

// Clean post-processes a raw model completion so that inserting it at the
// cursor produces a sensible buffer. It ports zsh_codex's heuristics:
//
//  1. strip markdown code fences (models ignore instructions sometimes)
//  2. strip an echoed "#!/bin/zsh" shebang
//  3. strip an echoed buffer prefix or current-line prefix
//  4. strip an echoed buffer suffix
//  5. trim surrounding newlines
//  6. if the cursor line is a comment, prepend "\n" so the generated
//     command lands on the next line
func Clean(completion string, in Input) string {
	completion = stripCodeFence(completion)
	completion = strings.TrimPrefix(completion, in.shebang())
	completion = strings.TrimPrefix(completion, "#!/bin/zsh\n")
	completion = strings.TrimPrefix(completion, "#!/bin/zsh")
	completion = strings.TrimPrefix(completion, "#!/bin/bash\n")
	completion = strings.TrimPrefix(completion, "#!/bin/bash")

	prefix := in.Prefix()
	linePrefix := in.LinePrefix()
	switch {
	case prefix != "" && strings.HasPrefix(completion, prefix):
		completion = completion[len(prefix):]
	case linePrefix != "" && strings.HasPrefix(completion, linePrefix):
		completion = completion[len(linePrefix):]
	}

	if suffix := in.Suffix(); suffix != "" && strings.HasSuffix(completion, suffix) {
		completion = completion[:len(completion)-len(suffix)]
	}

	completion = strings.Trim(completion, "\n")

	if strings.HasPrefix(strings.TrimSpace(linePrefix), "#") {
		completion = "\n" + completion
	}
	return completion
}

// stripCodeFence removes wrapping markdown code fences (``` or ```zsh ...).
// Handles both a single wrapping fence and multiple consecutive fenced blocks.
// Also strips inline single-backtick wrapping (e.g. `ls -la`).
func stripCodeFence(s string) string {
	trimmed := strings.TrimSpace(s)

	// Handle inline single-backtick wrapping: `command`
	if len(trimmed) >= 2 && trimmed[0] == '`' && trimmed[len(trimmed)-1] == '`' &&
		!strings.HasPrefix(trimmed, "```") && !strings.Contains(trimmed[1:len(trimmed)-1], "`") {
		return trimmed[1 : len(trimmed)-1]
	}

	if !strings.HasPrefix(trimmed, "```") {
		return s
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 2 {
		return s
	}
	last := len(lines) - 1
	if strings.TrimSpace(lines[last]) != "```" {
		return s
	}

	// Collect content from possibly multiple fenced blocks.
	var result []string
	inFence := false
	for _, line := range lines {
		tl := strings.TrimSpace(line)
		if strings.HasPrefix(tl, "```") && !inFence {
			inFence = true
			continue
		}
		if tl == "```" && inFence {
			inFence = false
			continue
		}
		if inFence {
			result = append(result, line)
		}
		// Lines between fenced blocks (e.g. explanatory text) are discarded.
	}
	if len(result) == 0 {
		// Fallback: original single-fence logic.
		return strings.Join(lines[1:last], "\n")
	}
	return strings.Join(result, "\n")
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
