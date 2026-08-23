package progress

import "context"

// Update is one progress event for a single batch item.
type Update struct {
	ID      string
	Label   string
	Status  string
	Current int64
	Total   int64
	Done    bool
	Err     error // fatal; fails the batch in strict mode
	Warn    error // non-fatal; shown on the row only
}

// Source produces Updates for one unit of work.
type Source interface {
	Run(ctx context.Context, updates chan<- Update) error
}
