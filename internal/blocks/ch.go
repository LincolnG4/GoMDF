package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// CH hierarchy types (ch_type).
const (
	CHGroup      = 0
	CHFunction   = 1
	CHStructure  = 2
	CHMapList    = 3
	CHInputVars  = 4
	CHOutputVars = 5
	CHLocalVars  = 6
	CHCalDefVars = 7
	CHCalRefVars = 8
)

// CHElement is one channel reference of a hierarchy node: a link triple
// to the channel's parent DG, parent CG and the CN itself.
type CHElement struct {
	DG int64
	CG int64
	CN int64
}

// CH is a channel hierarchy block: a node of the logical channel tree.
type CH struct {
	// Links
	CHNext    int64       // ch_ch_next: next sibling
	CHFirst   int64       // ch_ch_first: first child
	TXName    int64       // ch_tx_name
	MDComment int64       // ch_md_comment
	Elements  []CHElement // ch_element[N]: DG/CG/CN link triples

	// Data
	ElementCount uint32 // ch_element_count
	Type         uint8  // ch_type
}

// DecodeCH decodes a channel hierarchy block at addr.
func DecodeCH(src source.Source, addr int64) (*CH, error) {
	r, err := decodeRaw(src, addr, IDCH, 4, 5)
	if err != nil {
		return nil, err
	}
	b := &CH{
		CHNext:    r.link(0),
		CHFirst:   r.link(1),
		TXName:    r.link(2),
		MDComment: r.link(3),

		ElementCount: le.Uint32(r.data[0:4]),
		Type:         r.data[4],
	}
	n := int(b.ElementCount)
	if 4+3*n > len(r.links) {
		return nil, blockErrf(IDCH, addr, "%w: element count %d exceeds %d links", ErrInvalidBlock, n, len(r.links))
	}
	b.Elements = make([]CHElement, n)
	for i := 0; i < n; i++ {
		b.Elements[i] = CHElement{
			DG: r.links[4+3*i],
			CG: r.links[4+3*i+1],
			CN: r.links[4+3*i+2],
		}
	}
	return b, nil
}
