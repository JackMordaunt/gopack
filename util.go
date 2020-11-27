package gopack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// cp copies src file to destination.
// If destination is a directory, the file will be copied into it.
// If destination doesn't exist it will be created as a file.
// If destination is a file an error will be returned.
func cp(src, dst string) error {
	if src == "" || dst == "" {
		return nil
	}
	var err error
	src, err = filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	srcf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer srcf.Close()
	_, err = os.Stat(filepath.Dir(dst))
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
			return fmt.Errorf("preparing %q: %w", filepath.Dir(dst), err)
		}
	}
	dstf, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("creating %q: %w", dst, err)
	}
	defer dstf.Close()
	if _, err := io.Copy(dstf, srcf); err != nil {
		return fmt.Errorf("copying data: %w", err)
	}
	return nil
}

// Finder finds files by name.
type Finder struct {
	Root  string
	IsDir bool
	// Rel indicates to return a relative path instead of an absolute path.
	Rel bool
}

// Find the first file with the given name recursively from the root.
//
// Returns the absolute path to the file, or an error if something went wrong
// while walking the file system.
//
// If path is empty then no file was found.
func (f Finder) Find(name string) (string, error) {
	var found string
	err := filepath.Walk(f.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() == f.IsDir && info.Name() == name {
			if f.Rel {
				path = filepath.Clean(strings.TrimPrefix(path, f.Root))
			}
			found = path
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking: %w", err)
	}
	if found == "" {
		return "", nil
	}
	if !f.Rel {
		found, err = filepath.Abs(found)
		if err != nil {
			return "", fmt.Errorf("resolving absolute path: %w", err)
		}
	}
	return found, nil
}
