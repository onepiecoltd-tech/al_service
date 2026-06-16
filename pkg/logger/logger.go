package logger

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

func Init(env string) {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:     slog.LevelDebug,
			AddSource: false,
		})
	}
	slog.SetDefault(slog.New(handler))
}
