package blocks

const (
	Version410 = 410
	Version420 = 420
	Byte       = 8
)

const (
	AtID = "##AT"
	CaID = "##CA"
	CcID = "##CC"
	CgID = "##CG"
	ChID = "##CH"
	CnID = "##CN"
	DgID = "##DG"
	DtID = "##DT"
	DvID = "##DV"
	EvID = "##EV"
	FhID = "##FH"
	HdID = "##HD"
	MdID = "##MD"
	SiID = "##SI"
	SrID = "##SR"
	TxID = "##TX"
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
	CcNoConversion          uint8 = 0
	CcLinear                uint8 = 1
	CcRational              uint8 = 2
	CcAlgebraic             uint8 = 3
	CcVVLookUpInterpolation uint8 = 4
	CcVVLookUp              uint8 = 5
	CcVrVLookUp             uint8 = 6
	CcVTLookUp              uint8 = 7
	CcVrTLookUp             uint8 = 8
	CcTVLookUp              uint8 = 9
	CcTTLookUp              uint8 = 10
	CcBitfield              uint8 = 11
)
