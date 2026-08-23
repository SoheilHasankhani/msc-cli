package doctor

// Status is one check outcome.
type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

// FixKind says what --fix may do.
type FixKind string

const (
	FixNone    FixKind = ""
	FixOverlay FixKind = "overlay"
	FixShim    FixKind = "shim"
	FixHosts   FixKind = "hosts"
	FixCerts   FixKind = "certs"
	FixOSTrust FixKind = "os-trust"
)

// Check is one row in the doctor report.
type Check struct {
	Name    string
	Status  Status
	Message string
	Next    string
	Fix     FixKind
}

// Report is the full doctor result.
type Report struct {
	Checks  []Check
	Fixed   []string
	Skipped []string
}

// HasFail reports whether any check failed.
func (r Report) HasFail() bool {
	for _, c := range r.Checks {
		if c.Status == StatusFail {
			return true
		}
	}
	return false
}
