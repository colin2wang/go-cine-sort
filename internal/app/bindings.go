// Package app defines the Wails event names, payloads and bridge helpers that
// connect the Go service layer to the Vue frontend.
package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"

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
func Events(app *application.App) service.Events {
	return service.Events{
		Progress: func(p model.TaskProgress) { app.Event.Emit(EventProgress, p) },
		Item:     func(r model.ProcessResult) { app.Event.Emit(EventItem, r) },
		Done:     func(s model.TaskSummary) { app.Event.Emit(EventTaskDone, s) },
		Error: func(message string) {
			app.Event.Emit(EventTaskError, TaskError{Message: message})
		},
	}
}

// LogSink forwards every leveled log entry to the frontend log console.
func LogSink(app *application.App) func(logging.Entry) {
	return func(entry logging.Entry) {
		app.Event.Emit(EventLog, entry)
	}
}
