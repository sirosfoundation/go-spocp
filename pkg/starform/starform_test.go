package starform

import (
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func TestWildcard(t *testing.T) {
	wc := &Wildcard{}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches atom",
			elem: sexp.NewAtom("test"),
			want: true,
		},
		{
			name: "matches list",
			elem: sexp.NewList("tag", sexp.NewAtom("value")),
			want: true,
		},
		{
			name: "matches nil",
			elem: nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wc.Match(tt.elem); got != tt.want {
				t.Errorf("Wildcard.Match() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test String()
	if got := wc.String(); got != "(1:*)" {
		t.Errorf("Wildcard.String() = %v, want (1:*)", got)
	}

	// Test type checks
	if !wc.IsStarForm() {
		t.Error("Wildcard.IsStarForm() = false, want true")
	}
	if wc.IsAtom() {
		t.Error("Wildcard.IsAtom() = true, want false")
	}
	if wc.IsList() {
		t.Error("Wildcard.IsList() = true, want false")
	}
}

func TestSet(t *testing.T) {
	set := &Set{
		Elements: []sexp.Element{
			sexp.NewAtom("read"),
			sexp.NewAtom("write"),
			sexp.NewAtom("execute"),
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches element in set",
			elem: sexp.NewAtom("read"),
			want: true,
		},
		{
			name: "matches another element in set",
			elem: sexp.NewAtom("write"),
			want: true,
		},
		{
			name: "does not match element not in set",
			elem: sexp.NewAtom("delete"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Match(tt.elem); got != tt.want {
				t.Errorf("Set.Match() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test String()
	str := set.String()
	if len(str) == 0 {
		t.Error("Set.String() returned empty string")
	}

	// Test type checks
	if !set.IsStarForm() {
		t.Error("Set.IsStarForm() = false, want true")
	}
	if set.IsAtom() {
		t.Error("Set.IsAtom() = true, want false")
	}
	if set.IsList() {
		t.Error("Set.IsList() = true, want false")
	}
}

func TestPrefix(t *testing.T) {
	prefix := &Prefix{Value: "test"}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches atom with prefix",
			elem: sexp.NewAtom("testing"),
			want: true,
		},
		{
			name: "matches exact value",
			elem: sexp.NewAtom("test"),
			want: true,
		},
		{
			name: "does not match atom without prefix",
			elem: sexp.NewAtom("other"),
			want: false,
		},
		{
			name: "does not match shorter string",
			elem: sexp.NewAtom("tes"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prefix.Match(tt.elem); got != tt.want {
				t.Errorf("Prefix.Match() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test String()
	if got := prefix.String(); got != "(1:*6:prefix4:test)" {
		t.Errorf("Prefix.String() = %v, want (1:*6:prefix4:test)", got)
	}

	// Test type checks
	if !prefix.IsStarForm() {
		t.Error("Prefix.IsStarForm() = false, want true")
	}
}

func TestSuffix(t *testing.T) {
	suffix := &Suffix{Value: ".txt"}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches atom with suffix",
			elem: sexp.NewAtom("file.txt"),
			want: true,
		},
		{
			name: "matches exact value",
			elem: sexp.NewAtom(".txt"),
			want: true,
		},
		{
			name: "does not match atom without suffix",
			elem: sexp.NewAtom("file.doc"),
			want: false,
		},
		{
			name: "does not match shorter string",
			elem: sexp.NewAtom("txt"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := suffix.Match(tt.elem); got != tt.want {
				t.Errorf("Suffix.Match() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test String()
	if got := suffix.String(); got != "(1:*6:suffix4:.txt)" {
		t.Errorf("Suffix.String() = %v, want (1:*6:suffix4:.txt)", got)
	}

	// Test type checks
	if !suffix.IsStarForm() {
		t.Error("Suffix.IsStarForm() = false, want true")
	}
}

func TestRange_Numeric(t *testing.T) {
	numRange := &Range{
		RangeType: RangeNumeric,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "10",
		},
		UpperBound: &RangeBound{
			Op:    OpLE,
			Value: "20",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches value in range",
			elem: sexp.NewAtom("15"),
			want: true,
		},
		{
			name: "matches lower bound",
			elem: sexp.NewAtom("10"),
			want: true,
		},
		{
			name: "matches upper bound",
			elem: sexp.NewAtom("20"),
			want: true,
		},
		{
			name: "does not match value below range",
			elem: sexp.NewAtom("5"),
			want: false,
		},
		{
			name: "does not match value above range",
			elem: sexp.NewAtom("25"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := numRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test String()
	str := numRange.String()
	if len(str) == 0 {
		t.Error("Range.String() returned empty string")
	}

	// Test type checks
	if !numRange.IsStarForm() {
		t.Error("Range.IsStarForm() = false, want true")
	}
}

func TestRange_Time(t *testing.T) {
	timeRange := &Range{
		RangeType: RangeTime,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "09:00:00",
		},
		UpperBound: &RangeBound{
			Op:    OpLE,
			Value: "17:00:00",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches time in range",
			elem: sexp.NewAtom("12:00:00"),
			want: true,
		},
		{
			name: "matches lower bound",
			elem: sexp.NewAtom("09:00:00"),
			want: true,
		},
		{
			name: "matches upper bound",
			elem: sexp.NewAtom("17:00:00"),
			want: true,
		},
		{
			name: "does not match time before range",
			elem: sexp.NewAtom("08:00:00"),
			want: false,
		},
		{
			name: "does not match time after range",
			elem: sexp.NewAtom("18:00:00"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v for time %v", got, tt.want, tt.elem)
			}
		})
	}
}

func TestRange_Alpha(t *testing.T) {
	alphaRange := &Range{
		RangeType: RangeAlpha,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "a",
		},
		UpperBound: &RangeBound{
			Op:    OpLE,
			Value: "z",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches alpha in range",
			elem: sexp.NewAtom("m"),
			want: true,
		},
		{
			name: "matches lower bound",
			elem: sexp.NewAtom("a"),
			want: true,
		},
		{
			name: "matches upper bound",
			elem: sexp.NewAtom("z"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := alphaRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test Type() methods for all star forms
func TestStarFormTypes(t *testing.T) {
	tests := []struct {
		name     string
		starForm StarForm
		wantType string
	}{
		{
			name:     "wildcard type",
			starForm: &Wildcard{},
			wantType: "wildcard",
		},
		{
			name:     "set type",
			starForm: &Set{Elements: []sexp.Element{sexp.NewAtom("a")}},
			wantType: "set",
		},
		{
			name:     "range type",
			starForm: &Range{RangeType: RangeNumeric},
			wantType: "range",
		},
		{
			name:     "prefix type",
			starForm: &Prefix{Value: "test"},
			wantType: "prefix",
		},
		{
			name:     "suffix type",
			starForm: &Suffix{Value: ".txt"},
			wantType: "suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.starForm.Type(); got != tt.wantType {
				t.Errorf("Type() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

// Test IsAtom/IsList methods for Range, Prefix, Suffix
func TestRangeIsAtomIsList(t *testing.T) {
	r := &Range{RangeType: RangeAlpha}

	if r.IsAtom() {
		t.Error("Range.IsAtom() = true, want false")
	}
	if r.IsList() {
		t.Error("Range.IsList() = true, want false")
	}
}

func TestPrefixIsAtomIsList(t *testing.T) {
	p := &Prefix{Value: "test"}

	if p.IsAtom() {
		t.Error("Prefix.IsAtom() = true, want false")
	}
	if p.IsList() {
		t.Error("Prefix.IsList() = true, want false")
	}
}

func TestSuffixIsAtomIsList(t *testing.T) {
	s := &Suffix{Value: ".txt"}

	if s.IsAtom() {
		t.Error("Suffix.IsAtom() = true, want false")
	}
	if s.IsList() {
		t.Error("Suffix.IsList() = true, want false")
	}
}

func TestRange_Date(t *testing.T) {
	dateRange := &Range{
		RangeType: RangeDate,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "2024-01-01T00:00:00Z",
		},
		UpperBound: &RangeBound{
			Op:    OpLE,
			Value: "2024-12-31T23:59:59Z",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches date in range",
			elem: sexp.NewAtom("2024-06-15T12:00:00Z"),
			want: true,
		},
		{
			name: "matches lower bound",
			elem: sexp.NewAtom("2024-01-01T00:00:00Z"),
			want: true,
		},
		{
			name: "matches upper bound",
			elem: sexp.NewAtom("2024-12-31T23:59:59Z"),
			want: true,
		},
		{
			name: "does not match date before range",
			elem: sexp.NewAtom("2023-12-31T23:59:59Z"),
			want: false,
		},
		{
			name: "does not match date after range",
			elem: sexp.NewAtom("2025-01-01T00:00:00Z"),
			want: false,
		},
		{
			name: "does not match invalid date format",
			elem: sexp.NewAtom("2024-06-15"),
			want: false,
		},
		{
			name: "does not match list element",
			elem: sexp.NewList("date", sexp.NewAtom("2024-06-15T12:00:00Z")),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dateRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v for date %v", got, tt.want, tt.elem)
			}
		})
	}
}

func TestRange_DateWithExclusiveBounds(t *testing.T) {
	// Test with GT (exclusive lower) and LT (exclusive upper)
	dateRange := &Range{
		RangeType: RangeDate,
		LowerBound: &RangeBound{
			Op:    OpGT,
			Value: "2024-01-01T00:00:00Z",
		},
		UpperBound: &RangeBound{
			Op:    OpLT,
			Value: "2024-12-31T23:59:59Z",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches date in range",
			elem: sexp.NewAtom("2024-06-15T12:00:00Z"),
			want: true,
		},
		{
			name: "does not match lower bound with GT",
			elem: sexp.NewAtom("2024-01-01T00:00:00Z"),
			want: false,
		},
		{
			name: "does not match upper bound with LT",
			elem: sexp.NewAtom("2024-12-31T23:59:59Z"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dateRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v for date %v", got, tt.want, tt.elem)
			}
		})
	}
}

func TestRange_DateWithInvalidBounds(t *testing.T) {
	// Test with invalid lower bound date format
	rangeInvalidLower := &Range{
		RangeType: RangeDate,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "invalid-date",
		},
	}

	got := rangeInvalidLower.Match(sexp.NewAtom("2024-06-15T12:00:00Z"))
	if got {
		t.Error("Range.Match() with invalid lower bound should return false")
	}

	// Test with invalid upper bound date format
	rangeInvalidUpper := &Range{
		RangeType: RangeDate,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "2024-01-01T00:00:00Z",
		},
		UpperBound: &RangeBound{
			Op:    OpLE,
			Value: "invalid-date",
		},
	}

	got = rangeInvalidUpper.Match(sexp.NewAtom("2024-06-15T12:00:00Z"))
	if got {
		t.Error("Range.Match() with invalid upper bound should return false")
	}
}

func TestRange_IPv4(t *testing.T) {
	ipRange := &Range{
		RangeType: RangeIPv4,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "192.168.0.0",
		},
		UpperBound: &RangeBound{
			Op:    OpLE,
			Value: "192.168.255.255",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches IP in range",
			elem: sexp.NewAtom("192.168.1.100"),
			want: true,
		},
		{
			name: "matches lower bound",
			elem: sexp.NewAtom("192.168.0.0"),
			want: true,
		},
		{
			name: "matches upper bound",
			elem: sexp.NewAtom("192.168.255.255"),
			want: true,
		},
		{
			name: "does not match IP below range",
			elem: sexp.NewAtom("192.167.255.255"),
			want: false,
		},
		{
			name: "does not match IP above range",
			elem: sexp.NewAtom("192.169.0.0"),
			want: false,
		},
		{
			name: "does not match list element",
			elem: sexp.NewList("ip", sexp.NewAtom("192.168.1.100")),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ipRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v for IP %v", got, tt.want, tt.elem)
			}
		})
	}
}

func TestRange_IPv6(t *testing.T) {
	ipRange := &Range{
		RangeType: RangeIPv6,
		LowerBound: &RangeBound{
			Op:    OpGE,
			Value: "2001:db8::",
		},
		UpperBound: &RangeBound{
			Op:    OpLE,
			Value: "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches lower bound",
			elem: sexp.NewAtom("2001:db8::"),
			want: true,
		},
		{
			name: "does not match IPv6 below range",
			elem: sexp.NewAtom("2001:db7:ffff::"),
			want: false,
		},
		{
			name: "does not match list element",
			elem: sexp.NewList("ip", sexp.NewAtom("2001:db8::")),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ipRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v for IPv6 %v", got, tt.want, tt.elem)
			}
		})
	}
}

func TestRange_String_AllTypes(t *testing.T) {
	tests := []struct {
		name    string
		rg      *Range
		wantLen bool // Just check it's not empty
	}{
		{
			name: "alpha range",
			rg: &Range{
				RangeType:  RangeAlpha,
				LowerBound: &RangeBound{Op: OpGE, Value: "a"},
				UpperBound: &RangeBound{Op: OpLE, Value: "z"},
			},
		},
		{
			name: "numeric range",
			rg: &Range{
				RangeType:  RangeNumeric,
				LowerBound: &RangeBound{Op: OpGT, Value: "0"},
				UpperBound: &RangeBound{Op: OpLT, Value: "100"},
			},
		},
		{
			name: "date range",
			rg: &Range{
				RangeType:  RangeDate,
				LowerBound: &RangeBound{Op: OpGE, Value: "2024-01-01T00:00:00Z"},
			},
		},
		{
			name: "time range",
			rg: &Range{
				RangeType:  RangeTime,
				UpperBound: &RangeBound{Op: OpLE, Value: "23:59:59"},
			},
		},
		{
			name: "ipv4 range",
			rg: &Range{
				RangeType:  RangeIPv4,
				LowerBound: &RangeBound{Op: OpGE, Value: "0.0.0.0"},
				UpperBound: &RangeBound{Op: OpLE, Value: "255.255.255.255"},
			},
		},
		{
			name: "ipv6 range",
			rg: &Range{
				RangeType:  RangeIPv6,
				LowerBound: &RangeBound{Op: OpGE, Value: "::"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rg.String()
			if len(got) == 0 {
				t.Errorf("Range.String() returned empty string")
			}
			// Verify it starts with the expected prefix
			if got[:7] != "(1:*5:r" {
				t.Errorf("Range.String() should start with (1:*5:range, got %v", got)
			}
		})
	}
}

func TestRange_NumericWithExclusiveBounds(t *testing.T) {
	// Test with GT (exclusive lower) and LT (exclusive upper)
	numRange := &Range{
		RangeType: RangeNumeric,
		LowerBound: &RangeBound{
			Op:    OpGT,
			Value: "10",
		},
		UpperBound: &RangeBound{
			Op:    OpLT,
			Value: "20",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches value in range",
			elem: sexp.NewAtom("15"),
			want: true,
		},
		{
			name: "does not match lower bound with GT",
			elem: sexp.NewAtom("10"),
			want: false,
		},
		{
			name: "does not match upper bound with LT",
			elem: sexp.NewAtom("20"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := numRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRange_AlphaWithExclusiveBounds(t *testing.T) {
	// Test with GT (exclusive lower) and LT (exclusive upper)
	alphaRange := &Range{
		RangeType: RangeAlpha,
		LowerBound: &RangeBound{
			Op:    OpGT,
			Value: "a",
		},
		UpperBound: &RangeBound{
			Op:    OpLT,
			Value: "z",
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches value in range",
			elem: sexp.NewAtom("m"),
			want: true,
		},
		{
			name: "does not match lower bound with GT",
			elem: sexp.NewAtom("a"),
			want: false,
		},
		{
			name: "does not match upper bound with LT",
			elem: sexp.NewAtom("z"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := alphaRange.Match(tt.elem); got != tt.want {
				t.Errorf("Range.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrefixMatchWithListElement(t *testing.T) {
	prefix := &Prefix{Value: "test"}

	// Match should return false for list elements
	list := sexp.NewList("tag", sexp.NewAtom("testing"))
	if prefix.Match(list) {
		t.Error("Prefix.Match() should return false for list element")
	}
}

func TestSuffixMatchWithListElement(t *testing.T) {
	suffix := &Suffix{Value: ".txt"}

	// Match should return false for list elements
	list := sexp.NewList("tag", sexp.NewAtom("file.txt"))
	if suffix.Match(list) {
		t.Error("Suffix.Match() should return false for list element")
	}
}

func TestElementsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    sexp.Element
		b    sexp.Element
		want bool
	}{
		{
			name: "equal atoms",
			a:    sexp.NewAtom("test"),
			b:    sexp.NewAtom("test"),
			want: true,
		},
		{
			name: "different atoms",
			a:    sexp.NewAtom("test1"),
			b:    sexp.NewAtom("test2"),
			want: false,
		},
		{
			name: "equal simple lists",
			a:    sexp.NewList("tag", sexp.NewAtom("value")),
			b:    sexp.NewList("tag", sexp.NewAtom("value")),
			want: true,
		},
		{
			name: "lists with different tags",
			a:    sexp.NewList("tag1", sexp.NewAtom("value")),
			b:    sexp.NewList("tag2", sexp.NewAtom("value")),
			want: false,
		},
		{
			name: "lists with different lengths",
			a:    sexp.NewList("tag", sexp.NewAtom("v1")),
			b:    sexp.NewList("tag", sexp.NewAtom("v1"), sexp.NewAtom("v2")),
			want: false,
		},
		{
			name: "lists with different elements",
			a:    sexp.NewList("tag", sexp.NewAtom("v1")),
			b:    sexp.NewList("tag", sexp.NewAtom("v2")),
			want: false,
		},
		{
			name: "nested equal lists",
			a:    sexp.NewList("outer", sexp.NewList("inner", sexp.NewAtom("value"))),
			b:    sexp.NewList("outer", sexp.NewList("inner", sexp.NewAtom("value"))),
			want: true,
		},
		{
			name: "nested different lists",
			a:    sexp.NewList("outer", sexp.NewList("inner", sexp.NewAtom("v1"))),
			b:    sexp.NewList("outer", sexp.NewList("inner", sexp.NewAtom("v2"))),
			want: false,
		},
		{
			name: "atom vs list",
			a:    sexp.NewAtom("test"),
			b:    sexp.NewList("test"),
			want: false,
		},
		{
			name: "list vs atom",
			a:    sexp.NewList("test"),
			b:    sexp.NewAtom("test"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := elementsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("elementsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetMatchWithLists(t *testing.T) {
	// Test Set matching with list elements
	set := &Set{
		Elements: []sexp.Element{
			sexp.NewList("action", sexp.NewAtom("read")),
			sexp.NewList("action", sexp.NewAtom("write")),
		},
	}

	tests := []struct {
		name string
		elem sexp.Element
		want bool
	}{
		{
			name: "matches list in set",
			elem: sexp.NewList("action", sexp.NewAtom("read")),
			want: true,
		},
		{
			name: "matches another list in set",
			elem: sexp.NewList("action", sexp.NewAtom("write")),
			want: true,
		},
		{
			name: "does not match list not in set",
			elem: sexp.NewList("action", sexp.NewAtom("delete")),
			want: false,
		},
		{
			name: "does not match atom when set has lists",
			elem: sexp.NewAtom("read"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Match(tt.elem); got != tt.want {
				t.Errorf("Set.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetType(t *testing.T) {
	set := &Set{Elements: []sexp.Element{sexp.NewAtom("a")}}
	if got := set.Type(); got != "set" {
		t.Errorf("Set.Type() = %v, want set", got)
	}
}
