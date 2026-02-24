package authhandler

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{"valid password", "SecurePass123!", false},
		{"valid with symbols", "MyP@ssw0rd!!99", false},
		{"too short", "Short1!", true},
		{"no uppercase", "alllowercase1!", true},
		{"no lowercase", "ALLUPPERCASE1!", true},
		{"no digit or symbol", "AbcDefGhIjKlMn", true},
		{"exactly 12 chars valid", "AbcDefGhIj1!", false},
		{"11 chars invalid", "AbcDefGhI1!", true},
		{"only letters mixed case", "AbCdEfGhIjKl", true},
		{"unicode symbols count", "AbcDefGhIjKl§", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.pw)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword(%q) error = %v, wantErr %v", tt.pw, err, tt.wantErr)
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("TestPassword123!")
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	// Bcrypt hashes start with $2a$ or $2b$.
	if hash[0] != '$' {
		t.Errorf("expected bcrypt hash prefix, got %q", hash[:4])
	}
}
