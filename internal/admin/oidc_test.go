package admin

import (
	"encoding/hex"
	"encoding/json"
	"testing"
)

func jsonUnmarshalHelper(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func testSecretKey() string {
	// 32-byte key as hex (64 hex chars).
	return hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
}

func TestEncryptDecryptSecret(t *testing.T) {
	key := testSecretKey()

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple", "my-client-secret"},
		{"empty", ""},
		{"long", "this-is-a-much-longer-secret-value-that-might-be-used-in-production-environments"},
		{"special chars", "s3cr3t!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptSecret(tt.plaintext, key)
			if err != nil {
				t.Fatalf("encryptSecret() error = %v", err)
			}

			if tt.plaintext != "" && encrypted == tt.plaintext {
				t.Error("encrypted value should differ from plaintext")
			}

			if tt.plaintext != "" && encrypted[:17] != "ENCRYPTED:AES256:" {
				t.Errorf("encrypted value should have ENCRYPTED:AES256: prefix, got %q", encrypted[:17])
			}

			decrypted, err := decryptSecret(encrypted, key)
			if err != nil {
				t.Fatalf("decryptSecret() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("decryptSecret() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptSecretDifferentNonce(t *testing.T) {
	key := testSecretKey()
	plaintext := "same-secret"

	enc1, err := encryptSecret(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := encryptSecret(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if enc1 == enc2 {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}

	// Both should decrypt to the same value.
	dec1, _ := decryptSecret(enc1, key)
	dec2, _ := decryptSecret(enc2, key)
	if dec1 != dec2 {
		t.Error("both ciphertexts should decrypt to the same plaintext")
	}
}

func TestDecryptSecretNotEncrypted(t *testing.T) {
	// decryptSecret should pass through non-encrypted strings.
	plain := "not-encrypted-value"
	result, err := decryptSecret(plain, testSecretKey())
	if err != nil {
		t.Fatalf("decryptSecret() error = %v", err)
	}
	if result != plain {
		t.Errorf("decryptSecret() = %q, want %q", result, plain)
	}
}

func TestEncryptSecretBadKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"too short", "abcdef"},
		{"not hex", "xyz-not-hex-0123456789abcdef0123456789abcdef012345"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptSecret("test", tt.key)
			if err == nil {
				t.Error("encryptSecret() should error with bad key")
			}
		})
	}
}

func TestDecryptSecretWrongKey(t *testing.T) {
	key1 := testSecretKey()
	key2 := hex.EncodeToString([]byte("abcdefghijklmnopabcdefghijklmnop"))

	encrypted, err := encryptSecret("my-secret", key1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = decryptSecret(encrypted, key2)
	if err == nil {
		t.Error("decryptSecret() should error when using wrong key")
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"some-secret", "bw_****...****"},
		{"ENCRYPTED:AES256:abc123", "bw_****...****"},
	}

	for _, tt := range tests {
		got := maskSecret(tt.input)
		if got != tt.want {
			t.Errorf("maskSecret(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNonNilSlice(t *testing.T) {
	var nilSlice []string
	result := nonNilSlice(nilSlice)
	if result == nil {
		t.Error("nonNilSlice(nil) should return non-nil slice")
	}
	if len(result) != 0 {
		t.Error("nonNilSlice(nil) should return empty slice")
	}

	existing := []string{"a", "b"}
	result = nonNilSlice(existing)
	if len(result) != 2 {
		t.Errorf("nonNilSlice() should preserve existing values, got %d", len(result))
	}
}

func TestLoadOIDCConfig(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   OIDCConfig
	}{
		{
			name:   "empty config",
			config: `{}`,
			want:   OIDCConfig{},
		},
		{
			name:   "no oidc section",
			config: `{"nightowl_api_url":"http://localhost"}`,
			want:   OIDCConfig{},
		},
		{
			name:   "with oidc",
			config: `{"oidc":{"issuer_url":"https://keycloak.local/realms/test","client_id":"bookowl","enabled":true}}`,
			want: OIDCConfig{
				IssuerURL: "https://keycloak.local/realms/test",
				ClientID:  "bookowl",
				Enabled:   true,
			},
		},
		{
			name:   "null config",
			config: "",
			want:   OIDCConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loadOIDCConfig([]byte(tt.config))
			if got.IssuerURL != tt.want.IssuerURL {
				t.Errorf("IssuerURL = %q, want %q", got.IssuerURL, tt.want.IssuerURL)
			}
			if got.ClientID != tt.want.ClientID {
				t.Errorf("ClientID = %q, want %q", got.ClientID, tt.want.ClientID)
			}
			if got.Enabled != tt.want.Enabled {
				t.Errorf("Enabled = %v, want %v", got.Enabled, tt.want.Enabled)
			}
		})
	}
}

func TestSaveOIDCConfig(t *testing.T) {
	existing := []byte(`{"nightowl_api_url":"http://localhost:8080"}`)
	cfg := OIDCConfig{
		IssuerURL: "https://keycloak.local/realms/test",
		ClientID:  "bookowl",
		Enabled:   true,
	}

	result, err := saveOIDCConfig(existing, cfg)
	if err != nil {
		t.Fatalf("saveOIDCConfig() error = %v", err)
	}

	// Verify the NightOwl config is preserved.
	got := loadOIDCConfig(result)
	if got.IssuerURL != cfg.IssuerURL {
		t.Errorf("IssuerURL = %q, want %q", got.IssuerURL, cfg.IssuerURL)
	}

	// Parse the full JSON to verify NightOwl config wasn't lost.
	var full map[string]interface{}
	if err := jsonUnmarshalHelper(result, &full); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if full["nightowl_api_url"] != "http://localhost:8080" {
		t.Error("saveOIDCConfig should preserve existing config keys")
	}
}

func TestOIDCTestDetailsParsing(t *testing.T) {
	// Verify the response struct can be created correctly.
	resp := OIDCTestResponse{
		OK:        true,
		LatencyMs: 42,
		Issuer:    "https://keycloak.local/realms/test",
		Details: OIDCTestDetails{
			DiscoveryOK:         true,
			JWKSURI:             "https://keycloak.local/realms/test/protocol/openid-connect/certs",
			JWKSOK:              true,
			ClientCredentialsOK: true,
			SupportedScopes:     []string{"openid", "profile", "email", "groups"},
			HasGroupsScope:      true,
		},
	}

	if !resp.OK {
		t.Error("expected OK to be true")
	}
	if resp.LatencyMs != 42 {
		t.Errorf("LatencyMs = %d, want 42", resp.LatencyMs)
	}
	if !resp.Details.HasGroupsScope {
		t.Error("expected HasGroupsScope to be true")
	}
	if len(resp.Details.SupportedScopes) != 4 {
		t.Errorf("SupportedScopes len = %d, want 4", len(resp.Details.SupportedScopes))
	}
}
