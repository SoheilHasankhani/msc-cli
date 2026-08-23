package completesvc

import (
	"fmt"
	"io"
	"sort"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
)

// BrandNames returns sorted registry project names for shell completion.
func BrandNames(dirs paths.Resolver) ([]string, error) {
	reg, err := registry.Load(dirs.RegistryFile())
	if err != nil {
		return nil, err
	}
	names := reg.Names()
	sort.Strings(names)
	return names, nil
}

// WriteBashBrandCompleters registers brand shims with the msc bash completion function.
func WriteBashBrandCompleters(w io.Writer, names []string) error {
	if len(names) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "# Brand shims use the same completion function as msc"); err != nil {
		return err
	}
	for _, name := range names {
		if name == "msc" {
			continue
		}
		if _, err := fmt.Fprintf(w, "complete -o default -F __start_msc %q\n", name); err != nil {
			return err
		}
	}
	return nil
}

// WriteZshBrandCompleters registers brand shims with the msc zsh completion function.
func WriteZshBrandCompleters(w io.Writer, names []string) error {
	if len(names) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "# Brand shims use the same completion function as msc"); err != nil {
		return err
	}
	for _, name := range names {
		if name == "msc" {
			continue
		}
		if _, err := fmt.Fprintf(w, "compdef _msc %q\n", name); err != nil {
			return err
		}
	}
	return nil
}

// WritePowerShellBrandCompleters registers brand commands with the msc completer block.
func WritePowerShellBrandCompleters(w io.Writer, names []string) error {
	if len(names) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "# Brand commands share msc's completer block"); err != nil {
		return err
	}
	for _, name := range names {
		if name == "msc" {
			continue
		}
		if _, err := fmt.Fprintf(w, "Register-ArgumentCompleter -CommandName %q -ScriptBlock ${__mscCompleterBlock}\n", name); err != nil {
			return err
		}
	}
	return nil
}
