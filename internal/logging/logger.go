package logging

import (
	"log/slog"
	"os"

	"space-wars-3002-text-generation/internal/config"
)

type Logger struct {
	*slog.Logger
}

func New(cfg *config.Config) *Logger {
	opts := &slog.HandlerOptions{
		Level: cfg.LogLevel(),
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &Logger{slog.New(handler)}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{l.Logger.With(args...)}
}

func (l *Logger) Debugf(msg string, fields map[string]interface{}) {
	if l.Enabled(nil, slog.LevelDebug) {
		args := make([]interface{}, 0, len(fields)*2)
		for k, v := range fields {
			args = append(args, k, v)
		}
		l.Debug(msg, args...)
	}
}

func (l *Logger) Infof(msg string, fields map[string]interface{}) {
	args := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	l.LogAttrs(nil, slog.LevelInfo, msg, args...)
}

func (l *Logger) Warnf(msg string, fields map[string]interface{}) {
	args := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	l.LogAttrs(nil, slog.LevelWarn, msg, args...)
}

func (l *Logger) Errorf(msg string, fields map[string]interface{}) {
	args := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	l.LogAttrs(nil, slog.LevelError, msg, args...)
}
