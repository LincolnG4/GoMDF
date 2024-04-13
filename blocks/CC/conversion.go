package CC

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/TX"
	"github.com/soniah/evaler"
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

type Algebraic struct {
	Info    Info
	Formula string
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

type ValueValue struct {
	Info   Info
	Keys   []float64
	Values []float64
	Type   uint8
}

type ValueRangeToValue struct {
	Info     Info
	KeyMin   []float64
	KeyMax   []float64
	Values   []float64
	Default  float64
	DataType uint8
}

type ValueRangeToText struct {
	Info     Info
	KeyMin   []float64
	KeyMax   []float64
	Links    []interface{}
	Default  interface{}
	DataType uint8
}

type ValueText struct {
	Info    Info
	Keys    []float64
	Links   []interface{}
	Default interface{}
}

type TextValue struct {
	Info    Info
	Values  []float64
	Keys    []string
	Default float64
}

type TextText struct {
	Info    Info
	Keys    []string
	Values  []string
	Default string
}

type BitfieldText struct {
	Info    Info
	Bitmask []float64
	Links   []float64
}

func New(file *os.File, startAdress int64) *Block {
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
		fmt.Println("error reading link section ccblock:", err)
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
			fmt.Println("error loading data from ccblock:", BinaryError)
		}
	}

	valcount := make([]float64, b.Data.ValCount)
	BinaryError := binary.Read(buf, binary.LittleEndian, valcount)
	if BinaryError != nil {
		fmt.Println("error loading data from ccblock:", BinaryError)
	}
	b.Data.Val = valcount

	return &b
}

// Get returns an conversion struct type
func (b *Block) Get(file *os.File, channelType uint8) Conversion {
	switch b.dataType() {
	case blocks.CcNoConversion:
		return nil
	case blocks.CcLinear:
		return b.GetLinear(file)
	case blocks.CcRational:
		return b.GetRational(file)
	case blocks.CcAlgebraic:
		return b.GetAlgebraic(file)
	case blocks.CcVVLookUpInterpolation, blocks.CcVVLookUp:
		return b.GetValueToValue(file)
	case blocks.CcVrVLookUp:
		return b.GetValueRangeToValue(file, channelType)
	case blocks.CcVTLookUp:
		return b.GetValueToText(file)
	case blocks.CcVrTLookUp:
		return b.GetValueRangeToText(file, channelType)
	case blocks.CcTVLookUp:
		return b.GetTextToValue(file)
	case blocks.CcTTLookUp:
		return b.GetTextToText(file)
	case blocks.CcBitfield:
		return b.GetBitfield(file)
	default:
		return nil
	}
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

// GetVVInterporlation returns value to value tabular look-up with interpolation
func (b *Block) GetValueToValue(file *os.File) Conversion {
	v := b.getVal()
	key, value := createKeyValueFloat64(&v)
	return &ValueValue{
		Info:   b.getInfo(file),
		Keys:   key,
		Values: value,
		Type:   b.dataType(),
	}
}

// GetVVInterporlation returns value to value tabular look-up with interpolation
func (b *Block) GetValueRangeToValue(file *os.File, channelType uint8) Conversion {
	v := b.getVal()
	keyMin, keyMax, value, def := createKeyMinMaxValue(&v)
	return &ValueRangeToValue{
		Info:     b.getInfo(file),
		KeyMin:   keyMin,
		KeyMax:   keyMax,
		Values:   value,
		Default:  def,
		DataType: channelType,
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
	formula := b.refToString(file)
	return &Algebraic{
		Info:    b.getInfo(file),
		Formula: formula[0].(string),
	}
}

func (b *Block) GetValueToText(file *os.File) Conversion {
	v := b.getVal()
	t := b.refToString(file)
	return &ValueText{
		Info:    b.getInfo(file),
		Keys:    v,
		Links:   t[:len(t)-2],
		Default: t[len(t)-1],
	}
}

func (b *Block) GetValueRangeToText(file *os.File, channelType uint8) Conversion {
	v := b.getVal()
	min, max := createKeyValueFloat64(&v)
	t := b.refToString(file)
	return &ValueRangeToText{
		Info:     b.getInfo(file),
		KeyMin:   min,
		KeyMax:   max,
		Links:    t[:len(t)-2],
		Default:  t[len(t)-1],
		DataType: channelType,
	}
}

func (b *Block) GetTextToValue(file *os.File) Conversion {
	v := b.getVal()
	t := b.refToString(file)
	keys := interfaceArrayToStringArray(t)
	return &TextValue{
		Info:    b.getInfo(file),
		Keys:    keys,
		Values:  v[:len(v)-1],
		Default: v[len(v)-1],
	}
}

func (b *Block) GetTextToText(file *os.File) Conversion {
	t := b.refToString(file)
	k := t[:len(t)-1]

	keys := interfaceArrayToStringArray(k)
	key, value := createKeyValueString(&keys)
	return &TextText{
		Info:    b.getInfo(file),
		Keys:    key,
		Values:  value,
		Default: t[len(t)-1].(string),
	}
}

func (b *Block) GetBitfield(file *os.File) Conversion {
	v := b.getVal()
	t := b.refToString(file)

	fmt.Println(t, v)
	return &BitfieldText{
		Info: b.getInfo(file),
	}
}

// linear formula with two parameters `(y=a*x+b)`
func (l *Linear) Apply(sample *[]interface{}) {
	s := *sample

	for i, v := range s {
		c := convertToFloat64(v)
		s[i] = c*l.P2 + l.P1
	}
}

// Rational formula with two parameters
// `(y=v1*x+v2*x+v3*x/v4*x+v5*x+v6*x)`
func (r *Rational) Apply(sample *[]interface{}) {
	s := *sample

	for i, v := range s {
		c := convertToFloat64(v)
		s[i] = (r.P1*math.Pow(c, 2) + r.P2*c + r.P3) / (r.P4*math.Pow(c, 2) + r.P5*c + r.P6)
	}
}

// Interporlation formula
func (a *Algebraic) Apply(sample *[]interface{}) {
	var x string = "X"

	s := *sample

	//Configure Formula
	m := make(map[string]string)

	for i, v := range s {
		c := convertToFloat64(v)
		m[x] = fmt.Sprintf("%f", c)
		r, err := evaler.EvalWithVariables(a.Formula, m)
		if err != nil {
			fmt.Println("error to convert formula")
		}
		result := evaler.BigratToFloat(r)
		s[i] = result
	}
}

func (vt *ValueText) Apply(sample *[]interface{}) {
	s := *sample

	for i, v := range s {
		c := convertToFloat64(v)
		for j := 0; j < len(vt.Keys); j++ {
			if c == vt.Keys[j] {
				switch v := vt.Links[j].(type) {
				case string:
					s[i] = vt.Links[j]
				case Conversion:
					a := []interface{}{c}
					v.Apply(&a)
					s[i] = a[0]
				}
				break
			}

			switch v := vt.Default.(type) {
			case string:
				s[i] = vt.Default
			case Conversion:
				a := []interface{}{c}
				v.Apply(&a)
				s[i] = a[0]
			}
		}
	}
}

func (vt *ValueRangeToText) Apply(sample *[]interface{}) {
	var f func(j int) bool
	var c float64

	s := *sample
	n := len(vt.KeyMin)

	if vt.DataType <= 3 {
		f = func(j int) bool {
			return vt.KeyMax[j] >= c
		}
	} else {
		f = func(j int) bool {
			return vt.KeyMax[j] > c
		}
	}

	for i, v := range s {
		c := convertToFloat64(v)
		index := sort.Search(n, f)

		if index != n && c >= vt.KeyMin[index] {
			switch v := vt.Links[index].(type) {
			case string:
				s[i] = v
			case Conversion:
				a := []interface{}{c}
				v.Apply(&a)
				s[i] = a[0]
			}

		} else {
			switch v := vt.Default.(type) {
			case string:
				s[i] = v
			case Conversion:
				a := []interface{}{c}
				v.Apply(&a)
				s[i] = a[0]
			}
		}
	}
}

func (vv *ValueValue) Apply(sample *[]interface{}) {
	if vv.Type == blocks.CcVVLookUpInterpolation {
		vv.withInterpolation(sample)
	}
	if vv.Type == blocks.CcVVLookUp {
		vv.withoutInterpolation(sample)
	}
}

func (vr *ValueRangeToValue) Apply(sample *[]interface{}) {
	var c float64
	var f func(j int) bool
	s := *sample
	n := len(vr.KeyMin)

	if vr.DataType <= 3 {
		f = func(j int) bool {
			return vr.KeyMax[j] >= c
		}
	} else {
		f = func(j int) bool {
			return vr.KeyMax[j] > c
		}
	}

	for i, v := range s {
		c = convertToFloat64(v)

		index := sort.Search(n, f)

		if index != n && c >= vr.KeyMin[index] {
			s[i] = vr.Values[index]
		} else {
			s[i] = vr.Default
		}
	}

}

func (vv ValueValue) withInterpolation(sample *[]interface{}) {
	var check int
	var c float64

	s := *sample
	n := len(vv.Keys)
	for i, v := range s {
		c = convertToFloat64(v)
		check = 0
		if c <= vv.Keys[check] {
			s[i] = vv.Values[check]
			continue
		}

		check = n - 1
		if c >= vv.Keys[check] {
			s[i] = vv.Values[check]
			continue
		}

		// Find the index i such that key[i] <= c < key[i+1]
		index := blocks.BinarySearch(vv.Keys, c)
		if index != -1 {
			s[i] = interpolate(c, vv.Keys[index], vv.Keys[index+1], vv.Values[index], vv.Values[index+1])
		}

	}
}

func (vv ValueValue) withoutInterpolation(sample *[]interface{}) {
	var c float64
	s := *sample
	n := len(vv.Keys)

	for i, v := range s {
		c = convertToFloat64(v)

		index := sort.Search(n, func(j int) bool {
			return vv.Keys[j] >= c
		})

		// Check if c is outside the range of keys
		if index == 0 {
			s[i] = vv.Values[0]
		} else if index == n {
			s[i] = vv.Values[n-1]
		} else {
			prev := vv.Keys[index-1]
			next := vv.Keys[index]

			if c-prev > next-c {
				s[i] = vv.Values[index]
			} else {
				s[i] = vv.Values[index-1]
			}
		}
	}

}

func (tv *TextValue) Apply(sample *[]interface{}) {
	s := *sample

	keyMap := make(map[string]float64)
	for j, k := range tv.Keys {
		keyMap[k] = tv.Values[j]
	}

	for i := range s {
		if val, ok := keyMap[s[i].(string)]; ok {
			s[i] = val
		} else {
			s[i] = tv.Default
		}
	}
}

func (tt *TextText) Apply(sample *[]interface{}) {
	s := *sample
	keyMap := make(map[string]string)
	for j, k := range tt.Keys {
		keyMap[k] = tt.Values[j]
	}

	for i, v := range s {
		if val, ok := keyMap[v.(string)]; ok {
			s[i] = val
		} else {
			s[i] = tt.Default
		}
	}
}

func (bt *BitfieldText) Apply(sample *[]interface{}) {
	fmt.Println("bitfieldtext convertion is not ready on the package, open a pull request")
}

func interpolate(x, x0, x1, y0, y1 float64) float64 {
	return y0 + (((x - x0) * (y1 - y0)) / (x1 - x0))
}

func convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		panic(fmt.Errorf("variable type %T: not numerical", v))
	}
}

func createKeyValueFloat64(val *[]float64) ([]float64, []float64) {
	v := *val
	lenV := len(v) / 2
	keys := make([]float64, lenV)
	vals := make([]float64, lenV)

	j, k := 0, 0
	for i := 0; i < len(v); i++ {
		if i%2 == 0 {
			keys[j] = v[i]
			j++
		} else {
			vals[k] = v[i]
			k++
		}
	}
	return keys, vals
}

func createKeyValueString(val *[]string) ([]string, []string) {
	v := *val
	lenV := len(v) / 2
	keys := make([]string, lenV)
	vals := make([]string, lenV)

	j, k := 0, 0
	for i := 0; i < len(v); i++ {
		if i%2 == 0 {
			keys[j] = v[i]
			j++
		} else {
			vals[k] = v[i]
			k++
		}
	}
	return keys, vals
}

func createKeyMinMaxValue(val *[]float64) ([]float64, []float64, []float64, float64) {
	v := *val
	lenV := len(v) / 3
	keyMin := make([]float64, lenV)
	keyMax := make([]float64, lenV)
	vals := make([]float64, lenV)

	// Array alternate cycle between MIN, MAX, Value
	j := 0
	for i := 0; i < len(v)-2; i += 3 {
		keyMin[j] = v[i]
		keyMax[j] = v[i+1]
		vals[j] = v[i+2]
		j++
	}
	// Last value is default
	def := v[len(v)-1]
	return keyMin, keyMax, vals, def
}

func (b *Block) getInfo(file *os.File) Info {
	return Info{
		Name:    b.name(file),
		Unit:    b.unit(file),
		Comment: b.comment(file),
	}
}

func (b *Block) refToString(file *os.File) []interface{} {
	var result interface{}

	ref := b.getRef()
	r := make([]interface{}, 0)

	for i := 0; i < len(ref); i++ {
		header, err := blocks.GetBlockType(file, ref[i])
		if err != nil {
			return nil
		}
		hId := string(header.ID[:])
		if hId == blocks.TxID || hId == blocks.MdID {
			result, err = TX.GetText(file, ref[i])
			if err != nil {
				return nil
			}
		}
		if hId == blocks.CcID {
			cc := New(file, ref[i])
			result = cc.Get(file, blocks.CcVrTLookUp)
		}

		r = append(r, result)
	}
	return r
}

func interfaceArrayToStringArray(interfaceArray []interface{}) []string {
	stringArray := make([]string, len(interfaceArray))
	for i, v := range interfaceArray {
		stringArray[i] = v.(string)
	}
	return stringArray
}

func (b *Block) getVal() []float64 {
	return b.Data.Val
}

func (b *Block) dataType() uint8 {
	return b.Data.Type
}

func (b *Block) getRef() []int64 {
	return b.Link.Ref
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
