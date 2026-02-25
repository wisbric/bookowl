package seed

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/bookowl/internal/authhandler"
	"github.com/wisbric/bookowl/internal/config"
	"github.com/wisbric/bookowl/internal/db"
	dbglobal "github.com/wisbric/bookowl/internal/db/global"
	dbtenant "github.com/wisbric/bookowl/internal/db/tenant"
)

const (
	devAPIKeyRaw = "bw_dev_seed_key_do_not_use_in_production"
	tenantSlug   = "acme"
	tenantName   = "Acme Corp"
)

// Run executes the seed (or seed-demo) mode.
func Run(ctx context.Context, cfg config.Config) error {
	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	if err := ensureGlobalSchema(ctx, pool); err != nil {
		return fmt.Errorf("ensuring global schema: %w", err)
	}

	tenantID, err := ensureTenant(ctx, pool)
	if err != nil {
		return fmt.Errorf("ensuring tenant: %w", err)
	}

	if err := ensureDevAPIKey(ctx, pool, tenantID); err != nil {
		return fmt.Errorf("ensuring dev API key: %w", err)
	}

	if err := ensureLocalAdmin(ctx, pool, cfg); err != nil {
		return fmt.Errorf("ensuring local admin: %w", err)
	}

	if err := ensureTenantSchema(ctx, pool); err != nil {
		return fmt.Errorf("ensuring tenant schema: %w", err)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO tenant_%s", tenantSlug)); err != nil {
		return fmt.Errorf("setting search_path: %w", err)
	}

	q := dbtenant.New(conn)

	users, err := ensureUsers(ctx, q)
	if err != nil {
		return fmt.Errorf("ensuring users: %w", err)
	}

	if err := ensureSystemTemplates(ctx, q); err != nil {
		return fmt.Errorf("ensuring system templates: %w", err)
	}

	slog.Info("seed complete", "tenant", tenantSlug, "users", len(users))

	if cfg.Mode == "seed-demo" {
		if err := seedDemo(ctx, q, users); err != nil {
			return fmt.Errorf("seeding demo data: %w", err)
		}
	}

	return nil
}

func ensureTenant(ctx context.Context, pool *pgxpool.Pool) ([16]byte, error) {
	gq := dbglobal.New(pool)

	t, err := gq.GetTenantBySlug(ctx, tenantSlug)
	if err == nil {
		slog.Info("tenant already exists", "slug", tenantSlug, "id", t.ID)
		return t.ID, nil
	}

	t, err = gq.CreateTenant(ctx, dbglobal.CreateTenantParams{
		Slug:   tenantSlug,
		Name:   tenantName,
		Config: json.RawMessage(`{}`),
	})
	if err != nil {
		return [16]byte{}, fmt.Errorf("creating tenant: %w", err)
	}

	slog.Info("created tenant", "slug", tenantSlug, "id", t.ID)
	return t.ID, nil
}

func ensureDevAPIKey(ctx context.Context, pool *pgxpool.Pool, tenantID [16]byte) error {
	gq := dbglobal.New(pool)

	hash := sha256.Sum256([]byte(devAPIKeyRaw))
	hashHex := hex.EncodeToString(hash[:])

	_, err := gq.GetAPIKeyByHash(ctx, hashHex)
	if err == nil {
		slog.Info("dev API key already exists")
		return nil
	}

	_, err = gq.CreateAPIKey(ctx, dbglobal.CreateAPIKeyParams{
		TenantID:    tenantID,
		KeyHash:     hashHex,
		KeyPrefix:   "bw_dev",
		Description: "Dev seed key — do not use in production",
		Role:        "admin",
	})
	if err != nil {
		return fmt.Errorf("creating API key: %w", err)
	}

	slog.Info("created dev API key")
	return nil
}

func ensureGlobalSchema(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := "migrations/global"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading global migrations dir: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, f := range upFiles {
		sql, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", f, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("running migration %s: %w", f, err)
			}
		}
		slog.Debug("applied global migration", "file", f)
	}
	return nil
}

func ensureLocalAdmin(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) error {
	gq := dbglobal.New(pool)

	// Check if local admin already exists.
	_, err := gq.GetLocalAdminByTenant(ctx, tenantSlug)
	if err == nil {
		slog.Info("local admin already exists", "tenant", tenantSlug)
		return nil
	}

	// Determine password.
	password := cfg.AdminPassword
	generated := false
	if password == "" {
		if cfg.DevMode || cfg.Mode == "seed-demo" {
			password = "bookowl-admin"
		} else {
			b := make([]byte, 12)
			if _, err := rand.Read(b); err != nil {
				return fmt.Errorf("generating random password: %w", err)
			}
			password = hex.EncodeToString(b)[:16]
			generated = true
		}
	}

	hash, err := authhandler.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hashing admin password: %w", err)
	}

	_, err = gq.CreateLocalAdmin(ctx, dbglobal.CreateLocalAdminParams{
		TenantSlug:   tenantSlug,
		Username:     "admin",
		PasswordHash: hash,
		MustChange:   true,
	})
	if err != nil {
		return fmt.Errorf("creating local admin: %w", err)
	}

	fmt.Println("✓ Local admin created")
	fmt.Println("  Username: admin")
	fmt.Printf("  Password: %s\n", password)
	if generated {
		fmt.Println("  ⚠ Save this password — it will not be shown again")
	}

	return nil
}

func ensureTenantSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := fmt.Sprintf("tenant_%s", tenantSlug)

	// Create schema if not exists.
	_, err := pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema))
	if err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}

	// Run tenant migrations in order.
	migrationsDir := "migrations/tenant"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
		return fmt.Errorf("setting search_path: %w", err)
	}

	for _, f := range upFiles {
		sql, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", f, err)
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			// Table already exists is fine — we're idempotent.
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("running migration %s: %w", f, err)
			}
		}
		slog.Debug("applied migration", "file", f)
	}
	conn.Release()

	return nil
}

type seedUser struct {
	name string
	user dbtenant.User
}

func ensureUsers(ctx context.Context, q *dbtenant.Queries) ([]seedUser, error) {
	type userDef struct {
		externalID  string
		email       string
		displayName string
		role        string
	}

	defs := []userDef{
		{"seed-stefan", "stefan@acme.example", "Stefan K.", "admin"},
		{"seed-max", "max@acme.example", "Max B.", "editor"},
		{"seed-anna", "anna@acme.example", "Anna L.", "editor"},
	}

	var users []seedUser
	for _, d := range defs {
		u, err := q.UpsertUser(ctx, dbtenant.UpsertUserParams{
			ExternalID:  d.externalID,
			Email:       d.email,
			DisplayName: d.displayName,
			Role:        d.role,
		})
		if err != nil {
			return nil, fmt.Errorf("upserting user %s: %w", d.displayName, err)
		}
		users = append(users, seedUser{name: d.displayName, user: u})
	}

	return users, nil
}

func ensureSystemTemplates(ctx context.Context, q *dbtenant.Queries) error {
	templatesDir := "internal/seed/templates"
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return fmt.Errorf("reading templates dir: %w", err)
	}

	type templateDef struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		DocType     string          `json:"doc_type"`
		Icon        string          `json:"icon"`
		SortOrder   int32           `json:"sort_order"`
		Content     json.RawMessage `json:"content"`
	}

	var seeded int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(templatesDir, e.Name()))
		if err != nil {
			return fmt.Errorf("reading template %s: %w", e.Name(), err)
		}

		var def templateDef
		if err := json.Unmarshal(data, &def); err != nil {
			return fmt.Errorf("parsing template %s: %w", e.Name(), err)
		}

		// Check if this system template already exists by name.
		_, err = q.GetSystemTemplateByName(ctx, def.Name)
		if err == nil {
			slog.Debug("system template already exists", "name", def.Name)
			continue
		}

		_, err = q.CreateTemplate(ctx, dbtenant.CreateTemplateParams{
			Name:        def.Name,
			Description: db.ValidText(def.Description),
			DocType:     def.DocType,
			Content:     def.Content,
			Icon:        db.NullText(&def.Icon),
			IsSystem:    true,
			IsGlobal:    true,
			SortOrder:   def.SortOrder,
		})
		if err != nil {
			return fmt.Errorf("creating system template %q: %w", def.Name, err)
		}
		seeded++
	}

	if seeded > 0 {
		slog.Info("seeded system templates", "count", seeded)
	}
	return nil
}

func seedDemo(ctx context.Context, q *dbtenant.Queries, users []seedUser) error {
	slog.Info("seeding demo data")

	stefan := users[0].user
	max := users[1].user
	anna := users[2].user

	createdBy := db.ValidUUID(stefan.ID)

	// --- Space: Platform Engineering ---
	platSpace, err := findOrCreateSpace(ctx, q, "Platform Engineering", "platform-engineering", "Central engineering documentation hub", "🏗️", createdBy)
	if err != nil {
		return err
	}

	for _, u := range users {
		_, _ = q.AddSpaceMember(ctx, dbtenant.AddSpaceMemberParams{
			SpaceID: platSpace.ID,
			UserID:  u.user.ID,
			Role:    "editor",
		})
	}

	k8sColl, err := findOrCreateCollection(ctx, q, platSpace.ID, "Kubernetes", "kubernetes", "☸️", 0, createdBy)
	if err != nil {
		return err
	}

	pmColl, err := findOrCreateCollection(ctx, q, platSpace.ID, "Post-mortems", "post-mortems", "🔥", 1, createdBy)
	if err != nil {
		return err
	}

	archColl, err := findOrCreateCollection(ctx, q, platSpace.ID, "Architecture", "architecture", "📐", 2, createdBy)
	if err != nil {
		return err
	}

	// --- Space: On-Call Runbooks ---
	oncallSpace, err := findOrCreateSpace(ctx, q, "On-Call Runbooks", "on-call-runbooks", "Operational runbooks for on-call engineers", "📟", createdBy)
	if err != nil {
		return err
	}

	for _, u := range users {
		_, _ = q.AddSpaceMember(ctx, dbtenant.AddSpaceMemberParams{
			SpaceID: oncallSpace.ID,
			UserID:  u.user.ID,
			Role:    "editor",
		})
	}

	alertsColl, err := findOrCreateCollection(ctx, q, oncallSpace.ID, "Alerts", "alerts", "🔔", 0, createdBy)
	if err != nil {
		return err
	}

	// --- Runbooks ---
	runbooks := []struct {
		title   string
		slug    string
		content string
		coll    dbtenant.Collection
		author  dbtenant.User
	}{
		{"Pod CrashLoopBackOff", "pod-crashloopbackoff", podCrashloopContent, alertsColl, stefan},
		{"Container OOMKilled", "container-oomkilled", oomKilledContent, alertsColl, stefan},
		{"TLS Certificate Expiry", "tls-cert-expiry", certExpiryContent, alertsColl, max},
		{"Node Not Ready", "node-not-ready", nodeNotReadyContent, alertsColl, max},
		{"DNS Resolution Failures", "dns-resolution-failures", dnsResolutionContent, alertsColl, anna},
		{"High CPU Usage Troubleshooting", "high-cpu-troubleshooting", highCPUContent, alertsColl, stefan},
		{"Ingress Troubleshooting Guide", "ingress-troubleshooting", ingressTroubleshootContent, alertsColl, max},
		{"Secrets and Credential Rotation", "secrets-rotation", secretsRotationContent, alertsColl, anna},
		{"PVC Stuck Pending", "pvc-stuck-pending", pvcStuckContent, k8sColl, anna},
		{"etcd High Latency", "etcd-high-latency", etcdLatencyContent, k8sColl, anna},
	}

	for i, rb := range runbooks {
		_, err := createDocIfNotExists(ctx, q, dbtenant.CreateDocumentParams{
			SpaceID:      rb.coll.SpaceID,
			CollectionID: db.ValidUUID(rb.coll.ID),
			Title:        rb.title,
			Slug:         rb.slug,
			Content:      json.RawMessage(rb.content),
			ContentText:  extractText(rb.content),
			DocType:      "runbook",
			Status:       "published",
			Tags:         []string{"runbook", "on-call"},
			Icon:         db.ValidText("📋"),
			Position:     int32(i),
			CreatedBy:    db.ValidUUID(rb.author.ID),
		})
		if err != nil {
			return fmt.Errorf("creating runbook %q: %w", rb.title, err)
		}
	}

	// --- 2 Post-mortems ---
	pms := []struct {
		title   string
		slug    string
		content string
		author  dbtenant.User
	}{
		{"Database Connection Pool Exhaustion — 2025-12-15", "db-pool-exhaustion-2025-12-15", pmPoolExhaustionContent, stefan},
		{"CDN Cache Invalidation Storm — 2026-01-08", "cdn-cache-storm-2026-01-08", pmCDNCacheContent, max},
	}

	for i, pm := range pms {
		_, err := createDocIfNotExists(ctx, q, dbtenant.CreateDocumentParams{
			SpaceID:      platSpace.ID,
			CollectionID: db.ValidUUID(pmColl.ID),
			Title:        pm.title,
			Slug:         pm.slug,
			Content:      json.RawMessage(pm.content),
			ContentText:  extractText(pm.content),
			DocType:      "post-mortem",
			Status:       "published",
			Tags:         []string{"post-mortem", "incident"},
			Icon:         db.ValidText("🔥"),
			Position:     int32(i),
			CreatedBy:    db.ValidUUID(pm.author.ID),
		})
		if err != nil {
			return fmt.Errorf("creating post-mortem %q: %w", pm.title, err)
		}
	}

	// --- Architecture, SOPs, and guides ---
	archDocs := []struct {
		title   string
		slug    string
		content string
		docType string
		tags    []string
		coll    dbtenant.Collection
		author  dbtenant.User
	}{
		{"Platform Service Overview", "platform-service-overview", liveContextDocContent, "page", []string{"architecture", "live-context"}, archColl, stefan},
		{"ADR-001: Kubernetes as Container Orchestration Platform", "adr-001-kubernetes", adrKubernetesContent, "page", []string{"architecture", "adr"}, archColl, stefan},
		{"Monitoring and Alerting Standards", "monitoring-alerting-standards", monitoringGuideContent, "page", []string{"architecture", "monitoring"}, archColl, max},
		{"On-Call Engineer Onboarding Guide", "on-call-onboarding", onboardingGuideContent, "sop", []string{"onboarding", "on-call"}, k8sColl, stefan},
		{"Production Deployment Procedure", "production-deployment-procedure", deploymentSOPContent, "sop", []string{"deployment", "sop"}, k8sColl, anna},
	}

	for i, doc := range archDocs {
		icon := "📄"
		if doc.docType == "sop" {
			icon = "📋"
		}
		_, err := createDocIfNotExists(ctx, q, dbtenant.CreateDocumentParams{
			SpaceID:      doc.coll.SpaceID,
			CollectionID: db.ValidUUID(doc.coll.ID),
			Title:        doc.title,
			Slug:         doc.slug,
			Content:      json.RawMessage(doc.content),
			ContentText:  extractText(doc.content),
			DocType:      doc.docType,
			Status:       "published",
			Tags:         doc.tags,
			Icon:         db.ValidText(icon),
			Position:     int32(i + 10), // Offset to avoid collision with runbooks
			CreatedBy:    db.ValidUUID(doc.author.ID),
		})
		if err != nil {
			return fmt.Errorf("creating doc %q: %w", doc.title, err)
		}
	}

	slog.Info("demo seed complete",
		"runbooks", len(runbooks),
		"post_mortems", len(pms),
		"docs", len(archDocs),
	)
	return nil
}

func findOrCreateSpace(ctx context.Context, q *dbtenant.Queries, name, slug, desc, icon string, createdBy pgtype.UUID) (dbtenant.Space, error) {
	sp, err := q.GetSpaceBySlug(ctx, slug)
	if err == nil {
		return sp, nil
	}
	return q.CreateSpace(ctx, dbtenant.CreateSpaceParams{
		Name:        name,
		Slug:        slug,
		Description: db.ValidText(desc),
		Icon:        db.ValidText(icon),
		IsPrivate:   pgtype.Bool{Bool: false, Valid: true},
		CreatedBy:   createdBy,
	})
}

func findOrCreateCollection(ctx context.Context, q *dbtenant.Queries, spaceID [16]byte, name, slug, icon string, pos int32, createdBy pgtype.UUID) (dbtenant.Collection, error) {
	coll, err := q.CreateCollection(ctx, dbtenant.CreateCollectionParams{
		SpaceID:   spaceID,
		Name:      name,
		Slug:      slug,
		Icon:      db.ValidText(icon),
		Position:  pos,
		CreatedBy: createdBy,
	})
	if err != nil && db.IsUniqueViolation(err) {
		// Already exists — look it up via list.
		colls, err2 := q.ListCollectionsBySpace(ctx, spaceID)
		if err2 != nil {
			return dbtenant.Collection{}, err2
		}
		for _, c := range colls {
			if c.Slug == slug {
				return c, nil
			}
		}
		return dbtenant.Collection{}, fmt.Errorf("collection %q created but not found", slug)
	}
	return coll, err
}

func createDocIfNotExists(ctx context.Context, q *dbtenant.Queries, params dbtenant.CreateDocumentParams) (dbtenant.Document, error) {
	existing, err := q.GetDocumentBySlug(ctx, dbtenant.GetDocumentBySlugParams{
		SpaceID: params.SpaceID,
		Slug:    params.Slug,
	})
	if err == nil {
		// Document exists — ensure collection_id is set correctly.
		if params.CollectionID.Valid && !existing.CollectionID.Valid {
			_ = q.MoveDocumentToCollection(ctx, dbtenant.MoveDocumentToCollectionParams{
				CollectionID: params.CollectionID,
				ID:           existing.ID,
			})
		}
		return existing, nil
	}
	return q.CreateDocument(ctx, params)
}

func extractText(contentJSON string) string {
	var doc struct {
		Content []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(contentJSON), &doc); err != nil {
		return ""
	}
	var parts []string
	for _, block := range doc.Content {
		for _, inline := range block.Content {
			if inline.Text != "" {
				parts = append(parts, inline.Text)
			}
		}
	}
	return strings.Join(parts, " ")
}
