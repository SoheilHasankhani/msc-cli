package dockerapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type dockerConfigFile struct {
	Auths       map[string]dockerAuthEntry `json:"auths"`
	CredsStore  string                     `json:"credsStore"`
	CredHelpers map[string]string          `json:"credHelpers"`
}

type dockerAuthEntry struct {
	Auth string `json:"auth"`
}

type registryAuthConfig struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	ServerAddress string `json:"serveraddress"`
}

type credentialHelperResponse struct {
	Username string `json:"Username"`
	Secret   string `json:"Secret"`
}

// RegistryAuthHeader returns the X-Registry-Auth value for an image ref, or "" if none.
func RegistryAuthHeader(ref string) (string, error) {
	cfg, err := loadDockerConfig()
	if err != nil {
		return "", err
	}
	auth, ok, err := lookupAuth(cfg, ref)
	if err != nil || !ok {
		return "", err
	}
	raw, err := json.Marshal(auth)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

func loadDockerConfig() (*dockerConfigFile, error) {
	dir := os.Getenv("DOCKER_CONFIG")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".docker")
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return &dockerConfigFile{Auths: map[string]dockerAuthEntry{}}, nil
		}
		return nil, err
	}
	var cfg dockerConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse docker config: %w", err)
	}
	if cfg.Auths == nil {
		cfg.Auths = map[string]dockerAuthEntry{}
	}
	return &cfg, nil
}

func lookupAuth(cfg *dockerConfigFile, ref string) (registryAuthConfig, bool, error) {
	host := registryHost(ref)
	if helper := cfg.CredHelpers[host]; helper != "" {
		if auth, err := credentialHelperGet(helper, host); err == nil {
			return auth, true, nil
		}
	}
	if cfg.CredsStore != "" {
		if auth, err := credentialHelperGet(cfg.CredsStore, host); err == nil {
			return auth, true, nil
		}
	}
	for _, key := range registryAuthKeys(host) {
		if entry, ok := cfg.Auths[key]; ok && entry.Auth != "" {
			user, pass, err := decodeBasicAuth(entry.Auth)
			if err != nil {
				return registryAuthConfig{}, false, err
			}
			return registryAuthConfig{
				Username:      user,
				Password:      pass,
				ServerAddress: host,
			}, true, nil
		}
	}
	return registryAuthConfig{}, false, nil
}

func credentialHelperGet(name, host string) (registryAuthConfig, error) {
	bin := "docker-credential-" + name
	cmd := exec.Command(bin, "get")
	cmd.Stdin = strings.NewReader(host)
	out, err := cmd.Output()
	if err != nil {
		return registryAuthConfig{}, err
	}
	var resp credentialHelperResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return registryAuthConfig{}, err
	}
	if resp.Username == "" && resp.Secret == "" {
		return registryAuthConfig{}, fmt.Errorf("empty credentials from %s", bin)
	}
	return registryAuthConfig{
		Username:      resp.Username,
		Password:      resp.Secret,
		ServerAddress: host,
	}, nil
}

func registryHost(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if !strings.Contains(ref, "/") {
		return "https://index.docker.io/v1/"
	}
	head := ref[:strings.Index(ref, "/")]
	if strings.ContainsAny(head, ".:") || head == "localhost" {
		return head
	}
	return "https://index.docker.io/v1/"
}

func registryAuthKeys(host string) []string {
	keys := []string{host, "https://" + host + "/v1/", host + "/v1/"}
	if !strings.HasPrefix(host, "https://") {
		keys = append(keys, "https://"+host)
	}
	return keys
}

func decodeBasicAuth(encoded string) (user, pass string, err error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	if !ok {
		return "", "", fmt.Errorf("invalid auth entry")
	}
	return user, pass, nil
}
