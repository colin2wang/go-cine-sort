// Package logging provides leveled logging to console, a daily file and an
// optional sink used to forward entries to the Wails frontend.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// ParseLevel maps a level name (case-insensitive) to a Level; unknown input
// falls back to LevelInfo.
func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return LevelDebug
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

// Entry is the payload emitted to the frontend.
type Entry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

type Logger struct {
	mu      sync.Mutex
	min     Level
	file    *os.File
	sink    func(Entry)
	console bool
}

// New creates a logger writing to <dir>/cine_sort_<date>.log. dir may be empty
// to disable file logging. sink may be replaced later with SetSink.
func New(dir string, sink func(Entry)) (*Logger, error) {
	l := &Logger{min: LevelInfo, sink: sink, console: true}
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}
		name := fmt.Sprintf("cine_sort_%s.log", time.Now().Format("2006-01-02"))
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		l.file = f
	}
	return l, nil
}

func (l *Logger) SetSink(sink func(Entry)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sink = sink
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.min = level
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

func (l *Logger) log(level Level, source, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	entry := Entry{
		Time:    time.Now().Format("15:04:05.000"),
		Level:   level.String(),
		Message: msg,
		Source:  source,
	}

	l.mu.Lock()
	if level < l.min {
		l.mu.Unlock()
		return
	}
	line := fmt.Sprintf("%s [%s] [%s] %s", entry.Time, entry.Level, entry.Source, entry.Message)
	if l.console {
		fmt.Fprintln(os.Stdout, line)
	}
	if l.file != nil {
		_, _ = fmt.Fprintln(l.file, line)
	}
	sink := l.sink
	l.mu.Unlock()

	if sink != nil {
		sink(entry)
	}
}

func (l *Logger) Debug(source, format string, args ...any) {
	l.log(LevelDebug, source, format, args...)
}
func (l *Logger) Info(source, format string, args ...any) { l.log(LevelInfo, source, format, args...) }
func (l *Logger) Warn(source, format string, args ...any) { l.log(LevelWarn, source, format, args...) }
func (l *Logger) Error(source, format string, args ...any) {
	l.log(LevelError, source, format, args...)
}
