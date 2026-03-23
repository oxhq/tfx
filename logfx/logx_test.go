package logfx

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/internal/testutil"
	writerpkg "github.com/oxhq/tfx/writer"
)

// Helper to reset global logger for tests
func resetGlobalLogger() {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = New(DefaultOptions())
}

func TestMain(m *testing.M) {
	// Ensure global logger is reset before and after tests
	resetGlobalLogger()
	code := m.Run()
	resetGlobalLogger()
	os.Exit(code)
}

func TestNew(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	logger := New(opts)

	if logger == nil {
		t.Fatal("New() returned nil")
	}

	// Check if console writer is added
	if len(logger.writers) != 1 {
		t.Errorf("Expected 1 writer, got %d", len(logger.writers))
	}

	// Check if output is set correctly
	logger.Info("test message")
	logger.Flush()
	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("Expected \"test message\" in output, got %q", buf.String())
	}
}

func TestNewWithFile(t *testing.T) {
	// Create a temporary file for logging
	file, err := os.CreateTemp("", "testlog*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()

	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.LogFile = file.Name()
	opts.Level = share.LevelDebug // Set the logger level to Debug
	opts.FileLevel = share.LevelDebug
	logger := New(opts)

	if logger == nil {
		t.Fatal("New() returned nil")
	}

	// Check if console and file writers are added
	if len(logger.writers) != 2 {
		t.Errorf("Expected 2 writers, got %d", len(logger.writers))
	}

	logger.Debug("file test message")
	logger.Flush()

	// Give the file writer a moment to write to the file
	time.Sleep(100 * time.Millisecond) // Increased sleep duration

	// Read from file to verify content
	fileContent, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(fileContent), "file test message") {
		t.Errorf("Expected \"file test message\" in file output, got %q", string(fileContent))
	}
}

func TestNewWithAsync(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Async = true
	opts.AsyncBuffer = 10
	logger := New(opts)

	if logger == nil {
		t.Fatal("New() returned nil")
	}

	// Check if writer is AsyncWriter
	if _, ok := logger.writers[0].(*writerpkg.AsyncWriter); !ok {
		t.Error("Expected writer to be AsyncWriter")
	}

	logger.Info("async test message")
	logger.Flush()
	time.Sleep(10 * time.Millisecond) // Give async writer time to process

	if !strings.Contains(buf.String(), "async test message") {
		t.Errorf("Expected \"async test message\" in output, got %q", buf.String())
	}
}

func TestAsyncWrappedConsoleReconfiguration(t *testing.T) {
	buf1 := &testutil.SafeBuffer{}
	buf2 := &testutil.SafeBuffer{}

	opts := DefaultOptions()
	opts.Output = buf1
	opts.Async = true
	opts.ForceColor = true
	logger := New(opts)

	logger.SetFormat(share.FormatJSON)
	logger.Info("first")
	logger.Flush()

	if !strings.Contains(buf1.String(), `"msg":"first"`) {
		t.Fatalf("Expected first message in initial buffer, got %q", buf1.String())
	}

	buf1.Reset()
	logger.SetOutput(buf2)
	logger.SetFormat(share.FormatText)
	logger.DisableTimestamp()
	logger.Info("second")
	logger.Flush()

	if buf1.String() != "" {
		t.Fatalf("Expected no new writes to original buffer, got %q", buf1.String())
	}
	if !strings.Contains(buf2.String(), "INFO") || !strings.Contains(buf2.String(), "second") {
		t.Fatalf("Expected second message in new buffer, got %q", buf2.String())
	}

	buf2.Reset()
	logger.SetOutput(buf2)
	logger.SetFormat(share.FormatBadge)
	logger.DisableTimestamp()

	logger.SetTheme(color.DefaultTheme)
	logger.Info("theme")
	logger.Flush()
	firstThemeOutput := buf2.String()

	buf2.Reset()
	logger.SetTheme(color.DraculaTheme)
	logger.Info("theme")
	logger.Flush()
	secondThemeOutput := buf2.String()

	if firstThemeOutput == secondThemeOutput {
		t.Fatalf("Expected theme change to affect wrapped console output, got %q", firstThemeOutput)
	}
}

func TestConfigureAndGetLogger(t *testing.T) {
	resetGlobalLogger()

	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Level = share.LevelDebug
	Configure(opts)

	logger := GetLogger()
	if logger == nil {
		t.Fatal("GetLogger() returned nil")
	}

	logger.Debug("configured message")
	logger.Flush()

	if !strings.Contains(buf.String(), "configured message") {
		t.Errorf("Expected \"configured message\" in output, got %q", buf.String())
	}

	if logger.options.Level != share.LevelDebug {
		t.Errorf("Expected level to be Debug, got %v", logger.options.Level)
	}
}

func TestSetLevel(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)

	logger.SetLevel(share.LevelWarn)

	logger.Info("info message") // Should not be logged
	logger.Warn("warn message") // Should be logged
	logger.Flush()

	if strings.Contains(buf.String(), "info message") {
		t.Error("Info message should not be logged")
	}
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("Warn message should be logged")
	}
}

func TestSetOutput(t *testing.T) {
	buf1 := &testutil.SafeBuffer{}
	buf2 := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf1)

	logger.Info("message to buf1")
	logger.Flush()

	if !strings.Contains(buf1.String(), "message to buf1") {
		t.Error("Expected message in buf1")
	}

	logger.SetOutput(buf2)
	buf1.Reset() // Clear buf1 to ensure no new writes go there

	logger.Info("message to buf2")
	logger.Flush()

	if strings.Contains(buf1.String(), "message to buf2") {
		t.Error("Message should not be in buf1")
	}
	if !strings.Contains(buf2.String(), "message to buf2") {
		t.Error("Expected message in buf2")
	}
}

func TestSetFormat(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)

	logger.SetFormat(share.FormatJSON)
	logger.Info("test")
	logger.Flush()

	if !strings.Contains(buf.String(), `"level":"INFO","msg":"test","time":"`) {
		t.Errorf("Expected JSON format, got %q", buf.String())
	}
}

func TestEnableDisableTimestamp(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)

	// Test EnableTimestamp
	logger.DisableTimestamp() // Ensure it's off first
	buf.Reset()
	logger.EnableTimestamp()
	logger.Info("timestamp test")
	logger.Flush()

	// Check the actual timestamp format used in badge format (RFC3339)
	currentTime := time.Now().Format(time.RFC3339)[:16] // Just the date and hour:minute part
	if !strings.Contains(buf.String(), "timestamp test") ||
		!strings.Contains(buf.String(), currentTime) {
		t.Errorf("Expected timestamp in output, got %q", buf.String())
	}
}

func TestSetTheme(t *testing.T) {
	logger := New(DefaultOptions())
	logger.SetTheme(color.DraculaTheme)
	if logger.options.Theme != color.DraculaTheme {
		t.Errorf("Expected theme to be DraculaTheme, got %v", logger.options.Theme)
	}
}

func TestThemeOptionsUseConcreteThemes(t *testing.T) {
	tests := []struct {
		name     string
		opt      LogOption
		expected color.ColorTheme
	}{
		{name: "nord", opt: WithNordTheme(), expected: color.NordTheme},
		{name: "github", opt: WithGitHubTheme(), expected: color.GitHubTheme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := LogWith(tt.opt)
			if logger.options.Theme != tt.expected {
				t.Fatalf("Expected theme %q, got %q", tt.expected.Name, logger.options.Theme.Name)
			}
		})
	}
}

func TestWithDevelopmentUsesUserCallerSite(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := LogWith(WithDevelopment(), WithDisableColor(true), WithOutput(buf))

	logger.Info("caller depth test")
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "logx_test.go:") {
		t.Fatalf("Expected user callsite in output, got %q", output)
	}
	if strings.Contains(output, "logx.go:") {
		t.Fatalf("Expected internal caller to be hidden, got %q", output)
	}
}

func TestAddWriter(t *testing.T) {
	buf1 := &testutil.SafeBuffer{}
	buf2 := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf1) // Default writer

	logger.AddWriter(writerpkg.NewConsoleWriter(buf2, writerpkg.ConsoleOptions{}))

	logger.Info("message to multiple writers")
	logger.Flush()

	if !strings.Contains(buf1.String(), "message to multiple writers") {
		t.Error("Expected message in buf1")
	}
	if !strings.Contains(buf2.String(), "message to multiple writers") {
		t.Error("Expected message in buf2")
	}

	// Test adding nil writer
	initialWriterCount := len(logger.writers)
	logger.AddWriter(nil)
	if len(logger.writers) != initialWriterCount {
		t.Errorf(
			"Expected writer count to remain %d, got %d",
			initialWriterCount,
			len(logger.writers),
		)
	}
}

func TestAddHook(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)

	hookCalled := false
	hook := func(entry *share.Entry) *share.Entry {
		hookCalled = true
		entry.Message = "hooked message"
		return entry
	}
	logger.AddHook(hook)

	logger.Info("original message")
	logger.Flush()

	if !hookCalled {
		t.Error("Hook was not called")
	}
	if !strings.Contains(buf.String(), "hooked message") {
		t.Errorf("Expected \"hooked message\" in output, got %q", buf.String())
	}

	// Test adding nil hook
	initialHookCount := len(logger.hooks)
	logger.AddHook(nil)
	if len(logger.hooks) != initialHookCount {
		t.Errorf("Expected hook count to remain %d, got %d", initialHookCount, len(logger.hooks))
	}
}

func TestWithFields(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)
	logger.options.DisableColor = true

	ctxLogger := logger.WithFields(share.Fields{"key": "value", "another": 123})
	ctxLogger.Info("message with fields")
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "message with fields") {
		t.Errorf("Expected message in output, got %q", output)
	}
	if !strings.Contains(output, "key=value") || !strings.Contains(output, "another=123") {
		t.Errorf("Expected fields in output, got %q", output)
	}
}

func TestWithContext(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)
	logger.SetFormat(share.FormatText)

	type testKeyType string
	ctx := context.WithValue(context.Background(), testKeyType("testKey"), "testValue")
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("message with context")
	logger.Flush()

	// Verifying context fields in output might require custom formatter or inspecting entry
	// For now, just ensure it doesn't panic and logs something
	if !strings.Contains(buf.String(), "message with context") {
		t.Errorf("Expected message in output, got %q", buf.String())
	}
}

func TestShouldLog(t *testing.T) {
	logger := New(DefaultOptions())
	logger.SetLevel(share.LevelInfo)

	if !logger.shouldLog(share.LevelInfo) {
		t.Error("Expected to log Info level")
	}
	if logger.shouldLog(share.LevelDebug) {
		t.Error("Did not expect to log Debug level")
	}
}

func TestCreateEntry(t *testing.T) {
	logger := New(DefaultOptions())
	logger.options.ShowCaller = true
	logger.options.CallerDepth = 1 // Adjust caller depth for this test

	entry := logger.createEntry(share.LevelInfo, "test message", share.Fields{"key": "value"})

	if entry.Level != share.LevelInfo {
		t.Errorf("Expected level Info, got %v", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected message \"test message\", got %q", entry.Message)
	}
	if entry.Fields["key"] != "value" {
		t.Errorf("Expected field key=value, got %v", entry.Fields["key"])
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if entry.Caller == nil || !strings.Contains(entry.Caller.File, "logx.go") {
		t.Errorf("Expected caller info, got %v", entry.Caller)
	}

	hookCalled := false
	logger.AddHook(func(e *share.Entry) *share.Entry {
		hookCalled = true
		e.Message = "modified message"
		return e
	})
	entry = logger.createEntry(share.LevelInfo, "original", nil)
	if !hookCalled {
		t.Error("Hook was not called")
	}
	if entry.Message != "modified message" {
		t.Errorf("Expected modified message, got %q", entry.Message)
	}
}

func TestGetCaller(t *testing.T) {
	logger := New(DefaultOptions())
	logger.options.CallerDepth = 1 // Adjust caller depth for this test

	caller := logger.getCaller()

	if caller == nil {
		t.Fatal("getCaller() returned nil")
	}
	if !strings.Contains(caller.File, "logx_test.go") {
		t.Errorf("Expected file to contain logx_test.go, got %q", caller.File)
	}
	if !strings.Contains(caller.Function, "TestGetCaller") {
		t.Errorf("Expected function to contain TestGetCaller, got %q", caller.Function)
	}
	if caller.Line == 0 {
		t.Error("Expected line number to be non-zero")
	}
}

func TestLogMethods(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)
	logger.SetLevel(share.LevelTrace)  // Enable all levels
	logger.SetFormat(share.FormatText) // Use text format for easier testing

	tests := []struct {
		name    string
		logFunc func(msg string)
		level   share.Level
	}{
		{"Trace", logger.Trace, share.LevelTrace},
		{"Debug", logger.Debug, share.LevelDebug},
		{"Info", logger.Info, share.LevelInfo},
		{"Warn", logger.Warn, share.LevelWarn},
		{"Error", logger.Error, share.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			t.Logf("Running test for %s", tt.name)
			logger.DisableTimestamp() // Disable timestamp for consistent output
			tt.logFunc(tt.name + " message")
			logger.Flush()

			output := buf.String()
			if !strings.Contains(output, tt.name+" message") {
				t.Errorf("Expected %q in output, got %q", tt.name+" message", output)
			}
			if !strings.Contains(output, tt.level.String()) {
				t.Errorf("Expected level %q in output, got %q", tt.level.String(), output)
			}
		})
	}

	// Test Success
	buf.Reset()
	logger.DisableTimestamp() // Disable timestamp for consistent output
	logger.Success("success message")
	logger.Flush()
	if !strings.Contains(buf.String(), "success message") ||
		!strings.Contains(buf.String(), share.LevelSuccess.String()) {
		t.Errorf("Expected success message, got %q", buf.String())
	}
}

func TestFatalAndPanic(t *testing.T) {
	// Test Fatal (requires mocking os.Exit)
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()

	var exitCalled bool
	osExit = func(code int) {
		exitCalled = true
	}

	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)

	logger.Fatal("fatal message")

	if !exitCalled {
		t.Error("Fatal did not call os.Exit")
	}

	logger.Flush()
	output := buf.String()
	if !strings.Contains(output, "fatal message") {
		t.Errorf("Expected \"fatal message\" in output, got %q", output)
	}
	if !strings.Contains(output, "Fatal") {
		t.Errorf("Expected \"Fatal\" in output, got %q", output)
	}

	// Test Panic (requires recovering from panic)
	buf.Reset()
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Panic did not cause panic")
			}
		}()
		logger.Panic("panic message")
	}()
	logger.Flush()
	output = buf.String()
	if !strings.Contains(output, "panic message") {
		t.Errorf("Expected \"panic message\" in output, got %q", output)
	}
	if !strings.Contains(output, "Panic") {
		t.Errorf("Expected \"Panic\" in output, got %q", output)
	}
}

func TestConditionalLogMethods(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)
	logger.SetLevel(share.LevelDebug) // Enable all levels
	logger.SetLevel(share.LevelDebug) // Enable all levels

	tests := []struct {
		name    string
		logFunc func(err error, msg string, args ...any) bool
		level   share.Level
	}{
		{"ErrorIf", logger.ErrorIf, share.LevelError},
		{"WarnIf", logger.WarnIf, share.LevelWarn},
		{"InfoIf", logger.InfoIf, share.LevelInfo},
		{"DebugIf", logger.DebugIf, share.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name+" with error", func(t *testing.T) {
			buf.Reset()
			err := errors.New("test error")
			result := tt.logFunc(err, tt.name+" message")
			logger.Flush()

			if !result {
				t.Errorf("Expected true, got false")
			}
			output := buf.String()
			// Check for message content and error
			if !strings.Contains(output, tt.name+" message") ||
				!strings.Contains(output, "test error") {
				t.Errorf("Expected %q message with error, got %q", tt.name, output)
			}
		})

		t.Run(tt.name+" without error", func(t *testing.T) {
			buf.Reset()
			result := tt.logFunc(nil, tt.name+" message")
			logger.Flush()

			if result {
				t.Errorf("Expected false, got true")
			}
			output := buf.String()
			if strings.Contains(output, tt.name+" message") {
				t.Errorf("Did not expect %q message, got %q", tt.name, output)
			}
		})
	}
}

func TestBadgeMethods(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)
	logger.SetFormat(share.FormatBadge)

	// Test Badge
	buf.Reset()
	logger.Badge("TEST", "badge message", color.Red)
	logger.Flush()
	if !strings.Contains(buf.String(), "TEST") || !strings.Contains(buf.String(), "badge message") {
		t.Errorf("Expected badge message, got %q", buf.String())
	}

	// Test BadgeWithOptions (uses global logger, need to configure it)
	resetGlobalLogger() // Reset to clean state
	opts := DefaultOptions()
	opts.Output = buf
	Configure(opts) // Configure global logger to use our test buffer

	buf.Reset()
	BadgeWithOptions("OPT", "options message", BadgeOptions{Foreground: color.Blue})
	GetLogger().Flush()
	if !strings.Contains(buf.String(), "OPT") ||
		!strings.Contains(buf.String(), "options message") {
		t.Errorf("Expected options badge message, got %q", buf.String())
	}
}

func TestGroup(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	logger := New(DefaultOptions())
	logger.SetOutput(buf)
	logger.SetFormat(share.FormatText) // Use text format for easier indentation check
	logger.DisableTimestamp()          // Disable timestamp for consistent output

	logger.Group("Main Group", func() {
		logger.Info("First item")
		logger.Group("Sub Group", func() {
			logger.Info("Sub item")
		})
		logger.Info("Second item")
	})
	logger.Flush()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Expected output (ignoring levels for simplicity)
	// Main Group
	//   First item
	//   Sub Group
	//     Sub item
	//   Second item

	if len(lines) < 5 {
		t.Fatalf("Expected at least 5 lines, got %d: %q", len(lines), output)
	}

	// Check content more flexibly since format might include level names
	if !strings.Contains(lines[0], "Main Group") {
		t.Errorf("Expected 'Main Group', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "First item") {
		t.Errorf("Expected 'First item', got %q", lines[1])
	}
	if !strings.Contains(lines[2], "Sub Group") {
		t.Errorf("Expected 'Sub Group', got %q", lines[2])
	}
	if !strings.Contains(lines[3], "Sub item") {
		t.Errorf("Expected 'Sub item', got %q", lines[3])
	}
	if !strings.Contains(lines[4], "Second item") {
		t.Errorf("Expected 'Second item', got %q", lines[4])
	}
}

func TestClose(t *testing.T) {
	mockWriter := &MockCloserWriter{}
	logger := New(DefaultOptions())
	logger.AddWriter(mockWriter)

	err := logger.Close()
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
	if !mockWriter.closed {
		t.Error("Writer's Close method was not called")
	}

	// Test Close with error from writer
	mockWriterWithError := &MockCloserWriter{returnError: true}
	logger = New(DefaultOptions())
	logger.AddWriter(mockWriterWithError)

	err = logger.Close()
	if err == nil {
		t.Error("Close() did not return an error when writer returned error")
	}
}

// MockCloserWriter is a mock implementation of share.Writer that also implements io.Closer
type MockCloserWriter struct {
	closed      bool
	returnError bool
}

func (m *MockCloserWriter) Write(entry *share.Entry) error {
	// Do nothing for this test
	return nil
}

func (m *MockCloserWriter) Close() error {
	m.closed = true
	if m.returnError {
		return errors.New("mock writer close error")
	}
	return nil
}
