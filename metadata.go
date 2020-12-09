package gopack

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"

	"git.sr.ht/~jackmordaunt/gopack/internal/ico"
	"git.sr.ht/~jackmordaunt/gopack/internal/util"
	"github.com/jackmordaunt/icns"
	"github.com/markbates/pkger"
)

// MetaData pulls together all platform specific metadata require to create a
// bundle.
//
// Note: we don't care about the structure of the platform files, hence they are
// represented as blobs.
//
// @Todo Generate plist, manifest, et al from common data (author, version,
// access requirements, etc).
// This would allow streamlined use, as the user wouldn't need to individually
// create such files; which is very much part of the mission statement of this
// software.
// If the user has already created those files for whatever reason, we can
// fallback to copying them.
type MetaData struct {
	// Icon contains the image data for the icon.
	Icon   image.Image
	Darwin struct {
		// ICNS contains icon encoded as ICNS.
		ICNS io.Reader
		// Plist contains the Info.plist metadata file.
		Plist io.Reader
	}
	Windows struct {
		// ICO contains icon encoded as ICO.
		ICO io.Reader
		// Manifest contains the windows manifest metadata file.
		Manifest io.Reader
	}
	Linux struct {
		// @Todo linux metadata stuff. flatpack, snap, appimage.
	}
}

// @OnHold this requires embed fs in next Go release 1.16 in feb 2021.
//go:embed default.png
var goIcon []byte

// Load metadata using defaults if not specified.
// For icons and other resources this means buffering the files in memory.
func (md *MetaData) Load(root string) error {
	finder := util.Finder{Root: root}
	if md.Icon == nil {
		icon, err := finder.Find("icon.png")
		if err != nil {
			return fmt.Errorf("finding icon: %w", err)
		}
		var iconby []byte
		if icon != "" {
			fmt.Printf("icon: %v\n", icon)
			iconby, err = ioutil.ReadFile(icon)
			if err != nil {
				return fmt.Errorf("reading icon: %w", err)
			}
		} else {
			// @Enhance do we want to handle errors at this level?
			// It seems that if we are returning errors then we ought not be
			// directly printing warnings to stdout, since to return errors is
			// to indicate that there is a more appropriate context to handle
			// them.
			// Given the scope of this program, the consequences are small either
			// way.
			fmt.Printf("warning: icon not found; using default\n")
			if len(goIcon) > 0 {
				iconby = goIcon
			} else {
				if err := func() error {
					// Fallback to another strategy of providing a default icon.
					defaultf, err := pkger.Open("/default.png")
					if err != nil {
						return fmt.Errorf("default icon not compiled in, see https://github.com/markbates/pkger")
					}
					defer defaultf.Close()
					iconby, err = ioutil.ReadAll(defaultf)
					if err != nil {
						return fmt.Errorf("reading default icon: %w", err)
					}
					return nil
				}(); err != nil {
					fmt.Printf("warning: %v\n", err)
				}
			}
		}
		if len(iconby) > 0 {
			img, err := png.Decode(bytes.NewBuffer(iconby))
			if err != nil {
				return fmt.Errorf("decoding icon: %w", err)
			}
			md.Icon = img
		}
	}
	if md.Darwin.ICNS == nil && md.Icon != nil {
		buffer := bytes.NewBuffer(nil)
		if err := icns.Encode(buffer, md.Icon); err != nil {
			return fmt.Errorf("icns: converting icon: %w", err)
		}
		md.Darwin.ICNS = util.NewCopyBuffer(buffer.Bytes())
	}
	if md.Darwin.Plist == nil {
		plist, err := finder.Find("Info.plist")
		if err != nil {
			return fmt.Errorf("Info.plist: %w", err)
		}
		if plist != "" {
			by, err := ioutil.ReadFile(plist)
			if err != nil {
				return fmt.Errorf("reading %s: %w", plist, err)
			}
			md.Darwin.Plist = util.NewCopyBuffer(by)
		}
	}
	if md.Windows.ICO == nil && md.Icon != nil {
		buffer := bytes.NewBuffer(nil)
		if err := ico.FromPNG(buffer, md.Icon); err != nil {
			return fmt.Errorf("ico: converting icon: %w", err)
		}
		md.Windows.ICO = util.NewCopyBuffer(buffer.Bytes())
	}
	if md.Windows.Manifest == nil {
		manifest, err := finder.Find("manifest")
		if err != nil {
			return fmt.Errorf("manifest: %w", err)
		}
		if manifest != "" {
			by, err := ioutil.ReadFile(manifest)
			if err != nil {
				return fmt.Errorf("reading %s: %w", manifest, err)
			}
			md.Windows.Manifest = util.NewCopyBuffer(by)
		}
	}
	return nil
}
