package secrets

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/squoyster/hydracast/internal/config"
)

type Resolver struct {
	cfg         config.SecretsConfig
	clientToken string
}

func NewResolver(cfg config.SecretsConfig) *Resolver {
	return &Resolver{cfg: cfg}
}

func (r *Resolver) Resolve(ref string) (string, error) {
	if !strings.HasPrefix(ref, "secret://") {
		return "", fmt.Errorf("invalid secret ref format: %s", ref)
	}

	// Optional "#key" selects a single field within the KV secret.
	path := strings.TrimPrefix(ref, "secret://")
	var key string
	if i := strings.IndexByte(path, '#'); i >= 0 {
		key = path[i+1:]
		path = path[:i]
	}

	if strings.HasPrefix(path, "openbao/") {
		return r.resolveOpenBao(path, key)
	}

	return "", fmt.Errorf("unknown secret scheme in ref: %s", ref)
}

func (r *Resolver) resolveOpenBao(path, key string) (string, error) {
	if r.cfg.OpenBao.Address == "" {
		if r.cfg.Fallback.Enabled {
			return r.resolveFileFallback(path, key)
		}
		return "", fmt.Errorf("openbao not configured and fallback disabled")
	}

	token, err := r.getOpenBaoToken()
	if err != nil {
		if r.cfg.Fallback.Enabled {
			return r.resolveFileFallback(path, key)
		}
		return "", fmt.Errorf("get openbao token: %w", err)
	}

	// path = "openbao/kv/hydracast/youtube/client" → mount=kv, secret=hydracast/youtube/client
	rel := strings.TrimPrefix(path, "openbao/")
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid openbao secret path: %s", path)
	}
	mount, secretPath := parts[0], parts[1]
	// ponytail: assumes KV-v2 mount ("/data/" segment); add v1 branch if a v1 mount appears.
	url := strings.TrimRight(r.cfg.OpenBao.Address, "/") + "/v1/" + mount + "/data/" + secretPath

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build openbao request: %w", err)
	}
	req.Header.Set("X-Vault-Token", token)
	if r.cfg.OpenBao.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", r.cfg.OpenBao.Namespace)
	}

	client := &http.Client{Timeout: r.cfg.OpenBao.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		if r.cfg.Fallback.Enabled {
			return r.resolveFileFallback(path, key)
		}
		return "", fmt.Errorf("openbao request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		if r.cfg.Fallback.Enabled {
			return r.resolveFileFallback(path, key)
		}
		return "", fmt.Errorf("openbao secret not found: %s", path)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if r.cfg.Fallback.Enabled {
			return r.resolveFileFallback(path, key)
		}
		return "", fmt.Errorf("openbao %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var out struct {
		Data struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode openbao response: %w", err)
	}
	if len(out.Data.Data) == 0 {
		return "", fmt.Errorf("openbao secret has no data: %s", path)
	}

	return selectField(out.Data.Data, key, path)
}

// selectField returns the named field of a KV secret, or the whole secret
// rendered as "k=v\n" lines when key is empty.
func selectField(m map[string]any, key, path string) (string, error) {
	if key == "" {
		return serializeKV(m), nil
	}
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("secret %s has no key %q", path, key)
	}
	return fmt.Sprint(v), nil
}

// serializeKV renders a multi-field secret as sorted "k=v\n" lines, which
// parseClientSecret (internal/app/auth.go) consumes. A single-field secret
// yields its value verbatim (so token/cookies/page-token read as raw values).
func serializeKV(m map[string]any) string {
	if len(m) == 1 {
		for _, v := range m {
			return fmt.Sprint(v)
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v\n", k, m[k])
	}
	return strings.TrimRight(b.String(), "\n")
}

func (r *Resolver) resolveFileFallback(path, key string) (string, error) {
	if !r.cfg.Fallback.Enabled {
		return "", fmt.Errorf("file fallback disabled")
	}

	filePath := r.cfg.Fallback.Root + "/" + strings.TrimPrefix(path, "openbao/kv/hydracast/")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read secret file %s: %w", filePath, err)
	}

	content := strings.TrimSpace(string(data))
	if key == "" {
		return content, nil
	}
	// Key requested: parse "k=v" lines (mirrors serializeKV output shape).
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"="), nil
		}
	}
	return "", fmt.Errorf("secret file %s has no key %q", filePath, key)
}

func (r *Resolver) getOpenBaoToken() (string, error) {
	if r.clientToken != "" {
		return r.clientToken, nil
	}
	if token := os.Getenv("BAO_TOKEN"); token != "" {
		r.clientToken = token
		return token, nil
	}
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		r.clientToken = token
		return token, nil
	}
	// AppRole login (preferred dynamic auth). Skipped if the creds file is absent.
	if r.cfg.OpenBao.AppRoleFile != "" {
		if _, err := os.Stat(r.cfg.OpenBao.AppRoleFile); err == nil {
			token, err := r.loginAppRole()
			if err != nil {
				return "", err
			}
			r.clientToken = token
			return token, nil
		}
	}
	// Static token file (last-resort fallback).
	if r.cfg.OpenBao.TokenFile != "" {
		if data, err := os.ReadFile(r.cfg.OpenBao.TokenFile); err == nil {
			token := strings.TrimSpace(string(data))
			r.clientToken = token
			return token, nil
		}
	}
	return "", fmt.Errorf("no openbao token: set BAO_TOKEN/VAULT_TOKEN, provide approle creds (%s), or static token file (%s)",
		r.cfg.OpenBao.AppRoleFile, r.cfg.OpenBao.TokenFile)
}

// loginAppRole authenticates via the AppRole auth method and returns a client
// token. The token lifetime is governed by the AppRole role (24h in this
// deployment); it is cached on the Resolver for the run, so no refresh logic.
func (r *Resolver) loginAppRole() (string, error) {
	roleID, secretID, err := r.readAppRoleCreds()
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]string{"role_id": roleID, "secret_id": secretID})
	authPath := r.cfg.OpenBao.AuthPath
	if authPath == "" {
		authPath = "approle"
	}
	url := strings.TrimRight(r.cfg.OpenBao.Address, "/") + "/v1/auth/" + authPath + "/login"

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build approle login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: r.cfg.OpenBao.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("approle login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("approle login %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var out struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode approle login response: %w", err)
	}
	if out.Auth.ClientToken == "" {
		return "", fmt.Errorf("approle login returned no client_token")
	}
	return out.Auth.ClientToken, nil
}

// readAppRoleCreds parses "role_id=...\nsecret_id=..." from the creds file.
func (r *Resolver) readAppRoleCreds() (roleID, secretID string, err error) {
	data, err := os.ReadFile(r.cfg.OpenBao.AppRoleFile)
	if err != nil {
		return "", "", fmt.Errorf("read approle creds %s: %w", r.cfg.OpenBao.AppRoleFile, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "role_id="):
			roleID = strings.TrimSpace(strings.TrimPrefix(line, "role_id="))
		case strings.HasPrefix(line, "secret_id="):
			secretID = strings.TrimSpace(strings.TrimPrefix(line, "secret_id="))
		}
	}
	if roleID == "" || secretID == "" {
		return "", "", fmt.Errorf("approle creds %s missing role_id or secret_id", r.cfg.OpenBao.AppRoleFile)
	}
	return roleID, secretID, nil
}

func Fingerprint(value string) string {
	h := sha256.Sum256([]byte(value))
	return fmt.Sprintf("sha256:%x", h[:4])
}

func Redact(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}
