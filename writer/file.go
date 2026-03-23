package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/oxhq/tfx/internal/share"
)

// FileWriter writes logs to files with rotation support
type FileWriter struct {
	filename    string
	file        *os.File
	options     FileOptions
	currentSize int64
	mu          sync.Mutex
}

// FileOptions configuration for file writer
type FileOptions struct {
	Level       share.Level
	Format      share.Format
	MaxSize     int64 // Maximum size in bytes before rotation
	MaxBackups  int   // Maximum number of backup files to keep
	MaxAge      int   // Maximum number of days to retain files
	Compress    bool  // Whether to compress rotated files
	Permissions os.FileMode
}

// DefaultFileOptions returns sensible defaults for file writing
func DefaultFileOptions() FileOptions {
	return FileOptions{
		Level:       share.LevelInfo,
		Format:      share.FormatText,
		MaxSize:     100 * 1024 * 1024, // 100MB
		MaxBackups:  3,
		MaxAge:      30, // 30 days
		Compress:    true,
		Permissions: 0o644,
	}
}

// NewFileWriter creates a new file writer
func NewFileWriter(filename string, opts FileOptions) (*FileWriter, error) {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file
	// #nosec G304 -- the log path is an explicit API input controlled by the caller.
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, opts.Permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	writer := &FileWriter{
		filename:    filename,
		file:        file,
		options:     opts,
		currentSize: stat.Size(),
	}

	// Clean up old files
	go writer.cleanup()

	return writer, nil
}

// Write writes a log entry to file
func (w *FileWriter) Write(entry *share.Entry) error {
	if entry.Level < w.options.Level {
		return nil
	}

	// Format the entry
	var output string
	switch w.options.Format {
	case share.FormatJSON:
		output = w.formatJSON(entry)
	case share.FormatText:
		output = w.formatText(entry)
	default:
		output = w.formatText(entry)
	}

	output += "\n"

	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if rotation is needed
	if w.needsRotation(int64(len(output))) {
		if err := w.rotate(); err != nil {
			return fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	// Write to file
	n, err := w.file.WriteString(output)
	if err != nil {
		return err
	}

	// Sync the file to ensure data is written to disk
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	w.currentSize += int64(n)
	return nil
}

// needsRotation checks if the file needs rotation
func (w *FileWriter) needsRotation(additionalSize int64) bool {
	return w.currentSize+additionalSize > w.options.MaxSize
}

// rotate rotates the current log file
func (w *FileWriter) rotate() error {
	// Close current file
	if err := w.file.Close(); err != nil {
		return err
	}

	// Generate backup filename
	backupName := w.generateBackupName()

	// Rename current file to backup
	if err := os.Rename(w.filename, backupName); err != nil {
		return err
	}

	// Compress if enabled
	if w.options.Compress {
		go w.compressFile(backupName)
	}

	// Create new file
	file, err := os.OpenFile(w.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, w.options.Permissions)
	if err != nil {
		return err
	}

	w.file = file
	w.currentSize = 0

	return nil
}

// generateBackupName generates a timestamped backup filename
func (w *FileWriter) generateBackupName() string {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	ext := filepath.Ext(w.filename)
	base := strings.TrimSuffix(w.filename, ext)
	return fmt.Sprintf("%s.%s%s", base, timestamp, ext)
}

// compressFile compresses a file (placeholder - would use gzip in real implementation)
func (w *FileWriter) compressFile(filename string) {
	// This is a placeholder for file compression
	// In a real implementation, you would:
	// 1. Open the file
	// 2. Create a .gz file
	// 3. Use gzip.Writer to compress
	// 4. Remove the original file
}

// cleanup removes old log files based on MaxAge and MaxBackups
func (w *FileWriter) cleanup() {
	dir := filepath.Dir(w.filename)
	base := filepath.Base(w.filename)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext)

	// Find all log files
	files, err := filepath.Glob(filepath.Join(dir, prefix+".*"+ext))
	if err != nil {
		return
	}

	// Sort files by modification time (newest first)
	fileInfos := make([]fileInfo, 0, len(files))
	for _, file := range files {
		if file == w.filename {
			continue // Skip current file
		}

		stat, err := os.Stat(file)
		if err != nil {
			continue
		}

		fileInfos = append(fileInfos, fileInfo{
			name:    file,
			modTime: stat.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.Before(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove files exceeding MaxBackups
	if w.options.MaxBackups > 0 && len(fileInfos) > w.options.MaxBackups {
		for _, file := range fileInfos[w.options.MaxBackups:] {
			_ = os.Remove(file.name)
		}
		fileInfos = fileInfos[:w.options.MaxBackups]
	}

	// Remove files exceeding MaxAge
	if w.options.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -w.options.MaxAge)
		for _, file := range fileInfos {
			if file.modTime.Before(cutoff) {
				_ = os.Remove(file.name)
			}
		}
	}
}

// formatJSON formats entry as JSON for file output
func (w *FileWriter) formatJSON(entry *share.Entry) string {
	// Simplified JSON formatting - in real implementation use json.Marshal
	parts := []string{
		fmt.Sprintf(`"timestamp":"%s"`, entry.Timestamp.Format(time.RFC3339)),
		fmt.Sprintf(`"level":"%s"`, entry.Level.String()),
		fmt.Sprintf(`"message":"%s"`, w.escapeJSON(entry.Message)),
	}

	// Add caller if available
	if entry.Caller != nil {
		parts = append(parts, fmt.Sprintf(`"caller":"%s:%d"`, entry.Caller.File, entry.Caller.Line))
	}

	// Add fields
	keys := make([]string, 0, len(entry.Fields))
	for key := range entry.Fields {
		if key == "badge" || key == "badge_color" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := entry.Fields[key]
		parts = append(parts, fmt.Sprintf(`"%s":"%s"`, key, w.escapeJSON(fmt.Sprintf("%v", value))))
	}

	return fmt.Sprintf("{%s}", strings.Join(parts, ","))
}

// formatText formats entry as plain text for file output
func (w *FileWriter) formatText(entry *share.Entry) string {
	var parts []string

	// Timestamp
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	parts = append(parts, timestamp)

	// Level
	parts = append(parts, fmt.Sprintf("%-5s", entry.Level.String()))

	// Caller
	if entry.Caller != nil {
		caller := fmt.Sprintf("%s:%d", w.shortFilename(entry.Caller.File), entry.Caller.Line)
		parts = append(parts, fmt.Sprintf("[%s]", caller))
	}

	// Message
	parts = append(parts, entry.Message)

	// Fields
	if len(entry.Fields) > 0 {
		keys := make([]string, 0, len(entry.Fields))
		for key := range entry.Fields {
			if key == "badge" || key == "badge_color" {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var fieldParts []string
		for _, key := range keys {
			value := entry.Fields[key]
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", key, value))
		}
		if len(fieldParts) > 0 {
			parts = append(parts, fmt.Sprintf("fields=(%s)", strings.Join(fieldParts, " ")))
		}
	}

	return strings.Join(parts, " ")
}

// Helper functions
func (w *FileWriter) escapeJSON(s string) string {
	// Simple JSON escaping - in real implementation use proper JSON escaping
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func (w *FileWriter) shortFilename(filename string) string {
	parts := strings.Split(filename, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return filename
}

// Close closes the file writer
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}

// fileInfo helper struct for sorting files
type fileInfo struct {
	name    string
	modTime time.Time
}
