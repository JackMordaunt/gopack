package rsrc

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/akavel/rsrc/binutil"
	"github.com/akavel/rsrc/ico"
)

// Embed icons into an output .syso file for consumption by the Go linker.
func Embed(output string, arch Arch, icons ...string) error {
	nextID := idGenerator()
	// Todo: if arch must be set after NewRSRC, then set it IN NewRSRC.
	coffData := NewRSRC(arch)
	for _, icon := range icons {
		if err := addIcon(coffData, icon, nextID); err != nil {
			return fmt.Errorf("adding icon: %w", err)
		}
	}
	coffData.Freeze()
	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()
	return write(coffData, out)
}

// on storing icons, see: http://blogs.msdn.com/b/oldnewthing/archive/2012/07/20/10331787.aspx
type iconGroup struct {
	ico.ICONDIR
	Entries []iconEntry
}

func (group iconGroup) Size() int64 {
	return int64(binary.Size(group.ICONDIR) + len(group.Entries)*binary.Size(group.Entries[0]))
}

type iconEntry struct {
	ico.IconDirEntryCommon
	Id uint16
}

func addIcon(out *Coff, icon string, newid func() uint16) error {
	f, err := os.Open(icon)
	if err != nil {
		return err
	}
	defer f.Close()
	icons, err := ico.DecodeHeaders(f)
	if err != nil {
		return fmt.Errorf("decoding header: %w", err)
	}
	if len(icons) > 0 {
		// RT_ICONs
		group := iconGroup{ICONDIR: ico.ICONDIR{
			Reserved: 0, // magic num.
			Type:     1, // magic num.
			Count:    uint16(len(icons)),
		}}
		for _, icon := range icons {
			id := newid()
			out.AddResource(RT_ICON, id, io.NewSectionReader(f, int64(icon.ImageOffset), int64(icon.BytesInRes)))
			group.Entries = append(group.Entries, iconEntry{icon.IconDirEntryCommon, id})
		}
		out.AddResource(RT_GROUP_ICON, newid(), group)
	}
	return nil
}

func write(coff *Coff, out io.Writer) error {
	w := binutil.Writer{W: out}
	if err := binutil.Walk(coff, func(v reflect.Value, path string) error {
		if binutil.Plain(v.Kind()) {
			w.WriteLE(v.Interface())
			return nil
		}
		vv, ok := v.Interface().(binutil.SizedReader)
		if ok {
			w.WriteFromSized(vv)
			return binutil.WALK_SKIP
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walking coff: %w", err)
	}
	if w.Err != nil {
		return fmt.Errorf("writing output: %s", w.Err)
	}
	return nil
}

func idGenerator() func() uint16 {
	id := uint16(0)
	return func() uint16 {
		id++
		return id
	}
}
