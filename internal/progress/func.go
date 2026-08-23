package progress

import "context"

// FuncSource is a one-shot Source built from a callback (e.g. git ls-remote).
type FuncSource struct {
	ID    string
	Label string
	Fn    func(ctx context.Context, emit func(Update)) error
}

// Run implements Source.
func (s FuncSource) Run(ctx context.Context, updates chan<- Update) error {
	if s.Fn == nil {
		return nil
	}
	emit := func(u Update) {
		if u.ID == "" {
			u.ID = s.ID
		}
		if u.Label == "" {
			u.Label = s.Label
		}
		updates <- u
	}
	return s.Fn(ctx, emit)
}
