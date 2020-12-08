package gopack

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// bundleWindows bundles a single binary application for windows.
//
// For now that just means copying it to some destination, since icon resources
// are compiled in prior.
func bundleWindows(dest string, binary io.Reader) error {
	by, err := ioutil.ReadAll(binary)
	if err != nil {
		return fmt.Errorf("buffering binary: %w", err)
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0777)
	if err := ioutil.WriteFile(dest, by, 0777); err != nil {
		return fmt.Errorf("writing binary to file: %w", err)
	}
	return nil
}
