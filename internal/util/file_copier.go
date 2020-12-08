package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Copier copies files from one path to another.
// Not recursive by default.
// Optional list of patterns to ignore via `strings.Contains`.
type Copier struct {
	Recursive bool
	Ignore    []string
}

// Copy files `from` into `to`.
func (c Copier) Copy(from, to string) error {
	if !c.Recursive {
		entries, err := ioutil.ReadDir(from)
		if err != nil {
			return fmt.Errorf("reading dir: %w", err)
		}
		for _, entry := range entries {
			from = filepath.Join(from, entry.Name())
			to = filepath.Join(to, entry.Name())
			if !entry.IsDir() && !c.ignore(from) {
				if err := cp(from, to); err != nil {
					return fmt.Errorf("copying file: %w", err)
				}
			}
		}
		return nil
	}
	if err := filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if c.ignore(path) {
			return nil
		}
		target := filepath.Join(to, strings.TrimPrefix(path, from))
		if info.IsDir() {
			_ = os.MkdirAll(target, 0777)
		} else {
			if err := cp(path, target); err != nil {
				return fmt.Errorf("copying file: %w", err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("copying files: %w", err)
	}
	return nil
}

// ignore path if it contains any of the ignore patterns.
func (c Copier) ignore(path string) bool {
	for _, pattern := range c.Ignore {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

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
	if _, err = os.Stat(filepath.Dir(dst)); os.IsNotExist(err) {
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
