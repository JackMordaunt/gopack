package gopack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func bundleLinux(dest, binary, icon string) error {
	by, err := ioutil.ReadFile(binary)
	if err != nil {
		return fmt.Errorf("reading binary file: %w", err)
	}
	path := filepath.Join(dest, filepath.Base(binary))
	_ = os.MkdirAll(filepath.Dir(path), 0777)
	if err := ioutil.WriteFile(path, by, 0777); err != nil {
		return fmt.Errorf("writing binary file: %w", err)
	}
	return nil
}
