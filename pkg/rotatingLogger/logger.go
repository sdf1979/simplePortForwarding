package rotatinglogger

import (
	"context"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type RotatingFileHandler struct {
	mu          sync.Mutex
	file        *os.File
	filePath    string
	baseDir     string
	currentHour int
}

func InitLogger(ctx context.Context, baseDir string) {
	baseDir, err := fullPath(baseDir)
	if err != nil {
		slog.Error("failed get full log directory", "error", err)
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		slog.Error("failed to create log directory", "error", err)
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				s := a.Value.Any().(*slog.Source)
				s.File = path.Base(s.File)
			}
			return a
		},
		Level: new(slog.LevelVar),
	}

	handler := NewRotatingFileHandler(baseDir)
	logger := slog.New(slog.NewJSONHandler(handler, opts))
	slog.SetDefault(logger)

	go startRotationAndSignalHandling(ctx, handler)
}

func NewRotatingFileHandler(baseDir string) *RotatingFileHandler {
	return &RotatingFileHandler{
		baseDir: baseDir,
	}
}

func (h *RotatingFileHandler) Write(p []byte) (n int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.rotateWithoutLock(); err != nil {
		return 0, err
	}
	return h.file.Write(p)
}

func (h *RotatingFileHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

func (h *RotatingFileHandler) rotate() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rotateWithoutLock()
}

func (h *RotatingFileHandler) rotateWithoutLock() error {
	now := time.Now()
	currentHour := now.Hour()

	if h.file != nil && currentHour == h.currentHour {
		return nil
	}

	newFilePath := filepath.Join(h.baseDir, now.Truncate(time.Hour).Format("2006-01-02_1504.log"))

	if h.file != nil {
		h.file.Close()
	}

	file, err := os.OpenFile(newFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	h.file = file
	h.filePath = newFilePath
	h.currentHour = currentHour

	go h.cleanupOldFiles()

	return nil
}

func (h *RotatingFileHandler) cleanupOldFiles() error {
	files, err := os.ReadDir(h.baseDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > 24*time.Hour {
			os.Remove(filepath.Join(h.baseDir, file.Name()))
		}
	}
	return nil
}

func fullPath(baseDir string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(exePath), baseDir), nil
}
