package livecontext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const maxResponseBody = 10 << 20 // 10 MB

// Client calls the NightOwl API for live context data.
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetOnCall(ctx context.Context, cfg TenantNightOwlConfig, rosterID string) (json.RawMessage, error) {
	u, err := url.JoinPath(cfg.APIURL, "/api/v1/rosters", rosterID, "oncall")
	if err != nil {
		return nil, fmt.Errorf("building URL: %w", err)
	}
	return c.doGet(ctx, cfg.APIKey, u)
}

func (c *Client) GetServiceAlerts(ctx context.Context, cfg TenantNightOwlConfig, serviceName string) (json.RawMessage, error) {
	base, err := url.Parse(cfg.APIURL)
	if err != nil {
		return nil, fmt.Errorf("parsing API URL: %w", err)
	}
	base.Path += "/api/v1/alerts"
	q := base.Query()
	q.Set("status", "firing")
	q.Set("service_name", serviceName)
	q.Set("limit", "10")
	base.RawQuery = q.Encode()
	return c.doGet(ctx, cfg.APIKey, base.String())
}

func (c *Client) GetActiveAlerts(ctx context.Context, cfg TenantNightOwlConfig, severity string, limit int) (json.RawMessage, error) {
	base, err := url.Parse(cfg.APIURL)
	if err != nil {
		return nil, fmt.Errorf("parsing API URL: %w", err)
	}
	base.Path += "/api/v1/alerts"
	q := base.Query()
	q.Set("status", "firing")
	q.Set("severity", severity)
	q.Set("limit", strconv.Itoa(limit))
	base.RawQuery = q.Encode()
	return c.doGet(ctx, cfg.APIKey, base.String())
}

func (c *Client) GetIncident(ctx context.Context, cfg TenantNightOwlConfig, incidentID string) (json.RawMessage, error) {
	u, err := url.JoinPath(cfg.APIURL, "/api/v1/incidents", incidentID)
	if err != nil {
		return nil, fmt.Errorf("building URL: %w", err)
	}
	return c.doGet(ctx, cfg.APIKey, u)
}

func (c *Client) doGet(ctx context.Context, apiKey, reqURL string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling NightOwl: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NightOwl returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return json.RawMessage(body), nil
}
