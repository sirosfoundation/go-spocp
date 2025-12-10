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
