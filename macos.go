package gopack

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kdomanski/iso9660"
)

// bundleMacOS creates a macOS .app bundleMacOS on disk rooted at dest.
// All paramaters are filepaths.
// NB: Will clobber destination if it is a directory, or error if it is a file.
func bundleMacOS(dest, binary, icon, plist, name string) error {
	var (
		app       = filepath.Join(dest, fmt.Sprintf("%s.app", name))
		contents  = filepath.Join(app, "Contents")
		macos     = filepath.Join(contents, "MacOS")
		resources = filepath.Join(contents, "Resources")
	)
	m, err := os.Stat(app)
	if os.IsNotExist(err) || m.IsDir() {
		if err := os.MkdirAll(app, 0777); err != nil {
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
	if err := cp(binary, filepath.Join(macos, filepath.Base(binary))); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}
	// TODO generate Info.plist from metadata.
	if err := cp(plist, filepath.Join(contents, "Info.plist")); err != nil {
		return fmt.Errorf("copying plist: %w", err)
	}
	if err := cp(icon, filepath.Join(resources, filepath.Base(icon))); err != nil {
		return fmt.Errorf("copying plist: %w", err)
	}
	if err := dmg(app, filepath.Dir(app), name); err != nil {
		return fmt.Errorf("creating disk image: %w", err)
	}
	return nil
}

// DmgMeta is the DMG metadata that appears at the end of the disk image.
type DmgMeta struct {
	Signature             [4]byte // magic 'koly'
	Version               uint32  // 4 (as of 2013)
	HeaderSize            uint32  // sizeof(this) =  512 (as of 2013)
	Flags                 uint32
	RunningDataForkOffset uint64
	DataForkOffset        uint64 // usually 0, beginning of file
	DataForkLength        uint64
	RsrcForkOffset        uint64 // resource fork offset and length
	RsrcForkLength        uint64
	SegmentNumber         uint32 // Usually 1, can be 0
	SegmentCount          uint32 // Usually 1, can be 0
	SegmentID             [128]byte
	DataChecksumType      uint32 // Data fork checksum
	DataChecksumSize      uint32
	DataChecksum          [32]uint32
	XMLOffset             uint64 // Position of XML property list in file
	XMLLength             uint64
	Reserved1             [120]byte
	ChecksumType          uint32 // Master checksum
	ChecksumSize          uint32
	Checksum              [32]uint32
	ImageVariant          uint32 // Unknown, commonly 1
	SectorCount           uint64
	reserved2             uint32
	reserved3             uint32
	reserved4             uint32
}

func (md DmgMeta) Bytes() []byte {
	return nil
}

// dmg creates an iso disk image of src and places it into dst.
//
// Note: DMG metadata is currently ignored.
func dmg(src, dst, id string) error {
	if id == "" {
		id = "unspecified"
	}
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s not a directory", src)
	}
	writer, err := iso9660.NewWriter()
	if err != nil {
		return fmt.Errorf("initialising writer: %w", err)
	}
	defer writer.Cleanup()
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", path, err)
		}
		defer f.Close()
		err = writer.AddFile(f, strings.Trim(path, filepath.Dir(src)))
		if err != nil {
			return fmt.Errorf("adding file %s: %w", path, err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("adding files from %s to image: %w", src, err)
	}
	b := bytes.NewBuffer(nil)
	err = writer.WriteTo(b, id)
	if err != nil {
		return fmt.Errorf("writing ISO image: %w", err)
	}
	if _, err := b.Write(DmgMeta{}.Bytes()); err != nil {
		return fmt.Errorf("writing dmg metadata")
	}
	outputFile, err := os.OpenFile(
		filepath.Join(dst, id+".dmg"),
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0644,
	)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	if _, err := io.Copy(outputFile, b); err != nil {
		return fmt.Errorf("writing to output file: %w", err)
	}
	err = outputFile.Close()
	if err != nil {
		return fmt.Errorf("closing output file: %w", err)
	}
	return nil
}
