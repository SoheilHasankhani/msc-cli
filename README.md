# msc

Brand-agnostic CLI for local microservice development environments.

One binary (`msc`) drives Docker Compose stacks, nginx routing, and Git sync for a **meta-repo** (a checkout with `msc.manifest.yml` that describes services, repos, and layout). Each project also gets a **brand command** — a thin shim at `~/.msc/shims/<brand>` that sets `MSC_PROJECT` and execs `msc`. Use that command for day-to-day work, or `msc --project <name> …` when calling the engine directly.

Examples below use **`myproject`** as a stand-in registered name and brand command. Replace it with yours after `init`.

## Quick start

### 1. Install `msc`

Pick **one** command for your operating system. Each script downloads the latest [GitHub Release](https://github.com/SoheilHasankhani/msc-cli/releases/latest) for your OS/arch, verifies `checksums.txt`, and installs the binary.

**Linux or macOS** (bash, zsh, or similar):

```bash
curl -fsSL https://raw.githubusercontent.com/SoheilHasankhani/msc-cli/main/scripts/install.sh | sh
```

**Windows** (PowerShell — not Command Prompt, not WSL for this step):

```powershell
irm https://raw.githubusercontent.com/SoheilHasankhani/msc-cli/main/scripts/install.ps1 | iex
```

Ensure the install directory is on your `PATH`, then open a **new terminal** and verify:

```
msc --version
```

The install scripts configure PATH automatically (shell startup files on Linux/macOS; user `Path` on Windows). You still need a **new terminal** before `msc` resolves without a full path.

| OS | Default install location |
|----|--------------------------|
| Linux / macOS | `~/.local/bin/msc` |
| Windows | `%LOCALAPPDATA%\msc\msc.exe` |

Pin a version, use a fork, or change the install directory: [Install](#install). Build from source: [CONTRIBUTING.md](CONTRIBUTING.md).

### 2. Connect a meta-repo

`msc init` is **per machine**: it registers the checkout in your local Project Registry, writes a brand shim, links it onto your `PATH`, and runs `doctor --fix`. It does **not** start the stack (`up` is separate).

Pick the path that matches your situation.

#### A. Join an existing team (meta-repo already on Git)

The project already has `msc.manifest.yml` committed. Clone the meta-repo, `cd` into it, and register:

**Linux or macOS:**

```bash
git clone git@host:group/meta.git ~/src/my-meta-repo
cd ~/src/my-meta-repo
msc init
myproject sync
myproject up
```

**Windows** (PowerShell):

```powershell
git clone git@host:group/meta.git C:\src\my-meta-repo
cd C:\src\my-meta-repo
msc init
myproject sync
myproject up
```

`--repo` is not needed here — `init` reads the URL from `git remote origin`. Open a **new terminal** after `init` if it updated your shell startup files.

If the brand name is already registered on your machine for a **different** checkout, pass `--as other-name` or run `msc projects relink myproject --path <new-path>`.

#### B. New meta-repo (clone + first registration)

Use `--repo` when the checkout does not exist yet and you want `init` to clone it for you:

**Linux or macOS:**

```bash
msc init --repo git@host:group/meta.git --path ~/src/my-meta-repo
```

**Windows** (PowerShell):

```powershell
msc init --repo git@host:group/meta.git --path C:\src\my-meta-repo
```

If the directory exists but has no `msc.manifest.yml` yet, pass `--repo` so the CLI can draft a manifest (never auto-commits — commit it yourself when ready). Then `myproject sync` and `myproject up`.

### 3. Day-to-day use

```text
myproject status
myproject sync
myproject up
myproject down
```

Replace `myproject` with your brand command (`brand.command` in `msc.manifest.yml`). Engine equivalent: `msc --project myproject …`.

## Install

The installer detects OS/arch, downloads the matching GitHub Release archive, verifies `checksums.txt`, installs `msc`, and configures PATH for new terminals (shell startup files on Linux/macOS; user `Path` on Windows). Open a **new terminal** before running `msc`.

### Linux or macOS

Run in a terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/SoheilHasankhani/msc-cli/main/scripts/install.sh | sh
```

Writes `msc` to `~/.local/bin/` by default.

### Windows

Run in **PowerShell**:

```powershell
irm https://raw.githubusercontent.com/SoheilHasankhani/msc-cli/main/scripts/install.ps1 | iex
```

Writes `msc.exe` to `%LOCALAPPDATA%\msc\` by default.

Do **not** run the `curl … | sh` command on Windows — it is for Unix shells only.

### Build from source

To run a checkout of this repository instead of a release binary, see [CONTRIBUTING.md](CONTRIBUTING.md).

### Updating after install

```
msc self-update          # replace this binary with the latest release
msc self-update --check  # report only
msc self-update --force  # reinstall even if versions already match
```

### Optional install overrides

**You do not need to set any of these for a normal install.** They are advanced overrides, passed **only when you run the install script** (prefix the variable on the same line as the command).

| Variable | When to use it |
|----------|----------------|
| `MSC_REPO` | Install from a fork or mirror (`owner/name` on GitHub). Default: `SoheilHasankhani/msc-cli`. |
| `MSC_VERSION` | Pin a specific release (for example `1.0.0` or `v1.0.0`) instead of latest. |
| `MSC_INSTALL_DIR` | Install somewhere other than the default directory (see table in Quick start). |
| `MSC_GITHUB_TOKEN` | GitHub API returns rate-limit errors during install; a token raises the limit. |

Example (Linux/macOS — pin version and custom directory):

```bash
MSC_VERSION=1.0.0 MSC_INSTALL_DIR=/opt/bin \
  curl -fsSL https://raw.githubusercontent.com/SoheilHasankhani/msc-cli/main/scripts/install.sh | sh
```

Example (Windows PowerShell):

```powershell
$env:MSC_VERSION = "1.0.0"
irm https://raw.githubusercontent.com/SoheilHasankhani/msc-cli/main/scripts/install.ps1 | iex
```

`MSC_RELEASES_REPO` is unrelated to install — it overrides which GitHub repo `msc self-update` queries at runtime (default: same as `MSC_REPO` / `SoheilHasankhani/msc-cli`). Only set it if you maintain a separate releases fork.

## Runtime requirements

| Requirement | Used by |
|-------------|---------|
| **Docker** Engine or Desktop | `up`, `down`, `switch`, `compose` |
| **Git + SSH agent** | `sync`, `git` passthrough |
| **Tools from the Manifest** (for example `dotnet`) | checked by `doctor` |

Docker: the client honors `DOCKER_HOST`, then the active Docker CLI context (Docker Desktop on Linux works without pointing at `/var/run/docker.sock`), then the native Engine socket. Git: the CLI uses your SSH agent and never stores a Git-host token.

Recommended meta-repo layout is `compose/docker-compose.yml`, `config/`, and `local/` for clones only (`layout.*` in `msc.manifest.yml`). Empty layout fields still default to the older `local/docker-compose.yml` + `local/config` paths so existing checkouts keep working.

## Commands

Project-scoped commands are shown through the brand shim (`myproject …`). Equivalent engine form: `msc --project myproject …`.

### Stack lifecycle

```
myproject up                    # pull images, then docker compose up -d
myproject up --no-pull          # start without pulling
myproject up --pull-only        # pull only; do not start
myproject down                  # stop the stack
myproject status                # Docker vs Source mode per service
```

Compose profiles default from `layout.compose_profile` in `msc.manifest.yml`; override with `--profile`.

### Source Mode (`switch`)

`switch` routes a service between **Docker Mode** (container upstream) and **Source Mode** (process on the host, via nginx).

```
myproject switch api              # toggle
myproject switch api --to source  # run from IDE; container stopped
myproject switch api --to docker  # run in Docker again
```

Source Mode writes `layout.config_dir/nginx/generated/upstreams.conf` and reloads nginx with **SIGHUP**. Static nginx config under `components/` is never rewritten. Your IDE process must listen on **`0.0.0.0:<source_port>`**, not only `127.0.0.1`.

On native Linux Engine, `host.docker.internal` comes from a generated compose overlay (`docker-compose.msc.yml`) with `extra_hosts: host.docker.internal:host-gateway`.

### Sync

`sync` probes Manifest repos with `git ls-remote` over SSH (parallel, 8s timeout, access cached **7 days**; `--refresh` to re-check). Repos you cannot read are skipped. VPN down on port 22 fails fast instead of hanging.

```
myproject sync                  # clone missing + pull --ff-only (accessible repos)
myproject sync --list           # inspect cloned / available repos
myproject sync --refresh        # re-check access, then sync
myproject sync wallet-api       # one repo
myproject sync --clone-only     # clone only
myproject sync --pull-only      # pull cloned repos only
```

Per-repo fast-forward failures print a one-line warning and the rest continue.

### Init and projects

`msc init` registers a meta-repo **on your machine** (local registry + brand shim). Use `--repo` with the meta-repo SSH URL and `--path` with the checkout directory.

| Situation | What to do |
|-----------|------------|
| Join a team; already cloned the meta-repo | `cd` into the checkout, run `msc init` (reads `git remote origin`) |
| Join a team; not cloned yet | `git clone …`, `cd` into it, `msc init` |
| New path; want the CLI to clone | `msc init --repo git@host:group/meta.git --path ~/src/my-meta-repo` |
| Manifest draft needs review | Commit `msc.manifest.yml` yourself; `init` never commits |
| Brand name already taken locally | `--as other-name`, or `msc projects relink` / `remove` |
| Moved the checkout on disk | `msc projects relink myproject --path /new/path` |

```
msc init                                          # inside an existing clone
msc init --repo git@host:group/meta.git --path ~/src/my-meta-repo
msc init --repo git@host:group/meta.git --path ~/src/other-checkout --as other
msc projects list
msc projects relink myproject --path /new/path
msc projects remove myproject
```

`init` registers locally, links the brand command onto `PATH`, and runs `doctor --fix`. It does not run `up`. With an existing clone, `--repo` is optional (taken from `git remote origin`); pass it when cloning a new path or drafting a manifest without a remote.

### Doctor

```
myproject doctor
myproject doctor --fix
```

Reports Git, SSH agent, Docker, hosts file block, machine `local-ca` + project wildcard leaf, OS trust fingerprint, Manifest prerequisites, and host-gateway overlay. `--fix` writes overlays and certs, upserts the `# msc-begin` hosts block, and installs the CA when the store fingerprint does not match (may re-invoke under sudo / UAC). It does not install Docker, Git, or .NET.

### Compose and git passthrough

Arguments after `compose` or after `git … --` are forwarded unchanged. The CLI only picks the working directory and compose `-f` files (plus the host-gateway overlay when present).

```
myproject compose ps
myproject compose logs --tail 20 api
myproject git -- status -sb
myproject git identity-api -- log -1 --oneline
```

### Shell completion

```
msc completion bash|zsh|fish|powershell
msc completion install
```

Installed automatically on `init`, `scripts/install.sh`, and `self-update`. Bash/zsh/PowerShell completion also registers each name in the Project Registry so brand commands tab-complete like `msc`. Open a **new terminal** after install (or `source ~/.config/msc/completion.bash` / `. $PROFILE` on Windows). Tab completion lists names only — use `--help` for details.

## Logs and support bundle

Commands append JSON-lines to `~/.config/msc/logs/msc.jsonl` on Linux (rotated locally; nothing is sent off-machine).

| Variable | Effect |
|----------|--------|
| `--verbose` / `MSC_LOG_LEVEL=debug` | More detail in logs |
| `MSC_LOG_DIR` | Override log directory |

```
msc support-bundle              # ./msc-support-<timestamp>.zip
msc support-bundle -o /tmp      # write zip to that directory
```

The zip contains recent `*.jsonl*` files and `meta.json`. It does not include the Project Registry, Manifest, or certificate keys.

## Terminal UI

On a TTY, `msc` uses [Charm](https://charm.sh/) for tables, spinners, and prompts (`doctor --fix`, `switch`, `init`, `self-update`, `sync`, `up`).

| Variable | Effect |
|----------|--------|
| `NO_COLOR` | Plain text (no ANSI styling) |
| `MSC_NO_PROMPT` | Skip interactive forms; print CLI hints |

Piped or CI output always uses plain lines — no full-screen bubbletea UI.

## Contributing

Developing or patching this repository: [CONTRIBUTING.md](CONTRIBUTING.md) (Go, Make, symlinks, tests, CI).

## License

MIT. See [LICENSE](LICENSE).
