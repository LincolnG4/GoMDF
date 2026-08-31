package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// DG is a data group block.
type DG struct {
	// Links
	DGNext    int64 // dg_dg_next
	CGFirst   int64 // dg_cg_first
	Data      int64 // dg_data: DT/DV/DZ/DL/LD/HL block with the records
	MDComment int64 // dg_md_comment

	// Data
	RecIDSize uint8 // dg_rec_id_size: 0 (sorted) or 1/2/4/8 bytes
}

// DecodeDG decodes a data group block at addr.
func DecodeDG(src source.Source, addr int64) (*DG, error) {
	r, err := decodeRaw(src, addr, IDDG, 4, 1)
	if err != nil {
		return nil, err
	}
	return &DG{
		DGNext:    r.link(0),
		CGFirst:   r.link(1),
		Data:      r.link(2),
		MDComment: r.link(3),
		RecIDSize: r.data[0],
	}, nil
}
