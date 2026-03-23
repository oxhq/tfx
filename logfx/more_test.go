package logfx

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/internal/testutil"
	writerpkg "github.com/oxhq/tfx/writer"
)

type stubFormatter struct{}

func (stubFormatter) Format(entry *share.Entry) ([]byte, error) {
	return []byte(entry.Message), nil
}

func withStringContextValue(
	ctx context.Context,
	key string,
	value string,
) context.Context {
	//nolint:staticcheck // logfx intentionally extracts fields from string context keys.
	return context.WithValue(ctx, key, value)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	defer func() {
		os.Stdout = oldStdout
		_ = reader.Close()
	}()

	os.Stdout = writer
	fn()
	_ = writer.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return string(data)
}

func TestContextHelpersAndAccessors(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Level = share.LevelTrace
	opts.Format = share.FormatText
	opts.Timestamp = false
	opts.DisableColor = true
	logger := New(opts)

	ctx := withStringContextValue(context.Background(), "request_id", "req-1")
	ctx = withStringContextValue(ctx, "user_id", "user-1")
	ctx = withStringContextValue(ctx, "session_id", "sess-1")
	ctx = withStringContextValue(ctx, "trace_id", "trace-1")
	ctx = withStringContextValue(ctx, "correlation_id", "corr-1")

	ctxLogger := logger.WithContext(ctx).
		WithField("component", "api").
		WithFields(share.Fields{"region": "mx"}).
		WithError(errors.New("boom")).
		WithUser("shadowed-user").
		WithRequestID("shadowed-request").
		WithSession("shadowed-session").
		WithTraceID("shadowed-trace")

	fields := ctxLogger.GetFields()
	fields["component"] = "mutated"
	if ctxLogger.GetFields()["component"] != "api" {
		t.Fatal("expected GetFields to return a copy")
	}
	if ctxLogger.GetContext() != ctx {
		t.Fatal("expected GetContext to return the attached context")
	}

	ctxLogger.Trace("trace context")
	ctxLogger.Debug("debug context")
	ctxLogger.Info("info context")
	ctxLogger.Warn("warn context")
	ctxLogger.Error("error context")
	ctxLogger.Success("success context")
	ctxLogger.Badge("CTX", "badge context", color.Green)
	logger.Flush()

	output := buf.String()
	for _, want := range []string{
		"trace context",
		"debug context",
		"info context",
		"warn context",
		"error context",
		"success context",
		"badge context",
		"component=api",
		"region=mx",
		"error=boom",
		"request_id=req-1",
		"user_id=user-1",
		"session_id=sess-1",
		"trace_id=trace-1",
		"correlation_id=corr-1",
		"request_id=shadowed-request",
		"user_id=shadowed-user",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}

	resetGlobalLogger()
	globalBuf := &testutil.SafeBuffer{}
	globalOpts := DefaultOptions()
	globalOpts.Output = globalBuf
	globalOpts.Level = share.LevelInfo
	globalOpts.Format = share.FormatText
	globalOpts.Timestamp = false
	globalOpts.DisableColor = true
	Configure(globalOpts)

	FromContext(ctx).Info("from context")
	FromContextWithFields(ctx, share.Fields{"source": "fields"}).Info("from context with fields")
	Flush()

	globalOut := globalBuf.String()
	if !strings.Contains(globalOut, "from context") ||
		!strings.Contains(globalOut, "from context with fields") ||
		!strings.Contains(globalOut, "source=fields") {
		t.Fatalf("unexpected global context output: %q", globalOut)
	}
}

func TestContextConditionalFatalAndPanicPaths(t *testing.T) {
	buf := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Level = share.LevelDebug
	opts.Format = share.FormatText
	opts.Timestamp = false
	opts.DisableColor = true
	logger := New(opts)
	ctxLogger := logger.WithFields(share.Fields{"scope": "ctx"})

	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	exitCalls := 0
	osExit = func(int) {
		exitCalls++
	}

	ctxLogger.Fatal("fatal context")
	ctxLogger.FatalIf(errors.New("fatal-if"), "fatal if")
	if exitCalls != 2 {
		t.Fatalf("expected 2 osExit calls, got %d", exitCalls)
	}

	checks := []struct {
		name string
		fn   func(error, string, ...any) bool
	}{
		{name: "error-if", fn: ctxLogger.ErrorIf},
		{name: "warn-if", fn: ctxLogger.WarnIf},
		{name: "info-if", fn: ctxLogger.InfoIf},
		{name: "debug-if", fn: ctxLogger.DebugIf},
	}
	for _, check := range checks {
		if !check.fn(errors.New(check.name), check.name) {
			t.Fatalf("expected %s to report true", check.name)
		}
		if check.fn(nil, check.name) {
			t.Fatalf("expected %s to report false on nil", check.name)
		}
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("expected context panic to panic")
			}
		}()
		ctxLogger.Panic("panic context")
	}()

	logger.Flush()
	output := buf.String()
	for _, want := range []string{
		"fatal context",
		"fatal if",
		"error-if",
		"warn-if",
		"info-if",
		"debug-if",
		"panic context",
		"scope=ctx",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestGlobalWrappersOptionsPresetsAndBadges(t *testing.T) {
	resetGlobalLogger()

	buf := &testutil.SafeBuffer{}
	extra := &testutil.SafeBuffer{}
	opts := DefaultOptions()
	opts.Output = buf
	opts.Level = share.LevelTrace
	opts.Format = share.FormatText
	opts.Timestamp = false
	opts.DisableColor = true
	Configure(opts)

	AddWriter(writerpkg.NewConsoleWriter(extra, writerpkg.ConsoleOptions{
		Level:        share.LevelTrace,
		Format:       share.FormatText,
		Timestamp:    false,
		DisableColor: true,
	}))
	AddHook(func(entry *share.Entry) *share.Entry {
		entry.Message = "[hook] " + entry.Message
		return entry
	})
	SetLevel(share.LevelTrace)
	SetOutput(buf)
	SetFormat(share.FormatText)
	EnableTimestamp()
	DisableTimestamp()
	SetTheme(color.NordTheme)
	extraWriter := writerpkg.NewConsoleWriter(extra, writerpkg.ConsoleOptions{
		Level:        share.LevelTrace,
		Format:       share.FormatText,
		Timestamp:    false,
		DisableColor: true,
	})
	AddWriter(extraWriter)

	Trace("trace global")
	Debug("debug global")
	Info("info global")
	Warn("warn global")
	Error("error global")
	Success("success global")
	WithFields(share.Fields{"kind": "fields"}).Info("with fields global")
	WithContext(withStringContextValue(context.Background(), "request_id", "global-req")).
		Info("with context global")

	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	exitCalls := 0
	osExit = func(int) { exitCalls++ }
	Fatal("fatal global %d", 1)
	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("expected global panic to panic")
			}
		}()
		Panic("panic global %d", 2)
	}()

	if noop := If(nil); !noop.noOp {
		t.Fatal("expected If(nil) to return a no-op fluent logger")
	}
	if fatalFluent := If(errors.New("fluent fatal")); fatalFluent.logger == nil ||
		fatalFluent.AsFatal().level != share.LevelFatal {
		t.Fatal("expected If(err).AsFatal() to configure a fatal fluent logger")
	}

	badgeCfg := defaultBadgeConfig()
	WithModernBadgeStyle(BadgeStyleEmoji)(&badgeCfg)
	WithBadgeColor(color.Yellow)(&badgeCfg)
	WithBadgeBackground(color.Blue)(&badgeCfg)
	WithBadgeLevel(share.LevelWarn)(&badgeCfg)
	WithBold()(&badgeCfg)
	WithItalic()(&badgeCfg)
	WithUnderline()(&badgeCfg)
	if badgeCfg.style != BadgeStyleEmoji || badgeCfg.color != color.Yellow ||
		badgeCfg.bgColor != color.Blue || badgeCfg.level != share.LevelWarn ||
		!badgeCfg.bold || !badgeCfg.italic || !badgeCfg.underline {
		t.Fatalf("unexpected badge config after options: %#v", badgeCfg)
	}

	badgeStdout := captureStdout(t, func() {
		ModernBadge("MOD", "modern badge", WithBadgeBackground(color.Green))
		SuccessBadge("OK", "success badge")
		ErrorBadge("ERR", "error badge")
		WarnBadge("WARN", "warn badge")
		InfoBadge("INFO", "info badge")
		DebugBadge("DBG", "debug badge")
		DatabaseBadge("database ok", true)
		APIBadge("api fail", false)
		CacheBadge("cache warm", true)
		AuthBadge("auth ok", true)
		SystemBadge("system badge")
		ConfigBadge("config badge")
		SecurityBadge("security warn", share.LevelWarn)
		SecurityBadge("security error", share.LevelError)
		SecurityBadge("security info", share.LevelInfo)
	})
	Flush()

	if exitCalls != 1 {
		t.Fatalf("expected exactly one osExit call from global Fatal, got %d", exitCalls)
	}

	out := buf.String()
	for _, want := range []string{
		"[hook] trace global",
		"[hook] debug global",
		"[hook] info global",
		"[hook] warn global",
		"[hook] error global",
		"[hook] success global",
		"[hook] with fields global",
		"[hook] with context global",
		"[hook] fatal global 1",
		"[hook] panic global 2",
		"kind=fields",
		"request_id=global-req",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected global output to contain %q, got %q", want, out)
		}
	}
	for _, want := range []string{
		"modern badge",
		"success badge",
		"error badge",
		"warn badge",
		"info badge",
		"database ok",
		"api fail",
		"cache warm",
		"auth ok",
		"system badge",
		"config badge",
		"security warn",
		"security error",
		"security info",
	} {
		if !strings.Contains(badgeStdout, want) {
			t.Fatalf("expected badge stdout to contain %q, got %q", want, badgeStdout)
		}
	}
	if !strings.Contains(extra.String(), "info global") {
		t.Fatalf("expected extra writer to receive output, got %q", extra.String())
	}

	tempFile := filepath.Join(t.TempDir(), "app.log")
	formatter := stubFormatter{}
	logger := LogWith(
		WithOutput(buf),
		WithLevel(share.LevelWarn),
		WithFormat(share.FormatCustom),
		WithTimestamp(false),
		WithTimeFormat("15:04"),
		WithColorMode(color.ModeANSI),
		WithTheme(color.GitHubTheme),
		WithForceColor(true),
		WithDisableColor(true),
		WithBadgeWidth(12),
		WithLogBadgeStyle(string(share.BadgeStyleModern)),
		WithCaller(true),
		WithCallerDepth(7),
		WithFileOutput(tempFile),
		WithFileLevel(share.LevelError),
		WithFileRotation(4, 2, 7),
		WithCustomFormatter(formatter),
		WithAsync(8),
	)
	if logger.options.Output != buf || logger.options.Level != share.LevelWarn ||
		logger.options.Format != share.FormatCustom || logger.options.Timestamp ||
		logger.options.TimeFormat != "15:04" || logger.options.ColorMode != color.ModeANSI ||
		logger.options.Theme != color.GitHubTheme || !logger.options.ForceColor ||
		!logger.options.DisableColor || logger.options.BadgeWidth != 12 ||
		logger.options.BadgeStyle != share.BadgeStyleModern || !logger.options.ShowCaller ||
		logger.options.CallerDepth != 7 || logger.options.LogFile != tempFile ||
		logger.options.FileLevel != share.LevelError || logger.options.MaxFileSize != 4 ||
		logger.options.MaxBackups != 2 || logger.options.MaxAge != 7 ||
		logger.options.CustomFormatter != formatter || !logger.options.Async ||
		logger.options.AsyncBuffer != 8 {
		t.Fatalf("unexpected logger options: %#v", logger.options)
	}

	if cfgLogger := LogWithConfig(DefaultOptions()); cfgLogger == nil {
		t.Fatal("expected LogWithConfig to return a logger")
	}
	if express := Log(); express == nil {
		t.Fatal("expected Log() to return a logger")
	}

	if jsonLogger := LogWith(
		WithJSON(),
		WithOutput(buf),
	); jsonLogger.options.Format != share.FormatJSON {
		t.Fatal("expected WithJSON to force JSON format")
	}
	if badgeLogger := LogWith(
		WithBadges(),
		WithOutput(buf),
	); badgeLogger.options.Format != share.FormatBadge {
		t.Fatal("expected WithBadges to force badge format")
	}
	if textLogger := LogWith(
		WithText(),
		WithOutput(buf),
	); textLogger.options.Format != share.FormatText {
		t.Fatal("expected WithText to force text format")
	}
	if debugLogger := LogWith(
		WithDebugLevel(),
		WithOutput(buf),
	); debugLogger.options.Level != share.LevelDebug {
		t.Fatal("expected WithDebugLevel to force debug level")
	}
	if infoLogger := LogWith(
		WithInfoLevel(),
		WithOutput(buf),
	); infoLogger.options.Level != share.LevelInfo {
		t.Fatal("expected WithInfoLevel to force info level")
	}
	if warnLogger := LogWith(
		WithWarnLevel(),
		WithOutput(buf),
	); warnLogger.options.Level != share.LevelWarn {
		t.Fatal("expected WithWarnLevel to force warn level")
	}
	if errorLogger := LogWith(
		WithErrorLevel(),
		WithOutput(buf),
	); errorLogger.options.Level != share.LevelError {
		t.Fatal("expected WithErrorLevel to force error level")
	}

	dev := DevLogger()
	prod := ProdLogger()
	testLogger := TestLogger()
	consoleLogger := ConsoleLogger()
	fileLogger := FileLogger(filepath.Join(t.TempDir(), "file-only.log"))
	structured := StructuredLogger()
	if dev.options.Level != share.LevelDebug || !dev.options.ShowCaller || !dev.options.ForceColor {
		t.Fatalf("unexpected dev logger options: %#v", dev.options)
	}
	if prod.options.Level != share.LevelInfo ||
		prod.options.Format != share.FormatJSON ||
		!prod.options.DisableColor {
		t.Fatalf("unexpected prod logger options: %#v", prod.options)
	}
	if testLogger.options.Format != share.FormatText || testLogger.options.Timestamp {
		t.Fatalf("unexpected test logger options: %#v", testLogger.options)
	}
	if consoleLogger.options.LogFile != "" || !consoleLogger.options.ForceColor {
		t.Fatalf("unexpected console logger options: %#v", consoleLogger.options)
	}
	if fileLogger.options.LogFile == "" || fileLogger.options.Format != share.FormatJSON {
		t.Fatalf("unexpected file logger options: %#v", fileLogger.options)
	}
	if structured.options.Format != share.FormatJSON || !structured.options.DisableColor {
		t.Fatalf("unexpected structured logger options: %#v", structured.options)
	}

	if stdoutLogger := LogWith(WithStdout()); stdoutLogger.options.Output == nil {
		t.Fatal("expected WithStdout to set an output")
	}
	if stderrLogger := LogWith(WithStderr()); stderrLogger.options.Output == nil {
		t.Fatal("expected WithStderr to set an output")
	}
	if material := LogWith(WithMaterialTheme()); material.options.Theme != color.MaterialTheme {
		t.Fatal("expected material theme")
	}
	if dracula := LogWith(WithDraculaTheme()); dracula.options.Theme != color.DraculaTheme {
		t.Fatal("expected dracula theme")
	}
	if nord := LogWith(WithNordTheme()); nord.options.Theme != color.NordTheme {
		t.Fatal("expected nord theme")
	}
	if github := LogWith(WithGitHubTheme()); github.options.Theme != color.GitHubTheme {
		t.Fatal("expected github theme")
	}
}
