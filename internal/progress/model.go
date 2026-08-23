package progress

// Bar is the testable per-item progress state (no terminal I/O).
type Bar struct {
	Label   string
	Status  string
	Percent float64
	Done    bool
	Err     error
	Warn    error
}

// Model holds one bar per Update.ID in first-seen order.
type Model struct {
	Order []string
	Bars  map[string]Bar
}

// Apply creates or updates a bar from an Update.
func (m *Model) Apply(u Update) {
	if m.Bars == nil {
		m.Bars = map[string]Bar{}
	}
	bar, ok := m.Bars[u.ID]
	if !ok {
		m.Order = append(m.Order, u.ID)
	}
	if u.Label != "" {
		bar.Label = u.Label
	}
	if u.Status != "" {
		bar.Status = u.Status
	}
	if bar.Label == "" {
		bar.Label = u.ID
	}
	if u.Total > 0 {
		bar.Percent = float64(u.Current) / float64(u.Total)
	}
	bar.Done = u.Done
	if u.Err != nil {
		bar.Err = u.Err
		bar.Warn = nil
	}
	if u.Warn != nil {
		bar.Warn = u.Warn
	}
	m.Bars[u.ID] = bar
}
