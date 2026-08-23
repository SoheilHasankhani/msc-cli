package progress

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type pullEvent struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	Error          string `json:"error"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

// DefaultDockerPullRetries is how many times a transient image pull failure is retried.
const DefaultDockerPullRetries = 3

// ErrDockerPullTransient marks a pull failure that may succeed on retry.
var ErrDockerPullTransient = errors.New("docker pull transient")

// ParsePullStream turns a Docker ImagePull JSON-lines stream into Updates.
// Malformed lines are ignored. Layer byte counts are summed for one bar.
func ParsePullStream(r io.Reader, id, label string, emit func(Update)) error {
	if emit == nil {
		emit = func(Update) {}
	}
	layers := map[string][2]int64{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev pullEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Error != "" {
			err := fmt.Errorf("%s", ev.Error)
			if IsRetryableDockerPullMessage(ev.Error) {
				err = fmt.Errorf("%s: %w", FormatDockerPullError(err), ErrDockerPullTransient)
			}
			return err
		}
		if ev.ID != "" && ev.ProgressDetail.Total > 0 {
			layers[ev.ID] = [2]int64{ev.ProgressDetail.Current, ev.ProgressDetail.Total}
		}
		var current, total int64
		for _, pair := range layers {
			current += pair[0]
			total += pair[1]
		}
		emit(Update{ID: id, Label: label, Current: current, Total: total})
	}
	if err := sc.Err(); err != nil {
		return maybeTransientPullErr(err)
	}
	emit(Update{ID: id, Label: label, Current: 1, Total: 1, Done: true})
	return nil
}

// Puller opens a Docker image-pull JSON stream.
type Puller interface {
	Pull(ctx context.Context, ref string) (io.ReadCloser, error)
}

// DockerPullSource pulls one image and emits aggregated progress.
type DockerPullSource struct {
	Ref       string
	Puller    Puller
	Retries   int      // 0 means DefaultDockerPullRetries
	ID        string   // optional display id; defaults to Ref
	Label     string   // optional display label; defaults to ShortImageRef(Ref)
	FanOutIDs []string // when set, each update is emitted for every id (label = id)
}

// Run implements Source.
func (s DockerPullSource) Run(ctx context.Context, updates chan<- Update) error {
	ids := s.emitIDs()
	label := s.emitLabel(ids[0])
	attempts := s.Retries
	if attempts <= 0 {
		attempts = DefaultDockerPullRetries
	}
	emit := func(u Update) {
		for _, id := range ids {
			u2 := u
			u2.ID = id
			if len(s.FanOutIDs) > 0 {
				u2.Label = id
			} else if u2.Label == "" {
				u2.Label = label
			}
			select {
			case updates <- u2:
			case <-ctx.Done():
			}
		}
	}

	var lastErr error
	for try := 1; try <= attempts; try++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if try > 1 {
			emit(Update{Current: 0, Total: 0})
		}
		lastErr = s.pullOnce(ctx, emit, label)
		if lastErr == nil {
			return nil
		}
		if try == attempts || !IsRetryableDockerPullError(lastErr) {
			emit(Update{Done: true, Err: lastErr})
			return lastErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(try) * 500 * time.Millisecond):
		}
	}
	return lastErr
}

func (s DockerPullSource) emitIDs() []string {
	if len(s.FanOutIDs) > 0 {
		return s.FanOutIDs
	}
	return []string{s.emitID()}
}

func (s DockerPullSource) emitID() string {
	if s.ID != "" {
		return s.ID
	}
	return s.Ref
}

func (s DockerPullSource) emitLabel(id string) string {
	if len(s.FanOutIDs) > 0 {
		return id
	}
	if s.Label != "" {
		return s.Label
	}
	return ShortImageRef(s.Ref)
}

func (s DockerPullSource) pullOnce(ctx context.Context, emit func(Update), label string) error {
	rc, err := s.Puller.Pull(ctx, s.Ref)
	if err != nil {
		return maybeTransientPullErr(err)
	}
	defer rc.Close()
	id := s.emitID()
	return ParsePullStream(rc, id, label, emit)
}

// IsRetryableDockerPullError reports whether another pull attempt may succeed.
func IsRetryableDockerPullError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, ErrInterrupted) {
		return false
	}
	if errors.Is(err, ErrDockerPullTransient) {
		return true
	}
	return IsRetryableDockerPullMessage(err.Error())
}

func maybeTransientPullErr(err error) error {
	if err == nil || !IsRetryableDockerPullMessage(err.Error()) {
		return err
	}
	if errors.Is(err, ErrDockerPullTransient) {
		return err
	}
	return fmt.Errorf("%s: %w", FormatDockerPullError(err), ErrDockerPullTransient)
}

// IsRetryableDockerPullMessage reports transient registry/network pull failures.
func IsRetryableDockerPullMessage(msg string) bool {
	if msg == "" {
		return false
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "pull access denied") ||
		strings.Contains(lower, "authorization failed") ||
		strings.Contains(lower, "no basic auth credentials") ||
		strings.Contains(lower, "manifest unknown") ||
		strings.Contains(lower, "repository does not exist") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "unknown blob") ||
		strings.Contains(lower, "interrupted") {
		return false
	}
	return strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connection timed out") ||
		strings.Contains(lower, "operation timed out") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "temporary failure") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "unexpected eof") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "500 internal server error") ||
		strings.Contains(lower, "502 bad gateway") ||
		strings.Contains(lower, "503 service unavailable") ||
		strings.Contains(lower, "504 gateway timeout") ||
		strings.Contains(lower, "tls handshake timeout") ||
		strings.Contains(lower, "network is unreachable")
}
