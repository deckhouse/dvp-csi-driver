package logger

import (
	"log/slog"
	"os"
)

func New(options ...Option) *slog.Logger {
	var handlerOptions slog.HandlerOptions

	for _, option := range options {
		switch option.(type) {
		case *DebugOption:
			handlerOptions.Level = slog.LevelDebug
		default:
		}
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &handlerOptions))
}
