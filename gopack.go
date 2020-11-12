package gopack

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jackmordaunt/icns"
)

// Pack the program rooted at path.
// The final package will be named name if specified.
// If pkg is empty, root will be treated as the package.
// Recursively searches for "icon.png" to use as the icon.
func Pack(root, pkg, name string) error {
	if pkg != "" {
		var err error
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
	dist := filepath.Join(root, "dist")
	artifacts, err := build(root, pkg, dist)
	if err != nil {
		return fmt.Errorf("building %q: %w", pkg, err)
	}
	icon, err := Finder{Root: root}.Find("icon.png")
	if err != nil {
		return fmt.Errorf("finding icon: %w", err)
	}
	if icon == "" {
		fmt.Printf("warning: icon not found (icon.png)")
	}
	for _, artifact := range artifacts {
		dir := filepath.Join(dist, artifact.Platform.String())
		switch artifact.Platform {
		case Darwin:
			plist, err := Finder{Root: root}.Find("Info.plist")
			if err != nil {
				return fmt.Errorf("finding Info.plist: %w", err)
			}
			if err := bundleMacOS(
				dir,
				artifact.Binary,
				icon,
				plist,
			); err != nil {
				return fmt.Errorf("bundling macos: %w", err)
			}
		case Windows:
			if err := bundleWindows(
				dir,
				artifact.Binary,
				icon,
				"",
			); err != nil {
				return fmt.Errorf("bundling windows: %w", err)
			}
		case Linux:
			if err := bundleLinux(
				dir,
				artifact.Binary,
				icon,
			); err != nil {
				return fmt.Errorf("bundling linux: %w", err)
			}
		}
	}
	return nil
}

// convertIcon converts the source png to icon and returns a path to it.
// TODO: handle .ico
func convertIcon(src, dst string) error {
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
		return fmt.Errorf("encoding icns: %w", err)
	}
	return nil
}

// Artifact associates a path to a binary with the platform it's intended for.
type Artifact struct {
	Binary   string
	Platform Platform
}

// Platform identifier for the platforms we care about.
type Platform int

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

// build the Go program rooted at path for each target and return a list of
// produced artifacts.
func build(root, path, output string) ([]Artifact, error) {
	var artifacts []Artifact
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute path: %w", err)
	}
	for _, target := range []Platform{Windows, Darwin, Linux} {
		bin := filepath.Join(output, target.String(), filepath.Base(path))
		if target == Windows {
			bin += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", bin, path)
		cmd.Dir = root
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOOS=%s", target))
		cmd.Env = append(cmd.Env, "GOARCH=amd64")
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("compiling %q for %q: %w: %s", path, target, err, string(out))
		}
		artifacts = append(artifacts, Artifact{
			Binary:   bin,
			Platform: target,
		})
	}
	return artifacts, nil
}
