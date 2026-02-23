package document

import (
	"encoding/json"
	"testing"
)

func TestExtractText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"simple paragraph",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`,
			"Hello world",
		},
		{
			"multiple paragraphs",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"First"}]},{"type":"paragraph","content":[{"type":"text","text":"Second"}]}]}`,
			"First Second",
		},
		{
			"heading and paragraph",
			`{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Title"}]},{"type":"paragraph","content":[{"type":"text","text":"Body text"}]}]}`,
			"Title Body text",
		},
		{
			"nested list items",
			`{"type":"doc","content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item one"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item two"}]}]}]}]}`,
			"item one item two",
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
		{
			"text with marks (bold)",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"bold text"}]}]}`,
			"bold text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractText(json.RawMessage(tt.in))
			if got != tt.want {
				t.Errorf("ExtractText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		in   string
		want int32
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  spaced  out  ", 2},
		{"one two three four five", 5},
		{"tabs\tand\nnewlines", 3},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := CountWords(tt.in)
			if got != tt.want {
				t.Errorf("CountWords(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
