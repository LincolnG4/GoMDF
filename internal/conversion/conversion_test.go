package conversion

import (
	"math"
	"testing"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

func compile(t *testing.T, cc *blocks.CC, refs []Ref, intIn bool) *Conversion {
	t.Helper()
	c, err := Compile(cc, refs, intIn)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLinear(t *testing.T) {
	c := compile(t, &blocks.CC{Type: blocks.CCLinear, Vals: []float64{10, 2}}, nil, false)
	dst := make([]float64, 3)
	c.Convert(dst, []float64{0, 1, -4})
	if dst[0] != 10 || dst[1] != 12 || dst[2] != 2 {
		t.Errorf("linear = %v", dst)
	}
	// Identity-linear compiles to nil (no-op).
	if c := compile(t, &blocks.CC{Type: blocks.CCLinear, Vals: []float64{0, 1}}, nil, false); c != nil {
		t.Error("identity linear not elided")
	}
}

func TestRational(t *testing.T) {
	// (x^2 + 0x + 0) / (0x^2 + 0x + 2) = x²/2
	c := compile(t, &blocks.CC{Type: blocks.CCRational, Vals: []float64{1, 0, 0, 0, 0, 2}}, nil, false)
	dst := make([]float64, 1)
	c.Convert(dst, []float64{4})
	if dst[0] != 8 {
		t.Errorf("rational = %v", dst[0])
	}
}

func TestAlgebraic(t *testing.T) {
	cases := map[string]struct{ x, want float64 }{
		"2*X + 1":         {3, 7},
		"X^2":             {4, 16},
		"-X":              {2, -2},
		"sqrt(X)":         {9, 3},
		"pow(X, 3)":       {2, 8},
		"(X+1)*(X-1)":     {3, 8},
		"ln(X)":           {math.E, 1},
		"log(X)":          {100, 2},
		"1.5e2 + X":       {1, 151},
		"sin(X)":          {0, 0},
		"2 * (X + 3) / 4": {1, 2},
	}
	for formula, tc := range cases {
		c := compile(t, &blocks.CC{Type: blocks.CCAlgebraic}, []Ref{{Text: formula}}, false)
		dst := make([]float64, 1)
		c.Convert(dst, []float64{tc.x})
		if math.Abs(dst[0]-tc.want) > 1e-12 {
			t.Errorf("%q(%v) = %v, want %v", formula, tc.x, dst[0], tc.want)
		}
	}
	for _, bad := range []string{"", "X +", "foo(X)", "X)", "2 ** X", "Y"} {
		if _, err := Compile(&blocks.CC{Type: blocks.CCAlgebraic}, []Ref{{Text: bad}}, false); err == nil {
			t.Errorf("accepted bad formula %q", bad)
		}
	}
}

func TestTables(t *testing.T) {
	vals := []float64{0, 0, 10, 100, 20, 200} // keys 0,10,20 -> 0,100,200
	interp := compile(t, &blocks.CC{Type: blocks.CCTabInterp, Vals: vals}, nil, false)
	dst := make([]float64, 4)
	interp.Convert(dst, []float64{-5, 5, 15, 25})
	want := []float64{0, 50, 150, 200}
	for i := range want {
		if dst[i] != want[i] {
			t.Errorf("interp[%d] = %v, want %v", i, dst[i], want[i])
		}
	}
	tab := compile(t, &blocks.CC{Type: blocks.CCTab, Vals: vals}, nil, false)
	tab.Convert(dst, []float64{4, 6, 10, 99})
	want = []float64{0, 100, 100, 200}
	for i := range want {
		if dst[i] != want[i] {
			t.Errorf("tab[%d] = %v, want %v", i, dst[i], want[i])
		}
	}
}

func TestRangeToValue(t *testing.T) {
	// [0,10) -> 1, [10,20) -> 2, default 99
	vals := []float64{0, 10, 1, 10, 20, 2, 99}
	c := compile(t, &blocks.CC{Type: blocks.CCRangeToValue, Vals: vals}, nil, false)
	dst := make([]float64, 4)
	c.Convert(dst, []float64{5, 10, 20, -1})
	if dst[0] != 1 || dst[1] != 2 || dst[2] != 99 || dst[3] != 99 {
		t.Errorf("range = %v", dst)
	}
	// Integer semantics: upper bound inclusive.
	ci := compile(t, &blocks.CC{Type: blocks.CCRangeToValue, Vals: vals}, nil, true)
	ci.Convert(dst[:1], []float64{20})
	if dst[0] != 2 {
		t.Errorf("int range upper = %v, want 2", dst[0])
	}
}

func TestValueToText(t *testing.T) {
	cc := &blocks.CC{Type: blocks.CCValueToText, Vals: []float64{1, 2}}
	refs := []Ref{{Text: "one"}, {Text: "two"}, {Text: "other"}}
	c := compile(t, cc, refs, false)
	if !c.Textual() {
		t.Fatal("not textual")
	}
	got := c.Strings([]float64{2, 1, 7})
	if got[0] != "two" || got[1] != "one" || got[2] != "other" {
		t.Errorf("v2t = %v", got)
	}
	// The last real entry must not be dropped (old off-by-one).
	if got := c.Strings([]float64{2}); got[0] != "two" {
		t.Errorf("last entry dropped: %v", got)
	}
}

func TestRangeToText(t *testing.T) {
	cc := &blocks.CC{Type: blocks.CCRangeToText, Vals: []float64{0, 10, 10, 20}}
	refs := []Ref{{Text: "low"}, {Text: "high"}, {Text: "def"}}
	c := compile(t, cc, refs, false)
	got := c.Strings([]float64{5, 15, 25})
	if got[0] != "low" || got[1] != "high" || got[2] != "def" {
		t.Errorf("r2t = %v", got)
	}
}

func TestTextTables(t *testing.T) {
	t2v := compile(t, &blocks.CC{Type: blocks.CCTextToValue, Vals: []float64{1, 2, -1}},
		[]Ref{{Text: "a"}, {Text: "b"}}, false)
	got := t2v.FromText([]string{"b", "a", "zz"})
	if got[0] != 2 || got[1] != 1 || got[2] != -1 {
		t.Errorf("t2v = %v", got)
	}
	t2t := compile(t, &blocks.CC{Type: blocks.CCTextToText},
		[]Ref{{Text: "a"}, {Text: "A"}, {Text: "b"}, {Text: "B"}, {Text: ""}}, false)
	gs := t2t.TextToText([]string{"a", "b", "c"})
	if gs[0] != "A" || gs[1] != "B" || gs[2] != "c" {
		t.Errorf("t2t = %v", gs)
	}
}

func TestNestedRef(t *testing.T) {
	// A numeric default ref makes the table numeric-dominant: physical
	// values are numbers, text-matched keys become NaN.
	nested := compile(t, &blocks.CC{Type: blocks.CCLinear, Vals: []float64{0, 0.5}}, nil, false)
	cc := &blocks.CC{Type: blocks.CCValueToText, Vals: []float64{0}}
	c := compile(t, cc, []Ref{{Text: "zero"}, {Conv: nested}}, false)
	if c.Textual() {
		t.Fatal("numeric-dominant table reported textual")
	}
	dst := make([]float64, 2)
	c.Convert(dst, []float64{0, 8})
	if !math.IsNaN(dst[0]) || dst[1] != 4 {
		t.Errorf("nested numeric = %v", dst)
	}

	// A textual default keeps the table textual; nested numeric refs on
	// matched keys are rendered as text.
	c2 := compile(t, cc, []Ref{{Conv: nested}, {Text: "def"}}, false)
	if !c2.Textual() {
		t.Fatal("textual table reported numeric")
	}
	got := c2.Strings([]float64{0, 8})
	if got[0] != "0" || got[1] != "def" {
		t.Errorf("nested textual = %v", got)
	}
}

func TestNilIdentity(t *testing.T) {
	var c *Conversion
	dst := make([]float64, 2)
	c.Convert(dst, []float64{1, 2})
	if dst[0] != 1 || dst[1] != 2 {
		t.Errorf("nil identity = %v", dst)
	}
	if c.Textual() || c.TextInput() {
		t.Error("nil conversion claims text")
	}
}

func FuzzFormula(f *testing.F) {
	for _, s := range []string{"2*X+1", "pow(X,2)", "sin(cos(X))", "((", "1e", "-", "X^X^X"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, formula string) {
		fn, err := compileFormula(formula)
		if err == nil {
			fn(1.0) // must not panic
		}
	})
}
