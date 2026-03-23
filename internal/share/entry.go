package share

import (
	"context"
	"time"
)

// CallerInfo represents information about the calling function
type CallerInfo struct {
	File     string
	Function string
	Line     int
}

// Entry represents a single log entry
type Entry struct {
	Level     Level
	Message   string
	Fields    Fields
	Timestamp time.Time
	Caller    *CallerInfo
	Context   context.Context
	IndentStr string
}

// Writer defines the interface for log writers
type Writer interface {
	Write(entry *Entry) error
	Close() error
}
