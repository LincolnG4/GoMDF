package mf4

import (
	"fmt"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

// ConversionKind identifies the conversion rule type (cc_type).
type ConversionKind uint8

const (
	ConvNone           ConversionKind = blocks.CCIdentity
	ConvLinear         ConversionKind = blocks.CCLinear
	ConvRational       ConversionKind = blocks.CCRational
	ConvAlgebraic      ConversionKind = blocks.CCAlgebraic
	ConvTableInterp    ConversionKind = blocks.CCTabInterp
	ConvTable          ConversionKind = blocks.CCTab
	ConvRangeToValue   ConversionKind = blocks.CCRangeToValue
	ConvValueToText    ConversionKind = blocks.CCValueToText
	ConvRangeToText    ConversionKind = blocks.CCRangeToText
	ConvTextToValue    ConversionKind = blocks.CCTextToValue
	ConvTextToText     ConversionKind = blocks.CCTextToText
	ConvBitfieldToText ConversionKind = blocks.CCBitfieldToText
)

func (k ConversionKind) String() string {
	switch k {
	case ConvNone:
		return "none"
	case ConvLinear:
		return "linear"
	case ConvRational:
		return "rational"
	case ConvAlgebraic:
		return "algebraic"
	case ConvTableInterp:
		return "table-interpolated"
	case ConvTable:
		return "table"
	case ConvRangeToValue:
		return "range-to-value"
	case ConvValueToText:
		return "value-to-text"
	case ConvRangeToText:
		return "range-to-text"
	case ConvTextToValue:
		return "text-to-value"
	case ConvTextToText:
		return "text-to-text"
	case ConvBitfieldToText:
		return "bitfield-to-text"
	}
	return fmt.Sprintf("conversion(%d)", uint8(k))
}

// Textual reports whether the conversion produces text values.
func (k ConversionKind) Textual() bool {
	switch k {
	case ConvValueToText, ConvRangeToText, ConvTextToText, ConvBitfieldToText:
		return true
	}
	return false
}

// ConversionInfo describes a channel's conversion rule for introspection.
// The conversion itself is applied by Channel.Read.
type ConversionInfo struct {
	Kind    ConversionKind
	Name    string
	Unit    string
	Comment string
	// Formula is the expression text for algebraic conversions.
	Formula string
	// Params are the cc_val parameters (meaning depends on Kind).
	Params []float64
}

func (f *File) conversionInfo(cc *blocks.CC) (ConversionInfo, error) {
	if cc == nil {
		return ConversionInfo{Kind: ConvNone}, nil
	}
	info := ConversionInfo{Kind: ConversionKind(cc.Type), Params: cc.Vals}
	var err error
	if info.Name, err = blocks.DecodeText(f.src, cc.TXName); err != nil {
		return info, err
	}
	if info.Unit, err = blocks.CommentText(f.src, cc.MDUnit); err != nil {
		return info, err
	}
	if info.Comment, err = blocks.CommentText(f.src, cc.MDComment); err != nil {
		return info, err
	}
	if cc.Type == blocks.CCAlgebraic && len(cc.Refs) > 0 {
		if info.Formula, err = blocks.DecodeText(f.src, cc.Refs[0]); err != nil {
			return info, err
		}
	}
	return info, nil
}
