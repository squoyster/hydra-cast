package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/squoyster/hydracast/internal/app"
	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/joblog"
	"github.com/squoyster/hydracast/internal/lock"
	"github.com/squoyster/hydracast/internal/secrets"
	"github.com/squoyster/hydracast/internal/store"
)

var (
	configPath string
	lockPath   string
	dryRun     bool
	jsonOutput bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "hydracast",
		Short: "Scheduled video syndication relay",
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "/data/config.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&lockPath, "lock-file", "/data/hydracast.lock", "lock file path to prevent overlapping runs")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would happen without making changes")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")

	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(scanCmd())
	rootCmd.AddCommand(jobsCmd())
	rootCmd.AddCommand(logCmd())
	rootCmd.AddCommand(retryCmd())
	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(secretsCmd())
	rootCmd.AddCommand(scrapeReelsCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	config.ApplyDefaults(cfg)
	return cfg, nil
}

func syncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Run scheduled sync (scan, download, transform, publish)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if errs := config.Validate(cfg); len(errs) > 0 {
				for _, e := range errs {
					cmd.PrintErrf("config error: %v\n", e)
				}
				os.Exit(1)
			}

			flock := lock.New(lockPath)

			if err := flock.TryLock(); err != nil {
				cmd.PrintErrf("lock: %v\n", err)
				os.Exit(0)
			}
			defer flock.Unlock()

			logger := joblog.New()
			resolver := secrets.NewResolver(cfg.Secrets)

			db, err := store.New(cfg.Storage.Database)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer db.Close()

			if err := db.Migrate(); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			return app.RunSync(cmd.Context(), cfg, db, resolver, logger, dryRun)
		},
	}
	return cmd
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			cmd.Printf("OK config: %s\n", configPath)

			errs := config.Validate(cfg)
			if len(errs) > 0 {
				for _, e := range errs {
					cmd.Printf("ERROR %v\n", e)
				}
				os.Exit(1)
			}

			cmd.Printf("OK database: %s\n", cfg.Storage.Database)

			for _, src := range cfg.Sources {
				if src.Enabled {
					cmd.Printf("OK source plugin: %s\n", src.Type)
				}
			}

			for _, t := range cfg.Transforms {
				if t.Enabled {
					cmd.Printf("OK transform: %s.%s\n", t.Type, t.Preset)
				}
			}

			for _, dst := range cfg.Destinations {
				if dst.Enabled {
					cmd.Printf("OK destination plugin: %s\n", dst.Type)
				}
			}

			cmd.Printf("OK secrets provider: %s\n", cfg.Secrets.Provider)

			return nil
		},
	}
	return cmd
}

func scanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan sources for new media",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if errs := config.Validate(cfg); len(errs) > 0 {
				for _, e := range errs {
					cmd.PrintErrf("config error: %v\n", e)
				}
				os.Exit(1)
			}

			logger := joblog.New()
			resolver := secrets.NewResolver(cfg.Secrets)

			db, err := store.New(cfg.Storage.Database)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer db.Close()

			if err := db.Migrate(); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			return app.RunScan(cmd.Context(), cfg, db, resolver, logger, dryRun)
		},
	}
	return cmd
}

func jobsCmd() *cobra.Command {
	var lastN int
	var failedOnly bool

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			db, err := store.New(cfg.Storage.Database)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer db.Close()

			return app.ListJobs(cmd.Context(), db, lastN, failedOnly, jsonOutput, cmd.OutOrStdout())
		},
	}

	cmd.Flags().IntVar(&lastN, "last", 20, "number of recent jobs to show")
	cmd.Flags().BoolVar(&failedOnly, "failed", false, "show only failed jobs")

	return cmd
}

func logCmd() *cobra.Command {
	var lastN int

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent job events",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			db, err := store.New(cfg.Storage.Database)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer db.Close()

			return app.ListEvents(cmd.Context(), db, lastN, jsonOutput, cmd.OutOrStdout())
		},
	}

	cmd.Flags().IntVar(&lastN, "last", 100, "number of recent events to show")

	return cmd
}

func retryCmd() *cobra.Command {
	var failedOnly bool

	cmd := &cobra.Command{
		Use:   "retry",
		Short: "Retry failed jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if errs := config.Validate(cfg); len(errs) > 0 {
				for _, e := range errs {
					cmd.PrintErrf("config error: %v\n", e)
				}
				os.Exit(1)
			}

			flock := lock.New(lockPath)

			if err := flock.TryLock(); err != nil {
				cmd.PrintErrf("lock: %v\n", err)
				os.Exit(0)
			}
			defer flock.Unlock()

			logger := joblog.New()
			resolver := secrets.NewResolver(cfg.Secrets)

			db, err := store.New(cfg.Storage.Database)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer db.Close()

			if err := db.Migrate(); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			return app.RetryFailed(cmd.Context(), cfg, db, resolver, logger, dryRun)
		},
	}

	cmd.Flags().BoolVar(&failedOnly, "failed", false, "retry only failed jobs")

	return cmd
}

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage destination authentication",
	}

	youtubeCmd := &cobra.Command{
		Use:   "youtube",
		Short: "Set up YouTube OAuth",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			logger := joblog.New()
			resolver := secrets.NewResolver(cfg.Secrets)

			db, err := store.New(cfg.Storage.Database)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer db.Close()

			if err := db.Migrate(); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			destinationName, _ := cmd.Flags().GetString("destination")
			if destinationName == "" {
				return fmt.Errorf("--destination flag is required")
			}

			return app.SetupYouTubeAuth(cmd.Context(), cfg, db, resolver, logger, destinationName, dryRun)
		},
	}

	youtubeCmd.Flags().String("destination", "", "destination name")
	_ = youtubeCmd.MarkFlagRequired("destination")

	cmd.AddCommand(youtubeCmd)
	return cmd
}

func secretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets",
	}

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check secret references",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			logger := joblog.New()
			resolver := secrets.NewResolver(cfg.Secrets)

			return app.CheckSecrets(cmd.Context(), cfg, resolver, logger, cmd.OutOrStdout())
		},
	}

	cmd.AddCommand(checkCmd)
	return cmd
}

func scrapeReelsCmd() *cobra.Command {
	var url string
	var sourceName string

	cmd := &cobra.Command{
		Use:   "scrape-reels",
		Short: "Scrape Facebook reel URLs from a page, download via yt-dlp, publish to configured destinations (unauthenticated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if errs := config.Validate(cfg); len(errs) > 0 {
				for _, e := range errs {
					cmd.PrintErrf("config error: %v\n", e)
				}
				os.Exit(1)
			}

			flock := lock.New(lockPath)

			if err := flock.TryLock(); err != nil {
				cmd.PrintErrf("lock: %v\n", err)
				os.Exit(0)
			}
			defer flock.Unlock()

			logger := joblog.New()
			resolver := secrets.NewResolver(cfg.Secrets)

			db, err := store.New(cfg.Storage.Database)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer db.Close()

			if err := db.Migrate(); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			opts := app.ScrapeReelsOptions{
				URL:        url,
				SourceName: sourceName,
			}

			return app.RunScrapeReels(cmd.Context(), cfg, db, resolver, opts, logger, dryRun)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Facebook page URL to scrape (required)")
	cmd.Flags().StringVar(&sourceName, "source", "", "configured source name whose route defines destinations (required)")
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("source")

	return cmd
}
