package CC

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
	"github.com/Pramod-Devireddy/go-exprtk"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	TxName    int64
	MdUnit    int64
	MdComment int64
	Inverse   int64
	Ref       []int64
}

type Data struct {
	Type        uint8
	Precision   uint8
	Flags       uint16
	RefCount    uint16
	ValCount    uint16
	PhyRangeMin float64
	PhyRangeMax float64
	Val         []float64
}

type Conversion interface {
	Apply(*[]interface{})
}

type Info struct {
	Name    string
	Unit    string
	Comment string
}

type Linear struct {
	Info Info
	P1   float64
	P2   float64
}

type Rational struct {
	Info Info
	P1   float64
	P2   float64
	P3   float64
	P4   float64
	P5   float64
	P6   float64
}

type Algebraic struct {
	Info    Info
	Formula string
}

type Interporlation struct {
	Info Info
	P1   []float64
}

func New(file *os.File, version uint16, startAdress int64) *Block {
	var b Block
	var err error

	b.Header = blocks.Header{}

	b.Header, err = blocks.GetHeader(file, startAdress, blocks.CcID)
	if err != nil {
		return b.BlankBlock()
	}

	//Calculates size of Link Block
	blockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	buffEach := make([]byte, blockSize)

	// Read the Link section from the binary file
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		fmt.Println("Error reading Link section:", err)
	}

	// Populate the Link fields dynamically based on version
	linkFields := []int64{}
	for i := 0; i < len(buffEach)/8; i++ {
		linkFields = append(linkFields, int64(binary.LittleEndian.Uint64(buffEach[i*8:(i+1)*8])))
	}

	b.Link = Link{
		TxName:    linkFields[0],
		MdUnit:    linkFields[1],
		MdComment: linkFields[2],
		Inverse:   linkFields[3],
		Ref:       linkFields[4:],
	}
	fmt.Printf("%+v\n", b.Link)

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	b.Data = Data{}
	buf := blocks.LoadBuffer(file, blockSize)

	names := make([]interface{}, 7)
	names[0] = &b.Data.Type
	names[1] = &b.Data.Precision
	names[2] = &b.Data.Flags
	names[3] = &b.Data.RefCount
	names[4] = &b.Data.ValCount
	names[5] = &b.Data.PhyRangeMin
	names[6] = &b.Data.PhyRangeMax

	for i := 0; i < len(names); i++ {
		BinaryError := binary.Read(buf, binary.LittleEndian, names[i])
		if BinaryError != nil {
			fmt.Println("ERROR", BinaryError)
		}
	}

	foo := make([]float64, b.Data.ValCount)
	BinaryError := binary.Read(buf, binary.LittleEndian, foo)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}
	b.Data.Val = foo

	fmt.Printf("%+v\n", b.Data)
	return &b
}

// Get returns an conversion struct type
func (b *Block) Get(file *os.File) Conversion {
	switch b.dataType() {
	case blocks.CcNoConversion:
		return nil
	case blocks.CcLinear:
		return b.GetLinear(file)
	case blocks.CcRational:
		return b.GetRational(file)
	case blocks.CcAlgebraic:
		return b.GetAlgebraic(file)
	case blocks.CcVVLookUpInterpolation:
		return nil
	case blocks.CcVVLookUp:
		return nil
	case blocks.CcVrVLookUp:
		return nil
	case blocks.CcVTLookUp:
		return nil
	case blocks.CcVrTLookUp:
		return nil
	case blocks.CcTVLookUp:
		return nil
	case blocks.CcTTLookUp:
		return nil
	case blocks.CcBitfield:
		return nil
	default:
		fmt.Println("OUSSSSS")
		return nil
	}
}

func (b *Block) getVal() []float64 {
	return b.Data.Val
}

// GetLinear returns linear conversion struct type
func (b *Block) GetLinear(file *os.File) Conversion {
	v := b.getVal()

	return &Linear{
		Info: b.getInfo(file),
		P1:   v[0],
		P2:   v[1],
	}
}

// GetRational returns rational conversion struct type
func (b *Block) GetRational(file *os.File) Conversion {
	v := b.getVal()

	return &Rational{
		Info: b.getInfo(file),
		P1:   v[0],
		P2:   v[1],
		P3:   v[2],
		P4:   v[3],
		P5:   v[4],
		P6:   v[5],
	}
}

func (b *Block) GetAlgebraic(file *os.File) Conversion {
	f := b.getRef()
	formula := b.refToString(file, f)
	fmt.Println(formula)
	return &Algebraic{
		Info:    b.getInfo(file),
		Formula: formula[0],
	}
}

func (b *Block) refToString(file *os.File, ref []int64) []string {
	var text string
	var err error

	r := make([]string, 0)
	for i := 0; i < len(ref); i++ {
		text, err = TX.GetText(file, ref[i])
		if err != nil {
			return []string{}
		}
		r = append(r, text)
	}
	return r
}

// linear formula with two parameters `(y=a*x+b)`
func (l *Linear) Apply(sample *[]interface{}) {
	s := *sample

	for i, v := range s {
		switch c := v.(type) {
		case int:
			s[i] = float64(c)*l.P2 + l.P1
		case float64:
			s[i] = c*l.P2 + l.P1
		default:
			fmt.Printf("Variable type %s: not numerical", c)
		}
	}
}

// Rational formula with two parameters
// `(y=v1*x+v2*x+v3*x/v4*x+v5*x+v6*x)`
func (r *Rational) Apply(sample *[]interface{}) {
	s := *sample

	for i, v := range s {
		switch c := v.(type) {
		case int:
			d := float64(c)
			s[i] = (r.P1*math.Pow(d, 2) + r.P2*d + r.P3) / (r.P4*math.Pow(d, 2) + r.P5*d + r.P6)
		case float64:
			s[i] = (r.P1*math.Pow(c, 2) + r.P2*c + r.P3) / (r.P4*math.Pow(c, 2) + r.P5*c + r.P6)
		default:
			fmt.Printf("Variable type %s: not numerical", c)
		}
	}
}

// Interporlation formula
func (a *Algebraic) Apply(sample *[]interface{}) {
	var result interface{}
	var err error
	var x string = "X"

	s := *sample

	exprtkObj := exprtk.NewExprtk()
	defer exprtkObj.Delete()

	exprtkObj.SetExpression(a.Formula)
	exprtkObj.AddDoubleVariable(x)
	err = exprtkObj.CompileExpression()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for i, v := range s {
		switch c := v.(type) {
		case int:
			exprtkObj.SetDoubleVariableValue(x, float64(c))
		case float64:
			exprtkObj.SetDoubleVariableValue(x, c)
		default:
			fmt.Printf("Variable type %s: not numerical", c)
		}
		result = exprtkObj.GetEvaluatedValue()
		s[i] = result
	}
}

// Interporlation formula
func (it *Interporlation) Apply(sample *[]interface{}) {
	s := *sample

	for i, v := range s {
		switch c := v.(type) {
		case int:
			d := float64(c)
			s[i] = d
		case float64:
			s[i] = c
		default:
			fmt.Printf("Variable type %s: not numerical", c)
		}
	}
}

func (b *Block) getInfo(file *os.File) Info {
	return Info{
		Name:    b.name(file),
		Unit:    b.unit(file),
		Comment: b.comment(file),
	}
}

func (b *Block) dataType() uint8 {
	return b.Data.Type
}

func (b *Block) name(file *os.File) string {
	if b.Link.TxName == 0 {
		return ""
	}

	t, err := TX.GetText(file, b.Link.TxName)
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) unit(file *os.File) string {
	if b.Link.MdUnit == 0 {
		return ""
	}

	t, err := TX.GetText(file, b.Link.MdUnit)
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) comment(file *os.File) string {
	if b.Link.MdComment == 0 {
		return ""
	}

	t, err := TX.GetText(file, b.Link.MdComment)
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) getRef() []int64 {
	return b.Link.Ref
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.CcID),
			Reserved:  [4]byte{},
			Length:    blocks.FhblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}
