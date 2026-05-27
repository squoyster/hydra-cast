package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/squoyster/hydracast/internal/config"
)

func TestFingerprint(t *testing.T) {
	fp := Fingerprint("test-secret")
	if len(fp) == 0 {
		t.Error("Fingerprint() returned empty string")
	}
	if fp[:6] != "sha256" {
		t.Errorf("Fingerprint() = %q, want sha256 prefix", fp)
	}
}

func TestRedact(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "sh****rt"},
		{"ab", "****"},
		{"", "****"},
		{"longer-secret-value", "lo****ue"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Redact(tt.input)
			if got != tt.want {
				t.Errorf("Redact(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveFileFallback(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "youtube", "client")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretPath, []byte("secret-value\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.SecretsConfig{
		Provider: "files",
		Fallback: config.FallbackConfig{
			Provider: "files",
			Enabled:  true,
			Root:     dir,
		},
	}

	r := NewResolver(cfg)

	val, err := r.resolveFileFallback("openbao/kv/hydracast/youtube/client")
	if err != nil {
		t.Fatalf("resolveFileFallback() error: %v", err)
	}
	if val != "secret-value" {
		t.Errorf("resolveFileFallback() = %q, want %q", val, "secret-value")
	}
}

func TestResolveFileFallbackDisabled(t *testing.T) {
	cfg := config.SecretsConfig{
		Provider: "openbao",
		Fallback: config.FallbackConfig{
			Enabled: false,
		},
	}

	r := NewResolver(cfg)

	_, err := r.resolveFileFallback("openbao/kv/hydracast/youtube/client")
	if err == nil {
		t.Error("expected error when fallback disabled")
	}
}

func TestResolveInvalidRef(t *testing.T) {
	r := NewResolver(config.SecretsConfig{})

	_, err := r.Resolve("not-a-secret-ref")
	if err == nil {
		t.Error("expected error for invalid ref")
	}
}

func TestGetOpenBaoTokenEnv(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		want  string
	}{
		{
			name: "BAO_TOKEN",
			env:  map[string]string{"BAO_TOKEN": "bao-token-value"},
			want: "bao-token-value",
		},
		{
			name: "VAULT_TOKEN",
			env:  map[string]string{"VAULT_TOKEN": "vault-token-value"},
			want: "vault-token-value",
		},
		{
			name: "BAO_TOKEN takes precedence",
			env:  map[string]string{"BAO_TOKEN": "bao", "VAULT_TOKEN": "vault"},
			want: "bao",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			r := NewResolver(config.SecretsConfig{})
			token, err := r.getOpenBaoToken()
			if err != nil {
				t.Fatalf("getOpenBaoToken() error: %v", err)
			}
			if token != tt.want {
				t.Errorf("getOpenBaoToken() = %q, want %q", token, tt.want)
			}
		})
	}
}
