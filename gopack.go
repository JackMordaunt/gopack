package gopack

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"git.sr.ht/~jackmordaunt/gopack/internal/util"
)

// Packer packs a project into native artifacts.
type Packer struct {
	Info      *ProjectInfo
	MetaData  MetaData
	Artifacts []Artifact
	// PreCompile is run prior to compiling.
	// Allows modification of the compilation environment, such as generating a
	// Windows resource file to be compiled in.
	PreCompile func(root string, md MetaData, t Target) error
}

// ProjectInfo contains data required to compile a Go project.
type ProjectInfo struct {
	// Name specifies the output name of the artifact.
	// Defaults to the package name if empty.
	Name string
	// Root path to Go project.
	// Defaults to current working directory.
	Root string
	// Name of package to build.
	// Defaults to root if empty.
	Pkg string
	// Dist is the output directory to place Artifacts.
	// Defaults to "dist".
	Dist string
	// Flags are values to pass to the compiler.
	Flags Flags
	// Targets lists all targets to compile for.
	Targets []Target
}

// Flags maps tooling flags to Targets.
type Flags map[Target]FlagSet

// Lookup the flags for a Target, returning an zero value if not FlagSet
// specified.
func (f Flags) Lookup(target Target) FlagSet {
	if t, ok := f[target]; ok {
		return t
	}
	return FlagSet{}
}

// Flags contains tooling flags.
type FlagSet struct {
	// Compiler flags.
	// go tool compile
	Compiler []string
	// Linker flags.
	// go tool link
	Linker []string
}

// Artifact associates a path to a binary with the platform it's intended for.
type Artifact struct {
	Binary io.Reader
	Target
}

type Target struct {
	Platform     Platform
	Architecture Architecture
}

// DefaultTargets is a static list of supported targets as a subset of output by
// `go tool dist list`.
var DefaultTargets = []Target{
	NewTarget("windows/386"),
	NewTarget("windows/amd64"),
	NewTarget("windows/arm"),
	NewTarget("darwin/amd64"),
	// NewTarget("darwin/arm64"),
	NewTarget("linux/386"),
	NewTarget("linux/amd64"),
	NewTarget("linux/arm"),
	NewTarget("linux/arm64"),
	NewTarget("js/wasm"),
}

func NewTarget(s string) Target {
	var (
		p Platform
		a Architecture
	)
	return Target{
		Platform:     p.FromStr(strings.Split(s, "/")[0]),
		Architecture: a.FromStr(strings.Split(s, "/")[1]),
	}
}

func (t Target) Ext() string {
	switch t.Platform {
	case Windows:
		return ".exe"
	case JS:
		return ".wasm"
	}
	return ""
}

func (t Target) String() string {
	return fmt.Sprintf("%s_%s", t.Platform, t.Architecture)
}

// Platform identifier for the platforms we care about.
type Platform uint8

// Platforms supported.
const (
	Windows Platform = iota
	Darwin
	Linux
	JS
)

func (p Platform) String() string {
	switch p {
	case Windows:
		return "windows"
	case Darwin:
		return "darwin"
	case Linux:
		return "linux"
	case JS:
		return "js"
	}
	return "unknown"
}

func (p *Platform) FromStr(s string) Platform {
	switch s {
	case "windows":
		*p = Windows
	case "darwin":
		*p = Darwin
	case "linux":
		*p = Linux
	case "js":
		*p = JS
	}
	return *p
}

func (p Platform) List() []Platform {
	return []Platform{Windows, Darwin, Linux}
}

// Architecture of the machine.
type Architecture uint8

// Architectures supported.
const (
	X86 Architecture = iota
	AMD64
	ARM
	ARM64
	WASM
)

func (a Architecture) String() string {
	switch a {
	case X86:
		return "386"
	case AMD64:
		return "amd64"
	case ARM:
		return "arm"
	case ARM64:
		return "arm64"
	case WASM:
		return "wasm"
	}
	return "unknown"
}

func (a *Architecture) FromStr(s string) Architecture {
	switch s {
	case "386":
		*a = X86
	case "amd64":
		*a = AMD64
	case "arm":
		*a = ARM
	case "arm64":
		*a = ARM64
	case "wasm":
		*a = WASM
	}
	return *a
}

func (a Architecture) List() []Architecture {
	return []Architecture{X86, AMD64, ARM, ARM64}
}

// Pack the binaries into native formats.
func (p Packer) Pack() error {
	if p.Info != nil {
		if err := p.MetaData.Load(p.Info.Root); err != nil {
			return fmt.Errorf("loading metadata: %w", err)
		}
		if p.MetaData.Icon == nil {
			fmt.Printf("warning: icon not found (icon.png)\n")
		}
		if err := p.Compile(); err != nil {
			return fmt.Errorf("compiling %s: %w", p.Info.Pkg, err)
		}
	}
	if len(p.Artifacts) == 0 {
		return fmt.Errorf("no artifacts to pack")
	}
	wg := &sync.WaitGroup{}
	for _, artifact := range p.Artifacts {
		var (
			artifact = artifact
			dir      = filepath.Join(p.Output(), artifact.Target.String())
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch artifact.Platform {
			case Darwin:
				if err := bundleMacOS(
					dir,
					p.Info.Name,
					artifact.Binary,
					p.MetaData.Darwin.ICNS,
					p.MetaData.Darwin.Plist,
				); err != nil {
					fmt.Printf("bundling macos: %s\n", err)
				}
			case Windows:
				if err := bundleWindows(
					filepath.Join(dir, fmt.Sprintf("%s.exe", p.Info.Name)),
					artifact.Binary,
				); err != nil {
					fmt.Printf("bundling windows: %s\n", err)
				}
			case Linux:
				fmt.Printf("warning: bundle ignored\n")
				// if err := bundleLinux(
				// 	dir,
				// 	artifact.Binary,
				// 	"",
				// ); err != nil {
				// 	fmt.Printf("bundling linux: %s\n", err)
				// }
			}
		}()
	}
	wg.Wait()
	return nil
}

// Compile the Go project.
// Requires Go toolchain to be installed.
// Compiles targets in parallel.
func (p *Packer) Compile() error {
	var (
		output = p.Output()
		err    error
	)
	if p.Info.Root == "" {
		p.Info.Root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolving current working directory: %w", err)
		}
	}
	if r, err := filepath.Abs(p.Info.Root); err != nil {
		return fmt.Errorf("resolving root: %w", err)
	} else {
		p.Info.Root = r
	}
	if p.Info.Pkg != "" {
		p.Info.Pkg, err = util.Finder{
			Root:  p.Info.Root,
			IsDir: true,
			Rel:   true,
		}.Find(p.Info.Pkg)
		if err != nil {
			return fmt.Errorf("finding package: %w", err)
		}
		if p.Info.Pkg == "" {
			return fmt.Errorf("package %q not found", p.Info.Pkg)
		}
	} else {
		p.Info.Pkg = p.Info.Root
	}
	var (
		sandbox = filepath.Join(os.TempDir(), "gopack")
		wg      = &sync.WaitGroup{}
		errs    = make(chan error, len(p.Info.Targets))
	)
	fmt.Printf("package: %s\n", p.Info.Pkg)
	fmt.Printf("sandbox: %s\n", sandbox)
	for _, target := range p.Info.Targets {
		target := target
		var (
			platform = target.Platform
			arch     = target.Architecture
			sandbox  = filepath.Join(sandbox, target.String())
			bin      = fmt.Sprintf(
				"%s%s",
				filepath.Join(
					sandbox,
					output,
					fmt.Sprintf("%s_%s", platform.String(), arch.String()),
					filepath.Base(p.Info.Pkg)),
				target.Ext())
		)
		if err := os.RemoveAll(sandbox); err != nil {
			log.Printf("cleaning sandbox: %v", err)
		}
		if err := os.MkdirAll(sandbox, 0777); err != nil {
			log.Printf("preparing sandbox: %v", err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := func() error {
				// ENHANCE can we make this more semantic? EG: "prepare sandbox".
				if err := (util.Copier{
					Recursive: true,
					Ignore:    []string{"dist", ".git"},
				}).Copy(p.Info.Root, sandbox); err != nil {
					return fmt.Errorf("creating sandbox: %w", err)
				}
				if p.PreCompile != nil {
					if err := p.PreCompile(sandbox, p.MetaData, target); err != nil {
						return fmt.Errorf("pre compile: %w", err)
					}
				}
				cmd := exec.Command(
					"go", "build",
					"-o", bin,
					"-ldflags", strings.Join(p.Info.Flags.Lookup(target).Linker, " "),
					"-gcflags", strings.Join(p.Info.Flags.Lookup(target).Compiler, " "),
					p.Info.Pkg,
				)
				cmd.Dir = sandbox
				cmd.Env = append(cmd.Env, fmt.Sprintf("GOOS=%s", platform))
				cmd.Env = append(cmd.Env, fmt.Sprintf("GOARCH=%s", arch))
				cmd.Env = append(cmd.Env, os.Environ()...)
				fmt.Printf("%s\n", cmd)
				if out, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("%s%w", func() string {
						if len(out) == 0 {
							return ""
						}
						return fmt.Sprintf("%s: ", strings.TrimSpace(string(out)))
					}(), err)
				}
				data, err := ioutil.ReadFile(bin)
				if err != nil {
					return fmt.Errorf("reading binary: %w", err)
				}
				p.Artifacts = append(p.Artifacts, Artifact{
					Binary: bytes.NewBuffer(data),
					Target: target,
				})
				return nil
			}(); err != nil {
				errs <- fmt.Errorf("%s: %w", target.String(), err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	if err := new(util.MultiError).FromChan(errs); !err.IsEmpty() {
		return err
	}
	return nil
}

// Output returns the output directory to place artifacts into.
func (p Packer) Output() string {
	if p.Info != nil {
		if p.Info.Dist == "" {
			p.Info.Dist = "dist"
		}
		return filepath.Join(p.Info.Root, p.Info.Dist)
	}
	return "dist"
}
