package complete

import (
	"testing"

	"github.com/elee1766/poundai/pkg/prompt"
)

func TestClean(t *testing.T) {
	tests := []struct {
		name       string
		completion string
		buffer     string
		cursor     int
		want       string
	}{
		{
			name:       "plain completion appended at end",
			completion: " -la",
			buffer:     "ls",
			cursor:     2,
			want:       " -la",
		},
		{
			name:       "strips echoed shebang and prefix",
			completion: "#!/bin/zsh\n\nls -la",
			buffer:     "ls",
			cursor:     2,
			want:       " -la",
		},
		{
			name:       "strips echoed full prefix",
			completion: "git commit -m 'fix'",
			buffer:     "git commit",
			cursor:     10,
			want:       " -m 'fix'",
		},
		{
			name:       "strips echoed suffix",
			completion: "find . -name '*.go' | xargs wc -l",
			buffer:     "find  | xargs wc -l",
			cursor:     5,
			want:       ". -name '*.go'",
		},
		{
			name:       "comment gets newline prepended",
			completion: "docker ps -a",
			buffer:     "# list all docker containers",
			cursor:     28,
			want:       "\ndocker ps -a",
		},
		{
			name:       "comment with echoed comment line",
			completion: "# list all docker containers\ndocker ps -a",
			buffer:     "# list all docker containers",
			cursor:     28,
			want:       "\ndocker ps -a",
		},
		{
			name:       "strips code fence",
			completion: "```zsh\nls -la\n```",
			buffer:     "ls",
			cursor:     2,
			want:       " -la",
		},
		{
			name:       "strips bare code fence",
			completion: "```\ntar -xzf archive.tar.gz\n```",
			buffer:     "tar ",
			cursor:     4,
			want:       "-xzf archive.tar.gz",
		},
		{
			name:       "multiline buffer uses line prefix",
			completion: "echo done",
			buffer:     "make build\necho ",
			cursor:     16,
			want:       "done",
		},
		{
			name:       "empty buffer",
			completion: "ls -la",
			buffer:     "",
			cursor:     0,
			want:       "ls -la",
		},
		{
			name:       "unicode cursor offset (byte semantics)",
			completion: "echo héllo world",
			buffer:     "echo héllo",
			cursor:     11, // byte offset: 'é' is 2 bytes, so "echo héllo" = 11 bytes
			want:       " world",
		},
		{
			name:       "cursor beyond buffer is clamped",
			completion: "ls -la",
			buffer:     "ls",
			cursor:     99,
			want:       " -la",
		},
		{
			name:       "trims surrounding newlines",
			completion: "\n\nls -la\n",
			buffer:     "",
			cursor:     0,
			want:       "ls -la",
		},
		{
			name:       "heredoc newlines preserved",
			completion: "cat <<EOF > config.env\nAPI_URL=https://api.example.com\nDEBUG=false\nEOF",
			buffer:     "# create config.env with a heredoc",
			cursor:     34,
			want:       "\ncat <<EOF > config.env\nAPI_URL=https://api.example.com\nDEBUG=false\nEOF",
		},
		{
			name:       "fenced multiline loop preserved",
			completion: "```zsh\nfor f in *.log; do\n  gzip \"$f\"\ndone\n```",
			buffer:     "# compress all logs",
			cursor:     19,
			want:       "\nfor f in *.log; do\n  gzip \"$f\"\ndone",
		},
		{
			name:       "multiple code fences",
			completion: "```zsh\nls -la\n```\nSome explanation\n```zsh\necho done\n```",
			buffer:     "",
			cursor:     0,
			want:       "ls -la\necho done",
		},
		{
			name:       "inline backtick fence",
			completion: "`ls -la`",
			buffer:     "",
			cursor:     0,
			want:       "ls -la",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := Input{Buffer: tt.buffer, Cursor: tt.cursor}
			got := Clean(tt.completion, in)
			if got != tt.want {
				t.Errorf("Clean(%q) = %q, want %q", tt.completion, got, tt.want)
			}
		})
	}
}

func TestInputSplitting(t *testing.T) {
	in := Input{Buffer: "abc\ndef", Cursor: 5}
	if got := in.Prefix(); got != "abc\nd" {
		t.Errorf("Prefix() = %q", got)
	}
	if got := in.Suffix(); got != "ef" {
		t.Errorf("Suffix() = %q", got)
	}
	if got := in.LinePrefix(); got != "d" {
		t.Errorf("LinePrefix() = %q", got)
	}
}

func TestSystemMessage(t *testing.T) {
	if got := SystemMessage("", "", ""); got != "" {
		t.Errorf("empty base should return empty, got %q", got)
	}
	if got := SystemMessage("base", "", ""); got != "base" {
		t.Errorf("base only = %q", got)
	}
	got := SystemMessage("base", "extra", "ctx")
	want := "base\n\nextra\n\nctx"
	if got != want {
		t.Errorf("SystemMessage = %q, want %q", got, want)
	}
}

func TestUserMessage(t *testing.T) {
	in := Input{Buffer: "ls", Cursor: 2}
	if got := in.UserMessage(); got != "#!/bin/zsh\n\nls" {
		t.Errorf("UserMessage() = %q", got)
	}

	bash := Input{Buffer: "ls", Cursor: 2, Shell: "bash"}
	if got := bash.UserMessage(); got != "#!/bin/bash\n\nls" {
		t.Errorf("bash UserMessage() = %q", got)
	}
}

func TestMessages(t *testing.T) {
	demos := []prompt.Demo{
		{Comment: "show disk space", Command: "df -h"},
	}
	in := Input{Buffer: "# list files", Cursor: 12}
	msgs := Messages("system prompt", demos, in)

	if len(msgs) != 4 {
		t.Fatalf("got %d messages, want 4", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "system prompt" {
		t.Errorf("system message = %+v", msgs[0])
	}
	if msgs[1].Role != "user" || msgs[1].Content != "#!/bin/zsh\n\n# show disk space" {
		t.Errorf("demo user = %+v", msgs[1])
	}
	if msgs[2].Role != "assistant" || msgs[2].Content != "df -h" {
		t.Errorf("demo assistant = %+v", msgs[2])
	}
	if msgs[3].Role != "user" || msgs[3].Content != "#!/bin/zsh\n\n# list files" {
		t.Errorf("user message = %+v", msgs[3])
	}
}

func TestMessagesDemoWithoutHash(t *testing.T) {
	demos := []prompt.Demo{
		{Comment: "list pods", Command: "kubectl get pods"},
	}
	in := Input{Buffer: "ls", Cursor: 2}
	msgs := Messages("sys", demos, in)
	// Comment without "#" should get "# " prepended.
	if msgs[1].Content != "#!/bin/zsh\n\n# list pods" {
		t.Errorf("demo user = %q", msgs[1].Content)
	}
}

func TestBashMessagesAndClean(t *testing.T) {
	in := Input{Buffer: "# list files", Cursor: 12, Shell: "bash"}
	msgs := Messages("sys", []prompt.Demo{{Comment: "show date", Command: "date"}}, in)
	if got := msgs[1].Content; got != "#!/bin/bash\n\n# show date" {
		t.Errorf("demo user = %q", got)
	}
	if got := msgs[3].Content; got != "#!/bin/bash\n\n# list files" {
		t.Errorf("user message = %q", got)
	}
	if got := Clean("#!/bin/bash\n\nls -la", in); got != "\nls -la" {
		t.Errorf("Clean() = %q", got)
	}
}
