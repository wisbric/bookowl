package integration

import (
	"encoding/json"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Hello World", "hello-world"},
		{"special chars", "Incident #42: DB Down!", "incident-42-db-down"},
		{"leading trailing dashes", "---hello---", "hello"},
		{"empty", "", ""},
		{"long string", string(make([]byte, 200)), ""},
		{"unicode", "café résumé", "caf-r-sum"},
		{"multiple spaces", "hello   world", "hello-world"},
		{
			"truncated at 80 chars",
			"this-is-a-very-long-title-that-will-definitely-exceed-eighty-characters-so-it-should-be-truncated",
			"this-is-a-very-long-title-that-will-definitely-exceed-eighty-characters-so-it-sh",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.in)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if len(got) > 80 {
				t.Errorf("slugify(%q) produced %d chars, want <= 80", tt.in, len(got))
			}
		})
	}
}

func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"quotes", `say "hi"`, `say \"hi\"`},
		{"newline", "line1\nline2", `line1\nline2`},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"tab", "col1\tcol2", `col1\tcol2`},
		{"empty", "", ""},
		{"unicode", "café", "café"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeJSON(tt.in)
			if got != tt.want {
				t.Errorf("escapeJSON(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildPostMortemContent(t *testing.T) {
	incident := PostMortemIncident{
		ID:         "inc-123",
		Title:      "Database Outage",
		Severity:   "critical",
		RootCause:  "Disk full",
		Solution:   "Added more disk space",
		CreatedAt:  "2025-01-15T10:00:00Z",
		ResolvedAt: "2025-01-15T12:00:00Z",
		ResolvedBy: "jane@example.com",
	}

	content := buildPostMortemContent(incident)

	// Must be valid JSON.
	if !json.Valid(content) {
		t.Fatal("buildPostMortemContent returned invalid JSON")
	}

	// Must contain substituted values.
	s := string(content)
	for _, want := range []string{
		"Database Outage",
		"critical",
		"Disk full",
		"Added more disk space",
		"2025-01-15T10:00:00Z",
		"jane@example.com",
	} {
		if !contains(s, want) {
			t.Errorf("content does not contain %q", want)
		}
	}

	// Must not contain unsubstituted placeholders.
	for _, placeholder := range []string{
		"{incident.title}",
		"{incident.severity}",
		"{incident.root_cause}",
		"{incident.solution}",
		"{incident.created_at}",
		"{incident.resolved_by}",
	} {
		if contains(s, placeholder) {
			t.Errorf("content still contains placeholder %q", placeholder)
		}
	}
}

func TestBuildPostMortemContent_SpecialChars(t *testing.T) {
	incident := PostMortemIncident{
		Title:     `Incident with "quotes" and \backslash`,
		Severity:  "major",
		RootCause: "Line1\nLine2",
		Solution:  "Fixed\tit",
		CreatedAt: "2025-01-15",
		ResolvedBy: "user",
	}

	content := buildPostMortemContent(incident)
	if !json.Valid(content) {
		t.Fatalf("buildPostMortemContent returned invalid JSON for special chars: %s", string(content))
	}
}

func TestRenderHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"paragraph with text",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`,
			"<p>Hello world</p>",
		},
		{
			"heading level 2",
			`{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Title"}]}]}`,
			"<h2>Title</h2>",
		},
		{
			"bold text",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"strong"}]}]}`,
			"<p><strong>strong</strong></p>",
		},
		{
			"italic text",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"italic"}],"text":"em"}]}]}`,
			"<p><em>em</em></p>",
		},
		{
			"code text",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"code"}],"text":"x"}]}]}`,
			"<p><code>x</code></p>",
		},
		{
			"strike text",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"strike"}],"text":"old"}]}]}`,
			"<p><s>old</s></p>",
		},
		{
			"link",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"link","attrs":{"href":"https://example.com"}}],"text":"click"}]}]}`,
			`<p><a href="https://example.com">click</a></p>`,
		},
		{
			"bullet list",
			`{"type":"doc","content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item 1"}]}]}]}]}`,
			"<ul><li><p>item 1</p></li></ul>",
		},
		{
			"ordered list",
			`{"type":"doc","content":[{"type":"orderedList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]}]}]}]}`,
			"<ol><li><p>first</p></li></ol>",
		},
		{
			"task list unchecked",
			`{"type":"doc","content":[{"type":"taskList","content":[{"type":"taskItem","attrs":{"checked":false},"content":[{"type":"text","text":"todo"}]}]}]}`,
			`<ul class="task-list"><li><input type="checkbox" disabled> todo</li></ul>`,
		},
		{
			"task list checked",
			`{"type":"doc","content":[{"type":"taskList","content":[{"type":"taskItem","attrs":{"checked":true},"content":[{"type":"text","text":"done"}]}]}]}`,
			`<ul class="task-list"><li><input type="checkbox" disabled checked> done</li></ul>`,
		},
		{
			"code block",
			`{"type":"doc","content":[{"type":"codeBlock","content":[{"type":"text","text":"fmt.Println()"}]}]}`,
			"<pre><code>fmt.Println()</code></pre>",
		},
		{
			"blockquote",
			`{"type":"doc","content":[{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"quoted"}]}]}]}`,
			"<blockquote><p>quoted</p></blockquote>",
		},
		{
			"horizontal rule",
			`{"type":"doc","content":[{"type":"horizontalRule"}]}`,
			"<hr>",
		},
		{
			"table",
			`{"type":"doc","content":[{"type":"table","content":[{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H1"}]}]},{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"C1"}]}]}]}]}]}`,
			"<table><tr><th><p>H1</p></th><td><p>C1</p></td></tr></table>",
		},
		{
			"image",
			`{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/images/abc","alt":"screenshot"}}]}`,
			`<img src="/api/v1/images/abc" alt="screenshot">`,
		},
		{
			"image no alt",
			`{"type":"doc","content":[{"type":"image","attrs":{"src":"https://example.com/img.png"}}]}`,
			`<img src="https://example.com/img.png" alt="">`,
		},
		{
			"hard break",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"line1"},{"type":"hardBreak"},{"type":"text","text":"line2"}]}]}`,
			"<p>line1<br>line2</p>",
		},
		{
			"callout info",
			`{"type":"doc","content":[{"type":"callout","attrs":{"type":"info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Note this"}]}]}]}`,
			`<div class="callout callout-info"><p>Note this</p></div>`,
		},
		{
			"callout warning",
			`{"type":"doc","content":[{"type":"callout","attrs":{"type":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Be careful"}]}]}]}`,
			`<div class="callout callout-warning"><p>Be careful</p></div>`,
		},
		{
			"html escape",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"<script>alert(1)</script>"}]}]}`,
			"<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>",
		},
		{
			"empty content",
			"",
			"",
		},
		{
			"invalid json",
			"not json",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderHTML(json.RawMessage(tt.in))
			if got != tt.want {
				t.Errorf("renderHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHtmlEscape(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hello", "hello"},
		{"<>&\"", "&lt;&gt;&amp;&quot;"},
		{"", ""},
		{"a < b & c > d", "a &lt; b &amp; c &gt; d"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := htmlEscape(tt.in)
			if got != tt.want {
				t.Errorf("htmlEscape(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSpaceName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"postmortems", "Postmortems"},
		{"incidents", "Incidents"},
		{"Runbooks", "Runbooks"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := spaceName(tt.in)
			if got != tt.want {
				t.Errorf("spaceName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
