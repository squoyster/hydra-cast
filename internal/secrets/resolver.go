package secrets

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/squoyster/hydracast/internal/config"
)

type Resolver struct {
	cfg config.SecretsConfig
}

func NewResolver(cfg config.SecretsConfig) *Resolver {
	return &Resolver{cfg: cfg}
}

func (r *Resolver) Resolve(ref string) (string, error) {
	if !strings.HasPrefix(ref, "secret://") {
		return "", fmt.Errorf("invalid secret ref format: %s", ref)
	}

	parts := strings.TrimPrefix(ref, "secret://")

	if strings.HasPrefix(parts, "openbao/") {
		return r.resolveOpenBao(parts)
	}

	return "", fmt.Errorf("unknown secret scheme in ref: %s", ref)
}

func (r *Resolver) resolveOpenBao(path string) (string, error) {
	if r.cfg.OpenBao.Address == "" {
		if r.cfg.Fallback.Enabled {
			return r.resolveFileFallback(path)
		}
		return "", fmt.Errorf("openbao not configured and fallback disabled")
	}

	token, err := r.getOpenBaoToken()
	if err != nil {
		if r.cfg.Fallback.Enabled {
			return r.resolveFileFallback(path)
		}
		return "", fmt.Errorf("get openbao token: %w", err)
	}

	_ = token

	return "", fmt.Errorf("openbao client not yet implemented (ref: %s)", path)
}

func (r *Resolver) resolveFileFallback(path string) (string, error) {
	if !r.cfg.Fallback.Enabled {
		return "", fmt.Errorf("file fallback disabled")
	}

	filePath := r.cfg.Fallback.Root + "/" + strings.TrimPrefix(path, "openbao/kv/hydracast/")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read secret file %s: %w", filePath, err)
	}

	return strings.TrimSpace(string(data)), nil
}

func (r *Resolver) getOpenBaoToken() (string, error) {
	if token := os.Getenv("BAO_TOKEN"); token != "" {
		return token, nil
	}
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		return token, nil
	}
	if r.cfg.OpenBao.TokenFile != "" {
		data, err := os.ReadFile(r.cfg.OpenBao.TokenFile)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}
	return "", fmt.Errorf("no openbao token found")
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
