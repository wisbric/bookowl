package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/bookowl/internal/config"
	"github.com/wisbric/bookowl/internal/db"
	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/pkg/document"
)

// NightOwlRunbook is a runbook fetched from the NightOwl API.
type NightOwlRunbook struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`  // Markdown
	Category string   `json:"category"` // Maps to collection
	Tags     []string `json:"tags"`
}

type nightOwlRunbookList struct {
	Items []NightOwlRunbook `json:"items"`
	Total int               `json:"total"`
}

// RunRunbookMigration imports NightOwl runbooks into BookOwl documents.
func RunRunbookMigration(ctx context.Context, cfg config.Config) error {
	if cfg.NightOwlAPIURL == "" || cfg.NightOwlAPIKey == "" {
		return fmt.Errorf("BOOKOWL_NIGHTOWL_API_URL and BOOKOWL_NIGHTOWL_API_KEY must be set")
	}

	slog.Info("starting NightOwl runbook migration",
		"nightowl_url", cfg.NightOwlAPIURL,
		"tenant", cfg.MigrateTenantSlug,
		"target_space", cfg.MigrateTargetSpace,
		"target_collection", cfg.MigrateTargetCollection,
	)

	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	// Resolve tenant.
	globalQ := dbglobal.New(pool)
	t, err := globalQ.GetTenantBySlug(ctx, cfg.MigrateTenantSlug)
	if err != nil {
		return fmt.Errorf("tenant %q not found: %w", cfg.MigrateTenantSlug, err)
	}

	// Set search_path.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	schema := fmt.Sprintf("tenant_%s", t.Slug)
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
		return fmt.Errorf("setting search_path: %w", err)
	}

	q := dbtenant.New(conn)

	// Find or create the target space.
	sp, err := q.GetSpaceBySlug(ctx, cfg.MigrateTargetSpace)
	if err != nil {
		if !db.IsNotFound(err) {
			return fmt.Errorf("looking up space: %w", err)
		}
		name := strings.ToUpper(cfg.MigrateTargetSpace[:1]) + cfg.MigrateTargetSpace[1:]
		sp, err = q.CreateSpace(ctx, dbtenant.CreateSpaceParams{
			Name:        name,
			Slug:        cfg.MigrateTargetSpace,
			Description: pgtype.Text{String: "Imported NightOwl runbooks", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating space: %w", err)
		}
		slog.Info("created space", "slug", cfg.MigrateTargetSpace, "id", sp.ID)
	}

	// Find or create the target collection.
	var collectionID pgtype.UUID
	if cfg.MigrateTargetCollection != "" {
		// Try to find existing — not a direct query, so create if unique violation.
		col, err := q.CreateCollection(ctx, dbtenant.CreateCollectionParams{
			SpaceID: sp.ID,
			Name:    strings.ToUpper(cfg.MigrateTargetCollection[:1]) + cfg.MigrateTargetCollection[1:],
			Slug:    cfg.MigrateTargetCollection,
		})
		if err != nil {
			if !db.IsUniqueViolation(err) {
				return fmt.Errorf("creating collection: %w", err)
			}
			// Collection already exists — that's fine, but we don't have a GetCollectionBySlug.
			// We'll just leave collectionID as null.
			slog.Info("collection already exists, continuing without assigning", "slug", cfg.MigrateTargetCollection)
		} else {
			collectionID = pgtype.UUID{Bytes: col.ID, Valid: true}
			slog.Info("created collection", "slug", cfg.MigrateTargetCollection, "id", col.ID)
		}
	}

	// Fetch runbooks from NightOwl.
	client := &http.Client{Timeout: 30 * time.Second}
	var imported, skipped, failed int
	offset := 0
	limit := 50

	for {
		runbooks, total, err := fetchRunbooks(ctx, client, cfg.NightOwlAPIURL, cfg.NightOwlAPIKey, limit, offset)
		if err != nil {
			return fmt.Errorf("fetching runbooks (offset=%d): %w", offset, err)
		}

		for _, rb := range runbooks {
			// Idempotency check: use nightowl_alert_id to store original runbook ID.
			_, err := q.GetDocumentByNightOwlAlertID(ctx, pgtype.Text{String: rb.ID, Valid: true})
			if err == nil {
				skipped++
				continue
			}

			content := markdownToTiptapJSON(rb.Content)
			contentText := document.ExtractText(content)
			wordCount := document.CountWords(contentText)
			slug := slugify(rb.Title)

			tags := rb.Tags
			if tags == nil {
				tags = []string{}
			}
			tags = append(tags, "migrated")

			_, err = q.CreateDocument(ctx, dbtenant.CreateDocumentParams{
				SpaceID:         sp.ID,
				CollectionID:    collectionID,
				Title:           rb.Title,
				Slug:            slug,
				Content:         content,
				ContentText:     contentText,
				DocType:         "runbook",
				Status:          "published",
				Tags:            tags,
				WordCount:       wordCount,
				NightowlAlertID: pgtype.Text{String: rb.ID, Valid: rb.ID != ""},
			})
			if err != nil {
				slog.Error("creating document", "title", rb.Title, "error", err)
				failed++
				continue
			}
			imported++
		}

		offset += limit
		if offset >= total {
			break
		}
	}

	slog.Info("runbook migration completed",
		"imported", imported,
		"skipped", skipped,
		"failed", failed,
	)
	return nil
}

func fetchRunbooks(ctx context.Context, client *http.Client, baseURL, apiKey string, limit, offset int) ([]NightOwlRunbook, int, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing URL: %w", err)
	}
	u.Path += "/api/v1/runbooks"
	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("calling NightOwl: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("NightOwl returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("reading response: %w", err)
	}

	var list nightOwlRunbookList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, 0, fmt.Errorf("parsing response: %w", err)
	}
	return list.Items, list.Total, nil
}

// markdownToTiptapJSON converts markdown text to a basic Tiptap JSON document.
// It handles the most common markdown constructs: headings, paragraphs, code blocks,
// bullet lists, ordered lists, and inline formatting (bold, italic, code).
func markdownToTiptapJSON(md string) json.RawMessage {
	if md == "" {
		return json.RawMessage(`{"type":"doc","content":[{"type":"paragraph"}]}`)
	}

	var nodes []map[string]any
	lines := strings.Split(md, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Fenced code block.
		if strings.HasPrefix(line, "```") {
			lang := strings.TrimPrefix(line, "```")
			lang = strings.TrimSpace(lang)
			var codeLines []string
			i++
			for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
				codeLines = append(codeLines, lines[i])
				i++
			}
			if i < len(lines) {
				i++ // skip closing ```
			}
			attrs := map[string]any{}
			if lang != "" {
				attrs["language"] = lang
			}
			node := map[string]any{
				"type":  "codeBlock",
				"attrs": attrs,
				"content": []map[string]any{
					{"type": "text", "text": strings.Join(codeLines, "\n")},
				},
			}
			nodes = append(nodes, node)
			continue
		}

		// Heading.
		if strings.HasPrefix(line, "#") {
			level := 0
			for _, c := range line {
				if c == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= 6 {
				text := strings.TrimSpace(line[level:])
				nodes = append(nodes, map[string]any{
					"type":  "heading",
					"attrs": map[string]any{"level": level},
					"content": []map[string]any{
						{"type": "text", "text": text},
					},
				})
				i++
				continue
			}
		}

		// Bullet list items.
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			var items []map[string]any
			for i < len(lines) && (strings.HasPrefix(lines[i], "- ") || strings.HasPrefix(lines[i], "* ")) {
				text := strings.TrimSpace(lines[i][2:])
				items = append(items, map[string]any{
					"type": "listItem",
					"content": []map[string]any{
						{"type": "paragraph", "content": inlineContent(text)},
					},
				})
				i++
			}
			nodes = append(nodes, map[string]any{
				"type":    "bulletList",
				"content": items,
			})
			continue
		}

		// Ordered list items.
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' {
			dotIdx := strings.Index(line, ". ")
			if dotIdx > 0 && dotIdx <= 3 {
				var items []map[string]any
				for i < len(lines) {
					l := lines[i]
					di := strings.Index(l, ". ")
					if di <= 0 || di > 3 || l[0] < '0' || l[0] > '9' {
						break
					}
					text := strings.TrimSpace(l[di+2:])
					items = append(items, map[string]any{
						"type": "listItem",
						"content": []map[string]any{
							{"type": "paragraph", "content": inlineContent(text)},
						},
					})
					i++
				}
				nodes = append(nodes, map[string]any{
					"type":    "orderedList",
					"content": items,
				})
				continue
			}
		}

		// Horizontal rule.
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			nodes = append(nodes, map[string]any{"type": "horizontalRule"})
			i++
			continue
		}

		// Blockquote.
		if strings.HasPrefix(line, "> ") {
			var quoteLines []string
			for i < len(lines) && strings.HasPrefix(lines[i], "> ") {
				quoteLines = append(quoteLines, strings.TrimPrefix(lines[i], "> "))
				i++
			}
			nodes = append(nodes, map[string]any{
				"type": "blockquote",
				"content": []map[string]any{
					{"type": "paragraph", "content": inlineContent(strings.Join(quoteLines, " "))},
				},
			})
			continue
		}

		// Empty line → skip.
		if trimmed == "" {
			i++
			continue
		}

		// Default: paragraph.
		nodes = append(nodes, map[string]any{
			"type":    "paragraph",
			"content": inlineContent(line),
		})
		i++
	}

	if len(nodes) == 0 {
		nodes = append(nodes, map[string]any{"type": "paragraph"})
	}

	doc := map[string]any{
		"type":    "doc",
		"content": nodes,
	}
	b, _ := json.Marshal(doc)
	return json.RawMessage(b)
}

// inlineContent parses inline markdown (bold, italic, code) into Tiptap text nodes.
func inlineContent(text string) []map[string]any {
	if text == "" {
		return []map[string]any{{"type": "text", "text": " "}}
	}

	var nodes []map[string]any
	remaining := text

	for len(remaining) > 0 {
		// Inline code: `code`
		if idx := strings.Index(remaining, "`"); idx >= 0 {
			end := strings.Index(remaining[idx+1:], "`")
			if end >= 0 {
				if idx > 0 {
					nodes = append(nodes, map[string]any{"type": "text", "text": remaining[:idx]})
				}
				code := remaining[idx+1 : idx+1+end]
				nodes = append(nodes, map[string]any{
					"type":  "text",
					"text":  code,
					"marks": []map[string]any{{"type": "code"}},
				})
				remaining = remaining[idx+1+end+1:]
				continue
			}
		}

		// Bold: **text**
		if idx := strings.Index(remaining, "**"); idx >= 0 {
			end := strings.Index(remaining[idx+2:], "**")
			if end >= 0 {
				if idx > 0 {
					nodes = append(nodes, map[string]any{"type": "text", "text": remaining[:idx]})
				}
				bold := remaining[idx+2 : idx+2+end]
				nodes = append(nodes, map[string]any{
					"type":  "text",
					"text":  bold,
					"marks": []map[string]any{{"type": "bold"}},
				})
				remaining = remaining[idx+2+end+2:]
				continue
			}
		}

		// Italic: *text*
		if idx := strings.Index(remaining, "*"); idx >= 0 {
			end := strings.Index(remaining[idx+1:], "*")
			if end >= 0 {
				if idx > 0 {
					nodes = append(nodes, map[string]any{"type": "text", "text": remaining[:idx]})
				}
				italic := remaining[idx+1 : idx+1+end]
				nodes = append(nodes, map[string]any{
					"type":  "text",
					"text":  italic,
					"marks": []map[string]any{{"type": "italic"}},
				})
				remaining = remaining[idx+1+end+1:]
				continue
			}
		}

		// Plain text.
		nodes = append(nodes, map[string]any{"type": "text", "text": remaining})
		break
	}

	if len(nodes) == 0 {
		nodes = append(nodes, map[string]any{"type": "text", "text": text})
	}
	return nodes
}

// slugify converts a title to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteRune('-')
		}
	}
	s = b.String()
	// Collapse multiple dashes.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = s[:80]
		s = strings.TrimRight(s, "-")
	}
	return s
}
