package blocks

// Block IDs.
const (
	IDHD = "##HD"
	IDMD = "##MD"
	IDTX = "##TX"
	IDFH = "##FH"
	IDCH = "##CH"
	IDAT = "##AT"
	IDEV = "##EV"
	IDDG = "##DG"
	IDCG = "##CG"
	IDSI = "##SI"
	IDCN = "##CN"
	IDCC = "##CC"
	IDCA = "##CA"
	IDDT = "##DT"
	IDSR = "##SR"
	IDRD = "##RD"
	IDSD = "##SD"
	IDDL = "##DL"
	IDDZ = "##DZ"
	IDHL = "##HL"
	// MDF 4.2 column-oriented storage.
	IDLD = "##LD"
	IDDV = "##DV"
	IDDI = "##DI"
	IDRV = "##RV"
	IDRI = "##RI"
)

// Channel data types (cn_data_type).
const (
	DTUintLE        = 0
	DTUintBE        = 1
	DTIntLE         = 2
	DTIntBE         = 3
	DTFloatLE       = 4
	DTFloatBE       = 5
	DTStringLatin   = 6
	DTStringUTF8    = 7
	DTStringUTF16LE = 8
	DTStringUTF16BE = 9
	DTByteArray     = 10
	DTMIMESample    = 11
	DTMIMEStream    = 12
	DTCANopenDate   = 13
	DTCANopenTime   = 14
	DTComplexLE     = 15
	DTComplexBE     = 16
)

// Channel types (cn_type).
const (
	CNFixedLength   = 0
	CNVLSD          = 1
	CNMaster        = 2
	CNVirtualMaster = 3
	CNSync          = 4
	CNMaxLength     = 5
	CNVirtualData   = 6
)

// Sync types (cn_sync_type / ev_sync_type).
const (
	SyncNone     = 0
	SyncTime     = 1
	SyncAngle    = 2
	SyncDistance = 3
	SyncIndex    = 4
)

// Conversion types (cc_type).
const (
	CCIdentity       = 0
	CCLinear         = 1
	CCRational       = 2
	CCAlgebraic      = 3
	CCTabInterp      = 4 // value to value, interpolation
	CCTab            = 5 // value to value, no interpolation
	CCRangeToValue   = 6
	CCValueToText    = 7
	CCRangeToText    = 8
	CCTextToValue    = 9
	CCTextToText     = 10
	CCBitfieldToText = 11 // MDF 4.2
)

// CN flag bits (cn_flags).
const (
	CNFlagAllInvalid    = 1 << 0
	CNFlagInvalBit      = 1 << 1
	CNFlagPrecision     = 1 << 2
	CNFlagValRange      = 1 << 3
	CNFlagLimitRange    = 1 << 4
	CNFlagExtLimitRange = 1 << 5
	CNFlagDiscrete      = 1 << 6
	CNFlagCalibration   = 1 << 7
	CNFlagCalculated    = 1 << 8
	CNFlagVirtual       = 1 << 9
	CNFlagBusEvent      = 1 << 10
	CNFlagMonotonous    = 1 << 11
	CNFlagDefaultX      = 1 << 12
)

// CG flag bits (cg_flags).
const (
	CGFlagVLSD         = 1 << 0
	CGFlagBusEvent     = 1 << 1
	CGFlagPlainBus     = 1 << 2
	CGFlagRemoteMaster = 1 << 3 // MDF 4.2
	CGFlagEvent        = 1 << 4 // MDF 4.2
)

// DL flag bits.
const DLFlagEqualLength = 1 << 0

// DZ zip types.
const (
	ZipDeflate          = 0
	ZipTransposeDeflate = 1
)
