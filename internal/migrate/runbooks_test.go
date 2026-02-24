package migrate

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
		{"special chars", "Pod CrashLoopBackOff!", "pod-crashloopbackoff"},
		{"leading/trailing", "---hello---", "hello"},
		{"empty", "", ""},
		{"numbers", "Alert Rule 42", "alert-rule-42"},
		{"multiple dashes", "hello   world", "hello-world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.in)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMarkdownToTiptapJSON_Empty(t *testing.T) {
	content := markdownToTiptapJSON("")
	if !json.Valid(content) {
		t.Fatalf("produced invalid JSON: %s", string(content))
	}
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["type"] != "doc" {
		t.Errorf("expected type=doc, got %v", doc["type"])
	}
}

func TestMarkdownToTiptapJSON_Heading(t *testing.T) {
	content := markdownToTiptapJSON("# Hello")
	if !json.Valid(content) {
		t.Fatalf("invalid JSON: %s", string(content))
	}
	var doc struct {
		Type    string `json:"type"`
		Content []struct {
			Type  string         `json:"type"`
			Attrs map[string]any `json:"attrs"`
		} `json:"content"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Content) != 1 {
		t.Fatalf("expected 1 node, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != "heading" {
		t.Errorf("expected heading, got %s", doc.Content[0].Type)
	}
	if level, ok := doc.Content[0].Attrs["level"].(float64); !ok || level != 1 {
		t.Errorf("expected level=1, got %v", doc.Content[0].Attrs["level"])
	}
}

func TestMarkdownToTiptapJSON_CodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hello\")\n```"
	content := markdownToTiptapJSON(md)
	if !json.Valid(content) {
		t.Fatalf("invalid JSON: %s", string(content))
	}
	var doc struct {
		Content []struct {
			Type  string         `json:"type"`
			Attrs map[string]any `json:"attrs"`
		} `json:"content"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Content) != 1 {
		t.Fatalf("expected 1 node, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != "codeBlock" {
		t.Errorf("expected codeBlock, got %s", doc.Content[0].Type)
	}
	if lang := doc.Content[0].Attrs["language"]; lang != "go" {
		t.Errorf("expected language=go, got %v", lang)
	}
}

func TestMarkdownToTiptapJSON_BulletList(t *testing.T) {
	md := "- item one\n- item two\n- item three"
	content := markdownToTiptapJSON(md)
	if !json.Valid(content) {
		t.Fatalf("invalid JSON: %s", string(content))
	}
	var doc struct {
		Content []struct {
			Type    string `json:"type"`
			Content []any  `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Content) != 1 {
		t.Fatalf("expected 1 node, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != "bulletList" {
		t.Errorf("expected bulletList, got %s", doc.Content[0].Type)
	}
	if len(doc.Content[0].Content) != 3 {
		t.Errorf("expected 3 items, got %d", len(doc.Content[0].Content))
	}
}

func TestMarkdownToTiptapJSON_OrderedList(t *testing.T) {
	md := "1. first\n2. second"
	content := markdownToTiptapJSON(md)
	if !json.Valid(content) {
		t.Fatalf("invalid JSON: %s", string(content))
	}
	var doc struct {
		Content []struct {
			Type    string `json:"type"`
			Content []any  `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Content) != 1 {
		t.Fatalf("expected 1 node, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != "orderedList" {
		t.Errorf("expected orderedList, got %s", doc.Content[0].Type)
	}
}

func TestMarkdownToTiptapJSON_Blockquote(t *testing.T) {
	md := "> Important note"
	content := markdownToTiptapJSON(md)
	if !json.Valid(content) {
		t.Fatalf("invalid JSON: %s", string(content))
	}
	var doc struct {
		Content []struct {
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Content[0].Type != "blockquote" {
		t.Errorf("expected blockquote, got %s", doc.Content[0].Type)
	}
}

func TestMarkdownToTiptapJSON_HorizontalRule(t *testing.T) {
	md := "---"
	content := markdownToTiptapJSON(md)
	if !json.Valid(content) {
		t.Fatalf("invalid JSON: %s", string(content))
	}
	var doc struct {
		Content []struct {
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Content[0].Type != "horizontalRule" {
		t.Errorf("expected horizontalRule, got %s", doc.Content[0].Type)
	}
}

func TestMarkdownToTiptapJSON_Mixed(t *testing.T) {
	md := "# Runbook\n\nCheck the **pod** status:\n\n```bash\nkubectl get pods\n```\n\n- Step 1\n- Step 2"
	content := markdownToTiptapJSON(md)
	if !json.Valid(content) {
		t.Fatalf("invalid JSON: %s", string(content))
	}
	var doc struct {
		Content []struct {
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	// Should have: heading, paragraph, codeBlock, bulletList (4 nodes)
	if len(doc.Content) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(doc.Content))
		for i, n := range doc.Content {
			t.Logf("  [%d] type=%s", i, n.Type)
		}
	}
}

func TestInlineContent_Bold(t *testing.T) {
	nodes := inlineContent("hello **world**")
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0]["text"] != "hello " {
		t.Errorf("expected 'hello ', got %v", nodes[0]["text"])
	}
	if nodes[1]["text"] != "world" {
		t.Errorf("expected 'world', got %v", nodes[1]["text"])
	}
}

func TestInlineContent_Code(t *testing.T) {
	nodes := inlineContent("run `kubectl get pods` now")
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[1]["text"] != "kubectl get pods" {
		t.Errorf("expected 'kubectl get pods', got %v", nodes[1]["text"])
	}
}

func TestInlineContent_Plain(t *testing.T) {
	nodes := inlineContent("just plain text")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0]["text"] != "just plain text" {
		t.Errorf("expected 'just plain text', got %v", nodes[0]["text"])
	}
}
