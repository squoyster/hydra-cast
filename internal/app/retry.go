package app

import (
	"context"
	"fmt"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/joblog"
	"github.com/squoyster/hydracast/internal/secrets"
	"github.com/squoyster/hydracast/internal/store"
)

func RetryFailed(ctx context.Context, cfg *config.Config, db *store.Store, resolver *secrets.Resolver, logger *joblog.Logger, dryRun bool) error {
	component := logger.WithComponent("retry")

	component.Info("retrying failed jobs")

	jobs, err := db.GetFailedJobs(ctx)
	if err != nil {
		return fmt.Errorf("query failed jobs: %w", err)
	}

	count := 0
	for _, j := range jobs {
		component.Info("retrying job", "job_id", j.ID, "title", j.Title)

		if dryRun {
			component.Info("would retry (dry run)", "job_id", j.ID)
			count++
			continue
		}

		_, err := db.DB().ExecContext(ctx,
			`UPDATE jobs SET status = 'download_pending', attempts = attempts + 1, error_message = NULL WHERE id = ?`,
			j.ID,
		)
		if err != nil {
			component.Error("failed to update job status", "job_id", j.ID, "error", err)
			continue
		}

		count++
	}

	component.Info("retry complete", "retried", count)
	return nil
}
