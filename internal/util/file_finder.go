package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Finder finds files by name.
type Finder struct {
	// Root folder to start search from.
	Root string
	// IsDir if we are looking for a directory.
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
	} else {
		if found[0] != '.' {
			found = "." + found
		}
	}
	return found, nil
}
