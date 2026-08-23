package paths

import "os"

func getenv(key string) string {
	return os.Getenv(key)
}

func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
