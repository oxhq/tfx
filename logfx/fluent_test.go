package logfx

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/internal/testutil"
)

// TestFluentLogger tests the fluent API
func TestFluentLogger(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	// Test basic If with error
	err := errors.New("test error")
	fluent := IfWith(logger, err)
	if fluent == nil {
		t.Fatal("If() returned nil")
	}

	// Test setting level
	fluent = fluent.AsError()
	if fluent.level != share.LevelError {
		t.Error("AsError() should set level to Error")
	}

	// Test message
	fluent.Msg("fluent error message")
	logger.Flush()

	out := buf.String()
	if !strings.Contains(out, "fluent error message") ||
		!strings.Contains(out, "error=test error") {
		t.Errorf(
			"expected output to contain 'fluent error message' and 'error=test error', got %s",
			out,
		)
	}
}

// TestFluentLoggerNilError tests fluent API with nil error
func TestFluentLoggerNilError(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	// Test If with nil error - should not log
	fluent := IfWith(logger, nil).AsError().WithField("test", "value")
	fluent.Msg("should not log")
	logger.Flush()

	if strings.Contains(buf.String(), "should not log") {
		t.Error("Fluent should not log when error is nil")
	}
}

// TestFluentLoggerLevels tests all fluent level methods
func TestFluentLoggerLevels(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	opts.Level = share.LevelTrace // Enable all log levels
	logger := New(opts)

	err := errors.New("test error")

	tests := []struct {
		name     string
		setupFn  func(*FluentLogger) *FluentLogger
		expected share.Level
	}{
		{"AsTrace", func(fl *FluentLogger) *FluentLogger { return fl.AsTrace() }, share.LevelTrace},
		{"AsDebug", func(fl *FluentLogger) *FluentLogger { return fl.AsDebug() }, share.LevelDebug},
		{"AsInfo", func(fl *FluentLogger) *FluentLogger { return fl.AsInfo() }, share.LevelInfo},
		{
			"AsSuccess",
			func(fl *FluentLogger) *FluentLogger { return fl.AsSuccess() },
			share.LevelSuccess,
		},
		{"AsWarn", func(fl *FluentLogger) *FluentLogger { return fl.AsWarn() }, share.LevelWarn},
		{"AsError", func(fl *FluentLogger) *FluentLogger { return fl.AsError() }, share.LevelError},
		// Skip AsFatal since Fatal calls os.Exit(1)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fluent := IfWith(logger, err)
			fluent = tt.setupFn(fluent)

			if fluent.level != tt.expected {
				t.Errorf("%s should set level to %v, got %v", tt.name, tt.expected, fluent.level)
			}

			buf.Reset()
			fluent.Msg("%s message", tt.name)
			logger.Flush()

			if !strings.Contains(buf.String(), tt.name+" message") ||
				!strings.Contains(buf.String(), "error=test error") {
				t.Errorf("%s message not found in output or error field missing", tt.name)
			}
		})
	}
}

// TestFluentLoggerAs tests the As method with numeric levels
func TestFluentLoggerAs(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("test error")

	// Test As with numeric level
	fluent := IfWith(logger, err).As(share.LevelWarn)
	if fluent.level != share.LevelWarn {
		t.Error("As() should set the specified level")
	}

	fluent.Msg("numeric level message")
	logger.Flush()

	if !strings.Contains(buf.String(), "numeric level message") ||
		!strings.Contains(buf.String(), "error=test error") {
		t.Error("Numeric level message not found or error field missing")
	}
}

// TestFluentLoggerFields tests field methods
func TestFluentLoggerFields(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("test error")

	// Test WithField
	fluent := IfWith(logger, err).AsInfo().WithField("key1", "value1")
	if fluent.fields["key1"] != "value1" {
		t.Error("WithField should add field")
	}

	// Test WithFields
	fluent = fluent.WithFields(share.Fields{
		"key2": "value2",
		"key3": "value3",
	})
	if fluent.fields["key2"] != "value2" || fluent.fields["key3"] != "value3" {
		t.Error("WithFields should add multiple fields")
	}

	// Test WithError (should overwrite existing error)
	newErr := errors.New("new error")
	fluent = fluent.WithError(newErr)
	if fluent.fields["error"] != newErr.Error() {
		t.Error("WithError should update error field")
	}

	fluent.Msg("fields test message")
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "fields test message") ||
		!strings.Contains(output, "key1=value1") ||
		!strings.Contains(output, "key2=value2") ||
		!strings.Contains(output, "error=new error") {
		t.Errorf("expected all fields and message in output, got %s", output)
	}
}

// TestFluentLoggerMsgIf tests conditional message
func TestFluentLoggerMsgIf(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("test error")
	fluent := IfWith(logger, err).AsInfo()

	// Test MsgIf with true condition
	fluent.MsgIf(true, "should log this")
	logger.Flush()

	if !strings.Contains(buf.String(), "should log this") ||
		!strings.Contains(buf.String(), "error=test error") {
		t.Error("MsgIf(true) should log message and error field")
	}

	buf.Reset()
	// Test MsgIf with false condition
	fluent.MsgIf(false, "should not log this")
	logger.Flush()

	if strings.Contains(buf.String(), "should not log this") {
		t.Error("MsgIf(false) should not log message")
	}
}

// TestFluentLoggerMsgf tests formatted message
func TestFluentLoggerMsgf(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("test error")
	fluent := IfWith(logger, err).AsInfo()

	fluent.Msgf("formatted %s with %d", "message", 42)
	logger.Flush()

	if !strings.Contains(buf.String(), "formatted message with 42") ||
		!strings.Contains(buf.String(), "error=test error") {
		t.Error("Msgf should format message correctly and include error field")
	}
}

// TestFluentLoggerSend tests Send method
func TestFluentLoggerSend(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("test error")
	fluent := IfWith(logger, err).AsInfo().WithField("test", "send")

	// Send without message should still log fields
	fluent.Send()
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "test=send") || !strings.Contains(output, "error=test error") {
		t.Error("Send should log fields and error field")
	}
}

// TestFluentLoggerChaining tests method chaining
func TestFluentLoggerChaining(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("chain error")

	// Test extensive chaining
	fluent := IfWith(logger, err).
		AsWarn().
		WithField("step", 1).
		WithFields(share.Fields{
			"process": "chaining",
			"test":    true,
		}).
		WithError(errors.New("updated error"))

	// Verify state
	if fluent.level != share.LevelWarn {
		t.Error("Chained level should be Warn")
	}
	if fluent.fields["step"] != 1 {
		t.Error("Chained field 'step' should be 1")
	}
	if fluent.fields["process"] != "chaining" {
		t.Error("Chained field 'process' should be 'chaining'")
	}

	fluent.Msg("chained message")
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "chained message") ||
		!strings.Contains(output, "step=1") ||
		!strings.Contains(output, "process=chaining") ||
		!strings.Contains(output, "error=updated error") {
		t.Errorf("expected all chained fields and message in output, got %s", output)
	}
}

// TestFluentLoggerNilFields tests handling of nil fields
func TestFluentLoggerNilFields(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("test error")

	// Test WithFields with nil
	fluent := IfWith(logger, err).AsInfo().WithFields(nil)
	// Should not crash

	// Test WithError with nil
	fluent = fluent.WithError(nil)
	// Should not crash

	fluent.Msg("nil fields test")
	logger.Flush()

	if !strings.Contains(buf.String(), "nil fields test") ||
		!strings.Contains(buf.String(), "error=test error") {
		t.Error("Should handle nil fields gracefully and include error field")
	}
}

// TestFluentLoggerFieldTypes tests various field value types
func TestFluentLoggerFieldTypes(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	logger := New(opts)

	err := errors.New("test error")

	fluent := IfWith(logger, err).AsInfo().
		WithField("string", "text").
		WithField("int", 42).
		WithField("float", 3.14).
		WithField("bool", true).
		WithField("nil", nil)

	fluent.Msg("field types test")
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "string=text") ||
		!strings.Contains(output, "int=42") ||
		!strings.Contains(output, "float=3.14") ||
		!strings.Contains(output, "bool=true") ||
		!strings.Contains(output, "error=test error") {
		t.Errorf("expected all field types and error in output, got %s", output)
	}
}

// TestFluentLoggerConcurrency tests concurrent usage
func TestFluentLoggerConcurrency(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Timestamp = false
	opts.ForceColor = false
	// opts.Async = true // Field doesn't exist
	logger := New(opts)

	err := errors.New("concurrent error")

	// Test concurrent fluent loggers
	done := make(chan bool, 10)
	for i := range 10 {
		go func(id int) {
			fluent := IfWith(logger, err).AsInfo().WithField("goroutine", id)
			fluent.Msgf("concurrent fluent message %d", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	time.Sleep(50 * time.Millisecond)
	logger.Flush()

	// Verify messages are present
	output := buf.String()
	for i := range 10 {
		expectedMsg := fmt.Sprintf("concurrent fluent message %d", i)
		expectedFields := fmt.Sprintf("goroutine=%d", i)
		if !strings.Contains(output, expectedMsg) || !strings.Contains(output, expectedFields) ||
			!strings.Contains(output, "error=concurrent error") {
			t.Errorf("Missing concurrent fluent message %d or fields in output: %s", i, output)
		}
	}
}
