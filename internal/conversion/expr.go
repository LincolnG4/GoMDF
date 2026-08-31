package conversion

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// exprFunc evaluates a compiled algebraic formula for one raw value.
type exprFunc func(x float64) float64

// compileFormula parses an MDF algebraic conversion formula (MCD-2 MC
// expression syntax subset: numbers, the variable X, + - * / ^, parens
// and common math functions) once into a closure tree.
func compileFormula(formula string) (exprFunc, error) {
	p := &exprParser{src: formula}
	f, err := p.parseExpr()
	if err != nil {
		return nil, fmt.Errorf("formula %q: %w", formula, err)
	}
	p.skipSpace()
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("formula %q: unexpected %q at %d", formula, p.src[p.pos:], p.pos)
	}
	return f, nil
}

type exprParser struct {
	src string
	pos int
}

func (p *exprParser) skipSpace() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t' || p.src[p.pos] == '\n' || p.src[p.pos] == '\r') {
		p.pos++
	}
}

func (p *exprParser) peek() byte {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return 0
	}
	return p.src[p.pos]
}

func (p *exprParser) parseExpr() (exprFunc, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek() {
		case '+':
			p.pos++
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) + r(x) }
		case '-':
			p.pos++
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) - r(x) }
		default:
			return left, nil
		}
	}
}

func (p *exprParser) parseTerm() (exprFunc, error) {
	left, err := p.parsePower()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek() {
		case '*':
			p.pos++
			right, err := p.parsePower()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) * r(x) }
		case '/':
			p.pos++
			right, err := p.parsePower()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) / r(x) }
		default:
			return left, nil
		}
	}
}

func (p *exprParser) parsePower() (exprFunc, error) {
	base, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if p.peek() == '^' {
		p.pos++
		exp, err := p.parsePower() // right associative
		if err != nil {
			return nil, err
		}
		b, e := base, exp
		return func(x float64) float64 { return math.Pow(b(x), e(x)) }, nil
	}
	return base, nil
}

func (p *exprParser) parseUnary() (exprFunc, error) {
	switch p.peek() {
	case '-':
		p.pos++
		f, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return func(x float64) float64 { return -f(x) }, nil
	case '+':
		p.pos++
		return p.parseUnary()
	}
	return p.parsePrimary()
}

var exprFuncs = map[string]func(float64) float64{
	"sqrt":  math.Sqrt,
	"abs":   math.Abs,
	"exp":   math.Exp,
	"ln":    math.Log,
	"log":   math.Log10,
	"sin":   math.Sin,
	"cos":   math.Cos,
	"tan":   math.Tan,
	"asin":  math.Asin,
	"acos":  math.Acos,
	"atan":  math.Atan,
	"sinh":  math.Sinh,
	"cosh":  math.Cosh,
	"tanh":  math.Tanh,
	"floor": math.Floor,
	"ceil":  math.Ceil,
}

func (p *exprParser) parsePrimary() (exprFunc, error) {
	c := p.peek()
	switch {
	case c == '(':
		p.pos++
		f, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, fmt.Errorf("missing ) at %d", p.pos)
		}
		p.pos++
		return f, nil
	case c >= '0' && c <= '9' || c == '.':
		start := p.pos
		for p.pos < len(p.src) {
			ch := p.src[p.pos]
			if ch >= '0' && ch <= '9' || ch == '.' {
				p.pos++
			} else if ch == 'e' || ch == 'E' {
				// exponent, possibly signed
				next := p.pos + 1
				if next < len(p.src) && (p.src[next] == '+' || p.src[next] == '-') {
					next++
				}
				if next < len(p.src) && p.src[next] >= '0' && p.src[next] <= '9' {
					p.pos = next
				} else {
					break
				}
			} else {
				break
			}
		}
		v, err := strconv.ParseFloat(p.src[start:p.pos], 64)
		if err != nil {
			return nil, fmt.Errorf("bad number at %d: %w", start, err)
		}
		return func(float64) float64 { return v }, nil
	case unicode.IsLetter(rune(c)):
		start := p.pos
		for p.pos < len(p.src) && (unicode.IsLetter(rune(p.src[p.pos])) || unicode.IsDigit(rune(p.src[p.pos])) || p.src[p.pos] == '_') {
			p.pos++
		}
		name := p.src[start:p.pos]
		// The raw value variable: X or X1.
		if name == "X" || name == "x" || name == "X1" {
			return func(x float64) float64 { return x }, nil
		}
		lower := strings.ToLower(name)
		if p.peek() == '(' {
			p.pos++
			args := []exprFunc{}
			if p.peek() != ')' {
				for {
					a, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, a)
					if p.peek() != ',' {
						break
					}
					p.pos++
				}
			}
			if p.peek() != ')' {
				return nil, fmt.Errorf("missing ) after %s(", name)
			}
			p.pos++
			if lower == "pow" && len(args) == 2 {
				b, e := args[0], args[1]
				return func(x float64) float64 { return math.Pow(b(x), e(x)) }, nil
			}
			if fn, ok := exprFuncs[lower]; ok && len(args) == 1 {
				a := args[0]
				return func(x float64) float64 { return fn(a(x)) }, nil
			}
			return nil, fmt.Errorf("unknown function %q with %d args", name, len(args))
		}
		return nil, fmt.Errorf("unknown identifier %q", name)
	case c == 0:
		return nil, fmt.Errorf("unexpected end of formula")
	default:
		return nil, fmt.Errorf("unexpected character %q at %d", c, p.pos)
	}
}
