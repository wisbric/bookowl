package comment

import (
	"testing"
)

func TestRenderBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "bold",
			input:    "this is **bold** text",
			contains: []string{"<strong>bold</strong>"},
		},
		{
			name:     "italic",
			input:    "this is _italic_ text",
			contains: []string{"<em>italic</em>"},
		},
		{
			name:     "inline code",
			input:    "use `kubectl get pods`",
			contains: []string{"<code>kubectl get pods</code>"},
		},
		{
			name:     "fenced code block",
			input:    "```\nfoo := bar\n```",
			contains: []string{"<pre><code>foo := bar"},
		},
		{
			name:     "mention",
			input:    "hey @stefan check this",
			contains: []string{`<a href="/profile/stefan">@stefan</a>`},
		},
		{
			name:     "auto-link URL",
			input:    "see https://example.com/page for details",
			contains: []string{`<a href="https://example.com/page">https://example.com/page</a>`},
		},
		{
			name:     "HTML sanitization",
			input:    "<script>alert('xss')</script>",
			contains: []string{"&lt;script&gt;"},
		},
		{
			name:     "combined formatting",
			input:    "**bold** and _italic_ and `code`",
			contains: []string{"<strong>bold</strong>", "<em>italic</em>", "<code>code</code>"},
		},
		{
			name:     "newlines become br",
			input:    "line1\nline2",
			contains: []string{"line1<br>line2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderBody(tt.input)
			for _, want := range tt.contains {
				if !containsStr(got, want) {
					t.Errorf("RenderBody(%q) = %q, want it to contain %q", tt.input, got, want)
				}
			}
		})
	}
}

func TestExtractMentions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single mention",
			input: "hey @stefan look at this",
			want:  []string{"stefan"},
		},
		{
			name:  "multiple mentions",
			input: "@alice and @bob should review",
			want:  []string{"alice", "bob"},
		},
		{
			name:  "duplicate mentions deduplicated",
			input: "@alice said @alice should do it",
			want:  []string{"alice"},
		},
		{
			name:  "no mentions",
			input: "just plain text",
			want:  []string{},
		},
		{
			name:  "mention with dots",
			input: "@user.name is here",
			want:  []string{"user.name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMentions(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractMentions(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ExtractMentions(%q)[%d] = %q, want %q", tt.input, i, v, tt.want[i])
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
