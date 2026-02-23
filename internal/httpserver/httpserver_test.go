package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespond(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
		wantBody   string
	}{
		{
			"200 with data",
			http.StatusOK,
			map[string]string{"key": "value"},
			200,
			`{"key":"value"}`,
		},
		{
			"201 with nil data",
			http.StatusCreated,
			nil,
			201,
			"",
		},
		{
			"404 with error body",
			http.StatusNotFound,
			ErrorBody{Error: "Not Found", Message: "resource missing"},
			404,
			`{"error":"Not Found","message":"resource missing"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Respond(w, tt.status, tt.data)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.data != nil {
				ct := w.Header().Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
				got := strings.TrimSpace(w.Body.String())
				if got != tt.wantBody {
					t.Errorf("body = %q, want %q", got, tt.wantBody)
				}
			}
		})
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusBadRequest, "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Error != "Bad Request" {
		t.Errorf("error = %q, want %q", body.Error, "Bad Request")
	}
	if body.Message != "invalid input" {
		t.Errorf("message = %q, want %q", body.Message, "invalid input")
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int32
		wantOffset int32
	}{
		{"defaults", "", 50, 0},
		{"custom limit", "limit=10", 10, 0},
		{"custom offset", "offset=20", 50, 20},
		{"both", "limit=25&offset=50", 25, 50},
		{"limit too high", "limit=500", 50, 0},
		{"limit negative", "limit=-1", 50, 0},
		{"limit zero", "limit=0", 50, 0},
		{"offset negative", "offset=-5", 50, 0},
		{"invalid limit", "limit=abc", 50, 0},
		{"invalid offset", "offset=xyz", 50, 0},
		{"limit at max", "limit=200", 200, 0},
		{"limit above max", "limit=201", 50, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/?"+tt.query, nil)
			p := ParsePagination(r)
			if p.Limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", p.Limit, tt.wantLimit)
			}
			if p.Offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", p.Offset, tt.wantOffset)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	t.Run("valid json", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"alice","age":30}`))
		var p payload
		if err := DecodeJSON(r, &p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Name != "alice" || p.Age != 30 {
			t.Errorf("decoded = %+v, want {Name:alice Age:30}", p)
		}
	})

	t.Run("nil body", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", nil)
		r.Body = nil
		var p payload
		err := DecodeJSON(r, &p)
		if err == nil {
			t.Fatal("expected error for nil body")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", strings.NewReader("not json"))
		var p payload
		err := DecodeJSON(r, &p)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"alice","unknown":"field"}`))
		var p payload
		err := DecodeJSON(r, &p)
		if err == nil {
			t.Fatal("expected error for unknown fields")
		}
	})
}
