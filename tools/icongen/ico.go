package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// ICOImage is one PNG-encoded frame of the icon container.
type ICOImage struct {
	Size int
	PNG  []byte
}

// EncodeICO writes a Windows ICO container with PNG-compressed frames.
// PNG frames are supported by Windows Vista and later, and avoid all DIB
// bit-packing / AND-mask quirks while preserving full alpha.
//
// Layout: ICONDIR (6 bytes) + N * ICONDIRENTRY (16 bytes) + concatenated PNGs.
func EncodeICO(images []ICOImage) ([]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("ICO 至少需要一个图像尺寸")
	}
	if len(images) > 0xffff {
		return nil, fmt.Errorf("ICO 图像数量超出限制")
	}

	var buf bytes.Buffer
	write16 := func(v uint16) { _ = binary.Write(&buf, binary.LittleEndian, v) }
	write32 := func(v uint32) { _ = binary.Write(&buf, binary.LittleEndian, v) }

	write16(0) // idReserved
	write16(1) // idType: 1 = icon
	write16(uint16(len(images)))

	offset := 6 + 16*len(images)
	for _, image := range images {
		if image.Size <= 0 || image.Size > 256 {
			return nil, fmt.Errorf("非法图标尺寸 %d（应为 1..256）", image.Size)
		}
		if len(image.PNG) == 0 {
			return nil, fmt.Errorf("尺寸 %d 的 PNG 数据为空", image.Size)
		}
		dimension := byte(image.Size)
		if image.Size == 256 {
			dimension = 0 // 0 encodes 256 in ICONDIRENTRY
		}
		buf.WriteByte(dimension) // bWidth
		buf.WriteByte(dimension) // bHeight
		buf.WriteByte(0)         // bColorCount: 0 for >= 8bpp
		buf.WriteByte(0)         // bReserved
		write16(1)               // wPlanes
		write16(32)              // wBitCount
		write32(uint32(len(image.PNG)))
		write32(uint32(offset))
		offset += len(image.PNG)
	}

	for _, image := range images {
		buf.Write(image.PNG)
	}
	return buf.Bytes(), nil
}
