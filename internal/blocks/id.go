package blocks

import (
	"bytes"
	"fmt"

	"github.com/LincolnG4/GoMDF/internal/source"
)

// IDSize is the fixed size of the identification block at file offset 0.
const IDSize = 64

// ID is the identification block (no common header; fixed 64 bytes at
// offset 0).
type ID struct {
	File             string // id_file: "MDF     " or "UnFinMF " for unfinalized files
	VersionString    string // id_vers, e.g. "4.10"
	Program          string // id_prog: creator tool
	Version          uint16 // id_ver: 400/410/420...
	UnfinalizedFlags uint16 // id_unfin_flags: standard flags for unfinalized files
	CustomUnfinFlags uint16 // id_custom_unfin_flags
}

// Unfinalized reports whether the file was not finalized by the writer.
func (id *ID) Unfinalized() bool { return id.File == "UnFinMF " }

// DecodeID reads the identification block at offset 0.
func DecodeID(src source.Source) (*ID, error) {
	buf, err := src.Slice(0, IDSize)
	if err != nil {
		return nil, fmt.Errorf("identification block: %w", err)
	}
	id := &ID{
		File:             string(buf[0:8]),
		VersionString:    string(bytes.TrimRight(buf[8:16], " \x00")),
		Program:          string(bytes.TrimRight(buf[16:24], " \x00")),
		Version:          le.Uint16(buf[28:30]),
		UnfinalizedFlags: le.Uint16(buf[60:62]),
		CustomUnfinFlags: le.Uint16(buf[62:64]),
	}
	if id.File != "MDF     " && id.File != "UnFinMF " {
		return nil, fmt.Errorf("%w: not an MDF file (id %q)", ErrInvalidBlock, buf[0:8])
	}
	return id, nil
}
