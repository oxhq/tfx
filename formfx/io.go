package formfx

import (
	"context"
	"io"
)

// Reader is an interface for reading input.
type Reader interface {
	ReadLine(ctx context.Context) (string, error)
}

// Writer is an interface for writing output.
type Writer interface {
	Write(p []byte) (n int, err error)
	WriteString(s string) (n int, err error)
	Flush()
}

// StdinReader implements Reader for os.Stdin.
type StdinReader struct {
	reader io.Reader
}

// NewStdinReader creates a new StdinReader.
func NewStdinReader(r io.Reader) *StdinReader {
	return &StdinReader{reader: r}
}

// ReadLine reads a line from stdin, supporting context for timeouts/cancellation.
func (r *StdinReader) ReadLine(ctx context.Context) (string, error) {
	// This is a simplified implementation. A real implementation would
	// handle line buffering, raw mode, and proper context cancellation.
	// For now, we'll just read from the underlying reader.
	data := make([]byte, 1024)
	n, err := r.reader.Read(data)
	if err != nil {
		return "", err
	}
	return string(data[:n]), nil
}

// StdoutWriter implements Writer for os.Stdout.
type StdoutWriter struct {
	writer io.Writer
}

// NewStdoutWriter creates a new StdoutWriter.
func NewStdoutWriter(w io.Writer) *StdoutWriter {
	return &StdoutWriter{writer: w}
}

// Write writes bytes to stdout.
func (w *StdoutWriter) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

// WriteString writes a string to stdout.
func (w *StdoutWriter) WriteString(s string) (n int, err error) {
	return w.writer.Write([]byte(s))
}

// Flush flushes the writer (no-op for now).
func (w *StdoutWriter) Flush() {
	// No-op for now. In a real implementation, this might flush a buffered writer.
}
