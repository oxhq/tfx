package logfx

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	writerpkg "github.com/oxhq/tfx/writer"
)

// Global logger instance
var (
	globalLogger *Logger
	globalMu     sync.RWMutex
	osExit       = os.Exit // Expose os.Exit for testing
)

func init() {
	globalLogger = New(DefaultOptions())
}

// Logger represents a logger instance
type Logger struct {
	options   LogOptions
	writers   []share.Writer
	hooks     []Hook
	ctx       context.Context
	mu        sync.RWMutex // Mutex for protecting options and writers
	wg        sync.WaitGroup
	indent    int
	indentStr string
}

// LogOptions configures the logger
type LogOptions struct {
	Level           share.Level
	Output          io.Writer
	Format          share.Format
	Timestamp       bool
	TimeFormat      string
	Theme           color.ColorTheme
	BadgeWidth      int
	BadgeStyle      share.BadgeStyle // Changed to share.BadgeStyle
	ShowCaller      bool
	CallerDepth     int
	ForceColor      bool
	DisableColor    bool
	LogFile         string
	FileLevel       share.Level
	MaxFileSize     int64
	MaxBackups      int
	MaxAge          int
	Async           bool
	AsyncBuffer     int
	ColorMode       color.Mode
	CustomFormatter share.Formatter
}

// DefaultOptions returns default logger options
func DefaultOptions() LogOptions {
	return LogOptions{
		Level:       share.LevelInfo,
		Output:      os.Stdout,
		Format:      share.FormatBadge,
		Timestamp:   true,
		TimeFormat:  time.RFC3339,
		Theme:       color.DefaultTheme,
		BadgeWidth:  8,
		BadgeStyle:  share.BadgeStyleDefault, // Changed to share.BadgeStyle
		ShowCaller:  false,
		CallerDepth: 4,
		LogFile:     "",
		FileLevel:   share.LevelInfo,
		MaxFileSize: 100, // MB
		MaxBackups:  5,
		MaxAge:      30, // days
		Async:       false,
		AsyncBuffer: 1000,
		ColorMode:   color.ModeTrueColor,
	}
}

// Hook is a function that can modify a log entry
type Hook func(*share.Entry) *share.Entry

// New creates a new logger instance
func New(opts LogOptions) *Logger {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	logger := &Logger{
		options: opts,
		writers: []share.Writer{},
		hooks:   []Hook{},
		ctx:     context.Background(),
	}

	// Add default console writer
	cwOpts := writerpkg.ConsoleOptions{
		Level:        opts.Level,
		Format:       opts.Format,
		Timestamp:    opts.Timestamp,
		TimeFormat:   opts.TimeFormat,
		Theme:        opts.Theme,
		BadgeWidth:   opts.BadgeWidth,
		BadgeStyle:   opts.BadgeStyle,
		ShowCaller:   opts.ShowCaller,
		ForceColor:   opts.ForceColor,
		DisableColor: opts.DisableColor,
	}
	consoleWriter := writerpkg.NewConsoleWriter(opts.Output, cwOpts)
	logger.AddWriter(consoleWriter)

	// Add file writer if specified
	if opts.LogFile != "" {
		fwOpts := writerpkg.DefaultFileOptions()
		fwOpts.Level = opts.FileLevel
		fwOpts.Format = opts.Format
		fwOpts.MaxSize = opts.MaxFileSize
		fwOpts.MaxBackups = opts.MaxBackups
		fwOpts.MaxAge = opts.MaxAge

		fileWriter, err := writerpkg.NewFileWriter(opts.LogFile, fwOpts)
		if err == nil {
			logger.writers = append(logger.writers, fileWriter)
		}
	}

	if opts.Async {
		for i, w := range logger.writers {
			logger.writers[i] = writerpkg.NewAsyncWriter(w, opts.AsyncBuffer)
		}
	}

	return logger
}

// Configure updates the global logger options
func Configure(opts LogOptions) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = New(opts)
}

// GetLogger returns the global logger
func GetLogger() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// SetLevel sets the minimum logging level
func (l *Logger) SetLevel(level share.Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.options.Level = level
	l.updateConsoleWritersLocked()
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.options.Output = w
	l.updateConsoleWritersLocked()
}

func (l *Logger) Flush() {
	l.wg.Wait() // Wait for any direct (non-async) writes

	// Flush asynchronous writers
	var asyncWg sync.WaitGroup
	l.mu.RLock()
	for _, wr := range l.writers {
		if asyncWriter, ok := wr.(*writerpkg.AsyncWriter); ok {
			asyncWg.Add(1)
			go func(aw *writerpkg.AsyncWriter) {
				defer asyncWg.Done()
				aw.Flush()
			}(asyncWriter)
		}
	}
	l.mu.RUnlock()
	asyncWg.Wait()
}

// SetFormat sets the output format
func (l *Logger) SetFormat(format share.Format) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.options.Format = format
	l.updateConsoleWritersLocked()
}

// EnableTimestamp enables timestamp in logs
func (l *Logger) EnableTimestamp() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.options.Timestamp = true
	l.updateConsoleWritersLocked()
}

// DisableTimestamp disables timestamp in logs
func (l *Logger) DisableTimestamp() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.options.Timestamp = false
	l.updateConsoleWritersLocked()
}

// SetTheme sets the color theme
func (l *Logger) SetTheme(theme color.ColorTheme) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.options.Theme = theme
	l.updateConsoleWritersLocked()
}

func (l *Logger) consoleOptions() writerpkg.ConsoleOptions {
	return writerpkg.ConsoleOptions{
		Level:        l.options.Level,
		Format:       l.options.Format,
		Timestamp:    l.options.Timestamp,
		TimeFormat:   l.options.TimeFormat,
		Theme:        l.options.Theme,
		BadgeWidth:   l.options.BadgeWidth,
		BadgeStyle:   l.options.BadgeStyle,
		ShowCaller:   l.options.ShowCaller,
		ForceColor:   l.options.ForceColor,
		DisableColor: l.options.DisableColor,
	}
}

func (l *Logger) updateConsoleWritersLocked() {
	cwOpts := l.consoleOptions()
	for _, wr := range l.writers {
		updateConsoleWriter(wr, func(cw *writerpkg.ConsoleWriter) {
			cw.UpdateOptions(l.options.Output, cwOpts)
		})
	}
}

func updateConsoleWriter(w share.Writer, fn func(*writerpkg.ConsoleWriter)) {
	switch writer := w.(type) {
	case *writerpkg.ConsoleWriter:
		fn(writer)
	case *writerpkg.AsyncWriter:
		if cw, ok := writer.UnderlyingWriter().(*writerpkg.ConsoleWriter); ok {
			fn(cw)
		}
	}
}

// AddWriter adds a new writer
func (l *Logger) AddWriter(writer share.Writer) {
	if writer == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writers = append(l.writers, writer)
}

// AddHook adds a new hook
func (l *Logger) AddHook(hook Hook) {
	if hook == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hook)
}

// WithFields creates a new context with fields
func (l *Logger) WithFields(fields share.Fields) *Context {
	return &Context{
		logger: l,
		fields: cloneShareFields(fields),
		ctx:    l.ctx,
	}
}

// WithContext creates a new context with the given context
func (l *Logger) WithContext(ctx context.Context) *Context {
	return &Context{
		logger: l,
		fields: make(map[string]any),
		ctx:    ctx,
	}
}

// shouldLog checks if the level should be logged
func (l *Logger) shouldLog(level share.Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.options.Level
}

// createEntry creates a log entry
func (l *Logger) createEntry(level share.Level, msg string, fields share.Fields) *share.Entry {
	entry := &share.Entry{
		Level:     level,
		Message:   msg,
		Fields:    fields,
		Timestamp: time.Now(),
		Context:   l.ctx,
		IndentStr: l.indentStr,
	}

	// Add caller info if enabled
	if l.options.ShowCaller {
		entry.Caller = l.getCaller()
	}

	// Apply hooks
	l.mu.RLock()
	hooks := l.hooks
	l.mu.RUnlock()

	for _, hook := range hooks {
		if entry != nil {
			if newEntry := hook(entry); newEntry != nil {
				entry = newEntry
			}
		}
	}

	return entry
}

// getCaller gets caller information
func (l *Logger) getCaller() *share.CallerInfo {
	pc, file, line, ok := runtime.Caller(l.options.CallerDepth)
	if !ok {
		return nil
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return nil
	}

	return &share.CallerInfo{
		File:     file,
		Function: fn.Name(),
		Line:     line,
	}
}

// log writes a log entry
func (l *Logger) log(level share.Level, msg string, fields share.Fields) {
	if !l.shouldLog(level) {
		return
	}

	entry := l.createEntry(level, msg, fields)
	l.dispatch(entry)
}

// Logging methods
func (l *Logger) Trace(msg string) {
	l.log(share.LevelTrace, msg, nil)
}

func (l *Logger) Debug(msg string) {
	l.log(share.LevelDebug, msg, nil)
}

func (l *Logger) Info(msg string) {
	l.log(share.LevelInfo, msg, nil)
}

func (l *Logger) Warn(msg string) {
	l.log(share.LevelWarn, msg, nil)
}

func (l *Logger) Error(msg string) {
	l.log(share.LevelError, msg, nil)
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.log(share.LevelFatal, fmt.Sprintf(msg, args...), nil)
	osExit(1)
}

func (l *Logger) Panic(msg string, args ...any) {
	msg = fmt.Sprintf(msg, args...)
	l.log(share.LevelPanic, msg, nil)
	panic(msg)
}

// Success is a convenience method for successful operations
func (l *Logger) Success(msg string, args ...any) {
	l.log(share.LevelSuccess, fmt.Sprintf(msg, args...), nil)
}

// FatalIf logs a fatal message if err is not nil and exits
func (l *Logger) FatalIf(err error, msg string, args ...any) {
	if err != nil {
		formattedMsg := fmt.Sprintf(msg, args...)
		l.log(
			share.LevelFatal,
			fmt.Sprintf("%s: %v", formattedMsg, err),
			share.Fields{"error": err.Error()},
		)
		osExit(1)
	}
}

// ErrorIf logs an error message if err is not nil and returns true if error occurred
func (l *Logger) ErrorIf(err error, msg string, args ...any) bool {
	if err != nil {
		formattedMsg := fmt.Sprintf(msg, args...)
		l.log(
			share.LevelError,
			fmt.Sprintf("%s: %v", formattedMsg, err),
			share.Fields{"error": err.Error()},
		)
		return true
	}
	return false
}

// WarnIf logs a warning message if err is not nil and returns true if error occurred
func (l *Logger) WarnIf(err error, msg string, args ...any) bool {
	if err != nil {
		formattedMsg := fmt.Sprintf(msg, args...)
		l.log(
			share.LevelWarn,
			fmt.Sprintf("%s: %v", formattedMsg, err),
			share.Fields{"error": err.Error()},
		)
		return true
	}
	return false
}

// InfoIf logs an info message if err is not nil and returns true if error occurred
func (l *Logger) InfoIf(err error, msg string, args ...any) bool {
	if err != nil {
		formattedMsg := fmt.Sprintf(msg, args...)
		l.log(
			share.LevelInfo,
			fmt.Sprintf("%s: %v", formattedMsg, err),
			share.Fields{"error": err.Error()},
		)
		return true
	}
	return false
}

// DebugIf logs a debug message if err is not nil and returns true if error occurred
func (l *Logger) DebugIf(err error, msg string, args ...any) bool {
	if err != nil {
		formattedMsg := fmt.Sprintf(msg, args...)
		l.log(
			share.LevelDebug,
			fmt.Sprintf("%s: %v", formattedMsg, err),
			share.Fields{"error": err.Error()},
		)
		return true
	}
	return false
}

// Badge creates a custom badge log
func (l *Logger) Badge(tag, msg string, color color.Color, args ...any) {
	l.log(share.LevelInfo, fmt.Sprintf(msg, args...), share.Fields{
		"badge":       tag,
		"badge_color": color,
	})
}

// Group creates a contextual log group.
func (l *Logger) Group(title string, fn func()) {
	l.Info(title)
	l.mu.Lock()
	l.indent++
	l.indentStr = strings.Repeat("  ", l.indent)
	l.mu.Unlock()

	fn()

	l.mu.Lock()
	l.indent--
	l.indentStr = strings.Repeat("  ", l.indent)
	l.mu.Unlock()
}

// Close closes all writers
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, writer := range l.writers {
		if err := writer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Global functions that use the global logger
func SetLevel(level share.Level)               { GetLogger().SetLevel(level) }
func SetOutput(w io.Writer)                    { GetLogger().SetOutput(w) }
func SetFormat(format share.Format)            { GetLogger().SetFormat(format) }
func EnableTimestamp()                         { GetLogger().EnableTimestamp() }
func DisableTimestamp()                        { GetLogger().DisableTimestamp() }
func SetTheme(theme color.ColorTheme)          { GetLogger().SetTheme(theme) }
func AddWriter(writer share.Writer)            { GetLogger().AddWriter(writer) }
func AddHook(hook Hook)                        { GetLogger().AddHook(hook) }
func WithFields(fields share.Fields) *Context  { return GetLogger().WithFields(fields) }
func WithContext(ctx context.Context) *Context { return GetLogger().WithContext(ctx) }

func Trace(msg string)                { GetLogger().Trace(msg) }
func Debug(msg string)                { GetLogger().Debug(msg) }
func Info(msg string)                 { GetLogger().Info(msg) }
func Warn(msg string)                 { GetLogger().Warn(msg) }
func Error(msg string)                { GetLogger().Error(msg) }
func Fatal(msg string, args ...any)   { GetLogger().Fatal(msg, args...) }
func Panic(msg string, args ...any)   { GetLogger().Panic(msg, args...) }
func Success(msg string, args ...any) { GetLogger().Success(msg, args...) }

// Global If variants
func FatalIf(err error, msg string, args ...any) { GetLogger().FatalIf(err, msg, args...) }

func ErrorIf(
	err error,
	msg string,
	args ...any,
) bool {
	return GetLogger().ErrorIf(err, msg, args...)
}

func WarnIf(
	err error,
	msg string,
	args ...any,
) bool {
	return GetLogger().WarnIf(err, msg, args...)
}

func InfoIf(
	err error,
	msg string,
	args ...any,
) bool {
	return GetLogger().InfoIf(err, msg, args...)
}

func DebugIf(
	err error,
	msg string,
	args ...any,
) bool {
	return GetLogger().DebugIf(err, msg, args...)
}

func Flush() { GetLogger().Flush() }

// --- MULTIPATH API: Three Entry Points ---

// 1. EXPRESS: Quick defaults
func Log() *Logger {
	return newLogger(DefaultOptions())
}

// 2. FLUENT: Functional options + DSL chaining support
func LogWith(opts ...LogOption) *Logger {
	cfg := DefaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}
	return newLogger(cfg)
}

// 3. INSTANTIATED: Config struct
func LogWithConfig(cfg LogOptions) *Logger {
	return newLogger(cfg)
}

// newLogger is the internal implementation
func newLogger(cfg LogOptions) *Logger {
	return New(cfg)
}

// LogOption is a functional option for logger configuration
type LogOption func(*LogOptions)

// --- FUNCTIONAL OPTIONS ---

// WithOutput sets the output writer
func WithOutput(w io.Writer) LogOption {
	return func(cfg *LogOptions) {
		cfg.Output = w
	}
}

// WithLevel sets the minimum logging level
func WithLevel(level share.Level) LogOption {
	return func(cfg *LogOptions) {
		cfg.Level = level
	}
}

// WithFormat sets the output format
func WithFormat(format share.Format) LogOption {
	return func(cfg *LogOptions) {
		cfg.Format = format
	}
}

// WithTimestamp enables/disables timestamps
func WithTimestamp(enabled bool) LogOption {
	return func(cfg *LogOptions) {
		cfg.Timestamp = enabled
	}
}

// WithTimeFormat sets the timestamp format
func WithTimeFormat(format string) LogOption {
	return func(cfg *LogOptions) {
		cfg.TimeFormat = format
	}
}

// WithColorMode sets the color mode
func WithColorMode(mode color.Mode) LogOption {
	return func(cfg *LogOptions) {
		cfg.ColorMode = mode
	}
}

// WithTheme sets the color theme
func WithTheme(theme color.ColorTheme) LogOption {
	return func(cfg *LogOptions) {
		cfg.Theme = theme
	}
}

// WithForceColor forces color output
func WithForceColor(force bool) LogOption {
	return func(cfg *LogOptions) {
		cfg.ForceColor = force
	}
}

// WithDisableColor disables all color output
func WithDisableColor(disable bool) LogOption {
	return func(cfg *LogOptions) {
		cfg.DisableColor = disable
	}
}

// WithBadgeWidth sets the badge width
func WithBadgeWidth(width int) LogOption {
	return func(cfg *LogOptions) {
		cfg.BadgeWidth = width
	}
}

// WithLogBadgeStyle sets the badge style
func WithLogBadgeStyle(style string) LogOption {
	return func(cfg *LogOptions) {
		cfg.BadgeStyle = share.BadgeStyle(style)
	}
}

// WithCaller enables/disables caller information
func WithCaller(enabled bool) LogOption {
	return func(cfg *LogOptions) {
		cfg.ShowCaller = enabled
	}
}

// WithCallerDepth sets the caller depth
func WithCallerDepth(depth int) LogOption {
	return func(cfg *LogOptions) {
		cfg.CallerDepth = depth
	}
}

// WithFileOutput enables file output
func WithFileOutput(filename string) LogOption {
	return func(cfg *LogOptions) {
		cfg.LogFile = filename
	}
}

// WithFileLevel sets the file logging level
func WithFileLevel(level share.Level) LogOption {
	return func(cfg *LogOptions) {
		cfg.FileLevel = level
	}
}

// WithFileRotation configures file rotation
func WithFileRotation(maxSize int64, maxBackups, maxAge int) LogOption {
	return func(cfg *LogOptions) {
		cfg.MaxFileSize = maxSize
		cfg.MaxBackups = maxBackups
		cfg.MaxAge = maxAge
	}
}

// WithCustomFormatter sets a custom formatter
func WithCustomFormatter(formatter share.Formatter) LogOption {
	return func(cfg *LogOptions) {
		cfg.CustomFormatter = formatter
	}
}

// WithAsync enables asynchronous logging
func WithAsync(bufferSize int) LogOption {
	return func(cfg *LogOptions) {
		cfg.Async = true
		cfg.AsyncBuffer = bufferSize
	}
}

// --- CONVENIENCE OPTIONS ---

// WithJSON enables JSON format
func WithJSON() LogOption {
	return WithFormat(share.FormatJSON)
}

// WithBadges enables badge format (default)
func WithBadges() LogOption {
	return WithFormat(share.FormatBadge)
}

// WithText enables plain text format
func WithText() LogOption {
	return WithFormat(share.FormatText)
}

// WithDebugLevel sets level to Debug
func WithDebugLevel() LogOption {
	return WithLevel(share.LevelDebug)
}

// WithInfoLevel sets level to Info
func WithInfoLevel() LogOption {
	return WithLevel(share.LevelInfo)
}

// WithWarnLevel sets level to Warn
func WithWarnLevel() LogOption {
	return WithLevel(share.LevelWarn)
}

// WithErrorLevel sets level to Error
func WithErrorLevel() LogOption {
	return WithLevel(share.LevelError)
}

// WithDevelopment configures logger for development
func WithDevelopment() LogOption {
	return func(cfg *LogOptions) {
		cfg.Level = share.LevelDebug
		cfg.Format = share.FormatBadge
		cfg.Timestamp = true
		cfg.ShowCaller = true
		cfg.CallerDepth = 4
		cfg.ForceColor = true
	}
}

// WithProduction configures logger for production
func WithProduction() LogOption {
	return func(cfg *LogOptions) {
		cfg.Level = share.LevelInfo
		cfg.Format = share.FormatJSON
		cfg.Timestamp = true
		cfg.ShowCaller = false
		cfg.DisableColor = true
	}
}

// WithConsoleOnly ensures only console output
func WithConsoleOnly() LogOption {
	return func(cfg *LogOptions) {
		cfg.LogFile = ""
	}
}

// WithStdout sets output to stdout
func WithStdout() LogOption {
	return WithOutput(os.Stdout)
}

// WithStderr sets output to stderr
func WithStderr() LogOption {
	return WithOutput(os.Stderr)
}

// --- THEME OPTIONS ---

// WithMaterialTheme uses Material Design colors
func WithMaterialTheme() LogOption {
	return WithTheme(color.MaterialTheme)
}

// WithDraculaTheme uses Dracula colors
func WithDraculaTheme() LogOption {
	return WithTheme(color.DraculaTheme)
}

// WithNordTheme uses Nord colors
func WithNordTheme() LogOption {
	return WithTheme(color.NordTheme)
}

// WithGitHubTheme uses GitHub colors
func WithGitHubTheme() LogOption {
	return WithTheme(color.GitHubTheme)
}

// --- PRESET CONSTRUCTORS ---

// DevLogger creates a development-focused logger
func DevLogger() *Logger {
	return LogWith(WithDevelopment())
}

// ProdLogger creates a production-focused logger
func ProdLogger() *Logger {
	return LogWith(WithProduction())
}

// TestLogger creates a logger suitable for testing
func TestLogger() *Logger {
	return LogWith(
		WithLevel(share.LevelDebug),
		WithFormat(share.FormatText),
		WithTimestamp(false),
		WithDisableColor(true),
	)
}

// ConsoleLogger creates a console-only logger with colors
func ConsoleLogger() *Logger {
	return LogWith(
		WithLevel(share.LevelInfo),
		WithFormat(share.FormatBadge),
		WithTimestamp(false),
		WithForceColor(true),
		WithConsoleOnly(),
	)
}

// FileLogger creates a file-only logger
func FileLogger(filename string) *Logger {
	return LogWith(
		WithLevel(share.LevelInfo),
		WithFormat(share.FormatJSON),
		WithTimestamp(true),
		WithFileOutput(filename),
		WithDisableColor(true),
	)
}

// StructuredLogger creates a JSON logger for structured logging
func StructuredLogger() *Logger {
	return LogWith(
		WithJSON(),
		WithTimestamp(true),
		WithLevel(share.LevelInfo),
		WithDisableColor(true),
	)
}
