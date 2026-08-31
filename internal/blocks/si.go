package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// SI source types (si_type).
const (
	SIOther = 0
	SIECU   = 1
	SIBus   = 2
	SIIO    = 3
	SITool  = 4
	SIUser  = 5
)

// SI bus types (si_bus_type).
const (
	BusNone     = 0
	BusOther    = 1
	BusCAN      = 2
	BusLIN      = 3
	BusMOST     = 4
	BusFlexRay  = 5
	BusKLine    = 6
	BusEthernet = 7
	BusUSB      = 8
)

// SI is a source information block.
type SI struct {
	// Links
	TXName    int64 // si_tx_name
	TXPath    int64 // si_tx_path
	MDComment int64 // si_md_comment

	// Data
	Type    uint8 // si_type
	BusType uint8 // si_bus_type
	Flags   uint8 // si_flags
}

// DecodeSI decodes a source information block at addr. addr == 0 returns
// (nil, nil).
func DecodeSI(src source.Source, addr int64) (*SI, error) {
	if addr == 0 {
		return nil, nil
	}
	r, err := decodeRaw(src, addr, IDSI, 3, 3)
	if err != nil {
		return nil, err
	}
	return &SI{
		TXName:    r.link(0),
		TXPath:    r.link(1),
		MDComment: r.link(2),
		Type:      r.data[0],
		BusType:   r.data[1],
		Flags:     r.data[2],
	}, nil
}
