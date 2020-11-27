package gopack

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"git.sr.ht/~jackmordaunt/gopack/ico"
	"github.com/akavel/rsrc/rsrc"
	"github.com/jackmordaunt/icns"
)

// Packer packs a project into native artifacts.
type Packer struct {
	Info      *ProjectInfo
	MetaData  MetaData
	Artifacts []Artifact
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
	Flags struct {
		// Compiler flags.
		// go tool compile
		Compiler []string
		// Linker flags.
		// go tool link
		Linker []string
	}
}

// MetaData contains paths to meta files such as icon and manifests.
//
// TODO Specify data in a common format (eg, yml).
// Grab common data author, description, etc and allow for custom data.
// Then, generate the platform specific files or skip that and directly embed
// the meta data (eg windows manifest).
//
type MetaData struct {
	Icon   string
	Darwin struct {
		ICNS  string
		Plist string
		// Note: anything else?
	}
	Windows struct {
		ICO      string
		Manifest string
	}
	Linux struct {
		// TODO flatpack, snap, appimage?
	}
}

// Artifact associates a path to a binary with the platform it's intended for.
type Artifact struct {
	Binary string
	Target
}

type Target struct {
	Platform     Platform
	Architecture Architecture
}

// Targets is a static list of supported targets as a subset of output by
// `go tool dist list`.
var Targets = []Target{
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
		if p.MetaData.Icon == "" {
			fmt.Printf("warning: icon not found (icon.png)")
		}
		if err := p.Compile(); err != nil {
			return fmt.Errorf("compiling: %w", err)
		}
	}
	if len(p.Artifacts) == 0 {
		return fmt.Errorf("no artifacts to pack")
	}
	if p.MetaData.Icon == "" {
		fmt.Printf("warning: icon not found (icon.png)\n")
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
					artifact.Binary,
					p.MetaData.Darwin.ICNS,
					p.MetaData.Darwin.Plist,
					p.Info.Name,
				); err != nil {
					fmt.Printf("bundling macos: %s\n", err)
				}
			case Windows:
				if err := bundleWindows(
					dir,
					artifact.Binary,
					p.MetaData.Windows.ICO,
					p.MetaData.Windows.Manifest,
				); err != nil {
					fmt.Printf("bundling windows: %s\n", err)
				}
			case Linux:
				if err := bundleLinux(
					dir,
					artifact.Binary,
					"",
				); err != nil {
					fmt.Printf("bundling linux: %s\n", err)
				}
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
		root   = p.Info.Root
		pkg    = p.Info.Pkg
		output = p.Output()
		err    error
	)
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolving current working directory: %w", err)
		}
	}
	if r, err := filepath.Abs(root); err != nil {
		return fmt.Errorf("resolving root: %w", err)
	} else {
		root = r
	}
	if pkg != "" {
		pkg, err = Finder{Root: root, IsDir: true}.Find(pkg)
		if err != nil {
			return fmt.Errorf("finding package: %w", err)
		}
		if pkg == "" {
			return fmt.Errorf("package %q not found", p.Info.Pkg)
		}
	} else {
		pkg = root
	}
	wg := &sync.WaitGroup{}
	errs := make(chan error, len(Targets))
	for _, target := range Targets {
		target := target
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := func() error {
				var (
					platform = target.Platform
					arch     = target.Architecture
					sandbox  = filepath.Join(os.TempDir(), "gopack", target.String())
					bin      = fmt.Sprintf(
						"%s%s",
						filepath.Join(
							output,
							fmt.Sprintf("%s_%s", platform.String(), arch.String()),
							filepath.Base(pkg)),
						target.Ext())
				)

				// ENHANCE can we make this more semantic?
				if err := (Copier{Recursive: true, Ignore: []string{"dist"}}).Copy(root, sandbox); err != nil {
					return fmt.Errorf("creating sandbox: %w", err)
				}

				// TODO reify "precompile" step to capture this edge cases.
				// ENHANCE editing PE binary data inline, without creating a .syso file would be
				// more robust.
				if platform == Windows {
					if p.MetaData.Windows.ICO != "" {
						if err := rsrc.Embed("rsrc.syso", "amd64", "", p.MetaData.Windows.ICO); err != nil {
							return fmt.Errorf("windows: creating icon resource: %w", err)
						}
					}
					defer os.Remove("rsrc.syso")
				}
				// TODO target-specific linker and compiler flags.
				cmd := exec.Command(
					"go", "build",
					"-ldflags", strings.Join(p.Info.Flags.Linker, " "),
					"-gcflags", strings.Join(p.Info.Flags.Compiler, " "),
					"-o", bin,
					pkg,
				)
				cmd.Dir = sandbox
				cmd.Env = append(cmd.Env, fmt.Sprintf("GOOS=%s", platform))
				cmd.Env = append(cmd.Env, fmt.Sprintf("GOARCH=%s", arch))
				cmd.Env = append(cmd.Env, os.Environ()...)
				if out, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("%q for %q: %w: %s", pkg, platform, err, string(out))
				}
				p.Artifacts = append(p.Artifacts, Artifact{
					Binary: bin,
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
	if err := new(MultiError).FromChan(errs); !err.IsEmpty() {
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

// Load metadata using defaults if not specified.
func (md *MetaData) Load(root string) error {
	finder := Finder{Root: root}
	if md.Icon == "" {
		icon, err := finder.Find("icon.png")
		if err != nil {
			return fmt.Errorf("icon: %w", err)
		}
		if icon != "" {
			md.Icon = icon
		}
	}
	if md.Darwin.ICNS == "" && md.Icon != "" {
		icns := filepath.Join(os.TempDir(), "gopack", "icon.icns")
		if err := convertIcon(md.Icon, icns); err != nil {
			return err
		}
		md.Darwin.ICNS = icns
	}
	if md.Windows.ICO == "" && md.Icon != "" {
		ico := filepath.Join(os.TempDir(), "gopack", "icon.ico")
		if err := convertIcon(md.Icon, ico); err != nil {
			return err
		}
		md.Windows.ICO = ico
	}
	if md.Darwin.Plist == "" {
		plist, err := finder.Find("Info.plist")
		if err != nil {
			return fmt.Errorf("Info.plist: %w", err)
		}
		md.Darwin.Plist = plist
	}
	if md.Windows.Manifest == "" {
		// TODO pattern matching search?
		manifest, err := finder.Find("manifest")
		if err != nil {
			return fmt.Errorf("manifest: %w", err)
		}
		md.Windows.Manifest = manifest
	}
	// TODO load linux meta data.
	return nil
}

// convertIcon converts the source png ether icns or ico based on dst file
// extension {.icns,ico}.
func convertIcon(src, dst string) error {
	switch filepath.Ext(dst) {
	case ".icns":
		if err := func() error {
			srcf, err := os.Open(src)
			if err != nil {
				return fmt.Errorf("opening source file: %w", err)
			}
			defer srcf.Close()
			img, err := png.Decode(srcf)
			if err != nil {
				return fmt.Errorf("decoding source png: %w", err)
			}
			_ = os.MkdirAll(filepath.Dir(dst), 0777)
			dstf, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				return fmt.Errorf("opening destination file: %w", err)
			}
			defer dstf.Close()
			if err := icns.Encode(dstf, img); err != nil {
				return fmt.Errorf("encoding: %w", err)
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("converting to .icns: %w", err)
		}
	case ".ico":
		if err := ico.FromPNG(src, dst); err != nil {
			return fmt.Errorf("converting to .ico: %w", err)
		}
	default:
		return fmt.Errorf("cannot handle %q", filepath.Ext(dst))
	}
	return nil
}

// MultiError combines a number of errors into a single error value.
type MultiError []error

func (me *MultiError) FromChan(errs chan error) *MultiError {
	for err := range errs {
		(*me) = append((*me), err)
	}
	return me
}

func (me MultiError) IsEmpty() bool {
	return len(me) == 0
}

func (me MultiError) Error() string {
	if len(me) == 1 {
		return me[0].Error()
	}
	var b strings.Builder
	b.WriteString("[\n")
	for ii, err := range me {
		fmt.Fprintf(&b, "\t%d: %s\n", ii+1, err)
	}
	b.WriteString("]\n")
	return b.String()
}

type Copier struct {
	Recursive bool
	Ignore    []string
}

func (c Copier) Copy(from, to string) error {
	return nil
}
