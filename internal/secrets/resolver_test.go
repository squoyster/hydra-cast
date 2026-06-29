package secrets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	val, err := r.resolveFileFallback("openbao/kv/hydracast/youtube/client", "")
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

	_, err := r.resolveFileFallback("openbao/kv/hydracast/youtube/client", "")
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
		name string
		env  map[string]string
		want string
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

func TestResolveOpenBaoKV2(t *testing.T) {
	var gotPath, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotPath = req.URL.Path
		gotToken = req.Header.Get("X-Vault-Token")
		fields := map[string]any{
			"client_id":     "the-id",
			"client_secret": "the-secret",
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"data": fields, "metadata": map[string]any{}},
		})
	}))
	defer srv.Close()

	t.Setenv("BAO_TOKEN", "tok")
	r := NewResolver(config.SecretsConfig{
		Provider: "openbao",
		OpenBao:  config.OpenBaoConfig{Address: srv.URL, Mount: "kv", AppPath: "hydracast"},
	})

	t.Run("key_selector", func(t *testing.T) {
		got, err := r.Resolve("secret://openbao/kv/hydracast/youtube/client#client_secret")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got != "the-secret" {
			t.Errorf("got %q, want the-secret", got)
		}
	})

	t.Run("whole_secret_serialized", func(t *testing.T) {
		got, err := r.Resolve("secret://openbao/kv/hydracast/youtube/client")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		want := "client_id=the-id\nclient_secret=the-secret"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	if gotPath != "/v1/kv/data/hydracast/youtube/client" {
		t.Errorf("requested path %q", gotPath)
	}
	if gotToken != "tok" {
		t.Errorf("token header %q", gotToken)
	}
}

func TestResolveOpenBaoMissingKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"data": map[string]any{"client_id": "the-id"}},
		})
	}))
	defer srv.Close()

	t.Setenv("BAO_TOKEN", "tok")
	r := NewResolver(config.SecretsConfig{
		Provider: "openbao",
		OpenBao:  config.OpenBaoConfig{Address: srv.URL},
	})

	_, err := r.Resolve("secret://openbao/kv/hydracast/youtube/client#nope")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestResolveFileFallbackKey(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "youtube", "client")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretPath, []byte("client_id=the-id\nclient_secret=the-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(config.SecretsConfig{
		Fallback: config.FallbackConfig{Enabled: true, Root: dir},
	})

	got, err := r.Resolve("secret://openbao/kv/hydracast/youtube/client#client_secret")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "the-secret" {
		t.Errorf("got %q, want the-secret", got)
	}
}

func TestResolveViaAppRoleLogin(t *testing.T) {
	var kvToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/auth/approle/login":
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			if body["role_id"] == "" || body["secret_id"] == "" {
				http.Error(w, "missing creds", http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"auth": map[string]any{"client_token": "approle-issued-token"},
			})
		case "/v1/kv/data/hydracast/youtube/client":
			kvToken = req.Header.Get("X-Vault-Token")
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"data": map[string]any{"client_secret": "the-secret"}},
			})
		default:
			http.NotFound(w, req)
		}
	}))
	defer srv.Close()

	credsFile := filepath.Join(t.TempDir(), "creds")
	if err := os.WriteFile(credsFile, []byte("role_id=r-123\nsecret_id=s-456\n"), 0600); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(config.SecretsConfig{
		Provider: "openbao",
		OpenBao: config.OpenBaoConfig{
			Address:     srv.URL,
			AppRoleFile: credsFile,
			AuthPath:    "approle",
		},
	})

	got, err := r.Resolve("secret://openbao/kv/hydracast/youtube/client#client_secret")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "the-secret" {
		t.Errorf("got %q, want the-secret", got)
	}
	if kvToken != "approle-issued-token" {
		t.Errorf("KV request used token %q, want approle-issued-token", kvToken)
	}
}
