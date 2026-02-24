package admin

import "testing"

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"long key", "ow_abcdefghijk12345", "ow_a****2345"},
		{"short key", "abc", "abc"},
		{"exact 8", "12345678", "12345678"},
		{"9 chars", "123456789", "1234****6789"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskKey(tt.in)
			if got != tt.want {
				t.Errorf("maskKey(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
