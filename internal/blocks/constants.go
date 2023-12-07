package blocks

const (
	Version420 = 420
	Byte       = 8
)

const (
	EvID = "##EV"
	AtID = "##AT"
	DgID = "##DG"
	CgID = "##CG"
	CnID = "##CN"
	TxID = "##TX"
	MdID = "##MD"
	FhID = "##FH"
	DtID = "##DT"
	HdID = "##HD"
	CaID = "##CA"
	CcID = "##CC"
	ChID = "##CH"
)

const (
	LinkSize    uint64 = 8
	HeaderSize  uint64 = 24
	IdblockSize int64  = 64
	HdblockSize uint64 = 104
	FhblockSize uint64 = 56
	AtblockSize uint64 = 96
	DgblockSize uint64 = 64
	CgblockSize uint64 = 104
	CnblockSize uint64 = 160
	ChblockSize uint64 = 160
	EvblockSize uint64 = 24
)
