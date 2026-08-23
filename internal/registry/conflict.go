package registry

import "fmt"

// RegisterKind is the outcome of Register.
type RegisterKind int

const (
	// RegisterNew means the name was unused.
	RegisterNew RegisterKind = iota
	// RegisterPathUpdated means the same project moved on disk.
	RegisterPathUpdated
	// RegisterBlocked means a different project already owns the name.
	RegisterBlocked
)

// RegisterResult describes what Register did (or refused to do).
type RegisterResult struct {
	Kind RegisterKind
}

// SameIdentity reports whether a and b are the same underlying project.
func SameIdentity(a, b ProjectEntry) bool {
	return a.GitRemote != "" && a.GitHostURL != "" &&
		a.GitRemote == b.GitRemote && a.GitHostURL == b.GitHostURL
}

// Register inserts or updates an entry. A name clash with a different project
// is blocked; the caller must retry with a different name (--as).
func (r *Registry) Register(name string, entry ProjectEntry) (RegisterResult, error) {
	if name == "" {
		return RegisterResult{}, fmt.Errorf("project name is required")
	}
	if r.Projects == nil {
		r.Projects = map[string]ProjectEntry{}
	}
	existing, ok := r.Projects[name]
	if !ok {
		r.Projects[name] = entry
		return RegisterResult{Kind: RegisterNew}, nil
	}
	if SameIdentity(existing, entry) {
		r.Projects[name] = entry
		return RegisterResult{Kind: RegisterPathUpdated}, nil
	}
	return RegisterResult{Kind: RegisterBlocked}, fmt.Errorf(
		"command name %q is already registered to a different project at %s; use --as <other-name>",
		name, existing.Path,
	)
}
