package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// AT flag bits.
const (
	ATFlagEmbedded   = 1 << 0
	ATFlagCompressed = 1 << 1
	ATFlagMD5Valid   = 1 << 2
)

// AT is an attachment block.
type AT struct {
	// Links
	ATNext     int64 // at_at_next
	TXFilename int64 // at_tx_filename
	TXMimetype int64 // at_tx_mimetype
	MDComment  int64 // at_md_comment

	// Data
	Flags        uint16   // at_flags
	CreatorIndex uint16   // at_creator_index
	MD5Checksum  [16]byte // at_md5_checksum
	OriginalSize uint64   // at_original_size
	EmbeddedSize uint64   // at_embedded_size
	EmbeddedAddr int64    // absolute file offset of embedded data
}

// DecodeAT decodes an attachment block at addr.
func DecodeAT(src source.Source, addr int64) (*AT, error) {
	r, err := decodeRaw(src, addr, IDAT, 4, 40)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &AT{
		ATNext:     r.link(0),
		TXFilename: r.link(1),
		TXMimetype: r.link(2),
		MDComment:  r.link(3),

		Flags:        le.Uint16(d[0:2]),
		CreatorIndex: le.Uint16(d[2:4]),
		OriginalSize: le.Uint64(d[24:32]),
		EmbeddedSize: le.Uint64(d[32:40]),
		EmbeddedAddr: addr + HeaderSize + int64(len(r.links))*LinkSize + 40,
	}
	copy(b.MD5Checksum[:], d[8:24])
	if b.EmbeddedSize > r.header.DataLen()-40 {
		return nil, blockErrf(IDAT, addr, "%w: embedded size %d exceeds block", ErrInvalidBlock, b.EmbeddedSize)
	}
	return b, nil
}
