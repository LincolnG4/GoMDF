// Package conversion compiles MDF channel conversion (CC) blocks into
// batch operations over typed slices.
//
// A conversion is compiled once per channel — table keys sorted, algebraic
// formulas parsed to a closure tree — and then applied to whole columns.
package conversion

import (
	"fmt"
	"math"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

// Ref is one resolved cc_ref entry: either the text of a TX block or a
// nested compiled conversion.
type Ref struct {
	Text string
	Conv *Conversion // non-nil when the ref was a nested CC block
}

// Conversion is a compiled conversion rule.
type Conversion struct {
	typ uint8

	numeric  func(dst, src []float64)     // types 1-6 (and identity)
	toText   func(src []float64) []string // types 7, 8, 11
	fromText func(src []string) []float64 // type 9
	textText func(src []string) []string  // type 10
}

// Textual reports whether the conversion outputs text.
func (c *Conversion) Textual() bool {
	return c != nil && c.toText != nil || c != nil && c.textText != nil
}

// TextInput reports whether the conversion consumes text (string channels).
func (c *Conversion) TextInput() bool {
	return c != nil && (c.typ == blocks.CCTextToValue || c.typ == blocks.CCTextToText)
}

// Convert applies a numeric conversion. dst and src may alias. A nil
// Conversion is the identity.
func (c *Conversion) Convert(dst, src []float64) {
	if c == nil || c.numeric == nil {
		copy(dst, src)
		return
	}
	c.numeric(dst, src)
}

// Strings applies a value/range/bitfield-to-text conversion.
func (c *Conversion) Strings(src []float64) []string {
	return c.toText(src)
}

// FromText applies a text-to-value conversion.
func (c *Conversion) FromText(src []string) []float64 { return c.fromText(src) }

// TextToText applies a text-to-text conversion.
func (c *Conversion) TextToText(src []string) []string { return c.textText(src) }

// convertOne applies a numeric conversion to a single value (used for
// nested refs).
func (c *Conversion) convertOne(x float64) float64 {
	if c == nil || c.numeric == nil {
		return x
	}
	var out [1]float64
	c.numeric(out[:], []float64{x})
	return out[0]
}

// stringOne renders one value through a textual nested conversion.
func (c *Conversion) stringOne(x float64) string {
	if c.toText != nil {
		return c.toText([]float64{x})[0]
	}
	return fmt.Sprintf("%v", c.convertOne(x))
}

// Compile builds a batch conversion from a CC block and its resolved
// references. intInput selects the inclusive range-compare semantics the
// spec mandates for integer channels. cc == nil returns nil (identity).
func Compile(cc *blocks.CC, refs []Ref, intInput bool) (*Conversion, error) {
	if cc == nil {
		return nil, nil
	}
	c := &Conversion{typ: cc.Type}
	v := cc.Vals
	need := func(n int) error {
		if len(v) < n {
			return fmt.Errorf("conversion type %d: %d values, want %d", cc.Type, len(v), n)
		}
		return nil
	}
	switch cc.Type {
	case blocks.CCIdentity:
		return nil, nil

	case blocks.CCLinear:
		if err := need(2); err != nil {
			return nil, err
		}
		b, a := v[0], v[1]
		if a == 1 && b == 0 {
			return nil, nil
		}
		c.numeric = func(dst, src []float64) {
			for i, x := range src {
				dst[i] = a*x + b
			}
		}

	case blocks.CCRational:
		if err := need(6); err != nil {
			return nil, err
		}
		p2, p1, p0, q2, q1, q0 := v[0], v[1], v[2], v[3], v[4], v[5]
		if p2 == 0 && p1 == 1 && p0 == 0 && q2 == 0 && q1 == 0 && q0 == 1 {
			return nil, nil
		}
		c.numeric = func(dst, src []float64) {
			for i, x := range src {
				dst[i] = (p2*x*x + p1*x + p0) / (q2*x*x + q1*x + q0)
			}
		}

	case blocks.CCAlgebraic:
		if len(refs) < 1 {
			return nil, fmt.Errorf("algebraic conversion without formula ref")
		}
		f, err := compileFormula(refs[0].Text)
		if err != nil {
			return nil, err
		}
		c.numeric = func(dst, src []float64) {
			for i, x := range src {
				dst[i] = f(x)
			}
		}

	case blocks.CCTabInterp, blocks.CCTab:
		n := len(v) / 2
		if n == 0 {
			return nil, fmt.Errorf("empty conversion table")
		}
		keys := make([]float64, n)
		vals := make([]float64, n)
		for i := 0; i < n; i++ {
			keys[i], vals[i] = v[2*i], v[2*i+1]
		}
		interp := cc.Type == blocks.CCTabInterp
		c.numeric = func(dst, src []float64) {
			for i, x := range src {
				dst[i] = tabLookup(keys, vals, x, interp)
			}
		}

	case blocks.CCRangeToValue:
		n := len(v) / 3
		if err := need(3*n + 1); err != nil {
			return nil, err
		}
		t := rangeTable{n: n, intIn: intInput}
		for i := 0; i < n; i++ {
			t.min = append(t.min, v[3*i])
			t.max = append(t.max, v[3*i+1])
		}
		vals := make([]float64, n)
		for i := 0; i < n; i++ {
			vals[i] = v[3*i+2]
		}
		def := v[3*n]
		c.numeric = func(dst, src []float64) {
			for i, x := range src {
				if j := t.find(x); j >= 0 {
					dst[i] = vals[j]
				} else {
					dst[i] = def
				}
			}
		}

	case blocks.CCValueToText:
		n := len(v)
		if len(refs) < n+1 {
			return nil, fmt.Errorf("value-to-text: %d refs for %d keys", len(refs), n)
		}
		keys := v
		texts := refs[:n]
		def := refs[n]
		if numericDominant(def) {
			c.numeric = func(dst, src []float64) {
				for i, x := range src {
					r := def
					for j, k := range keys {
						if x == k {
							r = texts[j]
							break
						}
					}
					dst[i] = renderNumericRef(r, x)
				}
			}
		} else {
			c.toText = func(src []float64) []string {
				out := make([]string, len(src))
				for i, x := range src {
					r := def
					for j, k := range keys {
						if x == k {
							r = texts[j]
							break
						}
					}
					out[i] = renderRef(r, x)
				}
				return out
			}
		}

	case blocks.CCRangeToText:
		n := len(v) / 2
		if len(refs) < n+1 {
			return nil, fmt.Errorf("range-to-text: %d refs for %d ranges", len(refs), n)
		}
		t := rangeTable{n: n, intIn: intInput}
		for i := 0; i < n; i++ {
			t.min = append(t.min, v[2*i])
			t.max = append(t.max, v[2*i+1])
		}
		texts := refs[:n]
		def := refs[n]
		if numericDominant(def) {
			c.numeric = func(dst, src []float64) {
				for i, x := range src {
					r := def
					if j := t.find(x); j >= 0 {
						r = texts[j]
					}
					dst[i] = renderNumericRef(r, x)
				}
			}
		} else {
			c.toText = func(src []float64) []string {
				out := make([]string, len(src))
				for i, x := range src {
					r := def
					if j := t.find(x); j >= 0 {
						r = texts[j]
					}
					out[i] = renderRef(r, x)
				}
				return out
			}
		}

	case blocks.CCTextToValue:
		n := len(refs)
		if err := need(n + 1); err != nil {
			return nil, err
		}
		m := make(map[string]float64, n)
		for i, r := range refs {
			m[r.Text] = v[i]
		}
		def := v[n]
		c.fromText = func(src []string) []float64 {
			out := make([]float64, len(src))
			for i, s := range src {
				if val, ok := m[s]; ok {
					out[i] = val
				} else {
					out[i] = def
				}
			}
			return out
		}

	case blocks.CCTextToText:
		n := len(refs) / 2
		m := make(map[string]string, n)
		for i := 0; i < n; i++ {
			m[refs[2*i].Text] = refs[2*i+1].Text
		}
		var def string
		hasDef := len(refs) > 2*n && refs[2*n].Text != ""
		if hasDef {
			def = refs[2*n].Text
		}
		c.textText = func(src []string) []string {
			out := make([]string, len(src))
			for i, s := range src {
				if t, ok := m[s]; ok {
					out[i] = t
				} else if hasDef {
					out[i] = def
				} else {
					out[i] = s
				}
			}
			return out
		}

	case blocks.CCBitfieldToText:
		// cc_val are bit masks (as uint64); each ref is typically a nested
		// value-to-text conversion applied to the masked value.
		masks := cc.RawVals
		if len(refs) < len(masks) {
			return nil, fmt.Errorf("bitfield-to-text: %d refs for %d masks", len(refs), len(masks))
		}
		c.toText = func(src []float64) []string {
			out := make([]string, len(src))
			for i, x := range src {
				raw := uint64(x)
				s := ""
				for j, m := range masks {
					part := renderRef(refs[j], float64(raw&m))
					if part == "" {
						continue
					}
					if s != "" {
						s += " | "
					}
					s += part
				}
				out[i] = s
			}
			return out
		}

	default:
		return nil, fmt.Errorf("unsupported conversion type %d", cc.Type)
	}
	return c, nil
}

// numericDominant reports whether the default ref of a value/range-to-
// text table is a numeric conversion. In that common automotive pattern
// (numeric signal with a few textual error codes) the physical values
// are numbers, so the whole conversion is treated as numeric and
// text-matched samples become NaN.
func numericDominant(def Ref) bool {
	return def.Conv != nil && !def.Conv.Textual()
}

// renderNumericRef resolves one ref numerically: nested numeric
// conversions are applied, text entries become NaN.
func renderNumericRef(r Ref, x float64) float64 {
	if r.Conv != nil && !r.Conv.Textual() {
		return r.Conv.convertOne(x)
	}
	return math.NaN()
}

// renderRef renders one ref for value x: plain text, or a nested
// conversion applied to x.
func renderRef(r Ref, x float64) string {
	if r.Conv != nil {
		return r.Conv.stringOne(x)
	}
	return r.Text
}
