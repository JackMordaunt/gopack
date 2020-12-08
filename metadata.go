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

// Load metadata using defaults if not specified.
// For icons and other resources this means buffering the files in memory.
func (md *MetaData) Load(root string) error {
	finder := util.Finder{Root: root}
	if md.Icon == nil {
		icon, err := finder.Find("icon.png")
		if err != nil {
			return fmt.Errorf("finding icon: %w", err)
		}
		if icon != "" {
			fmt.Printf("icon: %v\n", icon)
			iconby, err := ioutil.ReadFile(icon)
			if err != nil {
				return fmt.Errorf("reading icon: %w", err)
			}
			img, err := png.Decode(bytes.NewBuffer(iconby))
			if err != nil {
				return fmt.Errorf("decoding icon: %w", err)
			}
			md.Icon = img
		} else {
			fmt.Printf("warning: icon not found\n")
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
	// TODO load linux meta data.
	return nil
}
