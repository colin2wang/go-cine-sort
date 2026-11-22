package media

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	mp4 "github.com/abema/go-mp4"
)

// Tags holds the editable MP4 iTunes metadata fields.
type Tags struct {
	Title   string `json:"title"`
	Comment string `json:"comment"`
}

var (
	boxTypeNam = mp4.StrToBoxType("\xa9nam")
	boxTypeCmt = mp4.StrToBoxType("\xa9cmt")
)

// IsMP4 reports whether path looks like an MP4-family file.
func IsMP4(path string) bool {
	ext := strings.ToLower(extName(path))
	return ext == ".mp4" || ext == ".m4v" || ext == ".mov"
}

func extName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

// ReadTags reads title (©nam) and comment (©cmt) from an MP4 file.
// Missing tags are returned as empty strings.
func ReadTags(path string) (*Tags, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	tags := &Tags{}
	_, err = mp4.ReadBoxStructure(f, func(h *mp4.ReadHandle) (interface{}, error) {
		if h.BoxInfo.Type == mp4.BoxTypeData() && len(h.Path) >= 2 {
			parent := h.Path[len(h.Path)-2]
			if parent == boxTypeNam || parent == boxTypeCmt {
				box, _, err := h.ReadPayload()
				if err != nil {
					return nil, err
				}
				if data, ok := box.(*mp4.Data); ok {
					if parent == boxTypeNam {
						tags.Title = string(data.Data)
					} else {
						tags.Comment = string(data.Data)
					}
				}
			}
		}
		_, err := h.Expand()
		return nil, err
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// UpdateTags writes title (©nam) and comment (©cmt) into an MP4 file's
// iTunes metadata. Empty arguments leave the corresponding tag untouched.
// The file is rebuilt (mdat streams are copied verbatim) and swapped into
// place on success.
func UpdateTags(path string, title, comment string) error {
	if !IsMP4(path) {
		return fmt.Errorf("标签编辑目前仅支持 MP4/M4V/MOV 文件")
	}
	if title == "" && comment == "" {
		return nil
	}

	in, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer in.Close()

	tmp, err := os.CreateTemp(tempDirOf(path), ".cinesort-*.mp4")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath)
	}()

	if err := rebuildWithTags(in, tmp, title, comment); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("写回文件失败: %w", err)
	}
	return nil
}

func tempDirOf(path string) string {
	idx := strings.LastIndexAny(path, `/\`)
	if idx < 0 {
		return "."
	}
	return path[:idx]
}

// rebuildWithTags walks the box tree; mdat and unsupported boxes are copied
// byte for byte, the ilst data boxes for ©nam/©cmt are replaced, and a
// missing metadata tree is created under moov.
func rebuildWithTags(in *os.File, out *os.File, title, comment string) error {
	w := mp4.NewWriter(out)
	var sawIlst bool

	_, err := mp4.ReadBoxStructure(in, func(h *mp4.ReadHandle) (interface{}, error) {
		boxType := h.BoxInfo.Type

		// Replace the payload of a data box nested in ©nam / ©cmt.
		if boxType == mp4.BoxTypeData() && len(h.Path) >= 2 {
			parent := h.Path[len(h.Path)-2]
			if (parent == boxTypeNam && title != "") || (parent == boxTypeCmt && comment != "") {
				value := title
				if parent == boxTypeCmt {
					value = comment
				}
				return replaceDataBox(w, h, value)
			}
		}

		if boxType == mp4.BoxTypeIlst() {
			sawIlst = true
		}

		if !h.BoxInfo.IsSupportedType() || boxType == mp4.BoxTypeMdat() {
			return nil, w.CopyBox(in, &h.BoxInfo)
		}

		if _, err := w.StartBox(&h.BoxInfo); err != nil {
			return nil, err
		}
		box, _, err := h.ReadPayload()
		if err != nil {
			return nil, err
		}
		if _, err := mp4.Marshal(w, box, h.BoxInfo.Context); err != nil {
			return nil, err
		}

		// Tag missing inside an existing ilst: append it after the children.
		if boxType == mp4.BoxTypeIlst() {
			if title != "" {
				if err := writeTagBox(w, boxTypeNam, title); err != nil {
					return nil, err
				}
			}
			if comment != "" {
				if err := writeTagBox(w, boxTypeCmt, comment); err != nil {
					return nil, err
				}
			}
		}

		if _, err := h.Expand(); err != nil {
			return nil, err
		}

		// No ilst anywhere: create moov.udta.meta.ilst from scratch.
		if boxType == mp4.BoxTypeMoov() && !sawIlst {
			if err := writeMetadataTree(w, title, comment); err != nil {
				return nil, err
			}
			sawIlst = true
		}
		return nil, nil
	})
	return err
}

func replaceDataBox(w *mp4.Writer, h *mp4.ReadHandle, value string) (interface{}, error) {
	payload := dataBoxBytes(value)
	bi := h.BoxInfo
	bi.Offset = 0
	bi.HeaderSize = 8
	bi.Size = uint64(8 + len(payload))
	bi.ExtendToEOF = false
	if _, err := w.StartBox(&bi); err != nil {
		return nil, err
	}
	if _, err := w.Write(payload); err != nil {
		return nil, err
	}
	if _, err := w.EndBox(); err != nil {
		return nil, err
	}
	return nil, nil
}

// writeTagBox emits one complete "©tag > data" box pair inside ilst.
func writeTagBox(w *mp4.Writer, tag mp4.BoxType, value string) error {
	return writeCompleteBox(w, tag, dataBoxBytes(value))
}

// dataBoxBytes builds a complete iTunes "data" box: UTF-8 text, no locale.
func dataBoxBytes(value string) []byte {
	payload := make([]byte, 8+len(value))
	binary.BigEndian.PutUint32(payload[0:4], 1) // DataType: UTF-8 text
	binary.BigEndian.PutUint32(payload[4:8], 0) // DataLang
	copy(payload[8:], value)
	return boxBytes(mp4.BoxTypeData(), payload)
}

// writeMetadataTree appends moov.udta.meta.ilst (with hdlr + requested tags)
// after the existing children of moov.
func writeMetadataTree(w *mp4.Writer, title, comment string) error {
	// hdlr: version/flags + predefined + handler type "mdir" + reserved + name.
	hdlrPayload := make([]byte, 25) // 4 + 4 + 4 + 12 + 1
	copy(hdlrPayload[8:12], "mdir")
	hdlrPayload[24] = 0 // null-terminated name
	hdlr := boxBytes(mp4.BoxTypeHdlr(), hdlrPayload)

	ilstPayload := make([]byte, 0, 256)
	if title != "" {
		ilstPayload = append(ilstPayload, boxBytes(boxTypeNam, dataBoxBytes(title))...)
	}
	if comment != "" {
		ilstPayload = append(ilstPayload, boxBytes(boxTypeCmt, dataBoxBytes(comment))...)
	}
	ilst := boxBytes(mp4.BoxTypeIlst(), ilstPayload)

	// meta is a FullBox: 1 byte version + 3 bytes flags before its payload.
	metaPayload := append([]byte{0, 0, 0, 0}, append(hdlr, ilst...)...)
	meta := boxBytes(mp4.BoxTypeMeta(), metaPayload)

	return writeCompleteBox(w, mp4.BoxTypeUdta(), meta)
}

// writeCompleteBox emits a finished box (header + payload).
func writeCompleteBox(w *mp4.Writer, t mp4.BoxType, payload []byte) error {
	if _, err := w.StartBox(&mp4.BoxInfo{Type: t, Size: uint64(8 + len(payload)), HeaderSize: 8}); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	_, err := w.EndBox()
	return err
}

// boxBytes builds a complete box (header + payload) as raw bytes.
func boxBytes(t mp4.BoxType, payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(out[0:4], uint32(len(out)))
	copy(out[4:8], t[:])
	copy(out[8:], payload)
	return out
}
