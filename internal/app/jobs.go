package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/squoyster/hydracast/internal/store"
)

type JobRow struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Title       string `json:"title"`
	FinishedAt  string `json:"finished_at"`
}

func ListJobs(ctx context.Context, db *store.Store, lastN int, failedOnly bool, jsonOut bool, w io.Writer) error {
	query := `
		SELECT j.id, j.status, mi.source_name, mi.title, j.finished_at
		FROM jobs j
		JOIN media_items mi ON j.media_item_id = mi.id
	`

	if failedOnly {
		query += " WHERE j.status IN ('failed', 'retryable_failed', 'permanent_failed')"
	}

	query += " ORDER BY j.id DESC LIMIT ?"

	rows, err := db.DB().QueryContext(ctx, query, lastN)
	if err != nil {
		return fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []JobRow
	for rows.Next() {
		var j JobRow
		var dest string
		if err := rows.Scan(&j.ID, &j.Status, &j.Source, &j.Title, &j.FinishedAt); err != nil {
			return fmt.Errorf("scan job: %w", err)
		}
		j.Destination = dest
		jobs = append(jobs, j)
	}

	if jsonOut {
		data, err := json.MarshalIndent(jobs, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal jobs: %w", err)
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	fmt.Fprintf(w, "%-6s %-12s %-16s %-16s %-24s %s\n", "ID", "STATUS", "SOURCE", "DESTINATION", "TITLE", "FINISHED")
	for _, j := range jobs {
		fmt.Fprintf(w, "%-6d %-12s %-16s %-16s %-24s %s\n", j.ID, j.Status, j.Source, j.Destination, j.Title, j.FinishedAt)
	}

	return nil
}

func ListEvents(ctx context.Context, db *store.Store, lastN int, jsonOut bool, w io.Writer) error {
	query := `
		SELECT id, timestamp, level, component, message
		FROM job_events
		ORDER BY id DESC
		LIMIT ?
	`

	rows, err := db.DB().QueryContext(ctx, query, lastN)
	if err != nil {
		return fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	type EventRow struct {
		ID        int64  `json:"id"`
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		Component string `json:"component"`
		Message   string `json:"message"`
	}

	var events []EventRow
	for rows.Next() {
		var e EventRow
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Level, &e.Component, &e.Message); err != nil {
			return fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}

	if jsonOut {
		data, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal events: %w", err)
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	fmt.Fprintf(w, "%-6s %-24s %-6s %-20s %s\n", "ID", "TIMESTAMP", "LEVEL", "COMPONENT", "MESSAGE")
	for _, e := range events {
		fmt.Fprintf(w, "%-6d %-24s %-6s %-20s %s\n", e.ID, e.Timestamp, e.Level, e.Component, e.Message)
	}

	return nil
}
