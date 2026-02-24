package image

import (
	"testing"

	"github.com/google/uuid"
)

func TestExtractIDFromImageURL(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name string
		src  string
		want uuid.UUID
	}{
		{"valid URL", "/api/v1/images/" + id.String(), id},
		{"with extension", "/api/v1/images/" + id.String() + ".jpg", id},
		{"with query string", "/api/v1/images/" + id.String() + "?w=100", id},
		{"external URL", "https://example.com/image.jpg", uuid.Nil},
		{"empty", "", uuid.Nil},
		{"wrong prefix", "/images/" + id.String(), uuid.Nil},
		{"invalid UUID", "/api/v1/images/not-a-uuid", uuid.Nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIDFromImageURL(tt.src)
			if got != tt.want {
				t.Errorf("extractIDFromImageURL(%q) = %v, want %v", tt.src, got, tt.want)
			}
		})
	}
}

func TestExtractImageIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			"no images",
			`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}`,
			0,
		},
		{
			"one image",
			`{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/images/` + id1.String() + `"}}]}`,
			1,
		},
		{
			"two images",
			`{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/images/` + id1.String() + `"}},{"type":"image","attrs":{"src":"/api/v1/images/` + id2.String() + `"}}]}`,
			2,
		},
		{
			"external image ignored",
			`{"type":"doc","content":[{"type":"image","attrs":{"src":"https://example.com/img.png"}}]}`,
			0,
		},
		{
			"mixed: one internal, one external",
			`{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/images/` + id1.String() + `"}},{"type":"image","attrs":{"src":"https://example.com/img.png"}}]}`,
			1,
		},
		{
			"nested in blockquote",
			`{"type":"doc","content":[{"type":"blockquote","content":[{"type":"image","attrs":{"src":"/api/v1/images/` + id1.String() + `"}}]}]}`,
			1,
		},
		{
			"empty content",
			"",
			0,
		},
		{
			"invalid JSON",
			"not json",
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := ExtractImageIDs([]byte(tt.content))
			if len(ids) != tt.want {
				t.Errorf("ExtractImageIDs() returned %d IDs, want %d", len(ids), tt.want)
			}
		})
	}
}

func TestExtractImageIDs_CorrectValues(t *testing.T) {
	id := uuid.New()
	content := `{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/images/` + id.String() + `"}}]}`

	ids := ExtractImageIDs([]byte(content))
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID, got %d", len(ids))
	}
	if ids[0] != id {
		t.Errorf("got ID %v, want %v", ids[0], id)
	}
}
