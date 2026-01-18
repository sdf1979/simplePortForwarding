package rotatinglogger

import (
	"context"
	"log/slog"
	"time"
)

func startRotationAndSignalHandling(ctx context.Context, h *RotatingFileHandler) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := h.rotate(); err != nil {
				slog.Error("Failed to rotate log file", "error", err)
			}
		case <-ctx.Done():
			slog.Info("file log close")
			time.Sleep(2 * time.Second)
			h.Close()
		}
	}
}
