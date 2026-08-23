package hostcerts

import (
	"os"
)

// WriteFile reads path, upserts this project's marked block, and writes it back.
func WriteFile(path, project string, names []string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	out := UpsertBlock(string(data), project, names)
	mode := os.FileMode(0o644)
	if err == nil {
		if st, statErr := os.Stat(path); statErr == nil {
			mode = st.Mode().Perm()
		}
	}
	return os.WriteFile(path, []byte(out), mode)
}
