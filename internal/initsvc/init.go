package initsvc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
	"github.com/SoheilHasankhani/msc-cli/internal/shim"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

// Options drives init.
type Options struct {
	RepoURL      string
	Path         string
	As           string
	Git          gitops.Runner
	OriginURL    func(ctx context.Context, repoPath string) (string, error)
	RegistryFile string
	ShimDir      string
	PathDirs     paths.Resolver
	EnginePath   string
	GOOS         string
	After        func(name, root string) error
	Stdout       io.Writer
	SpinnerOut   io.Writer
	PromptAs     func(brand, existingPath string) (string, error)
	ConfirmDraft func(manifestPath string) (bool, error)
}

// Result is the outcome of a successful init.
type Result struct {
	Name          string
	Path          string
	Shim          string
	CommandPath   string
	ShellFiles    []string
	WroteManifest bool
	CommitHint    string
	RegisterKind  registry.RegisterKind
}

// Run clones (if needed), loads or drafts a Manifest, registers the project, and writes a shim.
func Run(ctx context.Context, opt Options) (Result, error) {
	if strings.TrimSpace(opt.Path) == "" {
		return Result{}, fmt.Errorf("--path is required (default is the current directory)")
	}
	if opt.Git == nil {
		opt.Git = gitops.Exec{}
	}
	if opt.GOOS == "" {
		opt.GOOS = runtime.GOOS
	}
	if opt.EnginePath == "" {
		opt.EnginePath = shim.EngineOnPATH()
	}
	pathDirs := opt.PathDirs
	if pathDirs.Home == "" {
		pathDirs = paths.Resolver{GOOS: opt.GOOS, Home: paths.Default().Home}
	}
	if pathDirs.GOOS == "" {
		pathDirs.GOOS = opt.GOOS
	}

	dest, err := filepath.Abs(opt.Path)
	if err != nil {
		return Result{}, err
	}

	repoURL, err := resolveRepoURL(ctx, opt, dest)
	if err != nil {
		return Result{}, err
	}
	opt.RepoURL = repoURL

	if _, err := ensureClone(ctx, opt, dest); err != nil {
		return Result{}, err
	}

	man, wrote, err := loadOrDraftManifest(dest, opt.RepoURL)
	if err != nil {
		return Result{}, err
	}
	if err := man.Validate(); err != nil {
		return Result{}, fmt.Errorf("manifest: %w", err)
	}

	if wrote && opt.ConfirmDraft != nil {
		ok, err := opt.ConfirmDraft(filepath.Join(dest, manifest.FileName))
		if err != nil {
			return Result{}, err
		}
		if !ok {
			return Result{}, fmt.Errorf("cancelled")
		}
	}

	name := man.Brand.Command
	if opt.As != "" {
		name = opt.As
	}

	remote, gitHost, err := gitops.ParseIdentity(opt.RepoURL)
	if err != nil {
		return Result{}, err
	}
	if origin, err := gitops.OriginURL(ctx, dest); err == nil && origin != "" {
		if r, g, err := gitops.ParseIdentity(origin); err == nil {
			remote, gitHost = r, g
		}
	}

	reg, err := registry.Load(opt.RegistryFile)
	if err != nil {
		return Result{}, err
	}
	entry := registry.ProjectEntry{
		Path:       dest,
		GitHostURL: gitHost,
		GitRemote:  remote,
	}
	var registerKind registry.RegisterKind
	for {
		kind, err := reg.Register(name, entry)
		if err == nil {
			if err := reg.Save(opt.RegistryFile); err != nil {
				return Result{}, err
			}
			registerKind = kind.Kind
			break
		}
		if kind.Kind == registry.RegisterBlocked && opt.PromptAs != nil {
			alt, perr := opt.PromptAs(name, reg.Projects[name].Path)
			if perr != nil {
				return Result{}, perr
			}
			name = alt
			continue
		}
		return Result{}, err
	}

	shimPath, err := shim.Write(opt.ShimDir, name, opt.EnginePath, opt.GOOS)
	if err != nil {
		return Result{}, err
	}

	installed, err := shim.InstallOnPATH(name, shimPath, pathDirs)
	if err != nil {
		return Result{}, err
	}

	res := Result{
		Name:          name,
		Path:          dest,
		Shim:          shimPath,
		CommandPath:   installed.CommandPath,
		ShellFiles:    installed.ShellFiles,
		WroteManifest: wrote,
		RegisterKind:  registerKind,
	}
	if wrote {
		res.CommitHint = manifest.CommitReminder(manifest.FileName)
	}

	if opt.After != nil {
		if err := opt.After(name, dest); err != nil {
			return res, err
		}
	}
	return res, nil
}

func resolveRepoURL(ctx context.Context, opt Options, dest string) (string, error) {
	if repo := strings.TrimSpace(opt.RepoURL); repo != "" {
		return repo, nil
	}
	originFn := opt.OriginURL
	if originFn == nil {
		originFn = gitops.OriginURL
	}
	if origin, err := originFn(ctx, dest); err == nil && origin != "" {
		return origin, nil
	}
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return "", fmt.Errorf("--repo is required when %q does not exist yet (SSH URL of the meta-repository)", dest)
	}
	return "", fmt.Errorf("could not determine repository URL from %q — clone the meta-repository first, or pass --repo", dest)
}

func ensureClone(ctx context.Context, opt Options, dest string) (bool, error) {
	if _, err := manifest.Find(dest); err == nil {
		return false, nil
	}
	info, err := os.Stat(dest)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return false, err
		}
		spinOut := opt.SpinnerOut
		if spinOut == nil {
			spinOut = opt.Stdout
		}
		clone := func(ctx context.Context) error {
			return opt.Git.Clone(ctx, opt.RepoURL, dest, opt.Stdout)
		}
		if spinOut != nil {
			if err := ui.WithSpinner(ctx, spinOut, "Cloning meta-repository", clone); err != nil {
				return false, err
			}
			return true, nil
		}
		if err := clone(ctx); err != nil {
			return false, err
		}
		return true, nil
	}
	if !info.IsDir() {
		return false, fmt.Errorf("path %s exists and is not a directory", dest)
	}
	// Existing checkout (empty or full) without a Manifest: register in place.
	// Never re-clone a developer's working tree. loadOrDraftManifest will draft.
	return false, nil
}

func loadOrDraftManifest(dest, repoURL string) (*manifest.Manifest, bool, error) {
	if path, err := manifest.Find(dest); err == nil {
		m, err := manifest.Load(path)
		return m, false, err
	}
	_, gitHost, err := gitops.ParseIdentity(repoURL)
	if err != nil {
		return nil, false, err
	}
	compose := readCompose(dest)
	dirs := listCloneDirs(dest)
	cmd := guessCommand(dest, repoURL)
	m := manifest.Suggest(manifest.SuggestInput{
		DisplayName:  cmd,
		Command:      cmd,
		GitHostBase:  gitHost,
		LocalDomain:  cmd + ".local",
		ComposeYAML:  compose,
		CloneDirs:    dirs,
		DefaultGroup: defaultGroup(repoURL),
	})
	if err := m.Save(filepath.Join(dest, manifest.FileName)); err != nil {
		return nil, false, err
	}
	return m, true, nil
}

func readCompose(dest string) string {
	candidates := []string{
		filepath.Join(dest, manifest.DefaultComposeFile),
		filepath.Join(dest, "docker-compose.yml"),
		filepath.Join(dest, "compose", "docker-compose.yml"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

func listCloneDirs(dest string) []string {
	root := filepath.Join(dest, manifest.DefaultClonesDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func guessCommand(dest, repoURL string) string {
	base := filepath.Base(strings.TrimSuffix(dest, string(filepath.Separator)))
	if base != "" && base != "." && base != "/" {
		return sanitizeCommand(base)
	}
	return sanitizeCommand(repoBase(repoURL))
}

func repoBase(repoURL string) string {
	s := strings.TrimSuffix(repoURL, ".git")
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func defaultGroup(repoURL string) string {
	s := strings.TrimSuffix(repoURL, ".git")
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		return s[i+1:]
	}
	return "group"
}

func sanitizeCommand(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for i, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if i == 0 && (r < 'a' || r > 'z') {
			b.WriteByte('p')
		}
		if ok {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "project"
	}
	return b.String()
}
