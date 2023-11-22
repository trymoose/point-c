package wglog

import (
	"fmt"
	"log/slog"
)

// Slog allows the created of a [device.Logger] backed by a given [slog.Logger].
// No args are passed to the slog logger. The message is set to the formatted values.
// Verbose is logged on Debug and errors on Error
func Slog(logger *slog.Logger) *Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &Logger{
		Verbosef: func(format string, args ...any) { logger.Debug(fmt.Sprintf(format, args...)) },
		Errorf:   func(format string, args ...any) { logger.Error(fmt.Sprintf(format, args...)) },
	}
}
