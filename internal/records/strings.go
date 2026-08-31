package records

import (
	"bytes"
	"unicode/utf16"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

func extractStrings(col *Column, spec ColumnSpec, buf []byte, recSize, count, nBytes int) {
	off := int(spec.ByteOffset)
	for r := 0; r < count; r++ {
		p := buf[r*recSize+off:]
		col.S = append(col.S, DecodeString(p[:nBytes], spec.DataType))
	}
}

// DecodeString converts raw string bytes of the given MDF data type to a
// Go (UTF-8) string, trimming at the terminating NUL.
func DecodeString(p []byte, dataType uint8) string {
	switch dataType {
	case blocks.DTStringUTF16LE, blocks.DTStringUTF16BE:
		bigE := dataType == blocks.DTStringUTF16BE
		u := make([]uint16, 0, len(p)/2)
		for i := 0; i+1 < len(p); i += 2 {
			var c uint16
			if bigE {
				c = uint16(p[i])<<8 | uint16(p[i+1])
			} else {
				c = uint16(p[i+1])<<8 | uint16(p[i])
			}
			if c == 0 {
				break
			}
			u = append(u, c)
		}
		return string(utf16.Decode(u))
	case blocks.DTStringLatin:
		if i := bytes.IndexByte(p, 0); i >= 0 {
			p = p[:i]
		}
		// Latin-1: each byte is the code point.
		r := make([]rune, len(p))
		for i, b := range p {
			r[i] = rune(b)
		}
		return string(r)
	default: // UTF-8
		if i := bytes.IndexByte(p, 0); i >= 0 {
			p = p[:i]
		}
		return string(p)
	}
}
