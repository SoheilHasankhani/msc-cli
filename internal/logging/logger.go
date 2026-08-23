// Package logging writes local structured JSON-lines logs and support bundles.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

const defaultName = "msc.jsonl"

// Options configure the file (or test) logger.
type Options struct {
	Dir        string
	Name       string
	Level      slog.Level
	Writer     io.Writer // if set, skip the rotating file
	MaxSizeMB  int
	MaxAgeDays int
	MaxBackups int
}

// New returns a JSON slog logger. The closer flushes the underlying file.
func New(opt Options) (*slog.Logger, io.Closer, error) {
	if opt.Level == 0 && opt.Writer == nil && opt.Dir == "" {
		opt.Level = slog.LevelInfo
	}
	w := opt.Writer
	var closer io.Closer = nopCloser{}
	if w == nil {
		if opt.Dir == "" {
			return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: opt.Level})), closer, nil
		}
		if err := os.MkdirAll(opt.Dir, 0o755); err != nil {
			return nil, nil, err
		}
		name := opt.Name
		if name == "" {
			name = defaultName
		}
		maxSize := opt.MaxSizeMB
		if maxSize <= 0 {
			maxSize = 5
		}
		maxAge := opt.MaxAgeDays
		if maxAge <= 0 {
			maxAge = 14
		}
		maxBackups := opt.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 5
		}
		lj := &lumberjack.Logger{
			Filename:   filepath.Join(opt.Dir, name),
			MaxSize:    maxSize,
			MaxAge:     maxAge,
			MaxBackups: maxBackups,
			LocalTime:  false,
		}
		w = lj
		closer = lj
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: opt.Level})
	return slog.New(h), closer, nil
}

// ParseLevel maps MSC_LOG_LEVEL / --verbose text to a slog level.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
