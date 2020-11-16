package gopack

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git.sr.ht/~jackmordaunt/gopack/ico"
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
// TODO: Specify data in a common format (eg, yml).
// Grab common data author, description, etc and allow for custom data.
// Then, generate the platform specific files or skip that and directly embed
// the meta data (eg windows manifest).
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
		// TODO: flatpack, snap, appimage?
	}
}

// Artifact associates a path to a binary with the platform it's intended for.
type Artifact struct {
	Binary   string
	Platform Platform
}

// Platform identifier for the platforms we care about.
type Platform uint8

const (
	Windows Platform = iota
	Darwin
	Linux
)

func (p Platform) String() string {
	switch p {
	case Windows:
		return "windows"
	case Darwin:
		return "darwin"
	case Linux:
		return "linux"
	}
	return ""
}

// Pack the binaries into native formats.
func (p Packer) Pack() error {
	if p.Info != nil {
		if err := p.Compile(); err != nil {
			return fmt.Errorf("compiling: %w", err)
		}
		if err := p.MetaData.Load(p.Info.Root); err != nil {
			return fmt.Errorf("loading metadata: %w", err)
		}
	}
	if len(p.Artifacts) == 0 {
		return fmt.Errorf("no artifacts to pack")
	}
	if p.MetaData.Icon == "" {
		fmt.Printf("warning: icon not found (icon.png)\n")
	}
	// TODO: parallelize builds.
	for _, artifact := range p.Artifacts {
		dir := filepath.Join(p.Output(), artifact.Platform.String())
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
			// if err := bundleLinux(
			// 	dir,
			// 	artifact.Binary,
			// 	icon,
			// ); err != nil {
			// 	return fmt.Errorf("bundling linux: %w", err)
			// }
		}
	}
	return nil
}

// Compile the Go project.
// Requires Go toolchain to be installed.
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
			return fmt.Errorf("package %q not found", pkg)
		}
	} else {
		pkg = root
	}
	for _, target := range []Platform{Windows, Darwin, Linux} {
		bin := filepath.Join(output, target.String(), filepath.Base(pkg))
		if target == Windows {
			bin += ".exe"
		}
		cmd := exec.Command(
			"go", "build",
			"-ldflags", strings.Join(p.Info.Flags.Linker, " "),
			"-gcflags", strings.Join(p.Info.Flags.Compiler, " "),
			"-o", bin,
			pkg,
		)
		cmd.Dir = root
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOOS=%s", target))
		cmd.Env = append(cmd.Env, "GOARCH=amd64")
		cmd.Env = append(cmd.Env, os.Environ()...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%q for %q: %w: %s", pkg, target, err, string(out))
		}
		p.Artifacts = append(p.Artifacts, Artifact{
			Binary:   bin,
			Platform: target,
		})
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
			return fmt.Errorf("converting icon to .icns: %w", err)
		}
		md.Darwin.ICNS = icns
	}
	if md.Windows.ICO == "" && md.Icon != "" {
		ico := filepath.Join(os.TempDir(), "gopack", "icon.ico")
		if err := convertIcon(md.Icon, ico); err != nil {
			return fmt.Errorf("converting icon to .ico: %w", err)
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
		// TODO: pattern matching search?
		manifest, err := finder.Find("manifest")
		if err != nil {
			return fmt.Errorf("manifest: %w", err)
		}
		md.Windows.Manifest = manifest
	}
	// TODO: load linux meta data.
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
