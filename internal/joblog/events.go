package joblog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

type Logger struct {
	*slog.Logger
}

func New() *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String("component", component)),
	}
}

func (l *Logger) WithMediaItemID(id int64) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.Int64("media_item_id", id)),
	}
}

type EventRecorder struct {
	logger *Logger
}

func NewEventRecorder(logger *Logger) *EventRecorder {
	return &EventRecorder{logger: logger}
}

type JobEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	JobID        *int64    `json:"job_id,omitempty"`
	Level        string    `json:"level"`
	Component    string    `json:"component"`
	Message      string    `json:"message"`
	ContextJSON  string    `json:"context_json,omitempty"`
}

func (r *EventRecorder) Record(ctx context.Context, level, component, message string, context map[string]any) error {
	event := JobEvent{
		Timestamp: time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
	}

	if context != nil {
		data, err := json.Marshal(context)
		if err != nil {
			return fmt.Errorf("marshal context: %w", err)
		}
		event.ContextJSON = string(data)
	}

	r.logger.Log(ctx, slogLevel(level), message)

	return nil
}

func slogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
