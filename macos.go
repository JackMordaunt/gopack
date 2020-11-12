package gopack

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// bundleMacOS creates a macOS .app bundleMacOS on disk rooted at dest.
// All paramaters are filepaths.
// NB: Will clobber destination if it is a directory, or error if it is a file.
func bundleMacOS(dest, binary, icon, plist string) error {
	var (
		contents  = filepath.Join(dest, "Contents")
		macos     = filepath.Join(contents, "MacOS")
		resources = filepath.Join(contents, "Resources")
	)
	m, err := os.Stat(dest)
	if os.IsNotExist(err) || m.IsDir() {
		os.RemoveAll(dest)
		if err := os.MkdirAll(dest, 0777); err != nil {
			return fmt.Errorf("preparing destination: %w", err)
		}
	} else if !m.IsDir() {
		return fmt.Errorf("destination %q: not a directory", dest)
	}
	if err := os.MkdirAll(macos, 0777); err != nil {
		return fmt.Errorf("preparing directory: %w", err)
	}
	if err := os.MkdirAll(resources, 0777); err != nil {
		return fmt.Errorf("preparing directory: %w", err)
	}
	if err := cp(binary, filepath.Join(macos, "kanban")); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}
	if err := cp(plist, filepath.Join(contents, "Info.plist")); err != nil {
		return fmt.Errorf("copying plist: %w", err)
	}
	if err := convertIcon(icon, filepath.Join(resources, "kanban.icns")); err != nil {
		return fmt.Errorf("converting icon to icns: %w", err)
	}
	switch runtime.GOOS {
	case "linux":
		if err := run(
			"genisoimage",
			"-V", "Kanban",
			"-D",
			"-R",
			"-apple",
			"-no-pad",
			"-o", "Kanban.dmg",
			filepath.Dir(dest),
		); err != nil {
			return fmt.Errorf("genisoimage: %w", err)
		}
	case "darwin":
		// dmg: | $(DMG_NAME)
		// $(DMG_NAME): $(APP_NAME)
		// 	@echo "Packing disk image..."
		// 	@ln -sf /Applications $(DMG_DIR)/Applications
		// 	@hdiutil create $(DMG_DIR)/$(DMG_NAME) \
		// 		-volname "Kanban" \
		// 		-fs HFS+ \
		// 		-srcfolder $(APP_DIR) \
		// 		-ov -format UDZO
		// 	@echo "Packed '$@' in '$(APP_DIR)'"
		if err := run(
			"hdiutil",
			"create",
			filepath.Join(filepath.Dir(dest), "Kanban.dmg"),
			"-volname", "Kanban",
			"-fs", "HFS+",
			"-srcfolder", dest,
			"-ov", "-format", "UDZO",
		); err != nil {
			return fmt.Errorf("hdiutil: %w", err)
		}
	case "windows":
		return fmt.Errorf("cannot create dmg on windows yet")
	default:
		return fmt.Errorf("cannot create dmg on %q", runtime.GOOS)
	}
	return nil
}
