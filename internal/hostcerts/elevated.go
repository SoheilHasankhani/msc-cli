package hostcerts

import (
	"fmt"
	"os"
	"strings"
)

// ApplyElevated is the hidden-command implementation. Destinations are forced to
// the real OS paths so a payload cannot point sudo at an arbitrary file.
func ApplyElevated(p Payload, goos string, run CommandRunner) error {
	switch p.Op {
	case OpWriteHosts:
		if strings.TrimSpace(p.Project) == "" {
			return fmt.Errorf("write-hosts: project is required")
		}
		p.HostsPath = SystemHostsPath(goos)
		return WriteFile(p.HostsPath, p.Project, p.Names)
	case OpInstallCA:
		if strings.TrimSpace(p.Project) == "" {
			return fmt.Errorf("install-ca: project is required")
		}
		if p.CAPath == "" {
			return fmt.Errorf("install-ca: ca_path is required")
		}
		if err := guardPath(p.CAPath); err != nil {
			return err
		}
		data, err := os.ReadFile(p.CAPath)
		if err != nil {
			return err
		}
		if _, err := ParseCertificatePEM(data); err != nil {
			return fmt.Errorf("install-ca: %w", err)
		}
		plan := OSTrustPlan(goos, p.CAPath)
		if goos == "windows" || goos == "darwin" {
			if run == nil {
				return fmt.Errorf("install-ca: missing command runner")
			}
			return run(plan.Tool, plan.Args...)
		}
		p.DestPath = plan.Dest
		if err := ApplyPayload(p); err != nil {
			return err
		}
		if plan.UpdateCmd != "" && run != nil {
			return run(plan.UpdateCmd, "-f")
		}
		return nil
	default:
		return fmt.Errorf("unknown elevated operation %q", p.Op)
	}
}
