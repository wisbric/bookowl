package comment

import (
	"html"
	"regexp"
	"strings"
)

var (
	boldRe       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe     = regexp.MustCompile(`_(.+?)_`)
	inlineCodeRe = regexp.MustCompile("`([^`]+)`")
	fencedCodeRe = regexp.MustCompile("(?s)```\\s*\n?(.*?)```")
	mentionRe    = regexp.MustCompile(`@([a-zA-Z0-9._-]+)`)
	urlRe        = regexp.MustCompile(`(https?://[^\s<>&"]+)`)
)

// RenderBody converts a comment body (Markdown-lite) to sanitised HTML.
// Input is HTML-escaped first, then known-safe formatting tags are injected.
func RenderBody(body string) string {
	// Escape HTML first to prevent injection.
	s := html.EscapeString(body)

	// Fenced code blocks (before inline transforms).
	s = fencedCodeRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := fencedCodeRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		return "<pre><code>" + inner[1] + "</code></pre>"
	})

	// Inline code (before bold/italic to avoid clashes).
	s = inlineCodeRe.ReplaceAllString(s, "<code>$1</code>")

	// Bold.
	s = boldRe.ReplaceAllString(s, "<strong>$1</strong>")

	// Italic.
	s = italicRe.ReplaceAllString(s, "<em>$1</em>")

	// @mentions → links.
	s = mentionRe.ReplaceAllString(s, `<a href="/profile/$1">@$1</a>`)

	// Auto-link URLs (that aren't already inside an href).
	s = autoLinkURLs(s)

	// Newlines → <br>.
	s = strings.ReplaceAll(s, "\n", "<br>")

	return s
}

// autoLinkURLs converts bare URLs into <a> tags, skipping URLs already inside href="...".
func autoLinkURLs(s string) string {
	result := urlRe.ReplaceAllStringFunc(s, func(match string) string {
		idx := strings.Index(s, match)
		if idx > 0 {
			before := s[:idx]
			if strings.HasSuffix(before, `href="`) || strings.HasSuffix(before, `">`) {
				return match
			}
		}
		return `<a href="` + match + `">` + match + `</a>`
	})
	return result
}

// ExtractMentions returns the usernames mentioned with @username in the body.
func ExtractMentions(body string) []string {
	matches := mentionRe.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	usernames := make([]string, 0, len(matches))
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			usernames = append(usernames, name)
		}
	}
	return usernames
}
