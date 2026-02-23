package httpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func Respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("encoding response", "error", err)
		}
	}
}

func RespondError(w http.ResponseWriter, status int, msg string) {
	Respond(w, status, ErrorBody{
		Error:   http.StatusText(status),
		Message: msg,
	})
}

type Pagination struct {
	Limit  int32
	Offset int32
}

func ParsePagination(r *http.Request) Pagination {
	p := Pagination{Limit: 50, Offset: 0}

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			p.Limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = int32(n)
		}
	}
	return p
}

type ListResponse[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func DecodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("empty request body")
	}
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func URLParamUUID(r *http.Request, name string) (uuid.UUID, error) {
	raw := chi.URLParam(r, name)
	if raw == "" {
		return uuid.Nil, fmt.Errorf("missing URL param %q", name)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID %q: %w", name, err)
	}
	return id, nil
}
