// gopack is a tool for creating OS specific installable packages for Windows,
// macOS and Linux.
package main

import (
	"fmt"
	"os"

	"git.sr.ht/~jackmordaunt/gopack"
)

func main() {
	if err := func() error {
		var (
			root string
			pkg  string
			name string
		)
		if len(os.Args) <= 1 {
			return fmt.Errorf("specify root of project")
		}
		root = os.Args[1]
		if len(os.Args) > 2 {
			pkg = os.Args[2]
		}
		if len(os.Args) > 3 {
			name = os.Args[3]
		}
		packer := gopack.Packer{
			Info: &gopack.ProjectInfo{
				Root: root,
				Pkg:  pkg,
				Name: name,
			},
		}
		if err := packer.Pack(); err != nil {
			return fmt.Errorf("bundling: %w", err)
		}
		return nil
	}(); err != nil {
		fmt.Printf("error: %v", err)
	}
}
