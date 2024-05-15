package blocks

const (
	Version410 = 410
	Version420 = 420
	Byte       = 8
)

const (
	AtID string = "##AT"
	CaID string = "##CA"
	CcID string = "##CC"
	CgID string = "##CG"
	ChID string = "##CH"
	CnID string = "##CN"
	DgID string = "##DG"
	DlID string = "##DL"
	DtID string = "##DT"
	DvID string = "##DV"
	DzID string = "##DZ"
	EvID string = "##EV"
	FhID string = "##FH"
	HdID string = "##HD"
	MdID string = "##MD"
	SdID string = "##SD"
	SiID string = "##SI"
	SrID string = "##SR"
	TxID string = "##TX"
)

const (
	VlsdEvent     string = "VLSD"
	BusEvent      string = "BUS_EVENT"
	Event         string = "EVENT"
	PlainBusEvent string = "PLAIN_BUS_EVENT"
)

const (
	OtherSource string = "OTHER"
	EcuSource   string = "ECU"
	BusSource   string = "BUS"
	IOSource    string = "I/O"
	ToolSource  string = "TOOL"
	UserSource  string = "USER"
)

const (
	NoBusType       string = "NONE"
	NotFitBusType   string = "OTHER"
	CanBusType      string = "CAN"
	LinBusType      string = "LIN"
	MostBusType     string = "MOST"
	FlexRayBusType  string = "FLEXRAY"
	KLineBusType    string = "K_LINE"
	EthernetBusType string = "ETHERNET"
	USBBusType      string = "USB"
)

var (
	SourceTypeMap map[uint8]string = map[uint8]string{
		0: OtherSource,
		1: EcuSource,
		2: BusSource,
		3: IOSource,
		4: ToolSource,
		5: UserSource,
	}

	BusTypeMap map[uint8]string = map[uint8]string{
		0: NoBusType,
		1: NotFitBusType,
		2: CanBusType,
		3: LinBusType,
		4: MostBusType,
		5: FlexRayBusType,
		6: KLineBusType,
		7: EthernetBusType,
		8: USBBusType,
	}
)

const (
	RemoteMaster string = "REMOTE_MASTER"
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

const (
	CcNoConversion uint8 = 0
	// Linear conversion
	CcLinear uint8 = 1

	// Rational conversion
	CcRational uint8 = 2

	// Algebraic conversion
	CcAlgebraic uint8 = 3

	// value to value tabular look-up with interpolation
	CcVVLookUpInterpolation uint8 = 4

	// value to value tabular look-up without interpolation
	CcVVLookUp uint8 = 5

	// value range to value tabular look-up
	CcVrVLookUp uint8 = 6

	// value to text/scale conversion tabular look-up
	CcVTLookUp uint8 = 7

	// value range to text/scale conversion tabular look-up
	CcVrTLookUp uint8 = 8

	// text to value tabular look-up
	CcTVLookUp uint8 = 9

	// text to text tabular look-up (translation)
	CcTTLookUp uint8 = 10

	// bitfield text table
	CcBitfield uint8 = 11
)
