// ico format encoding.
// Modified from https://github.com/wailsapp/wails project.
package ico

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"

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

// FromPNG generates an ico from a source png.
func FromPNG(pngfile string, iconfile string) error {
	sizes := []int{256, 128, 64, 48, 32, 16}

	pngf, err := os.Open(pngfile)
	if err != nil {
		return err
	}
	defer pngf.Close()

	pngdata, err := png.Decode(pngf)
	if err != nil {
		return err
	}

	icons := []Container{}

	for _, size := range sizes {
		rect := image.Rect(0, 0, int(size), int(size))
		rawdata := image.NewRGBA(rect)
		scale := draw.CatmullRom
		scale.Scale(rawdata, rect, pngdata, pngdata.Bounds(), draw.Over, nil)

		icondata := new(bytes.Buffer)
		writer := bufio.NewWriter(icondata)
		err = png.Encode(writer, rawdata)
		if err != nil {
			return err
		}
		writer.Flush()

		imgSize := size
		if imgSize >= 256 {
			imgSize = 0
		}

		data := icondata.Bytes()

		icn := Container{
			Header: Descriptor{
				width:  uint8(imgSize),
				height: uint8(imgSize),
				planes: 1,
				bpp:    32,
				size:   uint32(len(data)),
			},
			Data: data,
		}
		icons = append(icons, icn)
	}

	outfile, err := os.Create(iconfile)
	if err != nil {
		return err
	}
	defer outfile.Close()

	ico := Header{
		imageType:  1,
		imageCount: uint16(len(sizes)),
	}
	err = binary.Write(outfile, binary.LittleEndian, ico)
	if err != nil {
		return err
	}

	offset := uint32(6 + 16*len(sizes))
	for _, icon := range icons {
		icon.Header.offset = offset
		err = binary.Write(outfile, binary.LittleEndian, icon.Header)
		if err != nil {
			return err
		}
		offset += icon.Header.size
	}
	for _, icon := range icons {
		_, err = outfile.Write(icon.Data)
		if err != nil {
			return err
		}
	}
	return nil
}
