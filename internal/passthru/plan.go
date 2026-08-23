package passthru

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const overlayName = "docker-compose.msc.yml"

// Spec is the exact subprocess to run. No docker/git flags are reinterpreted.
type Spec struct {
	Name string
	Args []string
	Dir  string
}

// Compose builds `docker compose -f <manifest> [-f overlay] <user args>` at the meta-repo root.
// Injecting -f is path resolution from the Manifest, not a reimplementation of compose flags.
func Compose(root, composeFile string, userArgs []string) (Spec, error) {
	if strings.TrimSpace(root) == "" {
		return Spec{}, fmt.Errorf("project root is required")
	}
	if strings.TrimSpace(composeFile) == "" {
		return Spec{}, fmt.Errorf("layout.compose_file is required")
	}
	args := []string{"compose", "-f", composeFile}
	overlayRel := path.Join(filepath.Dir(composeFile), overlayName)
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(overlayRel))); err == nil {
		args = append(args, "-f", overlayRel)
	}
	args = append(args, userArgs...)
	return Spec{Name: "docker", Args: args, Dir: root}, nil
}

// ParseGit splits `git [<repo>] -- <args>` tokens. The `--` is required.
func ParseGit(tokens []string) (repo string, gitArgs []string, err error) {
	dash := -1
	for i, t := range tokens {
		if t == "--" {
			dash = i
			break
		}
	}
	if dash < 0 {
		return "", nil, fmt.Errorf(`missing "--" before git arguments

  git -- <git args>              meta-repository
  git <repo> -- <git args>       cloned service repo

Examples:
  git -- status -sb
  git identity-api -- log -1 --oneline

The "--" keeps git flags (e.g. -h, --help) out of msc. For git's own help in a clone:
  git doctor -- --help`)
	}
	left := tokens[:dash]
	gitArgs = tokens[dash+1:]
	switch len(left) {
	case 0:
		return "", gitArgs, nil
	case 1:
		return left[0], gitArgs, nil
	default:
		return "", nil, fmt.Errorf("usage: git [<repo>] -- <git args>")
	}
}

// Git resolves cwd to the meta-repo or a cloned Manifest repo.
func Git(root, clonesDir string, knownRepos []string, tokens []string) (Spec, error) {
	repo, gitArgs, err := ParseGit(tokens)
	if err != nil {
		return Spec{}, err
	}
	if repo == "" {
		return Spec{Name: "git", Args: gitArgs, Dir: root}, nil
	}
	ok := false
	for _, n := range knownRepos {
		if n == repo {
			ok = true
			break
		}
	}
	if !ok {
		return Spec{}, fmt.Errorf("unknown repo %q; use a Manifest repo name or omit it for the meta-repository", repo)
	}
	dest := filepath.Join(clonesDir, repo)
	if _, err := os.Stat(filepath.Join(dest, ".git")); err != nil {
		return Spec{}, fmt.Errorf("repo %q is not cloned — run: sync %s", repo, repo)
	}
	return Spec{Name: "git", Args: gitArgs, Dir: dest}, nil
}
