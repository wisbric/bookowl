package integration

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"

	"github.com/wisbric/bookowl/internal/db"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
	"github.com/wisbric/bookowl/pkg/document"
	"github.com/wisbric/bookowl/pkg/tenant"
)

// Handler serves /api/v1/integration/ endpoints called by NightOwl.
type Handler struct {
	docSvc    *document.Service
	publicURL string // optional frontend base URL for integration links
}

func NewHandler(docSvc *document.Service, publicURL string) *Handler {
	return &Handler{docSvc: docSvc, publicURL: strings.TrimRight(publicURL, "/")}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(requireAdmin)
	r.Get("/runbooks", h.ListRunbooks)
	r.Get("/runbooks/{id}", h.GetRunbook)
	r.Post("/post-mortems", h.CreatePostMortem)
	r.Get("/search", h.Search)
	return r
}

// requireAdmin ensures the caller has role=admin (service account key).
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := auth.FromContext(r.Context())
		if identity == nil || identity.Role != "admin" {
			httpserver.RespondError(w, http.StatusForbidden, "error", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Runbook types ---

type RunbookListItem struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Tags      []string  `json:"tags"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RunbookDetail struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	ContentText string    `json:"content_text"`
	ContentHTML string    `json:"content_html"`
	URL         string    `json:"url"`
	Tags        []string  `json:"tags"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SearchResultItem struct {
	ID      uuid.UUID `json:"id"`
	Title   string    `json:"title"`
	Excerpt string    `json:"excerpt"`
	URL     string    `json:"url"`
	Score   float32   `json:"score"`
}

// --- Post-mortem types ---

type PostMortemRequest struct {
	Title     string             `json:"title"`
	SpaceSlug string             `json:"space_slug"`
	Incident  PostMortemIncident `json:"incident"`
}

type PostMortemIncident struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	RootCause  string `json:"root_cause"`
	Solution   string `json:"solution"`
	CreatedAt  string `json:"created_at"`
	ResolvedAt string `json:"resolved_at"`
	ResolvedBy string `json:"resolved_by"`
}

type PostMortemResponse struct {
	ID    uuid.UUID `json:"id"`
	URL   string    `json:"url"`
	Title string    `json:"title"`
}

// --- Handlers ---

func (h *Handler) ListRunbooks(w http.ResponseWriter, r *http.Request) {
	q := dbtenant.New(tenant.ConnFromContext(r.Context()))
	pg, _ := httpserver.ParseOffsetParams(r)

	query := r.URL.Query().Get("q")

	if query != "" {
		rows, err := q.SearchRunbooks(r.Context(), dbtenant.SearchRunbooksParams{
			Query:        query,
			ResultLimit:  int32(pg.PageSize),
			ResultOffset: int32(pg.Offset),
		})
		if err != nil {
			slog.Error("searching runbooks", "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "error", "search failed")
			return
		}
		items := make([]RunbookListItem, len(rows))
		for i, row := range rows {
			items[i] = RunbookListItem{
				ID:        row.ID,
				Title:     row.Title,
				Slug:      row.Slug,
				Tags:      row.Tags,
				URL:       h.buildDocURL(r, row.SpaceID, row.ID),
				UpdatedAt: row.UpdatedAt,
			}
		}
		httpserver.Respond(w, http.StatusOK, httpserver.OffsetPage[RunbookListItem]{
			Items:      items,
			TotalItems: len(items),
			PageSize:   pg.PageSize,
			Page:       pg.Page,
		})
		return
	}

	total, err := q.CountRunbooks(r.Context())
	if err != nil {
		slog.Error("counting runbooks", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
		return
	}

	rows, err := q.ListRunbooksWithSpace(r.Context(), dbtenant.ListRunbooksWithSpaceParams{
		Limit:  int32(pg.PageSize),
		Offset: int32(pg.Offset),
	})
	if err != nil {
		slog.Error("listing runbooks", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
		return
	}

	items := make([]RunbookListItem, len(rows))
	for i, row := range rows {
		items[i] = RunbookListItem{
			ID:        row.ID,
			Title:     row.Title,
			Slug:      row.Slug,
			Tags:      row.Tags,
			URL:       h.buildDocURL(r, row.SpaceID, row.ID),
			UpdatedAt: row.UpdatedAt,
		}
	}
	httpserver.Respond(w, http.StatusOK, httpserver.OffsetPage[RunbookListItem]{
		Items:      items,
		TotalItems: int(total),
		PageSize:   pg.PageSize,
		Page:       pg.Page,
	})
}

func (h *Handler) GetRunbook(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.URLParamUUID(r, "id")
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	q := dbtenant.New(tenant.ConnFromContext(r.Context()))
	row, err := q.GetRunbookWithSpace(r.Context(), id)
	if err != nil {
		if db.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "error", "runbook not found")
			return
		}
		slog.Error("getting runbook", "id", id, "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
		return
	}

	httpserver.Respond(w, http.StatusOK, RunbookDetail{
		ID:          row.ID,
		Title:       row.Title,
		Slug:        row.Slug,
		ContentText: row.ContentText,
		ContentHTML: renderHTML(row.Content),
		URL:         h.buildDocURL(r, row.SpaceID, row.ID),
		Tags:        row.Tags,
		UpdatedAt:   row.UpdatedAt,
	})
}

func (h *Handler) CreatePostMortem(w http.ResponseWriter, r *http.Request) {
	var req PostMortemRequest
	if !httpserver.DecodeAndValidate(w, r, &req) {

		return
	}
	if req.Title == "" || req.SpaceSlug == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "error", "title and space_slug are required")
		return
	}

	ctx := r.Context()
	conn := tenant.ConnFromContext(ctx)
	q := dbtenant.New(conn)

	// Find or create the target space.
	sp, err := q.GetSpaceBySlug(ctx, req.SpaceSlug)
	if err != nil {
		if !db.IsNotFound(err) {
			slog.Error("looking up space for post-mortem", "slug", req.SpaceSlug, "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "error", "internal error")
			return
		}
		// Create the space.
		sp, err = q.CreateSpace(ctx, dbtenant.CreateSpaceParams{
			Name:        spaceName(req.SpaceSlug),
			Slug:        req.SpaceSlug,
			Description: pgtype.Text{String: "Post-mortem documents created from NightOwl incidents", Valid: true},
		})
		if err != nil {
			slog.Error("creating space for post-mortem", "slug", req.SpaceSlug, "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to create space")
			return
		}
	}

	// Try to use the system post-mortem template from the DB; fall back to hardcoded.
	content := buildPostMortemFromDB(ctx, q, req.Incident)
	contentText := document.ExtractText(content)
	wordCount := document.CountWords(contentText)
	docSlug := slugify(req.Title)

	incidentID := req.Incident.ID

	doc, err := q.CreateDocument(ctx, dbtenant.CreateDocumentParams{
		SpaceID:            sp.ID,
		Title:              req.Title,
		Slug:               docSlug,
		Content:            content,
		ContentText:        contentText,
		DocType:            "post-mortem",
		Status:             "published",
		Tags:               []string{"post-mortem", req.Incident.Severity},
		WordCount:          wordCount,
		NightowlIncidentID: pgtype.Text{String: incidentID, Valid: incidentID != ""},
	})
	if err != nil {
		slog.Error("creating post-mortem document", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "failed to create post-mortem")
		return
	}

	httpserver.Respond(w, http.StatusCreated, PostMortemResponse{
		ID:    doc.ID,
		URL:   h.buildDocURL(r, sp.ID, doc.ID),
		Title: doc.Title,
	})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "error", "query parameter 'q' is required")
		return
	}

	pg, _ := httpserver.ParseOffsetParams(r)
	q := dbtenant.New(tenant.ConnFromContext(r.Context()))

	rows, err := q.SearchRunbooks(r.Context(), dbtenant.SearchRunbooksParams{
		Query:        query,
		ResultLimit:  int32(pg.PageSize),
		ResultOffset: int32(pg.Offset),
	})
	if err != nil {
		slog.Error("integration search failed", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "error", "search failed")
		return
	}

	items := make([]SearchResultItem, len(rows))
	for i, row := range rows {
		items[i] = SearchResultItem{
			ID:      row.ID,
			Title:   row.Title,
			Excerpt: string(row.Excerpt),
			URL:     h.buildDocURL(r, row.SpaceID, row.ID),
			Score:   row.Rank,
		}
	}
	httpserver.Respond(w, http.StatusOK, map[string]any{"items": items})
}

// --- Helpers ---

func (h *Handler) buildDocURL(r *http.Request, spaceID uuid.UUID, docID uuid.UUID) string {
	if h.publicURL != "" {
		return fmt.Sprintf("%s/spaces/%s/docs/%s", h.publicURL, spaceID, docID)
	}
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	return fmt.Sprintf("%s://%s/spaces/%s/docs/%s", scheme, host, spaceID, docID)
}

func spaceName(slug string) string {
	if slug == "" {
		return slug
	}
	return strings.ToUpper(slug[:1]) + slug[1:]
}
