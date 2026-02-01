// Package logger provides a structured logger using Zap with Sentry integration.
package logger

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger configuration.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, console
	Output string // stdout, stderr, or file path
}

// SentryConfig holds Sentry configuration.
type SentryConfig struct {
	Enabled     bool
	DSN         string
	Environment string
	SampleRate  float64
}

// Logger wraps zap.Logger with Sentry integration.
type Logger struct {
	*zap.Logger
	sentryEnabled bool
}

// New creates a new Logger instance with optional Sentry integration.
func New(cfg Config, sentryCfg SentryConfig) (*Logger, error) {
	// Initialize Sentry if enabled
	if sentryCfg.Enabled && sentryCfg.DSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryCfg.DSN,
			Environment:      sentryCfg.Environment,
			SampleRate:       sentryCfg.SampleRate,
			AttachStacktrace: true,
		})
		if err != nil {
			return nil, err
		}
	}

	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create encoder based on format
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create output writer
	var output zapcore.WriteSyncer
	switch cfg.Output {
	case "stdout", "":
		output = zapcore.AddSync(os.Stdout)
	case "stderr":
		output = zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	// Create core with Sentry hook if enabled
	core := zapcore.NewCore(encoder, output, level)
	if sentryCfg.Enabled {
		core = zapcore.NewTee(core, newSentryCore(level))
	}

	// Build logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		Logger:        zapLogger,
		sentryEnabled: sentryCfg.Enabled,
	}, nil
}

// Sync flushes any buffered log entries and Sentry events.
func (l *Logger) Sync() error {
	if l.sentryEnabled {
		sentry.Flush(2 * time.Second)
	}

	return l.Logger.Sync()
}

// With creates a child logger with the given fields.
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		Logger:        l.Logger.With(fields...),
		sentryEnabled: l.sentryEnabled,
	}
}

// sentryCore implements zapcore.Core to send errors to Sentry.
type sentryCore struct {
	zapcore.LevelEnabler
	fields []zapcore.Field
}

func newSentryCore(level zapcore.Level) *sentryCore {
	return &sentryCore{
		LevelEnabler: level,
		fields:       make([]zapcore.Field, 0),
	}
}

func (c *sentryCore) With(fields []zapcore.Field) zapcore.Core {
	return &sentryCore{
		LevelEnabler: c.LevelEnabler,
		fields:       append(c.fields, fields...),
	}
}

func (c *sentryCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// Only send errors and above to Sentry
	if entry.Level >= zapcore.ErrorLevel {
		return checked.AddCore(entry, c)
	}

	return checked
}

func (c *sentryCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Build Sentry event
	event := sentry.NewEvent()
	event.Level = zapLevelToSentry(entry.Level)
	event.Message = entry.Message
	event.Logger = entry.LoggerName
	event.Timestamp = entry.Time

	// Add fields as extra data
	allFields := append(c.fields, fields...)
	event.Extra = fieldsToMap(allFields)

	// Capture event
	sentry.CaptureEvent(event)

	return nil
}

func (c *sentryCore) Sync() error {
	sentry.Flush(2 * time.Second)

	return nil
}

// zapLevelToSentry converts zap level to Sentry level.
func zapLevelToSentry(level zapcore.Level) sentry.Level {
	switch level {
	case zapcore.DebugLevel:
		return sentry.LevelDebug
	case zapcore.InfoLevel:
		return sentry.LevelInfo
	case zapcore.WarnLevel:
		return sentry.LevelWarning
	case zapcore.ErrorLevel:
		return sentry.LevelError
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return sentry.LevelFatal
	default:
		return sentry.LevelInfo
	}
}

// fieldsToMap converts zap fields to a map for Sentry extra data.
func fieldsToMap(fields []zapcore.Field) map[string]interface{} {
	m := make(map[string]interface{})
	for _, f := range fields {
		switch f.Type {
		case zapcore.StringType:
			m[f.Key] = f.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			m[f.Key] = f.Integer
		case zapcore.Float64Type, zapcore.Float32Type:
			m[f.Key] = float64(f.Integer)
		case zapcore.BoolType:
			m[f.Key] = f.Integer == 1
		case zapcore.DurationType:
			m[f.Key] = time.Duration(f.Integer).String()
		default:
			if f.Interface != nil {
				m[f.Key] = f.Interface
			}
		}
	}

	return m
}
