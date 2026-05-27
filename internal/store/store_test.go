package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	return s
}

func TestNewStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer s.Close()

	if s.db == nil {
		t.Error("db is nil")
	}
}

func TestMigrate(t *testing.T) {
	s := newTestStore(t)

	tables := []string{"media_items", "jobs", "publish_results", "job_events"}
	for _, table := range tables {
		var count int
		err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestUpsertMediaItem(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.UpsertMediaItem(ctx, "test-source", "facebook_page_videos", "ext-001", "https://example.com", "Test Video", "video", "fp-001", "", nil)
	if err != nil {
		t.Fatalf("UpsertMediaItem() error: %v", err)
	}
	if id == 0 {
		t.Error("UpsertMediaItem() returned 0")
	}

	id2, err := s.UpsertMediaItem(ctx, "test-source", "facebook_page_videos", "ext-001", "https://example.com", "Test Video", "video", "fp-001", "", nil)
	if err != nil {
		t.Fatalf("UpsertMediaItem() duplicate error: %v", err)
	}
	if id2 != id {
		t.Errorf("UpsertMediaItem() duplicate returned different id: %d != %d", id2, id)
	}
}

func TestCreateJob(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	mediaID, err := s.UpsertMediaItem(ctx, "test-source", "facebook_page_videos", "ext-001", "https://example.com", "Test Video", "video", "fp-001", "", nil)
	if err != nil {
		t.Fatalf("UpsertMediaItem() error: %v", err)
	}

	jobID, err := s.CreateJob(ctx, mediaID, "sync", "download_pending")
	if err != nil {
		t.Fatalf("CreateJob() error: %v", err)
	}
	if jobID == 0 {
		t.Error("CreateJob() returned 0")
	}
}

func TestUpdateJobStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	mediaID, _ := s.UpsertMediaItem(ctx, "test-source", "facebook_page_videos", "ext-001", "https://example.com", "Test Video", "video", "fp-001", "", nil)
	jobID, _ := s.CreateJob(ctx, mediaID, "sync", "download_pending")

	if err := s.UpdateJobStatus(ctx, jobID, "failed", "test error"); err != nil {
		t.Fatalf("UpdateJobStatus() error: %v", err)
	}

	var status, errMsg string
	err := s.db.QueryRow("SELECT status, error_message FROM jobs WHERE id = ?", jobID).Scan(&status, &errMsg)
	if err != nil {
		t.Fatalf("query job error: %v", err)
	}
	if status != "failed" {
		t.Errorf("status = %q, want %q", status, "failed")
	}
	if errMsg != "test error" {
		t.Errorf("error_message = %q, want %q", errMsg, "test error")
	}
}

func TestRecordEvent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	jobID := int64(1)
	if err := s.RecordEvent(ctx, &jobID, "info", "sync", "test event", `{"key":"value"}`); err != nil {
		t.Fatalf("RecordEvent() error: %v", err)
	}

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM job_events").Scan(&count)
	if err != nil {
		t.Fatalf("query events error: %v", err)
	}
	if count != 1 {
		t.Errorf("event count = %d, want 1", count)
	}
}

func TestPruneEvents(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	jobID := int64(1)
	for i := 0; i < 5; i++ {
		if err := s.RecordEvent(ctx, &jobID, "info", "sync", "event", ""); err != nil {
			t.Fatalf("RecordEvent() error: %v", err)
		}
	}

	if err := s.PruneEvents(ctx, 2); err != nil {
		t.Fatalf("PruneEvents() error: %v", err)
	}

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM job_events").Scan(&count)
	if err != nil {
		t.Fatalf("query events error: %v", err)
	}
	if count != 2 {
		t.Errorf("event count after prune = %d, want 2", count)
	}
}

func TestGetFailedJobs(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	mediaID, _ := s.UpsertMediaItem(ctx, "test-source", "facebook_page_videos", "ext-001", "https://example.com", "Test Video", "video", "fp-001", "", nil)
	jobID, _ := s.CreateJob(ctx, mediaID, "sync", "failed")
	_ = s.UpdateJobStatus(ctx, jobID, "failed", "test error")

	jobs, err := s.GetFailedJobs(ctx)
	if err != nil {
		t.Fatalf("GetFailedJobs() error: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("GetFailedJobs() len = %d, want 1", len(jobs))
	}
	if jobs[0].Title != "Test Video" {
		t.Errorf("GetFailedJobs()[0].Title = %q, want %q", jobs[0].Title, "Test Video")
	}
}

func TestUpsertMediaItemWithPublishedAt(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	pubTime := time.Now().UTC()
	id, err := s.UpsertMediaItem(ctx, "test-source", "facebook_page_videos", "ext-002", "https://example.com", "Test Video 2", "video", "fp-002", "", &pubTime)
	if err != nil {
		t.Fatalf("UpsertMediaItem() error: %v", err)
	}
	if id == 0 {
		t.Error("UpsertMediaItem() returned 0")
	}
}
