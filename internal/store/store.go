package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/squoyster/hydracast/internal/source"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate() error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for _, f := range entries {
		if f.IsDir() {
			continue
		}
		path := "migrations/" + f.Name()
		data, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f.Name(), err)
		}

		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("exec migration %s: %w", f.Name(), err)
		}
	}

	return nil
}

type FailedJob struct {
	ID          int64
	MediaItemID int64
	SourceName  string
	ExternalID  string
	SourceURL   string
	Title       string
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) UpsertMediaItem(ctx context.Context, sourceName, sourceType, externalID, sourceURL, title, mediaType, fingerprint, rawMetadata string, publishedAt *time.Time) (int64, error) {
	var existingID sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM media_items WHERE source_name = ? AND external_id = ?`,
		sourceName, externalID,
	).Scan(&existingID)

	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("query existing media: %w", err)
	}

	if existingID.Valid {
		return existingID.Int64, nil
	}

	var pubAt *string
	if publishedAt != nil {
		s := publishedAt.Format(time.RFC3339)
		pubAt = &s
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO media_items (source_name, source_type, external_id, source_url, title, media_type, published_at, detected_at, fingerprint, raw_metadata_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceName, sourceType, externalID, sourceURL, title, mediaType, pubAt, time.Now().UTC().Format(time.RFC3339), fingerprint, rawMetadata,
	)
	if err != nil {
		return 0, fmt.Errorf("insert media item: %w", err)
	}

	return result.LastInsertId()
}

func (s *Store) CreateJob(ctx context.Context, mediaItemID int64, jobType, status string) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (media_item_id, job_type, status, started_at, attempts) VALUES (?, ?, ?, ?, 0)`,
		mediaItemID, jobType, status, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("create job: %w", err)
	}

	return result.LastInsertId()
}

func (s *Store) UpdateJobStatus(ctx context.Context, jobID int64, status, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = ?, finished_at = ?, error_message = ? WHERE id = ?`,
		status, now, errorMessage, jobID,
	)
	return err
}

func (s *Store) RecordEvent(ctx context.Context, jobID *int64, level, component, message string, contextJSON string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO job_events (timestamp, job_id, level, component, message, context_json) VALUES (?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339), jobID, level, component, message, contextJSON,
	)
	return err
}

func (s *Store) PruneEvents(ctx context.Context, maxRetention int) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM job_events WHERE id NOT IN (SELECT id FROM job_events ORDER BY id DESC LIMIT ?)`,
		maxRetention,
	)
	return err
}

// ListPendingItems returns up to limit media_items that have no job yet (never
// attempted), in insertion order. Items with a job (published or failed) are
// excluded — failed items are owned by `retry --failed`. This is the durable
// work queue: a scan upserts items, later runs drain them here.
func (s *Store) ListPendingItems(ctx context.Context, limit int) ([]source.MediaItem, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.source_name, m.source_type, m.external_id, m.source_url, m.title, m.media_type, m.detected_at, m.fingerprint, m.raw_metadata_json
		 FROM media_items m
		 WHERE NOT EXISTS (SELECT 1 FROM jobs j WHERE j.media_item_id = m.id)
		 ORDER BY m.id ASC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query pending items: %w", err)
	}
	defer rows.Close()

	var items []source.MediaItem
	for rows.Next() {
		var m source.MediaItem
		var detectedAt string
		var title, rawMetadata sql.NullString
		if err := rows.Scan(&m.ID, &m.SourceName, &m.SourceType, &m.ExternalID, &m.SourceURL, &title, &m.MediaType, &detectedAt, &m.Fingerprint, &rawMetadata); err != nil {
			return nil, fmt.Errorf("scan pending item: %w", err)
		}
		m.Title = title.String
		m.RawMetadata = rawMetadata.String
		if t, err := time.Parse(time.RFC3339, detectedAt); err == nil {
			m.DetectedAt = t
		}
		items = append(items, m)
	}
	return items, nil
}

func (s *Store) GetFailedJobs(ctx context.Context) ([]FailedJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT j.id, j.media_item_id, mi.source_name, mi.external_id, mi.source_url, mi.title
		 FROM jobs j
		 JOIN media_items mi ON j.media_item_id = mi.id
		 WHERE j.status IN ('failed', 'retryable_failed')
		 ORDER BY j.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed jobs: %w", err)
	}
	defer rows.Close()

	var jobs []FailedJob
	for rows.Next() {
		var j FailedJob
		if err := rows.Scan(&j.ID, &j.MediaItemID, &j.SourceName, &j.ExternalID, &j.SourceURL, &j.Title); err != nil {
			return nil, fmt.Errorf("scan failed job: %w", err)
		}
		jobs = append(jobs, j)
	}

	return jobs, nil
}
