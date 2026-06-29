package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/joblog"
	"github.com/squoyster/hydracast/internal/secrets"
	"github.com/squoyster/hydracast/internal/store"
)

func SetupYouTubeAuth(ctx context.Context, cfg *config.Config, db *store.Store, resolver *secrets.Resolver, logger *joblog.Logger, destinationName string, dryRun bool) error {
	component := logger.WithComponent("auth.youtube")

	var dstCfg *config.DestinationConfig
	for i := range cfg.Destinations {
		if cfg.Destinations[i].Name == destinationName && cfg.Destinations[i].Type == "youtube" {
			dstCfg = &cfg.Destinations[i]
			break
		}
	}

	if dstCfg == nil {
		return fmt.Errorf("destination %q not found or not a youtube type", destinationName)
	}

	if dstCfg.ClientIDRef == "" {
		return fmt.Errorf("destination %q missing client_id_ref", destinationName)
	}
	if dstCfg.ClientSecretRef == "" {
		return fmt.Errorf("destination %q missing client_secret_ref", destinationName)
	}

	if dstCfg.TokenRef == "" {
		return fmt.Errorf("destination %q missing token_ref", destinationName)
	}

	component.Info("setting up YouTube OAuth", "destination", destinationName)

	if dryRun {
		fmt.Printf("Would initiate OAuth flow for destination: %s\n", destinationName)
		fmt.Printf("Client ID ref: %s\n", dstCfg.ClientIDRef)
		fmt.Printf("Client secret ref: %s\n", dstCfg.ClientSecretRef)
		fmt.Printf("Token ref: %s\n", dstCfg.TokenRef)
		return nil
	}

	clientID, err := resolver.Resolve(dstCfg.ClientIDRef)
	if err != nil {
		return fmt.Errorf("resolve client id: %w", err)
	}
	clientSecretVal, err := resolver.Resolve(dstCfg.ClientSecretRef)
	if err != nil {
		return fmt.Errorf("resolve client secret: %w", err)
	}

	if clientID == "" || clientSecretVal == "" {
		return fmt.Errorf("client_id and client_secret must not be empty")
	}

	credDir := filepath.Join(cfg.Storage.WorkDir, ".youtube-creds")
	if err := os.MkdirAll(credDir, 0700); err != nil {
		return fmt.Errorf("create cred dir: %w", err)
	}
	defer os.RemoveAll(credDir)

	clientSecretFile := filepath.Join(credDir, "client_secret.json")
	if err := writeClientSecret(clientSecretFile, clientID, clientSecretVal); err != nil {
		return fmt.Errorf("write client secret: %w", err)
	}

	tokenFile := filepath.Join(credDir, "token.json")

	ytDlpPath := cfg.Downloaders.YtDlp.Binary
	if ytDlpPath == "" {
		ytDlpPath = "/usr/local/bin/yt-dlp"
	}

	args := []string{
		"--username", "oauth2",
		"--password", "",
		"--netrc-location", credDir,
		"--extractor-args", "youtube:client_id=" + clientID + ",client_secret=" + clientSecretVal,
		"--simulate",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	}

	cmd := exec.CommandContext(ctx, ytDlpPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Starting YouTube OAuth flow...")
	fmt.Println("Please open the following URL in your browser and authorize access:")
	fmt.Println()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("oauth flow failed: %w", err)
	}

	if _, err := os.Stat(tokenFile); err == nil {
		component.Info("OAuth token obtained", "destination", destinationName)
		fmt.Println("\nOAuth flow complete. Token saved.")
	}

	return nil
}

func writeClientSecret(path, clientID, clientSecret string) error {
	content := fmt.Sprintf(`{
  "installed": {
    "client_id": "%s",
    "client_secret": "%s",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token"
  }
}
`, clientID, clientSecret)

	return os.WriteFile(path, []byte(content), 0600)
}

func CheckSecrets(ctx context.Context, cfg *config.Config, resolver *secrets.Resolver, logger *joblog.Logger, w io.Writer) error {
	component := logger.WithComponent("secrets")
	component.Info("checking secrets")

	var refs []string
	for _, src := range cfg.Sources {
		if src.Downloader == "yt_dlp" && cfg.Downloaders.YtDlp.CookiesRef != "" {
			refs = append(refs, cfg.Downloaders.YtDlp.CookiesRef)
		}
	}

	for _, dst := range cfg.Destinations {
		if dst.ClientSecretRef != "" {
			refs = append(refs, dst.ClientSecretRef)
		}
		if dst.TokenRef != "" {
			refs = append(refs, dst.TokenRef)
		}
		if dst.PageTokenRef != "" {
			refs = append(refs, dst.PageTokenRef)
		}
	}

	allOK := true
	for _, ref := range refs {
		val, err := resolver.Resolve(ref)
		if err != nil {
			fmt.Fprintf(w, "ERROR %s: %v\n", ref, err)
			allOK = false
		} else {
			fp := secrets.Fingerprint(val)
			fmt.Fprintf(w, "OK    %s (%s)\n", ref, fp)
		}
	}

	if allOK {
		fmt.Fprintf(w, "\nAll %d secret references resolved.\n", len(refs))
	} else {
		fmt.Fprintf(w, "\nSome secret references failed.\n")
	}

	return nil
}
