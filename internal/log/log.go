package log

import (
	"os"

	"golang.org/x/exp/slog"
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

func NewTextLogger() *Logger {
	return &Logger{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})),
	}
}
