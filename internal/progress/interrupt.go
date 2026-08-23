package progress

import "errors"

// ErrInterrupted is returned when the user aborts a progress batch (e.g. Ctrl+C).
var ErrInterrupted = errors.New("interrupted")
