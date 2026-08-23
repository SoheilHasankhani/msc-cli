package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/elevate"
	"github.com/SoheilHasankhani/msc-cli/internal/hostcerts"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/shim"
)

// Options configures doctor. Nil function fields use production implementations.
type Options struct {
	Project         *project.Context
	Fix             bool
	LookPath        func(string) (string, error)
	DockerPing      func(context.Context) error
	SSHStatus       func() (agent, identities bool, err error)
	HostsText       string
	EnsureOverlay   func() (wrote bool, err error)
	Elevate         elevate.Elevator
	NoElevate       bool
	ApplyHosts      func(project string, names []string) error
	EnsureCerts     func() (hostcerts.Bundle, error)
	TrustCA         func(caPath string) error
	TrustNSS        func(caPath string) error
	MachineCertsDir string
	TrustOK         func(caPath string) bool
}

// Run executes machine checks, and project checks when Project is set.
func Run(ctx context.Context, opt Options) (Report, error) {
	if opt.LookPath == nil {
		opt.LookPath = exec.LookPath
	}
	if opt.DockerPing == nil {
		opt.DockerPing = pingDocker
	}
	if opt.SSHStatus == nil {
		opt.SSHStatus = sshStatus
	}

	var r Report
	r.Checks = append(r.Checks, checkBinary(opt, "git", "install Git and ensure it is on PATH"))
	r.Checks = append(r.Checks, checkDocker(ctx, opt))
	r.Checks = append(r.Checks, checkSSH(opt))

	if opt.Project == nil || opt.Project.Manifest == nil {
		return r, nil
	}

	for _, extra := range extraPrereqs(opt.Project.Manifest.Prerequisites) {
		r.Checks = append(r.Checks, checkBinary(opt, extra, fmt.Sprintf("install %s (listed in Manifest prerequisites) or remove it from the Manifest", extra)))
	}

	hostsCheck, names := checkHosts(opt)
	r.Checks = append(r.Checks, projectChecks(opt, hostsCheck)...)

	if opt.Fix {
		applyFixes(ctx, &r, opt, names)
		// Re-evaluate so the table and exit code match the machine after repairs.
		hostsCheck, _ = checkHosts(opt)
		r.Checks = append(r.Checks[:nonProjectCheckCount(r)], projectChecks(opt, hostsCheck)...)
	}
	return r, nil
}

func nonProjectCheckCount(r Report) int {
	n := 0
	for _, c := range r.Checks {
		switch c.Name {
		case "hosts", "certs", "os-trust", "overlay", "shim":
			return n
		default:
			n++
		}
	}
	return n
}

func projectChecks(opt Options, hostsCheck Check) []Check {
	return []Check{
		hostsCheck,
		checkCerts(opt),
		checkOSTrust(opt),
		checkOverlay(opt),
		checkShim(opt),
	}
}

func extraPrereqs(listed []string) []string {
	core := map[string]bool{"docker": true, "git": true, "ssh": true}
	var out []string
	seen := map[string]bool{}
	for _, p := range listed {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || core[p] || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func checkBinary(opt Options, name, next string) Check {
	if _, err := opt.LookPath(name); err != nil {
		return Check{Name: name, Status: StatusFail, Message: name + " is not on PATH", Next: next, Fix: FixNone}
	}
	return Check{Name: name, Status: StatusPass, Message: name + " found"}
}

func checkDocker(ctx context.Context, opt Options) Check {
	if err := opt.DockerPing(ctx); err != nil {
		return Check{Name: "docker", Status: StatusFail, Message: "cannot reach Docker Engine", Next: "install Docker and start the engine (Desktop or docker-ce)", Fix: FixNone}
	}
	return Check{Name: "docker", Status: StatusPass, Message: "Docker Engine reachable"}
}

func checkSSH(opt Options) Check {
	agent, ids, err := opt.SSHStatus()
	if err != nil {
		return Check{Name: "ssh", Status: StatusFail, Message: err.Error(), Next: "install OpenSSH and retry", Fix: FixNone}
	}
	if !ids {
		if !agent {
			return Check{Name: "ssh", Status: StatusFail, Message: "SSH agent is not running and no private keys found in ~/.ssh", Next: "add a key to ~/.ssh or start ssh-agent and run ssh-add", Fix: FixNone}
		}
		return Check{Name: "ssh", Status: StatusFail, Message: "SSH agent has no identities", Next: "ssh-add your SSH key and register its public key with your Git host", Fix: FixNone}
	}
	if agent {
		return Check{Name: "ssh", Status: StatusPass, Message: "SSH agent has at least one key"}
	}
	return Check{Name: "ssh", Status: StatusPass, Message: "SSH private keys found in ~/.ssh"}
}

func checkHosts(opt Options) (Check, []string) {
	nginxText, err := hostcerts.ReadNginxDir(nginxcfg.ComponentsDir(opt.Project.ConfigDir()))
	if err != nil {
		return Check{Name: "hosts", Status: StatusWarn, Message: "nginx config not readable: " + err.Error(), Next: "create layout.config_dir/nginx/components", Fix: FixNone}, nil
	}
	names := hostcerts.CollectHostnames(opt.Project.Manifest.LocalDomain, nginxText)
	text := opt.HostsText
	if text == "" {
		path := hostcerts.SystemHostsPath(runtime.GOOS)
		data, err := os.ReadFile(path)
		if err != nil {
			return Check{Name: "hosts", Status: StatusFail, Message: "cannot read " + path, Next: "doctor --fix (needs elevation)", Fix: FixHosts}, names
		}
		text = string(data)
	}
	miss := hostcerts.Missing(text, names)
	if len(miss) == 0 {
		return Check{Name: "hosts", Status: StatusPass, Message: fmt.Sprintf("%d hostnames present", len(names)), Fix: FixHosts}, names
	}
	return Check{
		Name:    "hosts",
		Status:  StatusFail,
		Message: "missing " + strings.Join(miss, ", "),
		Next:    "doctor --fix (needs elevation to write the hosts file)",
		Fix:     FixHosts,
	}, names
}

func checkCerts(opt Options) Check {
	domain := ""
	if opt.Project.Manifest != nil {
		domain = opt.Project.Manifest.LocalDomain
	}
	b := certPaths(opt, domain)
	if domain != "" && hostcerts.Valid(b, domain) == nil {
		return Check{Name: "certs", Status: StatusPass, Message: "machine local-ca + " + hostcerts.LeafBase(domain) + " leaf present", Fix: FixCerts}
	}
	return Check{Name: "certs", Status: StatusFail, Message: "machine local-ca or project wildcard leaf missing", Next: "doctor --fix generates the machine CA, copies local-ca.crt, and writes the domain leaf", Fix: FixCerts}
}

func checkOSTrust(opt Options) Check {
	domain := ""
	if opt.Project.Manifest != nil {
		domain = opt.Project.Manifest.LocalDomain
	}
	b := certPaths(opt, domain)
	if _, err := os.Stat(b.CACrt); err != nil {
		return Check{Name: "os-trust", Status: StatusFail, Message: "machine local-ca is not in the OS trust store", Next: "doctor --fix installs the CA (needs elevation)", Fix: FixOSTrust}
	}
	if trustOK(opt, b.CACrt) {
		return Check{Name: "os-trust", Status: StatusPass, Message: "OS trust store matches machine local-ca"}
	}
	return Check{Name: "os-trust", Status: StatusFail, Message: "OS trust store does not match machine local-ca", Next: "doctor --fix reinstalls the CA (needs elevation)", Fix: FixOSTrust}
}

func checkOverlay(opt Options) Check {
	compose := opt.Project.ComposeFile()
	data, err := os.ReadFile(compose)
	if err != nil {
		return Check{Name: "overlay", Status: StatusWarn, Message: "compose file not readable", Next: "check layout.compose_file", Fix: FixOverlay}
	}
	if nginxcfg.HasHostGateway(string(data)) {
		return Check{Name: "overlay", Status: StatusPass, Message: "host.docker.internal is defined in compose"}
	}
	overlay := filepath.Join(filepath.Dir(compose), nginxcfg.OverlayFileName)
	if existing, err := os.ReadFile(overlay); err == nil && nginxcfg.HasHostGateway(string(existing)) {
		return Check{Name: "overlay", Status: StatusPass, Message: "host-gateway overlay present"}
	}
	return Check{Name: "overlay", Status: StatusFail, Message: "nginx cannot resolve host.docker.internal on native Engine", Next: "doctor --fix writes " + nginxcfg.OverlayFileName, Fix: FixOverlay}
}

func checkShim(opt Options) Check {
	brand := opt.Project.Name
	if opt.Project.Manifest.Brand.Command != "" {
		brand = opt.Project.Manifest.Brand.Command
	}
	dirs := paths.Default()
	shimDir := dirs.ShimDir()
	if shim.ValidBrandShim(shimDir, brand, runtime.GOOS) {
		return Check{Name: "shim", Status: StatusPass, Message: brand + " brand shim is installed"}
	}
	msg := brand + " shim is missing or not a shell script"
	if runtime.GOOS == "windows" {
		msg = brand + " shim is missing or not a cmd launcher"
	}
	return Check{Name: "shim", Status: StatusFail, Message: msg, Next: "doctor --fix rewires the brand shim to exec msc with MSC_PROJECT", Fix: FixShim}
}

func applyFixes(ctx context.Context, r *Report, opt Options, names []string) {
	applyShimFix(r, opt)
	if runtime.GOOS == "windows" && opt.Project != nil {
		applyWindowsShellHooks(r, opt)
	}
	// Non-elevated work first so a sudo prompt cannot block cert/overlay writes.
	applyOverlayFix(r, opt)
	certsPass := checkStatus(*r, "certs") == StatusPass
	trustPass := checkStatus(*r, "os-trust") == StatusPass
	priorCA, _ := hostcerts.FileFingerprint(certPaths(opt, opt.Project.Manifest.LocalDomain).CACrt)
	var bundle *hostcerts.Bundle
	for _, c := range r.Checks {
		if c.Fix != FixCerts {
			continue
		}
		b, err := ensureCerts(opt)
		if err != nil {
			r.Skipped = append(r.Skipped, "certs: "+err.Error())
			break
		}
		bundle = &b
		if !certsPass {
			r.Fixed = append(r.Fixed, "certs machine local-ca + wildcard leaf")
		}
	}
	for _, c := range r.Checks {
		if c.Fix != FixHosts || c.Status == StatusPass {
			continue
		}
		if err := applyHosts(ctx, opt, names); err != nil {
			r.Skipped = append(r.Skipped, "hosts: "+fixErr(err))
			continue
		}
		r.Fixed = append(r.Fixed, "hosts additive msc-begin block")
	}
	if bundle == nil {
		return
	}
	afterCA, _ := hostcerts.FileFingerprint(bundle.CACrt)
	if trustPass && (priorCA == "" || priorCA == afterCA) {
		return
	}
	if err := trustCA(ctx, opt, bundle.CACrt); err != nil {
		r.Skipped = append(r.Skipped, "certs OS trust: "+fixErr(err))
	} else {
		if afterCA != "" {
			_ = hostcerts.WriteTrustStamp(machineCertsDir(opt), afterCA)
		}
		r.Fixed = append(r.Fixed, "certs CA installed in OS trust store")
	}
	if err := trustNSS(opt, bundle.CACrt); err != nil {
		r.Skipped = append(r.Skipped, "certs NSS: "+err.Error())
	} else if runtime.GOOS == "linux" {
		r.Fixed = append(r.Fixed, "certs CA installed in Chrome NSS db")
	}
}

func checkStatus(r Report, name string) Status {
	for _, c := range r.Checks {
		if c.Name == name {
			return c.Status
		}
	}
	return StatusFail
}

func applyShimFix(r *Report, opt Options) {
	if opt.Project == nil {
		return
	}
	brand := opt.Project.Name
	if opt.Project.Manifest.Brand.Command != "" {
		brand = opt.Project.Manifest.Brand.Command
	}
	needs := shim.BrandShimNeedsRefresh(brand, paths.Default())
	if !needs {
		for _, c := range r.Checks {
			if c.Fix == FixShim && c.Status != StatusPass {
				needs = true
				break
			}
		}
	}
	if !needs {
		return
	}
	refreshBrandShim(r, opt)
}

func refreshBrandShim(r *Report, opt Options) {
	brand := opt.Project.Name
	if opt.Project.Manifest.Brand.Command != "" {
		brand = opt.Project.Manifest.Brand.Command
	}
	dirs := paths.Default()
	engine := shim.ResolveEnginePath()
	if runtime.GOOS == "windows" {
		if _, err := shim.EnsureWindowsEngineCommand(dirs.BinDir(), engine); err != nil {
			r.Skipped = append(r.Skipped, "msc path: "+err.Error())
		} else {
			engine = shim.WindowsEngineLaunchPath(dirs.BinDir(), engine)
		}
	}
	shimPath, err := shim.Write(dirs.ShimDir(), brand, engine, runtime.GOOS)
	if err != nil {
		r.Skipped = append(r.Skipped, "shim: "+err.Error())
		return
	}
	if _, err := shim.InstallOnPATH(brand, shimPath, dirs); err != nil {
		r.Skipped = append(r.Skipped, "shim path: "+err.Error())
		return
	}
	r.Fixed = append(r.Fixed, "shim "+brand)
}

func applyWindowsShellHooks(r *Report, opt Options) {
	if !opt.Fix {
		return
	}
	dirs := paths.Default()
	linked, err := shim.EnsureWindowsEngineCommand(dirs.BinDir(), shim.ResolveEnginePath())
	if err != nil {
		r.Skipped = append(r.Skipped, "msc path: "+err.Error())
	} else if linked {
		r.Fixed = append(r.Fixed, "msc on PATH")
	}
	_, changed, err := shim.RefreshWindowsShellHooks(dirs.Home, dirs.BinDir())
	if err != nil {
		r.Skipped = append(r.Skipped, "powershell hooks: "+err.Error())
		return
	}
	if changed {
		r.Fixed = append(r.Fixed, "powershell brand commands")
	}
}

func applyOverlayFix(r *Report, opt Options) {
	for _, c := range r.Checks {
		if c.Fix != FixOverlay || c.Status == StatusPass {
			continue
		}
		fn := opt.EnsureOverlay
		if fn == nil {
			fn = func() (bool, error) {
				_, wrote, err := nginxcfg.EnsureHostGateway(opt.Project.Root, opt.Project.Manifest.Layout.ComposeFile)
				return wrote, err
			}
		}
		wrote, err := fn()
		if err != nil {
			r.Skipped = append(r.Skipped, "overlay: "+err.Error())
			return
		}
		if wrote {
			r.Fixed = append(r.Fixed, "overlay "+nginxcfg.OverlayFileName)
		}
	}
}

func applyHosts(ctx context.Context, opt Options, names []string) error {
	if opt.NoElevate {
		return fmt.Errorf("skipped (--no-elevate); re-run doctor --fix in a terminal")
	}
	if opt.ApplyHosts != nil {
		return opt.ApplyHosts(opt.Project.Name, names)
	}
	if opt.Elevate == nil {
		return fmt.Errorf("needs elevation — re-run doctor --fix")
	}
	return withPayload(ctx, opt.Elevate, "update the hosts file for "+opt.Project.Name, hostcerts.Payload{
		Op:        hostcerts.OpWriteHosts,
		Project:   opt.Project.Name,
		Names:     names,
		HostsPath: hostcerts.SystemHostsPath(runtime.GOOS),
	})
}

func machineCertsDir(opt Options) string {
	if opt.MachineCertsDir != "" {
		return opt.MachineCertsDir
	}
	return paths.Default().CertsDir()
}

func certPaths(opt Options, domain string) hostcerts.Bundle {
	return hostcerts.Paths(machineCertsDir(opt), hostcerts.Dir(opt.Project.ConfigDir()), domain)
}

func trustOK(opt Options, caPath string) bool {
	if opt.TrustOK != nil {
		return opt.TrustOK(caPath)
	}
	return hostcerts.OSTrustMatches(runtime.GOOS, caPath, machineCertsDir(opt))
}

func ensureCerts(opt Options) (hostcerts.Bundle, error) {
	if opt.EnsureCerts != nil {
		return opt.EnsureCerts()
	}
	return hostcerts.Ensure(machineCertsDir(opt), hostcerts.Dir(opt.Project.ConfigDir()), opt.Project.Manifest.LocalDomain)
}

func trustCA(ctx context.Context, opt Options, caPath string) error {
	if opt.NoElevate {
		return fmt.Errorf("skipped (--no-elevate); re-run doctor --fix in a terminal")
	}
	if opt.TrustCA != nil {
		return opt.TrustCA(caPath)
	}
	if opt.Elevate == nil {
		return fmt.Errorf("needs elevation — re-run doctor --fix")
	}
	return withPayload(ctx, opt.Elevate, "install the msc local CA", hostcerts.Payload{
		Op:       hostcerts.OpInstallCA,
		Project:  opt.Project.Name,
		CAPath:   caPath,
		DestPath: hostcerts.LinuxCADest(),
	})
}

func trustNSS(opt Options, caPath string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if opt.NoElevate {
		return fmt.Errorf("skipped (--no-elevate); re-run doctor --fix in a terminal")
	}
	if opt.TrustNSS != nil {
		return opt.TrustNSS(caPath)
	}
	if opt.Elevate == nil {
		return fmt.Errorf("re-run doctor --fix to add the CA to Chrome NSS")
	}
	if hostcerts.LookPathNSS() != nil {
		return fmt.Errorf("certutil not on PATH — install libnss3-tools for Chrome")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return hostcerts.InstallNSS(home, caPath, hostcerts.ExecRunner, hostcerts.InteractiveRunner)
}

func withPayload(ctx context.Context, el elevate.Elevator, desc string, p hostcerts.Payload) error {
	f, err := os.CreateTemp("", "msc-elevated-*.json")
	if err != nil {
		return err
	}
	path := f.Name()
	_ = f.Close()
	if err := hostcerts.WritePayloadFile(path, p); err != nil {
		return err
	}
	return el.RunElevated(ctx, desc, []string{"__elevated-do", "--payload", path})
}

func fixErr(err error) string {
	if elevate.IsNeedTTY(err) {
		return err.Error()
	}
	return err.Error()
}

func pingDocker(ctx context.Context) error {
	e, err := dockerapi.NewEngine()
	if err != nil {
		return err
	}
	defer e.Close()
	return e.Ping(ctx)
}

func sshStatus() (agent, identities bool, err error) {
	if _, err := exec.LookPath("ssh-add"); err != nil {
		return false, false, fmt.Errorf("ssh-add is not on PATH")
	}
	cmd := exec.Command("ssh-add", "-l")
	out, runErr := cmd.CombinedOutput()
	text := strings.ToLower(string(out))
	if runErr == nil {
		return true, true, nil
	}
	if strings.Contains(text, "could not open a connection") ||
		strings.Contains(text, "connecting to agent") ||
		strings.Contains(text, "ssh_auth_sock") {
		agent = false
	} else if strings.Contains(text, "no identities") || strings.Contains(text, "the agent has no identities") {
		agent = true
	} else if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
		// ssh-add -l exits 1 for no identities on some platforms without that phrase.
		agent = true
	} else if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 2 {
		agent = false
	} else {
		return false, false, fmt.Errorf("ssh-add -l: %s", strings.TrimSpace(string(out)))
	}
	if sshPrivateKeysAvailable() {
		return agent, true, nil
	}
	return agent, false, nil
}

// sshPrivateKeysAvailable reports whether ~/.ssh has a standard private key file.
// OpenSSH reads these directly when no agent is running (common on Windows).
func sshPrivateKeysAvailable() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	return sshPrivateKeysIn(filepath.Join(home, ".ssh"))
}

func sshPrivateKeysIn(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch e.Name() {
		case "id_rsa", "id_ed25519", "id_ecdsa", "id_dsa":
			return true
		}
	}
	return false
}
