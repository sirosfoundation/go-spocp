package compare

import (
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
	"github.com/sirosfoundation/go-spocp/pkg/starform"
)

// Test cases from section 5.2 Example 1 of the spec
func TestSpecExample1Comparison(t *testing.T) {
	// x = (http (page index.html)(action GET)(user olav))
	x := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("GET")),
		sexp.NewList("user", sexp.NewAtom("olav")),
	)

	// y = (http (page index.html)(action GET)(user))
	y := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("GET")),
		sexp.NewList("user"),
	)

	// x <= y should be true
	if !LessPermissive(x, y) {
		t.Error("Expected x <= y to be true")
	}

	// z = (http (page index.html)(action)(user olav))
	z := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action"),
		sexp.NewList("user", sexp.NewAtom("olav")),
	)

	// x <= z should be true
	if !LessPermissive(x, z) {
		t.Error("Expected x <= z to be true")
	}

	// y and z should be unrelated (y <= z and z <= y both false)
	if LessPermissive(y, z) {
		t.Error("Expected y <= z to be false (unrelated)")
	}
	if LessPermissive(z, y) {
		t.Error("Expected z <= y to be false (unrelated)")
	}
}

// Test from section 6 examples
func TestSpecSection6Examples(t *testing.T) {
	// (fruit apple large red) <= (fruit apple)
	s1 := sexp.NewList("fruit",
		sexp.NewAtom("apple"),
		sexp.NewAtom("large"),
		sexp.NewAtom("red"),
	)
	t1 := sexp.NewList("fruit", sexp.NewAtom("apple"))

	if !LessPermissive(s1, t1) {
		t.Error("Expected (fruit apple large red) <= (fruit apple)")
	}

	// (fruit apple (size large) red) <= (fruit apple (size) red)
	s2 := sexp.NewList("fruit",
		sexp.NewAtom("apple"),
		sexp.NewList("size", sexp.NewAtom("large")),
		sexp.NewAtom("red"),
	)
	t2 := sexp.NewList("fruit",
		sexp.NewAtom("apple"),
		sexp.NewList("size"),
		sexp.NewAtom("red"),
	)

	if !LessPermissive(s2, t2) {
		t.Error("Expected (fruit apple (size large) red) <= (fruit apple (size) red)")
	}
}

func TestAtomComparison(t *testing.T) {
	tests := []struct {
		name     string
		s        sexp.Element
		t        sexp.Element
		expected bool
	}{
		{
			"equal atoms",
			sexp.NewAtom("test"),
			sexp.NewAtom("test"),
			true,
		},
		{
			"different atoms",
			sexp.NewAtom("test"),
			sexp.NewAtom("other"),
			false,
		},
		{
			"atom vs wildcard",
			sexp.NewAtom("anything"),
			&starform.Wildcard{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive(%v, %v) = %v, want %v",
					sexp.AdvancedForm(tt.s),
					sexp.AdvancedForm(tt.t),
					result, tt.expected)
			}
		})
	}
}

func TestPrefixComparison(t *testing.T) {
	// From spec section 5.3.4: (file (* prefix conf))
	// This should match any file starting with "conf"

	tests := []struct {
		name     string
		s        sexp.Element
		t        sexp.Element
		expected bool
	}{
		{
			"atom matches prefix",
			sexp.NewAtom("config.txt"),
			&starform.Prefix{Value: "conf"},
			true,
		},
		{
			"atom doesn't match prefix",
			sexp.NewAtom("data.txt"),
			&starform.Prefix{Value: "conf"},
			false,
		},
		{
			"more specific prefix <= less specific prefix",
			&starform.Prefix{Value: "config"},
			&starform.Prefix{Value: "conf"},
			true,
		},
		{
			"less specific prefix not <= more specific",
			&starform.Prefix{Value: "conf"},
			&starform.Prefix{Value: "config"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSuffixComparison(t *testing.T) {
	// From spec section 5.3.5: (file (* suffix pdf))

	tests := []struct {
		name     string
		s        sexp.Element
		t        sexp.Element
		expected bool
	}{
		{
			"atom matches suffix",
			sexp.NewAtom("document.pdf"),
			&starform.Suffix{Value: "pdf"},
			true,
		},
		{
			"atom doesn't match suffix",
			sexp.NewAtom("document.txt"),
			&starform.Suffix{Value: "pdf"},
			false,
		},
		{
			"more specific suffix <= less specific suffix",
			&starform.Suffix{Value: ".pdf"},
			&starform.Suffix{Value: "pdf"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSetComparison(t *testing.T) {
	// From spec section 5.3.2: (* set apple orange lemon)

	set := &starform.Set{
		Elements: []sexp.Element{
			sexp.NewAtom("apple"),
			sexp.NewAtom("orange"),
			sexp.NewAtom("lemon"),
		},
	}

	tests := []struct {
		name     string
		s        sexp.Element
		t        sexp.Element
		expected bool
	}{
		{
			"atom in set",
			sexp.NewAtom("apple"),
			set,
			true,
		},
		{
			"atom not in set",
			sexp.NewAtom("banana"),
			set,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRangeComparison(t *testing.T) {
	// From spec: (* range numeric ge 10 le 15)
	// which is the same as (* set 10 11 12 13 14 15)

	numRange := &starform.Range{
		RangeType: starform.RangeNumeric,
		LowerBound: &starform.RangeBound{
			Op:    starform.OpGE,
			Value: "10",
		},
		UpperBound: &starform.RangeBound{
			Op:    starform.OpLE,
			Value: "15",
		},
	}

	tests := []struct {
		name     string
		s        sexp.Element
		t        sexp.Element
		expected bool
	}{
		{
			"value in range",
			sexp.NewAtom("12"),
			numRange,
			true,
		},
		{
			"value below range",
			sexp.NewAtom("5"),
			numRange,
			false,
		},
		{
			"value above range",
			sexp.NewAtom("20"),
			numRange,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestListOrderMatters(t *testing.T) {
	// From spec section 6: order is absolutely vital

	s1 := sexp.NewList("fruit",
		sexp.NewAtom("apple"),
		sexp.NewAtom("large"),
		sexp.NewAtom("red"),
	)

	// Different order - should NOT match
	t1 := sexp.NewList("fruit",
		sexp.NewAtom("apple"),
		sexp.NewAtom("red"),
		sexp.NewAtom("large"),
	)

	if LessPermissive(s1, t1) {
		t.Error("Order matters: (fruit apple large red) should NOT be <= (fruit apple red large)")
	}

	// (apple (weight 100)(color red)) is not <= (apple (color red)(weight 100))
	s2 := sexp.NewList("apple",
		sexp.NewList("weight", sexp.NewAtom("100")),
		sexp.NewList("color", sexp.NewAtom("red")),
	)

	t2 := sexp.NewList("apple",
		sexp.NewList("color", sexp.NewAtom("red")),
		sexp.NewList("weight", sexp.NewAtom("100")),
	)

	if LessPermissive(s2, t2) {
		t.Error("Order matters in nested lists too")
	}
}

// Tests for compareRanges function
func TestCompareRanges(t *testing.T) {
	tests := []struct {
		name     string
		s        *starform.Range
		tRange   *starform.Range
		expected bool
	}{
		{
			name: "equal ranges",
			s: &starform.Range{
				RangeType:  starform.RangeNumeric,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "10"},
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "15"},
			},
			tRange: &starform.Range{
				RangeType:  starform.RangeNumeric,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "10"},
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "15"},
			},
			expected: true,
		},
		{
			name: "different range types",
			s: &starform.Range{
				RangeType:  starform.RangeNumeric,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "10"},
			},
			tRange: &starform.Range{
				RangeType:  starform.RangeAlpha,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "10"},
			},
			expected: false,
		},
		{
			name: "T has lower bound, S doesn't - S NOT <= T",
			s: &starform.Range{
				RangeType:  starform.RangeNumeric,
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "20"},
			},
			tRange: &starform.Range{
				RangeType:  starform.RangeNumeric,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "10"},
			},
			expected: false,
		},
		{
			name: "T has upper bound, S doesn't - S NOT <= T",
			s: &starform.Range{
				RangeType:  starform.RangeNumeric,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "5"},
			},
			tRange: &starform.Range{
				RangeType:  starform.RangeNumeric,
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "15"},
			},
			expected: false,
		},
		{
			name: "narrower upper bound fails",
			s: &starform.Range{
				RangeType:  starform.RangeAlpha,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "c"},
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "z"},
			},
			tRange: &starform.Range{
				RangeType:  starform.RangeAlpha,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "c"},
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "m"},
			},
			expected: false,
		},
		{
			name: "S lower bound higher than T lower bound fails",
			s: &starform.Range{
				RangeType:  starform.RangeAlpha,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "a"},
			},
			tRange: &starform.Range{
				RangeType:  starform.RangeAlpha,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "m"},
			},
			expected: false,
		},
		{
			name: "S within T bounds succeeds",
			s: &starform.Range{
				RangeType:  starform.RangeAlpha,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "m"},
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "p"},
			},
			tRange: &starform.Range{
				RangeType:  starform.RangeAlpha,
				LowerBound: &starform.RangeBound{Op: starform.OpGE, Value: "a"},
				UpperBound: &starform.RangeBound{Op: starform.OpLE, Value: "z"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.tRange)
			if result != tt.expected {
				t.Errorf("LessPermissive(%v, %v) = %v, want %v",
					tt.s.String(), tt.tRange.String(), result, tt.expected)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input sexp.Element
	}{
		{
			name:  "atom unchanged",
			input: sexp.NewAtom("test"),
		},
		{
			name:  "list unchanged",
			input: sexp.NewList("tag", sexp.NewAtom("value")),
		},
		{
			name:  "wildcard unchanged",
			input: &starform.Wildcard{},
		},
		{
			name: "set unchanged for now",
			input: &starform.Set{
				Elements: []sexp.Element{
					sexp.NewAtom("a"),
					sexp.NewAtom("b"),
				},
			},
		},
		{
			name:  "prefix unchanged",
			input: &starform.Prefix{Value: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.input)
			if result == nil {
				t.Error("Normalize() returned nil")
			}
		})
	}
}

func TestWildcardComparisons(t *testing.T) {
	wildcard := &starform.Wildcard{}

	tests := []struct {
		name     string
		s        sexp.Element
		t        sexp.Element
		expected bool
	}{
		{
			name:     "wildcard <= wildcard",
			s:        wildcard,
			t:        wildcard,
			expected: true,
		},
		{
			name:     "wildcard NOT <= atom",
			s:        wildcard,
			t:        sexp.NewAtom("test"),
			expected: false,
		},
		{
			name:     "wildcard NOT <= range",
			s:        wildcard,
			t:        &starform.Range{RangeType: starform.RangeNumeric},
			expected: false,
		},
		{
			name:     "wildcard NOT <= prefix",
			s:        wildcard,
			t:        &starform.Prefix{Value: "test"},
			expected: false,
		},
		{
			name:     "prefix <= wildcard",
			s:        &starform.Prefix{Value: "test"},
			t:        wildcard,
			expected: true,
		},
		{
			name:     "range <= wildcard",
			s:        &starform.Range{RangeType: starform.RangeNumeric},
			t:        wildcard,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSetAsSource(t *testing.T) {
	// Rule 8: S is a set, all elements of S <= T
	sourceSet := &starform.Set{
		Elements: []sexp.Element{
			sexp.NewAtom("apple"),
			sexp.NewAtom("apricot"),
		},
	}

	tests := []struct {
		name     string
		t        sexp.Element
		expected bool
	}{
		{
			name:     "set <= wildcard",
			t:        &starform.Wildcard{},
			expected: true,
		},
		{
			name:     "set NOT <= atom (only one element matches)",
			t:        sexp.NewAtom("apple"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(sourceSet, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive(set, %v) = %v, want %v", tt.t, result, tt.expected)
			}
		})
	}
}

func TestMixedStarFormComparisons(t *testing.T) {
	// Test comparing different star form types (that should return false)
	tests := []struct {
		name     string
		s        sexp.Element
		t        sexp.Element
		expected bool
	}{
		{
			name:     "prefix NOT <= suffix",
			s:        &starform.Prefix{Value: "test"},
			t:        &starform.Suffix{Value: "test"},
			expected: false,
		},
		{
			name:     "suffix NOT <= prefix",
			s:        &starform.Suffix{Value: "test"},
			t:        &starform.Prefix{Value: "test"},
			expected: false,
		},
		{
			name:     "range NOT <= prefix",
			s:        &starform.Range{RangeType: starform.RangeNumeric},
			t:        &starform.Prefix{Value: "test"},
			expected: false,
		},
		{
			name:     "prefix NOT <= range",
			s:        &starform.Prefix{Value: "10"},
			t:        &starform.Range{RangeType: starform.RangeNumeric},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LessPermissive(tt.s, tt.t)
			if result != tt.expected {
				t.Errorf("LessPermissive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestListWithDifferentTags(t *testing.T) {
	s := sexp.NewList("apple", sexp.NewAtom("red"))
	tList := sexp.NewList("orange", sexp.NewAtom("red"))

	if LessPermissive(s, tList) {
		t.Error("Lists with different tags should not be comparable")
	}
}

func TestShorterListNotLessPermissive(t *testing.T) {
	// S has fewer elements than T
	s := sexp.NewList("fruit", sexp.NewAtom("apple"))
	tList := sexp.NewList("fruit", sexp.NewAtom("apple"), sexp.NewAtom("large"))

	if LessPermissive(s, tList) {
		t.Error("Shorter list should NOT be <= longer list")
	}
}
