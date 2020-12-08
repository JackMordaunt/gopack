// gopack is a tool for creating OS specific installable packages for Windows,
// macOS and Linux.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~jackmordaunt/gopack"
	"github.com/akavel/rsrc/rsrc"
)

func main() {
	// TODO
	// - Accept dir to binaries, parse binary formats to figure what platform
	// they are intended for.
	// - Allow for zero config execution.
	if err := func() error {
		var (
			root    string
			pkg     string
			name    string
			targets []gopack.Target
		)
		args, named := parse(os.Args[1:])
		if len(args) <= 1 {
			return fmt.Errorf("specify root of project")
		}
		root = args[0]
		if len(args) > 1 {
			pkg = args[1]
		}
		if len(args) > 2 {
			name = args[2]
		}
		if name == "" {
			name = pkg
		}
		if tlist, ok := named["targets"]; ok {
			for _, t := range strings.Split(tlist, ",") {
				targets = append(targets, gopack.NewTarget(strings.TrimSpace(t)))
			}
		}
		if len(targets) == 0 {
			targets = gopack.DefaultTargets
		}
		packer := gopack.Packer{
			Info: &gopack.ProjectInfo{
				Root:    root,
				Pkg:     pkg,
				Name:    name,
				Targets: targets,
				Flags: map[gopack.Target]gopack.FlagSet{
					gopack.NewTarget("windows/amd64"): {
						Linker: []string{"-H windowsgui"},
					},
					gopack.NewTarget("windows/386"): {
						Linker: []string{"-H windowsgui"},
					},
				},
			},
			PreCompile: func(root string, md gopack.MetaData, t gopack.Target) error {
				// @Enhance editing PE binary data inline, without creating a .syso file could be
				// more robust.
				if t.Platform == gopack.Windows && md.Windows.ICO != nil {
					var (
						resource = filepath.Join(root, "rsrc.syso")
						path     = filepath.Join(os.TempDir(), "gopack", "icon.ico")
					)
					buffer, err := ioutil.ReadAll(md.Windows.ICO)
					if err != nil {
						return fmt.Errorf("reading ico data: %w", err)
					}
					if err := ioutil.WriteFile(path, buffer, 0777); err != nil {
						return fmt.Errorf("writing icon file to temporary location: %w", err)
					}
					if err := rsrc.Embed(resource, t.Architecture.String(), "", path); err != nil {
						return fmt.Errorf("creating icon resource: %w", err)
					}
					return nil
				}
				return nil
			},
		}
		if err := packer.Pack(); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		fmt.Printf("error: %v", err)
	}
}

// parse produces a list of positional and named arguments.
// An argument is named if it has a "-" prefix.
func parse(args []string) ([]string, map[string]string) {
	var (
		positional = []string{}
		named      = map[string]string{}
	)
	for ii := 0; ii < len(args); ii++ {
		arg := args[ii]
		if isNamed := strings.HasPrefix(arg, "-"); isNamed {
			// either it's combined via = or whitespace
			if parts := strings.Split(arg, "="); len(parts) > 2 {
				named[strings.Trim(parts[0], "-")] = parts[1]
			} else {
				named[strings.Trim(arg, "-")] = args[ii+1]
				ii++
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return positional, named
}
