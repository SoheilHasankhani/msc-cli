package progress

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// git writes one phase per line and updates the same line with CR.
var clonePctRE = regexp.MustCompile(`(?i)(Receiving objects|Resolving deltas|Checking out files|Updating files):\s+(\d+)%`)

// ParseCloneProgress turns `git clone --progress` / `git pull --progress` stderr
// into Updates on a single 0–100 bar. Receiving is 0–80, resolving 80–95,
// checkout 95–100 so the bar never drops when git starts a new phase.
// Unrecognized lines are ignored; the caller emits Done.
func ParseCloneProgress(r io.Reader, id, label string, emit func(Update)) error {
	if emit == nil {
		emit = func(Update) {}
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	sc.Split(scanGitProgress)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		m := clonePctRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pct, _ := strconv.Atoi(m[2])
		cur := clonePhaseProgress(m[1], pct)
		emit(Update{ID: id, Label: label, Current: cur, Total: 100})
	}
	_ = sc.Err()
	return nil
}

func clonePhaseProgress(phase string, pct int) int64 {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "receiving objects":
		return int64(pct * 80 / 100)
	case "resolving deltas":
		return 80 + int64(pct*15/100)
	default:
		return 95 + int64(pct*5/100)
	}
}

// scanGitProgress splits on CR or LF so in-place git --progress updates are seen.
func scanGitProgress(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] != '\n' && data[i] != '\r' {
			continue
		}
		adv := i + 1
		if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			adv++
		}
		return adv, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// GitCloner runs a clone and writes --progress output to progress.
type GitCloner interface {
	Clone(ctx context.Context, repoURL, destPath string, progress io.Writer) error
}

// GitCloneSource clones one repo and emits progress from git stderr.
type GitCloneSource struct {
	ID     string
	Label  string
	URL    string
	Dest   string
	Cloner GitCloner
}

// Run implements Source.
func (s GitCloneSource) Run(ctx context.Context, updates chan<- Update) error {
	updates <- Update{ID: s.ID, Label: s.Label, Current: 0, Total: 100}
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Cloner.Clone(ctx, s.URL, s.Dest, pw)
		_ = pw.Close()
	}()
	_ = ParseCloneProgress(pr, s.ID, s.Label, func(u Update) {
		updates <- u
	})
	if err := <-errCh; err != nil {
		updates <- Update{ID: s.ID, Label: s.Label, Current: 100, Total: 100, Done: true, Err: err}
		return err
	}
	updates <- Update{ID: s.ID, Label: s.Label, Current: 100, Total: 100, Done: true}
	return nil
}
