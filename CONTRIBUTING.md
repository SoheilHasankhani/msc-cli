# Contributing to msc-cli

How to set up a development environment, build from source, and smoke-test changes against a real meta-repo checkout.

Before opening a PR, run the unit suite and exercise project-scoped commands (`up`, `status`, `switch`, …) against a meta-repo you can access — the same flow end users follow through their brand shim.

Examples use **`myproject`** as the registered project name and brand command. Substitute the name from your `init` / registry.

## Development environment

Install and verify the tools below **before** cloning. The Makefile is the supported build entrypoint — **`make` must work** on your machine.

### Required

| Tool | Version | Verify |
|------|---------|--------|
| [Go](https://go.dev/dl/) | **1.26+** (see `go.mod`) | `go version` |
| [Git](https://git-scm.com/) | recent | `git --version` |
| [Make](https://www.gnu.org/software/make/) | GNU Make or BSD make | `make --version` |

Quick check (all must succeed):

```bash
go version    # go1.26 or later
git --version
make --version
make -C ~/src/msc-cli help
```

### Required for stack integration tests

Not needed for `make test`, but required to run `up`, `switch`, etc. against a live stack:

| Tool | Verify |
|------|--------|
| [Docker](https://docs.docker.com/get-docker/) Engine or Desktop | `docker version` |
| SSH agent with access to your Git host | `ssh -T git@host` |

### Lint and format

Same checks as the CI **lint** job. No separate `golangci-lint` install — `make` downloads **v2.9.0** via `go run` on first use.

| Command | Purpose |
|---------|---------|
| `make lint` | Run all linters + format checks (must pass before push) |
| `make fmt` | Auto-fix `gofmt` / `goimports` issues |
| `make lint-fix` | Run linters and apply auto-fixes where supported |

Without Make:

```bash
go run scripts/lint.go
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 fmt
```

### Optional tools

| Tool | Purpose | Verify |
|------|---------|--------|
| [Delve](https://github.com/go-delve/delve) | debugger | `dlv version` |
| Go extension (Cursor / VS Code) | test CodeLens, Delve launch | — |

Unit tests do not need Docker. Local-git tests use `file://` bare repos.

## Setup

### 1. Clone the repositories

```bash
git clone git@host:group/msc-cli.git ~/src/msc-cli
git clone git@host:group/meta.git ~/src/my-meta-repo
```

Adjust paths and remotes to match your machine.

### 2. Build the engine

**Linux / macOS:**

```bash
cd ~/src/msc-cli
make build    # writes bin/msc (gitignored)
./bin/msc --version
```

**Windows (PowerShell):**

```powershell
cd C:\src\msc-cli
make build    # writes bin\msc.exe (gitignored)
.\bin\msc.exe --version
```

### 3. Put `msc` on your PATH (recommended)

Keep your shell on the **latest `make build` output** without copying binaries after every change.

#### Linux / macOS — symlink

```bash
mkdir -p ~/.local/bin
ln -sf ~/src/msc-cli/bin/msc ~/.local/bin/msc
export PATH="$HOME/.local/bin:$PATH"   # add to ~/.bashrc / ~/.zshrc to persist
```

Verify:

```bash
readlink -f "$(command -v msc)"   # should end in .../msc-cli/bin/msc
msc --version
```

#### Windows — first-time (machine not configured yet)

Use **PowerShell**, not Command Prompt. Until step 3 finishes, `msc` and `myproject` are **not** on `PATH` — call the built exe by path.

**1. Build** (step 2 above):

```powershell
cd C:\src\msc-cli
make build
.\bin\msc.exe --version
```

**2. Register the meta-repo** from its checkout. This is what puts `msc` on `PATH`: it writes `%LOCALAPPDATA%\msc\msc.exe` (symlink) or `msc.cmd` pointing at `C:\src\msc-cli\bin\msc.exe`, installs the brand command (`myproject.cmd` + a PowerShell function), adds that folder to your user `PATH`, updates `$PROFILE`, and runs `doctor --fix`.

```powershell
cd C:\src\my-meta-repo
C:\src\msc-cli\bin\msc.exe init
```

Accept the UAC prompt if doctor asks to trust the local CA or update hosts.

If the meta-repo is **already registered** and you only need PATH / shims repaired (new machine clone of msc-cli, broken profile, etc.):

```powershell
C:\src\msc-cli\bin\msc.exe doctor --project myproject --fix
```

**3. Reload the shell** so this session sees PATH and the brand function:

```powershell
. $PROFILE
```

Or close the window and open a new PowerShell. Old terminals keep the previous PATH until you do this.

**4. Verify:**

```powershell
Get-Command msc | Select-Object -ExpandProperty Source
# expect ...\AppData\Local\msc\msc.exe  or  ...\msc.cmd

msc --version
myproject --help
```

If `msc` is still not found, `$PROFILE` did not load — run `. $PROFILE` again and confirm `Test-Path $PROFILE`.

**5. Daily loop** after that is only a rebuild. The AppData link still points at `bin\msc.exe`:

```powershell
cd C:\src\msc-cli
make build
msc --version          # new binary
myproject doctor       # same engine, brand mode
```

Do **not** copy `bin\msc.exe` over `myproject`. The brand command is a launcher, not a second engine. Repair with `myproject doctor --fix` (or the full-path `doctor --fix` in step 2 if `myproject` is missing).

A real `msc.exe` symlink needs [Developer Mode](https://learn.microsoft.com/en-us/windows/apps/get-started/developer-mode-features-and-debugging). Without it, `msc.cmd` is enough. The build output must stay named `msc.exe` — an extensionless `msc` can trigger Windows “Open with”.

#### Daily loop (both platforms)

```bash
cd ~/src/msc-cli && make build   # Linux / macOS
```

```powershell
cd C:\src\msc-cli; make build    # Windows
```

No `cp`, no reinstall — PATH (or the Unix symlink / Windows AppData link) always resolves to the fresh build output.

### 4. Register the meta-repo (local)

Contributors typically **join an existing meta-repo** that already has `msc.manifest.yml` in Git. Clone it (step 1), `cd` into it, and register your machine — same as [README §2A](README.md#a-join-an-existing-team-meta-repo-already-on-git).

**Linux / macOS** (after the step 3 symlink, `msc` is on PATH):

```bash
cd ~/src/my-meta-repo
msc init
```

**Windows:** you already ran `init` in [step 3](#windows--first-time-machine-not-configured-yet) via `C:\src\msc-cli\bin\msc.exe init`. Skip this unless you have not done that yet.

Open a **new terminal** if `init` updated shell startup files, then:

```bash
myproject sync
myproject up
```

Use `msc --project myproject …` when debugging the engine; use the brand command for end-user paths.

**Do not** copy `bin/msc` (or `bin/msc.exe`) over the brand command — that replaces the launcher with a second engine binary and breaks `MSC_PROJECT`. Repair with `myproject doctor --fix`.

### 5. Smoke-test against the meta-repo

```bash
cd ~/src/my-meta-repo
myproject doctor
myproject up --no-pull
myproject status
myproject sync --list
myproject switch api --to source
myproject switch api --to docker
```

Use `msc --project myproject …` when debugging the engine; use the brand command for the same paths end users take.

## Run without installing (optional)

```bash
cd ~/src/msc-cli
make run ARGS='--project myproject status'
go run ./cmd/msc --project myproject up
```

Prefer the [PATH workflow](#3-put-msc-on-your-path-recommended) when you iterate daily.

## Tests

This project uses **test-driven development (TDD)** for internal logic: write a failing test, implement the smallest change that passes, then refactor. Tests are the main safety net for a CLI that reads local config, spawns subprocesses, and behaves differently in **engine mode** (`msc`) vs **brand mode** (`myproject` via `MSC_PROJECT`).

### Running tests

There is **one** supported way to run the suite. It works on Windows and Linux/macOS:

```bash
make test
```

`make test` runs `go run scripts/runtests.go`, which:

- runs **all** unit tests (`go test ./... -count=1`)
- points `HOME`, `XDG_CONFIG_HOME`, `APPDATA`, `USERPROFILE`, and `LOCALAPPDATA` at a throwaway directory so tests cannot read your real registry or shims
- is the same isolation CI uses on Ubuntu, macOS, and Windows

Use `make test` before every commit or PR. Do not rely on a second “short” or “isolated” target — those no longer exist.

**Focused work** (one package or one test) can still use `go test` directly. That does **not** isolate your config home unless the test calls `testenv.IsolateUserConfig`. After you finish iterating, run `make test`.

```bash
go test ./internal/syncsvc/ -count=1 -run TestPlan
go test ./internal/cli/ -run TestGitMissingDashDash -count=1 -v
```

**Other Makefile targets** (not alternate test modes):

| Command | Purpose |
|---------|---------|
| `make lint` | `golangci-lint` v2.9.0 via `go run` (same as CI) |
| `make fmt` | Auto-fix formatting (`gofmt`, `goimports`) |
| `make lint-fix` | Lint with `--fix` |
| `make generate` | Regenerate mocks after changing `//go:generate` |
| `make cover` | Coverage report (`coverage.out` + per-function summary) |
| `make ci` | Local pre-push: `go mod tidy`, `go generate`, `make test`, `make lint`, build |

#### Editor / IDE

CodeLens and **Debug package tests** (see [Editor setup](#editor-setup-local-not-committed)) are fine while you write a test. They use your normal environment, so they are not a substitute for `make test`.

#### What unit tests do *not* cover

Unit tests do **not** require Docker, a running stack, or network access (git tests use `file://` bare repos or fakes). Validating `up`, `switch`, etc. against a real meta-repo is a **manual smoke test** — see [§5 Smoke-test](#5-smoke-test-against-the-meta-repo).

---

### Writing new tests

#### Where tests live

| Code | Tests | What to assert |
|------|-------|----------------|
| `internal/syncsvc`, `internal/registry`, … | Same package `*_test.go` | Business rules, error messages, edge cases — with **injected** dependencies (fake git, explicit `paths.Resolver`, temp dirs). |
| `internal/cli` | `internal/cli/*_test.go` | Cobra wiring only: flags, help text, that the right service function is invoked. Keep tests short. |
| Shared fixtures | `testdata/` | Static YAML, nginx snippets, manifest samples — never mutated in place. |
| Reusable isolation | `internal/testenv/` | Helpers every package can import (`IsolateUserConfig`, `InstallBrandProject`). |

**Rule:** if a test needs more than a few lines of setup to describe *behavior*, the behavior probably belongs in a service package, not in `internal/cli`.

#### TDD workflow (concrete)

1. **Red** — add `TestYourBehavior` in the package that owns the logic (not in `cli` unless it is purely flag parsing).
2. **Green** — minimal implementation; prefer passing interfaces/fakes over global state.
3. **Refactor** — clean up without changing test expectations.

Example shape for a service test:

```go
func TestPlanUsesCacheWhenFresh(t *testing.T) {
    t.Parallel()
    root := t.TempDir()
    // explicit inputs — no paths.Default() unless you called testenv.IsolateUserConfig(t)
    ...
}
```

Example shape for a brand-mode CLI test:

```go
func TestComposeCompleteServicesAfterLogs(t *testing.T) {
    testenv.InstallBrandProject(t, "myproject")
    cmd := NewRootCmd()
    // ...
}
```

#### Naming and structure

- Name tests `Test<Unit>_<Scenario>` (e.g. `TestParseGit_MissingDashDash`).
- Use `t.Helper()` in setup helpers.
- Prefer table-driven tests for classification / parsing rules.
- Call `t.Parallel()` when the test is self-contained (no shared mutable globals). Do **not** parallelize tests that mutate process-wide state without isolation.

#### Policies (required)

| Policy | Why |
|--------|-----|
| **Hermetic tests** — must pass with an empty config home | CI runners and new contributors have no `~/.config/msc/projects.yml`. A test that passes only because *you* ran `msc init` will break in CI and waste review time. |
| **`testenv.IsolateUserConfig(t)`** when using `paths.Default()`, registry, logs, shims, or completion install paths | Redirects `HOME` / `XDG_CONFIG_HOME` / `APPDATA` / `USERPROFILE` / `LOCALAPPDATA` into `t.TempDir()` for the duration of the test. |
| **`testenv.InstallBrandProject(t, name)`** for brand-mode CLI tests | Setting `MSC_PROJECT` alone is not enough — commands still resolve the project via the registry and manifest on disk. |
| **Explicit `paths.Resolver{Home: …}`** in service-layer tests | Keeps tests independent of env vars and documents which paths matter. |
| **Fixtures in `testdata/`, copies in `t.TempDir()`** | Avoids cross-test pollution and accidental edits to committed files. |
| **Fake or stub external tools** (`fakeGit`, mock docker client) | Unit tests must not require Docker daemon, a real Git host, or sudo. Skip or mock when the real binary is absent (`gitops` skips if `git` not on `PATH`). |
| **No assertions on the developer's registry or manifest** | Never assume project `"isos"` (or any name) is registered locally unless the test created it in an isolated home. |
| **Regenerate mocks after interface changes** | `make generate` + commit; CI fails if generated code is stale. |

#### Do / Don't (quick reference)

| Do | Don't |
|---|---|
| Call `testenv.IsolateUserConfig(t)` when touching user config paths | Assume `~/.config/msc/projects.yml` exists |
| Call `testenv.InstallBrandProject(t, "mybrand")` for `MSC_PROJECT` / brand shims | Set only `t.Setenv("MSC_PROJECT", "isos")` without registering the project |
| Inject fakes (`fakeGit`, `LookPath` stubs in doctor tests) | Call real `docker` / `git clone` over the network in unit tests |
| Add a helper to `internal/testenv/` when two or more tests need the same fixture | Copy-paste env setup into every test file |
| Run `make test` before pushing | Rely only on IDE "Run test" on a machine with a populated registry |

#### Adding a new fixture helper

When several tests need the same meta-repo layout:

1. Add a function to `internal/testenv/` (e.g. `InstallBrandProject`, or a new `InstallProjectWithManifest(t, name, manifestPath)`).
2. Document in a one-line comment what env vars it sets and what files it creates.
3. Use it from tests — keeps isolation consistent and gives CI the same setup as local isolated runs.

---

### Why isolation matters (real example)

Two CLI tests used `t.Setenv("MSC_PROJECT", "isos")` and expected compose/git behavior tied to services in a manifest. They **passed locally** because the developer had already registered `isos` via `msc init`. They **failed on GitHub Actions** (clean home, no registry) on all three OS jobs.

Fix: register the project inside the test via `testenv.InstallBrandProject`, and run CI with an empty config home so regressions are caught automatically.

**Takeaway:** a test that passes on your laptop but fails for a new clone is not a reliable test — treat that as a bug in the test, not in CI.

---

### Pre-PR checklist

- [ ] `make ci` passes (includes test + lint)
- [ ] `make generate` — no unexpected diff (if you changed interfaces)
- [ ] New tests that touch config/registry/brand mode use `internal/testenv`
- [ ] Business logic tested in the owning `internal/<pkg>` package, not only via CLI
- [ ] Manual smoke test against a meta-repo if you changed user-visible command behavior

Local pre-push (same as most of the checklist):

```bash
make ci
```

## Editor setup (local, not committed)

`.vscode/` and `.cursor/` are gitignored.

**Delve from a terminal:**

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/msc -- --version
```

**Optional `.vscode/launch.json`:**

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch msc",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/msc",
      "args": ["--help"],
      "cwd": "${workspaceFolder}"
    },
    {
      "name": "Debug package tests",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/internal/cli"
    }
  ]
}
```

## Repository layout

```
cmd/msc/              entrypoint (thin)
internal/cli/         Cobra wiring only — no business logic
internal/version/     build-time version identity
internal/manifest/    Project Manifest load/save/validate
internal/registry/    local project registry + conflict/repair
internal/stack/       up (pull+compose) / down
internal/nginxcfg/    generated upstream overlay, host-gateway overlay, SIGHUP reload
internal/switchsvc/   Source/Docker mode switch orchestration
internal/syncsvc/     access-aware sync plan + parallel clone
internal/doctor/      machine/project health checks + --fix
internal/shim/        brand command launchers (sh / cmd) that exec msc
internal/initsvc/     init: clone, Manifest, registry, shim
internal/update/      GitHub release plan, checksums, self-update
internal/logging/     local JSON-lines logs + support-bundle zip
internal/testenv/     hermetic test helpers (isolated config home, fixtures)
testdata/             fixtures for unit/integration/E2E tests
scripts/              install.sh / install.ps1; runtests.go (`make test`); lint.go (`make lint`)
.github/workflows/    CI and release
```

`internal/cli` stays thin: parse flags, call another package, print output. Testable logic belongs in sibling packages. Run `make help` for all Makefile targets.

## CI / CD

One workflow (`.github/workflows/ci.yml`) handles both checks and publishing:

| Event | Jobs that run | Release |
|---|---|---|
| Pull request | lint, test (Ubuntu / macOS / Windows), generate | skipped |
| Push to `main` | same checks | skipped |
| Tag `v*` | same checks **on that tag**, then GoReleaser | only if every check succeeded |

`release` uses `needs: [lint, test, generate]`. A failing or cancelled check never publishes a GitHub Release. Checks run on the tagged commit itself — that is the artifact source of truth, not a previous green run on `main`.

Local pre-push check (see [Pre-PR checklist](#pre-pr-checklist) in Tests):

```bash
make ci
```

### Releasing (maintainers)

1. Land the commit on `main` with green CI (required status checks on the branch ruleset, if enabled).
2. Tag that commit and push the tag:

   ```bash
   git tag -a v1.0.2 -m "…"
   git push origin v1.0.2
   ```

3. The tag run re-executes lint / test / generate. If they pass, GoReleaser publishes linux/darwin/windows archives and `checksums.txt`. Then `scripts/install.sh` and `msc self-update` can verify the checksums.

Do not delete and recreate a published tag unless you also delete the matching GitHub Release first. Prefer the next patch version.

## Module path

Go module: `github.com/SoheilHasankhani/msc-cli`. GitHub remote: `SoheilHasankhani/msc-cli`.
