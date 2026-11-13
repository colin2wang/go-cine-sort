// Package app defines the Wails event names, payloads and bridge helpers that
// connect the Go service layer to the Vue frontend.
package app

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"go-cine-sort/internal/logging"
	"go-cine-sort/internal/model"
	"go-cine-sort/internal/service"
)

const (
	EventLog       = "log"
	EventProgress  = "progress"
	EventItem      = "item-result"
	EventTaskDone  = "task-done"
	EventTaskError = "task-error"
)

// TaskError is the payload for EventTaskError.
type TaskError struct {
	Message string `json:"message"`
}

// Events returns the callbacks used by the service pipeline.
func Events(ctx context.Context) service.Events {
	return service.Events{
		Progress: func(p model.TaskProgress) { wailsruntime.EventsEmit(ctx, EventProgress, p) },
		Item:     func(r model.ProcessResult) { wailsruntime.EventsEmit(ctx, EventItem, r) },
		Done:     func(s model.TaskSummary) { wailsruntime.EventsEmit(ctx, EventTaskDone, s) },
		Error: func(message string) {
			wailsruntime.EventsEmit(ctx, EventTaskError, TaskError{Message: message})
		},
	}
}

// LogSink forwards every leveled log entry to the frontend log console.
func LogSink(ctx context.Context) func(logging.Entry) {
	return func(entry logging.Entry) {
		wailsruntime.EventsEmit(ctx, EventLog, entry)
	}
}
