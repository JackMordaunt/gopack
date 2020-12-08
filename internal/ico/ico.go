// ico format encoding.
// Modified from https://github.com/wailsapp/wails project.
package ico

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"

	"golang.org/x/image/draw"
)

type Container struct {
	Header Descriptor
	Data   []byte
}

type Header struct {
	_          uint16
	imageType  uint16
	imageCount uint16
}

type Descriptor struct {
	width  uint8
	height uint8
	_      uint8 // colors
	_      uint8
	planes uint16
	bpp    uint16
	size   uint32
	offset uint32
}

// FromPNG generates an ico from a source png and encodes it into an arbitrary
// reader.
func FromPNG(dst io.Writer, src image.Image) error {
	var (
		sizes = []int{256, 128, 64, 48, 32, 16}
		icons = []Container{}
	)
	for _, size := range sizes {
		var (
			rect   = image.Rect(0, 0, int(size), int(size))
			raw    = image.NewRGBA(rect)
			buffer = bytes.NewBuffer(nil)
			scale  = draw.CatmullRom
		)
		scale.Scale(raw, rect, src, src.Bounds(), draw.Over, nil)
		if err := png.Encode(buffer, raw); err != nil {
			return fmt.Errorf("encoding png data into ico: %w", err)
		}
		imgSize := size
		if imgSize >= 256 {
			imgSize = 0
		}
		data := buffer.Bytes()
		icons = append(icons, Container{
			Header: Descriptor{
				width:  uint8(imgSize),
				height: uint8(imgSize),
				planes: 1,
				bpp:    32,
				size:   uint32(len(data)),
			},
			Data: data,
		})
	}
	if err := binary.Write(dst, binary.LittleEndian, Header{
		imageType:  1,
		imageCount: uint16(len(sizes)),
	}); err != nil {
		return fmt.Errorf("writing ico header: %w", err)
	}
	offset := uint32(6 + 16*len(sizes))
	for _, icon := range icons {
		icon.Header.offset = offset
		if err := binary.Write(dst, binary.LittleEndian, icon.Header); err != nil {
			return fmt.Errorf("writing icon headers: %w", err)
		}
		offset += icon.Header.size
	}
	for _, icon := range icons {
		if _, err := dst.Write(icon.Data); err != nil {
			return fmt.Errorf("writing icon data: %w", err)
		}
	}
	return nil
}
