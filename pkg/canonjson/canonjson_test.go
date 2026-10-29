package canonjson

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCanonicalizeRFC8785Examples(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			// RFC 8785 §3.2.2 example with the "numbers" member removed: the
			// non-integer numbers are outside our subset. String escaping and
			// literal handling are unchanged.
			name: "rfc8785-3.2.2-without-numbers",
			in: `{
  "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/",
  "literals": [null, true, false]
}`,
			want: `{"literals":[null,true,false],"string":"€$\u000f\nA'B\"\\\\\"/"}`,
		},
		{
			// RFC 8785 §3.2.3 sorting example. The emoji (U+1F600, UTF-16
			// D83D DE00) must sort before U+FB33 even though its code point is
			// larger.
			name: "rfc8785-3.2.3-sorting",
			in: `{
  "\u20ac": "Euro Sign",
  "\r": "Carriage Return",
  "\ufb33": "Hebrew Letter Dalet With Dagesh",
  "1": "One",
  "\ud83d\ude00": "Emoji: Grinning Face",
  "\u0080": "Control",
  "\u00f6": "Latin Small Letter O With Diaeresis"
}`,
			want: "{\"\\r\":\"Carriage Return\",\"1\":\"One\",\"\u0080\":\"Control\",\"ö\":\"Latin Small Letter O With Diaeresis\",\"€\":\"Euro Sign\",\"😀\":\"Emoji: Grinning Face\",\"\ufb33\":\"Hebrew Letter Dalet With Dagesh\"}",
		},
		{
			// RFC 8785 test data "arrays.json".
			name: "rfc8785-arrays",
			in:   `[56, {"d": true, "10": null, "1": [ ]}]`,
			want: `[56,{"1":[],"10":null,"d":true}]`,
		},
		{
			// RFC 8785 test data "structures.json" with 56.0 replaced by 56.
			name: "rfc8785-structures-integer",
			in:   `{"1": {"f": {"f": "hi","F": 5} ,"\n": 56},"10": { },"": "empty","a": { },"111": [ {"e": "yes","E": "no" } ],"A": { }}`,
			want: `{"":"empty","1":{"\n":56,"f":{"F":5,"f":"hi"}},"10":{},"111":[{"E":"no","e":"yes"}],"A":{},"a":{}}`,
		},
		{
			// RFC 8785 test data "unicode.json": no Unicode normalization.
			name: "rfc8785-unicode",
			in:   `{"Unnormalized Unicode":"A\u030a"}`,
			want: "{\"Unnormalized Unicode\":\"A\u030a\"}",
		},
		{
			// RFC 8785 test data "weird.json".
			name: "rfc8785-weird",
			in: `{
  "\u20ac": "Euro Sign",
  "\r": "Carriage Return",
  "\u000a": "Newline",
  "1": "One",
  "\u0080": "Control\u007f",
  "\ud83d\ude02": "Smiley",
  "\u00f6": "Latin Small Letter O With Diaeresis",
  "\ufb33": "Hebrew Letter Dalet With Dagesh",
  "</script>": "Browser Challenge"
}`,
			want: "{\"\\n\":\"Newline\",\"\\r\":\"Carriage Return\",\"1\":\"One\",\"</script>\":\"Browser Challenge\",\"\u0080\":\"Control\u007f\",\"ö\":\"Latin Small Letter O With Diaeresis\",\"€\":\"Euro Sign\",\"😂\":\"Smiley\",\"\ufb33\":\"Hebrew Letter Dalet With Dagesh\"}",
		},
		{
			// RFC 8785 Appendix B integer values inside the subset.
			name: "rfc8785-appendixB-integers",
			in:   `[0, 9007199254740991, -9007199254740991, 1, -1]`,
			want: `[0,9007199254740991,-9007199254740991,1,-1]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Canonicalize([]byte(tc.in))
			if err != nil {
				t.Fatalf("Canonicalize: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("got  %q\nwant %q", got, tc.want)
			}
			// Canonicalization is idempotent.
			again, err := Canonicalize(got)
			if err != nil || string(again) != string(got) {
				t.Fatalf("not idempotent: %q %v", again, err)
			}
		})
	}
}

func TestEscaping(t *testing.T) {
	cases := map[string]string{
		`"\u0000\u0001\u001f"`: `"\u0000\u0001\u001f"`,
		`"\b\f\n\r\t"`:         `"\b\f\n\r\t"`,
		`"\u0008\u000C"`:       `"\b\f"`,
		`"\/"`:                 `"/"`,
		`"\u2028\u2029"`:       "\"\u2028\u2029\"", // emitted literally, as JSON.stringify does
		`"<>&'"`:               `"<>&'"`,           // no HTML escaping
		`"\u007f"`:             "\"\u007f\"",
		`"\uD834\uDD1E"`:       `"𝄞"`,
		`"\ud834\udd1e"`:       `"𝄞"`,
		`""`:                   `""`,
		`"é"`:                  `"é"`,
	}
	for in, want := range cases {
		got, err := Canonicalize([]byte(in))
		if err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestWhitespaceAndLiterals(t *testing.T) {
	got, err := Canonicalize([]byte(" \t\r\n{ \"b\" : [ true , false , null ] , \"a\" : -5 } \n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"a":-5,"b":[true,false,null]}` {
		t.Fatalf("got %s", got)
	}
	for _, in := range []string{`null`, `true`, `false`, `0`, `"x"`, `[]`, `{}`} {
		got, err := Canonicalize([]byte(in))
		if err != nil || string(got) != in {
			t.Errorf("%s -> %s %v", in, got, err)
		}
	}
}

func TestRejections(t *testing.T) {
	cases := []struct {
		in   string
		want error
	}{
		{`1.0`, ErrNonInteger},
		{`1.5`, ErrNonInteger},
		{`1e3`, ErrNonInteger},
		{`1E3`, ErrNonInteger},
		{`[0.1]`, ErrNonInteger},
		{`{"a":2e-3}`, ErrNonInteger},
		{`-0`, ErrNegativeZero},
		{`9007199254740992`, ErrIntegerRange},
		{`-9007199254740992`, ErrIntegerRange},
		{`18446744073709551615`, ErrIntegerRange},
		{`12345678901234567890123`, ErrIntegerRange},
		{`{"a":1,"a":2}`, ErrDuplicateKey},
		{`{"a":1,"\u0061":2}`, ErrDuplicateKey}, // duplicate after unescaping
		{`{"x":{"b":1,"b":1}}`, ErrDuplicateKey},
		{"\"\xff\"", ErrInvalidUTF8},
		{"\"\xc3\x28\"", ErrInvalidUTF8},
		{"\"\xed\xa0\x80\"", ErrInvalidUTF8}, // UTF-8 encoded surrogate
		{"{\"\xfe\":1}", ErrInvalidUTF8},
		{`"\ud800"`, ErrLoneSurrogate},
		{`"\udc00"`, ErrLoneSurrogate},
		{`"\ud800x"`, ErrLoneSurrogate},
		{`"\ud800\u0041"`, ErrLoneSurrogate},
		{`"\udfff\ud800"`, ErrLoneSurrogate},
		{``, ErrSyntax},
		{` `, ErrSyntax},
		{`{`, ErrSyntax},
		{`[`, ErrSyntax},
		{`[1,]`, ErrSyntax},
		{`{"a":1,}`, ErrSyntax},
		{`{"a" 1}`, ErrSyntax},
		{`{a:1}`, ErrSyntax},
		{`[1 2]`, ErrSyntax},
		{`{"a":1 "b":2}`, ErrSyntax},
		{`01`, ErrSyntax},
		{`-`, ErrSyntax},
		{`+1`, ErrSyntax},
		{`.5`, ErrSyntax},
		{`tru`, ErrSyntax},
		{`nul`, ErrSyntax},
		{`NaN`, ErrSyntax},
		{`"abc`, ErrSyntax},
		{`"\x"`, ErrSyntax},
		{`"\u12"`, ErrSyntax},
		{`"\u12g4"`, ErrSyntax},
		{`"\ud800\u12zz"`, ErrSyntax},
		{`"\`, ErrSyntax},
		{"\"a\tb\"", ErrSyntax},
		{"\"a\nb\"", ErrSyntax},
		{`{} {}`, ErrSyntax},
		{`1 2`, ErrSyntax},
		{"\xef\xbb\xbf{}", ErrSyntax},
		{"\u00a0{}", ErrSyntax},
		{strings.Repeat("[", MaxDepth+1) + strings.Repeat("]", MaxDepth+1), ErrTooDeep},
		{strings.Repeat(`{"a":`, MaxDepth+1) + "1" + strings.Repeat("}", MaxDepth+1), ErrTooDeep},
	}
	for _, tc := range cases {
		got, err := Canonicalize([]byte(tc.in))
		if err == nil {
			t.Errorf("%q: accepted as %q", tc.in, got)
			continue
		}
		if !errors.Is(err, tc.want) {
			t.Errorf("%q: error %v, want %v", tc.in, err, tc.want)
		}
		var e *Error
		if !errors.As(err, &e) || e.Error() == "" {
			t.Errorf("%q: error %T is not *Error", tc.in, err)
		}
	}
}

func TestDepthLimitBoundary(t *testing.T) {
	in := strings.Repeat("[", MaxDepth) + strings.Repeat("]", MaxDepth)
	if _, err := Canonicalize([]byte(in)); err != nil {
		t.Fatalf("depth %d rejected: %v", MaxDepth, err)
	}
}

func TestMarshal(t *testing.T) {
	type inner struct {
		Z string `json:"z"`
		A int    `json:"a"`
	}
	v := struct {
		B   []inner        `json:"b"`
		A   map[string]any `json:"a"`
		Nil *int           `json:"nil"`
	}{
		B: []inner{{Z: "<&>", A: 1}},
		A: map[string]any{"y": true, "x": nil},
	}
	got, err := Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":{"x":null,"y":true},"b":[{"a":1,"z":"<&>"}],"nil":null}`
	if string(got) != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if _, err := Marshal(1.5); !errors.Is(err, ErrNonInteger) {
		t.Fatalf("Marshal(1.5) = %v", err)
	}
	if _, err := Marshal(uint64(1) << 60); !errors.Is(err, ErrIntegerRange) {
		t.Fatalf("Marshal(2^60) = %v", err)
	}
	if _, err := Marshal(make(chan int)); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("Marshal(chan) = %v", err)
	}
}

func TestHashHex(t *testing.T) {
	raw := []byte(`{"b":"2","payload_sha256":"ignored","a":1}`)
	got, err := HashHex(raw, "payload_sha256")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(`{"a":1,"b":"2"}`))
	if want := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	// Changing the excluded member does not change the hash.
	got2, _ := HashHex([]byte(`{"a":1,"payload_sha256":"other","b":"2"}`), "payload_sha256")
	if got2 != got {
		t.Fatal("excluded member affected hash")
	}
	// Only top-level members are excluded.
	nested, _ := HashHex([]byte(`{"x":{"payload_sha256":"v"}}`), "payload_sha256")
	full, _ := HashHex([]byte(`{"x":{"payload_sha256":"v"}}`))
	if nested != full {
		t.Fatal("nested member was excluded")
	}
	// Absent names are ignored; no exclusion equals plain canonical hash.
	plain, _ := HashHex([]byte(`[1]`))
	s := sha256.Sum256([]byte(`[1]`))
	if plain != hex.EncodeToString(s[:]) {
		t.Fatal("plain hash")
	}
	if _, err := HashHex([]byte(`[1]`), "x"); !errors.Is(err, ErrNotObject) {
		t.Fatalf("non-object with exclusions: %v", err)
	}
	if _, err := HashHex([]byte(`{"a":1.5}`), "x"); !errors.Is(err, ErrNonInteger) {
		t.Fatalf("invalid input: %v", err)
	}
	if _, err := Hash([]byte(`{`)); !errors.Is(err, ErrSyntax) {
		t.Fatalf("Hash invalid: %v", err)
	}
}

func TestCanonicalizeWithoutKeepsOriginal(t *testing.T) {
	raw := []byte(`{"keep":1,"drop":2}`)
	got, err := CanonicalizeWithout(raw, "drop", "missing")
	if err != nil || string(got) != `{"keep":1}` {
		t.Fatalf("%s %v", got, err)
	}
	if string(raw) != `{"keep":1,"drop":2}` {
		t.Fatal("input modified")
	}
}

// TestAgreesWithEncodingJSON checks that canonical output parses back to the
// same value with encoding/json.
func TestAgreesWithEncodingJSON(t *testing.T) {
	in := `{"z":[1,-2,"\u00e9\ud83d\ude00\n"],"a":{"c":null,"b":false}}`
	got, err := Canonicalize([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal([]byte(in), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &b); err != nil {
		t.Fatal(err)
	}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Fatalf("value changed: %s vs %s", ja, jb)
	}
}

func FuzzCanonicalize(f *testing.F) {
	for _, s := range []string{`{"a":1}`, `[1,"x",null]`, `"\ud83d\ude00"`, `{"b":{"a":[]}}`, `-1`, `1.0`, `{"a":1,"a":2}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		out, err := Canonicalize(in)
		if err != nil {
			return
		}
		// Output must be valid JSON, canonical (idempotent) and value-equal.
		if !json.Valid(out) {
			t.Fatalf("invalid output %q for %q", out, in)
		}
		again, err := Canonicalize(out)
		if err != nil || string(again) != string(out) {
			t.Fatalf("not idempotent: %q -> %q (%v)", out, again, err)
		}
	})
}
