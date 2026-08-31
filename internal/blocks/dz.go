package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// DZ is a zipped data block header. The compressed payload is NOT loaded
// here; DataAddr/DataLength locate it for lazy decompression.
type DZ struct {
	OrgBlockType  string // dz_org_block_type: "DT", "DV", "SD", "RD" (without "##")
	ZipType       uint8  // dz_zip_type
	ZipParameter  uint32 // dz_zip_parameter: transposition column width for ZipTransposeDeflate
	OrgDataLength uint64 // dz_org_data_length: uncompressed size
	DataLength    uint64 // dz_data_length: compressed size
	DataAddr      int64  // absolute file offset of the compressed bytes
}

// DecodeDZ decodes a zipped data block header at addr.
func DecodeDZ(src source.Source, addr int64) (*DZ, error) {
	r, err := decodeRaw(src, addr, IDDZ, 0, 24)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &DZ{
		OrgBlockType:  string(d[0:2]),
		ZipType:       d[2],
		ZipParameter:  le.Uint32(d[4:8]),
		OrgDataLength: le.Uint64(d[8:16]),
		DataLength:    le.Uint64(d[16:24]),
		DataAddr:      addr + HeaderSize + 24,
	}
	if b.DataLength > r.header.DataLen()-24 {
		return nil, blockErrf(IDDZ, addr, "%w: dz_data_length %d exceeds block", ErrInvalidBlock, b.DataLength)
	}
	return b, nil
}
