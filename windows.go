package gopack

import (
	"debug/pe"
	"fmt"
	"os"
)

// # Cursor and Icon resources (PE format) .rsrc section.
//
// The system handles each icon and cursor as a single file. However, these are
// stored in .res files and in executable files as a group of icon resources or
// a group of cursor resources. The file formats of icon and cursor resources are
// similar. In the .res file a resource group header follows all of the individual
// icon or cursor group components.

// The format of each icon component closely resembles the format of the .ico
// file. Each icon image is stored in a BITMAPINFO structure followed by the color
// device-independent bitmap (DIB) bits of the icon's XOR mask. The monochrome DIB
// bits of the icon's AND mask follow the color DIB bits.

// The format of each cursor component resembles the format of the .cur file.
// Each cursor image is stored in a BITMAPINFO structure followed by the monochrom
// DIB bits of the cursor's XOR mask, and then by the monochrome DIB bits of the
// cursor's AND mask. Note that there is a difference in the bitmaps of the two
// resources: Unlike icons, cursor XOR masks do not have color DIB bits. Although
// the bitmaps of the cursor masks are monochrome and do not have DIB headers or
// color tables, the bits are still in DIB format with respect to alignment and
// direction. Another significant difference between cursors and icons is that
// cursors have a hotspot and icons do not.

// The group header for both icon and cursor resources consists of a NEWHEADER
// structure plus one or more RESDIR structures. There is one RESDIR structure for
// each icon or cursor. The group header contains the information an application
// needs to select the correct icon or cursor to display. Both the group header
// and the data that repeats for each icon or cursor in the group have a fixed
// length. This allows the application to randomly access the information.

func bundleWindows(dest, binary, icon, manifest string) error {
	// load binary via debug/pe
	// append data to resource section (.rsrc), creating it if it doesn't exist
	// what should the data be?
	pefile, err := pe.Open(binary)
	if err != nil {
		return fmt.Errorf("opening PE binary: %w", err)
	}
	iconfile, err := os.Open(icon)
	if err != nil {
		return fmt.Errorf("opening icon: %w", err)
	}
	defer iconfile.Close()
	pefile.Sections = append(pefile.Sections, &pe.Section{
		SectionHeader: pe.SectionHeader{
			Name: ".rsrc",
		},
		ReaderAt: iconfile,
	})

	// Add icon data to PE .rsrc section: how?
	// Write PE back out: how?
	return nil
}
