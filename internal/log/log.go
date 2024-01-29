package log

import (
	"os"

	"log/slog"

	"github.com/charmbracelet/log"
)

var (
	level = new(slog.LevelVar)
)

func init() {
	slog.SetDefault(NewTextLogger().Logger)
}

func SetLevel(l slog.Level) {
	level.Set(l)
}

type Logger struct {
	*slog.Logger
}

func (l *Logger) With(v ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(v...),
	}
}

func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		Logger: l.Logger.WithGroup(name),
	}
}

func NewTextLogger() *Logger {
	var l log.Level
	switch level.Level() {
	case slog.LevelDebug:
		l = log.DebugLevel
	case slog.LevelInfo:
		l = log.InfoLevel
	case slog.LevelError:
		l = log.ErrorLevel
	case slog.LevelWarn:
		l = log.WarnLevel
	}

	return &Logger{
		Logger: slog.New(log.NewWithOptions(os.Stderr, log.Options{Level: l})),
	}
}
