package log

import (
	"os"

	"golang.org/x/exp/slog"
)

var (
	level = new(slog.LevelVar)
)

func init() {
	slog.SetDefault(NewTextLogger())
}

func SetLevel(l slog.Level) {
	level.Set(l)
}

func NewTextLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
