package conversion

import "sort"

// tabLookup implements value-to-value tables (cc_type 4 and 5). keys are
// sorted ascending per spec. Outside the key range the edge value is
// used.
func tabLookup(keys, vals []float64, x float64, interp bool) float64 {
	n := len(keys)
	if x <= keys[0] {
		return vals[0]
	}
	if x >= keys[n-1] {
		return vals[n-1]
	}
	// First index with key > x; x lies in [keys[i-1], keys[i]).
	i := sort.SearchFloat64s(keys, x)
	if keys[i] == x {
		return vals[i]
	}
	if interp {
		k0, k1 := keys[i-1], keys[i]
		v0, v1 := vals[i-1], vals[i]
		return v0 + (v1-v0)*(x-k0)/(k1-k0)
	}
	// No interpolation: value of the closest key; ties take the lower.
	if x-keys[i-1] <= keys[i]-x {
		return vals[i-1]
	}
	return vals[i]
}

// rangeTable implements range lookup for cc_type 6 and 8. Per spec the
// upper bound is inclusive for integer inputs and exclusive for reals.
type rangeTable struct {
	min, max []float64
	n        int
	intIn    bool
}

// find returns the first matching range index or -1.
func (t *rangeTable) find(x float64) int {
	for i := 0; i < t.n; i++ {
		if t.intIn {
			if t.min[i] <= x && x <= t.max[i] {
				return i
			}
		} else if t.min[i] <= x && x < t.max[i] {
			return i
		}
	}
	return -1
}
