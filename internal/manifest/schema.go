package manifest

// FileName is the canonical Project Manifest filename at a meta-repo root.
// Brand-specific names such as .isos-cli.manifest.yml are not used: the engine
// is brand-agnostic and the brand lives inside the file.
const FileName = "msc.manifest.yml"

// Default layout paths stay on the pre-migration isos tree so existing
// checkouts without an explicit layout block keep working.
const (
	DefaultComposeFile    = "local/docker-compose.yml"
	DefaultConfigDir      = "local/config"
	DefaultClonesDir      = "local"
	DefaultComposeProfile = "standard"
)

// Recommended layout after Phase 10 (compose/ + config/, clones stay in local/).
const (
	RecommendedComposeFile = "compose/docker-compose.yml"
	RecommendedConfigDir   = "config"
)

// Manifest is the committed, shared project description.
type Manifest struct {
	Brand         BrandInfo   `yaml:"brand"`
	GitHost       GitHostInfo `yaml:"git_host"`
	LocalDomain   string      `yaml:"local_domain"`
	Prerequisites []string    `yaml:"prerequisites"`
	Layout        Layout      `yaml:"layout"`
	Repos         []RepoDef   `yaml:"repos"`
}

// Layout locates stack files inside the meta-repo.
type Layout struct {
	ComposeFile    string `yaml:"compose_file"`
	ComposeProfile string `yaml:"compose_profile"`
	ConfigDir      string `yaml:"config_dir"`
	ClonesDir      string `yaml:"clones_dir"`
}

// ApplyDefaults fills empty layout fields with the current-isos paths.
func (m *Manifest) ApplyDefaults() {
	if m.Layout.ComposeFile == "" {
		m.Layout.ComposeFile = DefaultComposeFile
	}
	if m.Layout.ConfigDir == "" {
		m.Layout.ConfigDir = DefaultConfigDir
	}
	if m.Layout.ClonesDir == "" {
		m.Layout.ClonesDir = DefaultClonesDir
	}
	if m.Layout.ComposeProfile == "" {
		m.Layout.ComposeProfile = DefaultComposeProfile
	}
	if m.Prerequisites == nil {
		m.Prerequisites = []string{"docker", "git", "ssh"}
	}
}

// BrandInfo names the project and its suggested shim command.
type BrandInfo struct {
	DisplayName string `yaml:"display_name"`
	Command     string `yaml:"command"`
}

// GitHostInfo identifies the Git host for this project (GitLab, GitHub, etc.).
type GitHostInfo struct {
	BaseURL string `yaml:"base_url"`
}

// RepoDef is one clone under local/, which may contain zero or more services.
type RepoDef struct {
	Name     string       `yaml:"name"`
	Git      string       `yaml:"git"`
	Services []ServiceDef `yaml:"services"`
}

// ServiceDef maps a compose service to a path inside the repo and a Source Mode port.
type ServiceDef struct {
	ComposeService string `yaml:"compose_service"`
	Path           string `yaml:"path"`
	SourcePort     int    `yaml:"source_port"`
}

// ComposeProfiles returns compose --profile values. CLI overrides beat the manifest default.
func (m *Manifest) ComposeProfiles(override []string) []string {
	if len(override) > 0 {
		return override
	}
	if m == nil || m.Layout.ComposeProfile == "" {
		return nil
	}
	return []string{m.Layout.ComposeProfile}
}
