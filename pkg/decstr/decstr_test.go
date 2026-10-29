package decstr

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestParseAccept(t *testing.T) {
	cases := []struct {
		in   string
		want uint64
	}{
		{"0", 0},
		{"1", 1},
		{"9", 9},
		{"10", 10},
		{"1234", 1234},
		{"9007199254740991", 1<<53 - 1}, // Number.MAX_SAFE_INTEGER
		{"9007199254740992", 1 << 53},
		{"9007199254740993", 1<<53 + 1}, // first integer a float64 cannot hold
		{"9223372036854775807", math.MaxInt64},
		{"9223372036854775808", math.MaxInt64 + 1},
		{"18446744073709551615", math.MaxUint64},
		{"10000000000000000000", 1e19},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := Parse(tc.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tc.in, err)
			}
			if uint64(got) != tc.want {
				t.Fatalf("Parse(%q) = %d, want %d", tc.in, got, tc.want)
			}
			if got.String() != tc.in {
				t.Fatalf("round trip %q -> %q", tc.in, got.String())
			}
			if !Valid(tc.in) {
				t.Fatalf("Valid(%q) = false", tc.in)
			}
		})
	}
}

func TestParseReject(t *testing.T) {
	cases := []struct {
		in      string
		wantErr error
	}{
		{"", ErrSyntax},
		{"-0", ErrSyntax},
		{"-1", ErrSyntax},
		{"+1", ErrSyntax},
		{"01", ErrSyntax},
		{"00", ErrSyntax},
		{"007", ErrSyntax},
		{" 1", ErrSyntax},
		{"1 ", ErrSyntax},
		{"\t1", ErrSyntax},
		{"1\n", ErrSyntax},
		{"1e3", ErrSyntax},
		{"1E3", ErrSyntax},
		{"1.0", ErrSyntax},
		{"1.", ErrSyntax},
		{".5", ErrSyntax},
		{"0x10", ErrSyntax},
		{"1_000", ErrSyntax},
		{"1,000", ErrSyntax},
		{"NaN", ErrSyntax},
		{"Infinity", ErrSyntax},
		{"null", ErrSyntax},
		{"１", ErrSyntax},                    // fullwidth digit one
		{"٣", ErrSyntax},                    // Arabic-Indic digit three
		{"18446744073709551616", ErrRange},  // max+1
		{"99999999999999999999", ErrRange},  // 20 digits, above max
		{"100000000000000000000", ErrRange}, // 21 digits
		{"000000000000000000000001", ErrSyntax},
		{"1234567890123456789012345 ", ErrSyntax},
	}
	for _, tc := range cases {
		t.Run(strconv.Quote(tc.in), func(t *testing.T) {
			got, err := Parse(tc.in)
			if err == nil {
				t.Fatalf("Parse(%q) = %d, want error", tc.in, got)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Parse(%q) error = %v, want %v", tc.in, err, tc.wantErr)
			}
			var se *SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("error %T is not *SyntaxError", err)
			}
			if se.Error() == "" || se.Reason == "" {
				t.Fatal("empty error text")
			}
			if Valid(tc.in) {
				t.Fatalf("Valid(%q) = true", tc.in)
			}
		})
	}
}

func TestSyntaxErrorTruncatesInput(t *testing.T) {
	long := "x" + string(make([]byte, 100))
	_, err := Parse(long)
	var se *SyntaxError
	if !errors.As(err, &se) {
		t.Fatalf("got %v", err)
	}
	if len(se.Input) > 40 {
		t.Fatalf("input not truncated: %d bytes", len(se.Input))
	}
}

func TestMustParse(t *testing.T) {
	if MustParse("42") != 42 {
		t.Fatal("MustParse(42)")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustParse did not panic")
		}
	}()
	MustParse("01")
}

type wrapper struct {
	Bytes U64  `json:"bytes"`
	Seq   *U64 `json:"seq,omitempty"`
}

func TestJSONRoundTrip(t *testing.T) {
	// 2^53+1 must survive the JSON round trip unchanged; a float64 would turn
	// it into 9007199254740992.
	for _, s := range []string{"0", "9007199254740993", "18446744073709551615"} {
		in := []byte(`{"bytes":"` + s + `"}`)
		var w wrapper
		if err := json.Unmarshal(in, &w); err != nil {
			t.Fatalf("unmarshal %s: %v", in, err)
		}
		if w.Bytes.String() != s {
			t.Fatalf("got %s want %s", w.Bytes, s)
		}
		out, err := json.Marshal(w)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != string(in) {
			t.Fatalf("round trip: got %s want %s", out, in)
		}
	}
}

func TestJSONRejectsNonStrings(t *testing.T) {
	cases := map[string]error{
		`{"bytes":123}`:                    ErrJSONType,
		`{"bytes":9007199254740993}`:       ErrJSONType,
		`{"bytes":-1}`:                     ErrJSONType,
		`{"bytes":null}`:                   ErrJSONType,
		`{"bytes":true}`:                   ErrJSONType,
		`{"bytes":false}`:                  ErrJSONType,
		`{"bytes":{}}`:                     ErrJSONType,
		`{"bytes":[]}`:                     ErrJSONType,
		`{"bytes":"01"}`:                   ErrSyntax,
		`{"bytes":""}`:                     ErrSyntax,
		`{"bytes":"1e3"}`:                  ErrSyntax,
		`{"bytes":"\u0031"}`:               ErrSyntax, // escaped "1" is not canonical
		`{"bytes":"18446744073709551616"}`: ErrRange,
	}
	for in, want := range cases {
		var w wrapper
		err := json.Unmarshal([]byte(in), &w)
		if err == nil {
			t.Errorf("%s: accepted", in)
			continue
		}
		if !errors.Is(err, want) {
			t.Errorf("%s: error %v, want %v", in, err, want)
		}
	}
}

func TestUnmarshalJSONDirect(t *testing.T) {
	var u U64
	for _, in := range []string{``, `"`, `x`, `"1`, `1"`} {
		if err := u.UnmarshalJSON([]byte(in)); !errors.Is(err, ErrJSONType) {
			t.Errorf("UnmarshalJSON(%q) = %v, want ErrJSONType", in, err)
		}
	}
	if err := u.UnmarshalJSON([]byte(`"12"`)); err != nil || u != 12 {
		t.Fatalf("got %d, %v", u, err)
	}
}

func TestText(t *testing.T) {
	var u U64
	if err := u.UnmarshalText([]byte("77")); err != nil || u != 77 {
		t.Fatalf("UnmarshalText: %d %v", u, err)
	}
	if err := u.UnmarshalText([]byte("-77")); err == nil {
		t.Fatal("UnmarshalText accepted -77")
	}
	if u != 77 {
		t.Fatal("failed UnmarshalText modified the receiver")
	}
	b, err := Max.MarshalText()
	if err != nil || string(b) != "18446744073709551615" {
		t.Fatalf("MarshalText: %s %v", b, err)
	}
	// Map keys use TextMarshaler.
	m := map[U64]int{5: 1}
	out, err := json.Marshal(m)
	if err != nil || string(out) != `{"5":1}` {
		t.Fatalf("map key: %s %v", out, err)
	}
	if Max.Uint64() != math.MaxUint64 {
		t.Fatal("Uint64")
	}
}

func TestScan(t *testing.T) {
	ok := []struct {
		src  any
		want U64
	}{
		{"0", 0},
		{"18446744073709551615", Max},
		{[]byte("9007199254740993"), 9007199254740993},
		{int64(0), 0},
		{int64(math.MaxInt64), math.MaxInt64},
		{uint64(math.MaxUint64), Max},
	}
	for _, tc := range ok {
		var u U64
		if err := u.Scan(tc.src); err != nil {
			t.Errorf("Scan(%#v): %v", tc.src, err)
			continue
		}
		if u != tc.want {
			t.Errorf("Scan(%#v) = %d want %d", tc.src, u, tc.want)
		}
	}
	bad := []struct {
		src  any
		want error
	}{
		{nil, ErrScanType},
		{int64(-1), ErrRange},
		{float64(1), ErrScanType},
		{"1.0", ErrSyntax},
		{[]byte("-5"), ErrSyntax},
		{"18446744073709551616", ErrRange},
		{true, ErrScanType},
	}
	for _, tc := range bad {
		var u U64
		if err := u.Scan(tc.src); !errors.Is(err, tc.want) {
			t.Errorf("Scan(%#v) = %v, want %v", tc.src, err, tc.want)
		}
	}
	// Interface conformance.
	var _ sql.Scanner = (*U64)(nil)
	var _ driver.Valuer = U64(0)
	var n sql.Null[U64]
	if err := n.Scan(nil); err != nil || n.Valid {
		t.Fatalf("sql.Null scan nil: %v %v", n, err)
	}
}

func TestValue(t *testing.T) {
	v, err := U64(9007199254740993).Value()
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := v.(string); !ok || s != "9007199254740993" {
		t.Fatalf("Value() = %#v", v)
	}
	if !driver.IsValue(v) {
		t.Fatal("not a driver.Value")
	}
}

func TestArithmetic(t *testing.T) {
	if s, err := U64(1).Add(2); err != nil || s != 3 {
		t.Fatalf("1+2 = %d %v", s, err)
	}
	if s, err := (Max - 1).Add(1); err != nil || s != Max {
		t.Fatalf("max-1+1 = %d %v", s, err)
	}
	if _, err := Max.Add(1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("max+1 err = %v", err)
	}
	if d, err := U64(5).Sub(5); err != nil || d != 0 {
		t.Fatalf("5-5 = %d %v", d, err)
	}
	if d, err := Max.Sub(1); err != nil || d != Max-1 {
		t.Fatalf("max-1 = %d %v", d, err)
	}
	if _, err := U64(0).Sub(1); !errors.Is(err, ErrUnderflow) {
		t.Fatalf("0-1 err = %v", err)
	}
	if s, err := Sum(); err != nil || s != 0 {
		t.Fatalf("Sum() = %d %v", s, err)
	}
	if s, err := Sum(1, 2, 3, 9007199254740990); err != nil || s != 9007199254740996 {
		t.Fatalf("Sum = %d %v", s, err)
	}
	if _, err := Sum(Max/2, Max/2, 2); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Sum overflow err = %v", err)
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"0", "1", "01", "-0", "+1", " 1", "1e3", "1.0", "",
		"9007199254740993", "18446744073709551615", "18446744073709551616", "\u0661"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := Parse(s)
		ref, refErr := strconv.ParseUint(s, 10, 64)
		canonical := refErr == nil && strconv.FormatUint(ref, 10) == s
		if canonical != (err == nil) {
			t.Fatalf("Parse(%q): err=%v, reference canonical=%v", s, err, canonical)
		}
		if err != nil {
			if !errors.Is(err, ErrSyntax) && !errors.Is(err, ErrRange) {
				t.Fatalf("Parse(%q): unexpected error class %v", s, err)
			}
			return
		}
		if uint64(v) != ref || v.String() != s {
			t.Fatalf("Parse(%q) = %d", s, v)
		}
		b, _ := json.Marshal(v)
		var back U64
		if err := json.Unmarshal(b, &back); err != nil || back != v {
			t.Fatalf("JSON round trip of %q: %v", s, err)
		}
	})
}
