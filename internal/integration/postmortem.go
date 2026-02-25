package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

// postMortemTemplate is the Tiptap JSON template from docs/04-editor.md section 8.
// Variables like {incident.title} are substituted before document creation.
const postMortemTemplate = `{
  "type": "doc",
  "content": [
    { "type": "heading", "attrs": {"level": 1}, "content": [{"type": "text", "text": "Post-Mortem: {incident.title}"}] },
    { "type": "paragraph", "content": [{"type": "text", "marks": [{"type": "bold"}], "text": "Date:"}, {"type": "text", "text": " {incident.created_at}"}] },
    { "type": "paragraph", "content": [{"type": "text", "marks": [{"type": "bold"}], "text": "Severity:"}, {"type": "text", "text": " {incident.severity}"}] },
    { "type": "paragraph", "content": [{"type": "text", "marks": [{"type": "bold"}], "text": "Resolved by:"}, {"type": "text", "text": " {incident.resolved_by}"}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Summary"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "{incident.solution}"}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Timeline"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "Add key events from the incident timeline here."}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Root Cause"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "{incident.root_cause}"}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Impact"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "Describe the business/user impact."}] },
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Action Items"}] },
    { "type": "taskList", "content": [
      { "type": "taskItem", "attrs": {"checked": false}, "content": [{"type": "text", "text": "Add follow-up actions here"}] }
    ]},
    { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Lessons Learned"}] },
    { "type": "paragraph", "content": [{"type": "text", "text": "What did we learn? What do we do differently next time?"}] }
  ]
}`

// buildPostMortemFromDB tries to use the system post-mortem template from the
// database. If not found, it falls back to the hardcoded template.
func buildPostMortemFromDB(ctx context.Context, q *dbtenant.Queries, incident PostMortemIncident) json.RawMessage {
	tmpl, err := q.GetSystemTemplateByDocType(ctx, "post-mortem")
	if err == nil {
		// Substitute {{variable}} placeholders in the template content.
		content := string(tmpl.Content)
		content = strings.ReplaceAll(content, "{{incident_title}}", escapeJSON(incident.Title))
		content = strings.ReplaceAll(content, "{{date}}", escapeJSON(incident.CreatedAt))
		content = strings.ReplaceAll(content, "{{severity}}", escapeJSON(incident.Severity))
		content = strings.ReplaceAll(content, "{{resolved_by}}", escapeJSON(incident.ResolvedBy))
		return json.RawMessage(content)
	}
	// Fall back to hardcoded template.
	return buildPostMortemContent(incident)
}

func buildPostMortemContent(incident PostMortemIncident) json.RawMessage {
	content := postMortemTemplate
	content = strings.ReplaceAll(content, "{incident.title}", escapeJSON(incident.Title))
	content = strings.ReplaceAll(content, "{incident.severity}", escapeJSON(incident.Severity))
	content = strings.ReplaceAll(content, "{incident.root_cause}", escapeJSON(incident.RootCause))
	content = strings.ReplaceAll(content, "{incident.solution}", escapeJSON(incident.Solution))
	content = strings.ReplaceAll(content, "{incident.created_at}", escapeJSON(incident.CreatedAt))
	content = strings.ReplaceAll(content, "{incident.resolved_by}", escapeJSON(incident.ResolvedBy))
	return json.RawMessage(content)
}

// escapeJSON escapes characters that would break a JSON string value.
func escapeJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return s
	}
	// json.Marshal wraps in quotes: "value" — strip them.
	return string(b[1 : len(b)-1])
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = s[:80]
		s = strings.TrimRight(s, "-")
	}
	return s
}

// renderHTML converts Tiptap/ProseMirror JSON to basic HTML for inline display.
func renderHTML(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}
	var node tiptapNode
	if err := json.Unmarshal(content, &node); err != nil {
		return ""
	}
	var b strings.Builder
	renderNode(&b, node)
	return b.String()
}

type tiptapNode struct {
	Type    string            `json:"type"`
	Text    string            `json:"text,omitempty"`
	Attrs   map[string]any    `json:"attrs,omitempty"`
	Marks   []tiptapMark      `json:"marks,omitempty"`
	Content []tiptapNode      `json:"content,omitempty"`
}

type tiptapMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

func renderNode(b *strings.Builder, n tiptapNode) {
	switch n.Type {
	case "doc":
		renderChildren(b, n)
	case "heading":
		level := 1
		if l, ok := n.Attrs["level"].(float64); ok {
			level = int(l)
		}
		fmt.Fprintf(b, "<h%d>", level)
		renderChildren(b, n)
		fmt.Fprintf(b, "</h%d>", level)
	case "paragraph":
		b.WriteString("<p>")
		renderChildren(b, n)
		b.WriteString("</p>")
	case "bulletList":
		b.WriteString("<ul>")
		renderChildren(b, n)
		b.WriteString("</ul>")
	case "orderedList":
		b.WriteString("<ol>")
		renderChildren(b, n)
		b.WriteString("</ol>")
	case "listItem":
		b.WriteString("<li>")
		renderChildren(b, n)
		b.WriteString("</li>")
	case "taskList":
		b.WriteString(`<ul class="task-list">`)
		renderChildren(b, n)
		b.WriteString("</ul>")
	case "taskItem":
		checked := ""
		if c, ok := n.Attrs["checked"].(bool); ok && c {
			checked = " checked"
		}
		fmt.Fprintf(b, `<li><input type="checkbox" disabled%s> `, checked)
		renderChildren(b, n)
		b.WriteString("</li>")
	case "codeBlock":
		b.WriteString("<pre><code>")
		renderChildren(b, n)
		b.WriteString("</code></pre>")
	case "blockquote":
		b.WriteString("<blockquote>")
		renderChildren(b, n)
		b.WriteString("</blockquote>")
	case "hardBreak":
		b.WriteString("<br>")
	case "image":
		src := ""
		if s, ok := n.Attrs["src"].(string); ok {
			src = htmlEscape(s)
		}
		alt := ""
		if a, ok := n.Attrs["alt"].(string); ok {
			alt = htmlEscape(a)
		}
		fmt.Fprintf(b, `<img src="%s" alt="%s">`, src, alt)
	case "calloutBlock":
		calloutType := "info"
		if t, ok := n.Attrs["type"].(string); ok {
			calloutType = htmlEscape(t)
		}
		fmt.Fprintf(b, `<div class="callout callout-%s">`, calloutType)
		renderChildren(b, n)
		b.WriteString("</div>")
	case "diagramBlock":
		xmlAttr := ""
		if v, ok := n.Attrs["xml"].(string); ok {
			xmlAttr = v
		}
		if xmlAttr == "" {
			return
		}
		b.WriteString(`<div class="diagram-block-placeholder" style="border:1px dashed #666;padding:12px;border-radius:4px;color:#999;font-size:13px;">`)
		b.WriteString(`[Diagram — view in BookOwl for interactive version]`)
		b.WriteString(`</div>`)
	case "horizontalRule":
		b.WriteString("<hr>")
	case "table":
		b.WriteString("<table>")
		renderChildren(b, n)
		b.WriteString("</table>")
	case "tableRow":
		b.WriteString("<tr>")
		renderChildren(b, n)
		b.WriteString("</tr>")
	case "tableCell":
		b.WriteString("<td>")
		renderChildren(b, n)
		b.WriteString("</td>")
	case "tableHeader":
		b.WriteString("<th>")
		renderChildren(b, n)
		b.WriteString("</th>")
	case "text":
		text := htmlEscape(n.Text)
		for _, m := range n.Marks {
			switch m.Type {
			case "bold":
				text = "<strong>" + text + "</strong>"
			case "italic":
				text = "<em>" + text + "</em>"
			case "code":
				text = "<code>" + text + "</code>"
			case "strike":
				text = "<s>" + text + "</s>"
			case "link":
				href := ""
				if h, ok := m.Attrs["href"].(string); ok {
					href = htmlEscape(h)
				}
				text = fmt.Sprintf(`<a href="%s">%s</a>`, href, text)
			}
		}
		b.WriteString(text)
	default:
		renderChildren(b, n)
	}
}

func renderChildren(b *strings.Builder, n tiptapNode) {
	for _, child := range n.Content {
		renderNode(b, child)
	}
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
