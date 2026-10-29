// Package canonjson implements RFC 8785 JSON Canonicalization Scheme (JCS)
// for the JSON subset used by Freedom VPN hashed and signed documents
// (AGENTS.md §3, Project Spec §10.5, §13.3, §13.10.3, §14.7, §16.2, §17.3).
//
// Supported values are objects, arrays, strings, true, false, null and
// integers within ±(2^53-1). Large integers such as bytes, sequence numbers
// and generations are carried as decimal strings (see package decstr), so the
// subset never needs the ECMAScript double formatting rules of RFC 8785 §3.2.2.3.
//
// The canonical form is produced exactly as RFC 8785 requires for this subset:
//
//   - no insignificant whitespace;
//   - object members sorted by the UTF-16 code units of their names;
//   - strings escaped as ECMAScript JSON.stringify does: \" \\ \b \t \n \f \r,
//     other control characters as \u00xx with lowercase hex, every other code
//     point emitted literally as UTF-8 (no Unicode normalization);
//   - integers in plain decimal without exponent, fraction or leading zeros.
//
// Input is parsed strictly. The following are rejected with typed errors:
// numbers with a fraction or exponent, "-0", integers outside ±(2^53-1),
// duplicate object member names (after unescaping), invalid UTF-8, lone
// UTF-16 surrogates in \u escapes, raw control characters inside strings, a
// byte order mark, trailing data, nesting deeper than MaxDepth, and input
// larger than MaxBytes.
package canonjson

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// MaxSafeInteger is 2^53-1, the largest integer magnitude allowed.
const MaxSafeInteger = 1<<53 - 1

// MaxDepth is the maximum nesting depth of arrays and objects.
const MaxDepth = 256

// MaxBytes is the maximum encoded input size: 1 MiB, matching the control
// HTTP body limit (§14.6, §16.2). Checked before parsing, sorting or hashing.
const MaxBytes = 1 << 20

// Sentinel errors. Every error returned by Canonicalize, Marshal and HashHex
// wraps exactly one of these, so callers can classify with errors.Is.
var (
	ErrSyntax          = errors.New("canonjson: invalid JSON syntax")
	ErrNonInteger      = errors.New("canonjson: number is not an integer (fraction or exponent present)")
	ErrNegativeZero    = errors.New("canonjson: negative zero is not allowed")
	ErrIntegerRange    = errors.New("canonjson: integer outside ±(2^53-1)")
	ErrDuplicateKey    = errors.New("canonjson: duplicate object member name")
	ErrInvalidUTF8     = errors.New("canonjson: invalid UTF-8")
	ErrLoneSurrogate   = errors.New("canonjson: lone UTF-16 surrogate in string escape")
	ErrTooLarge        = errors.New("canonjson: input exceeds MaxBytes")
	ErrTooDeep         = errors.New("canonjson: nesting too deep")
	ErrNotObject       = errors.New("canonjson: top-level value is not an object")
	ErrUnsupportedType = errors.New("canonjson: value cannot be encoded as JSON")
)

// Error carries the byte offset of a parse failure and wraps a sentinel.
type Error struct {
	Offset int    // byte offset into the input where the problem was found
	Detail string // optional extra context (never contains document values longer than 32 bytes)
	err    error
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%v at offset %d: %s", e.err, e.Offset, e.Detail)
	}
	return fmt.Sprintf("%v at offset %d", e.err, e.Offset)
}

func (e *Error) Unwrap() error { return e.err }

type kind uint8

const (
	kNull kind = iota
	kTrue
	kFalse
	kNumber
	kString
	kArray
	kObject
)

type member struct {
	name  string
	value *node
}

type node struct {
	kind    kind
	num     int64
	str     string
	arr     []*node
	members []member
}

// Canonicalize parses raw JSON in the supported subset and returns its RFC
// 8785 canonical form.
func Canonicalize(raw []byte) ([]byte, error) {
	n, err := parse(raw)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Grow(len(raw))
	n.write(&buf)
	return buf.Bytes(), nil
}

// Marshal encodes v with encoding/json and canonicalizes the result. Values
// that encoding/json renders as non-integers or as integers outside
// ±(2^53-1) are rejected; use decstr.U64 for large integers.
func Marshal(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedType, err)
	}
	return Canonicalize(raw)
}

// HashHex returns the lowercase hexadecimal SHA-256 of the canonical form of
// raw after removing the named top-level object members. If any names are
// given, raw must be a JSON object. Names that are absent are ignored.
//
// This computes the usage event payload_sha256 (§10.5, excluding
// "payload_sha256") and the update result payload_hash (§13.10.3, excluding
// "payload_hash").
func HashHex(raw []byte, excludeTopLevel ...string) (string, error) {
	sum, err := Hash(raw, excludeTopLevel...)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum[:]), nil
}

// Hash is like HashHex but returns the raw digest.
func Hash(raw []byte, excludeTopLevel ...string) ([sha256.Size]byte, error) {
	c, err := CanonicalizeWithout(raw, excludeTopLevel...)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(c), nil
}

// CanonicalizeWithout is like Canonicalize but first removes the named
// top-level object members. If any names are given, raw must be an object.
func CanonicalizeWithout(raw []byte, excludeTopLevel ...string) ([]byte, error) {
	n, err := parse(raw)
	if err != nil {
		return nil, err
	}
	if len(excludeTopLevel) > 0 {
		if n.kind != kObject {
			return nil, &Error{Offset: 0, err: ErrNotObject}
		}
		drop := make(map[string]bool, len(excludeTopLevel))
		for _, k := range excludeTopLevel {
			drop[k] = true
		}
		kept := n.members[:0:0]
		for _, m := range n.members {
			if !drop[m.name] {
				kept = append(kept, m)
			}
		}
		n.members = kept
	}
	var buf bytes.Buffer
	buf.Grow(len(raw))
	n.write(&buf)
	return buf.Bytes(), nil
}

// ---- parser ----

type parser struct {
	data  []byte
	pos   int
	depth int
}

func parse(raw []byte) (*node, error) {
	if len(raw) > MaxBytes {
		return nil, &Error{Offset: 0, err: ErrTooLarge}
	}
	p := &parser{data: raw}
	if bytes.HasPrefix(raw, []byte("\xef\xbb\xbf")) {
		return nil, p.errf(ErrSyntax, "byte order mark")
	}
	p.skipWS()
	n, err := p.value()
	if err != nil {
		return nil, err
	}
	p.skipWS()
	if p.pos != len(p.data) {
		return nil, p.errf(ErrSyntax, "trailing data after top-level value")
	}
	return n, nil
}

func (p *parser) errf(sentinel error, detail string) error {
	return &Error{Offset: p.pos, Detail: detail, err: sentinel}
}

func (p *parser) skipWS() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *parser) value() (*node, error) {
	if p.pos >= len(p.data) {
		return nil, p.errf(ErrSyntax, "unexpected end of input")
	}
	switch c := p.data[p.pos]; {
	case c == '{':
		return p.object()
	case c == '[':
		return p.array()
	case c == '"':
		s, err := p.string()
		if err != nil {
			return nil, err
		}
		return &node{kind: kString, str: s}, nil
	case c == 't':
		return p.literal("true", kTrue)
	case c == 'f':
		return p.literal("false", kFalse)
	case c == 'n':
		return p.literal("null", kNull)
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	default:
		return nil, p.errf(ErrSyntax, fmt.Sprintf("unexpected character %q", c))
	}
}

func (p *parser) literal(lit string, k kind) (*node, error) {
	if !bytes.HasPrefix(p.data[p.pos:], []byte(lit)) {
		return nil, p.errf(ErrSyntax, "invalid literal")
	}
	p.pos += len(lit)
	return &node{kind: k}, nil
}

func (p *parser) enter() error {
	p.depth++
	if p.depth > MaxDepth {
		return p.errf(ErrTooDeep, "")
	}
	return nil
}

func (p *parser) object() (*node, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	p.pos++ // '{'
	n := &node{kind: kObject}
	seen := map[string]bool{}
	p.skipWS()
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		p.depth--
		return n, nil
	}
	for {
		p.skipWS()
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			return nil, p.errf(ErrSyntax, "expected object member name")
		}
		keyPos := p.pos
		name, err := p.string()
		if err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, &Error{Offset: keyPos, err: ErrDuplicateKey}
		}
		seen[name] = true
		p.skipWS()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return nil, p.errf(ErrSyntax, "expected ':'")
		}
		p.pos++
		p.skipWS()
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		n.members = append(n.members, member{name: name, value: v})
		p.skipWS()
		if p.pos >= len(p.data) {
			return nil, p.errf(ErrSyntax, "unterminated object")
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
		case '}':
			p.pos++
			p.depth--
			return n, nil
		default:
			return nil, p.errf(ErrSyntax, "expected ',' or '}'")
		}
	}
}

func (p *parser) array() (*node, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	p.pos++ // '['
	n := &node{kind: kArray}
	p.skipWS()
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		p.depth--
		return n, nil
	}
	for {
		p.skipWS()
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		n.arr = append(n.arr, v)
		p.skipWS()
		if p.pos >= len(p.data) {
			return nil, p.errf(ErrSyntax, "unterminated array")
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
		case ']':
			p.pos++
			p.depth--
			return n, nil
		default:
			return nil, p.errf(ErrSyntax, "expected ',' or ']'")
		}
	}
}

func (p *parser) number() (*node, error) {
	start := p.pos
	neg := false
	if p.data[p.pos] == '-' {
		neg = true
		p.pos++
	}
	digitsStart := p.pos
	for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
		p.pos++
	}
	digits := p.data[digitsStart:p.pos]
	if len(digits) == 0 {
		return nil, p.errf(ErrSyntax, "invalid number")
	}
	if p.pos < len(p.data) {
		if c := p.data[p.pos]; c == '.' || c == 'e' || c == 'E' {
			return nil, &Error{Offset: start, err: ErrNonInteger}
		}
	}
	if len(digits) > 1 && digits[0] == '0' {
		return nil, &Error{Offset: start, Detail: "leading zero", err: ErrSyntax}
	}
	if len(digits) > 16 { // 2^53-1 has 16 digits
		return nil, &Error{Offset: start, err: ErrIntegerRange}
	}
	v, err := strconv.ParseInt(string(digits), 10, 64)
	if err != nil || v > MaxSafeInteger {
		return nil, &Error{Offset: start, err: ErrIntegerRange}
	}
	if neg {
		if v == 0 {
			return nil, &Error{Offset: start, err: ErrNegativeZero}
		}
		v = -v
	}
	return &node{kind: kNumber, num: v}, nil
}

func hexVal(c byte) (rune, bool) {
	switch {
	case c >= '0' && c <= '9':
		return rune(c - '0'), true
	case c >= 'a' && c <= 'f':
		return rune(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return rune(c-'A') + 10, true
	}
	return 0, false
}

func (p *parser) hex4() (rune, error) {
	if p.pos+4 > len(p.data) {
		return 0, p.errf(ErrSyntax, "truncated \\u escape")
	}
	var r rune
	for i := 0; i < 4; i++ {
		h, ok := hexVal(p.data[p.pos+i])
		if !ok {
			return 0, p.errf(ErrSyntax, "invalid \\u escape")
		}
		r = r<<4 | h
	}
	p.pos += 4
	return r, nil
}

// string parses a JSON string starting at the opening quote and returns its
// unescaped value as valid UTF-8.
func (p *parser) string() (string, error) {
	p.pos++ // opening quote
	var sb []byte
	for {
		if p.pos >= len(p.data) {
			return "", p.errf(ErrSyntax, "unterminated string")
		}
		c := p.data[p.pos]
		switch {
		case c == '"':
			p.pos++
			return string(sb), nil
		case c == '\\':
			escPos := p.pos
			p.pos++
			if p.pos >= len(p.data) {
				return "", p.errf(ErrSyntax, "unterminated escape")
			}
			e := p.data[p.pos]
			p.pos++
			switch e {
			case '"', '\\', '/':
				sb = append(sb, e)
			case 'b':
				sb = append(sb, '\b')
			case 'f':
				sb = append(sb, '\f')
			case 'n':
				sb = append(sb, '\n')
			case 'r':
				sb = append(sb, '\r')
			case 't':
				sb = append(sb, '\t')
			case 'u':
				r, err := p.hex4()
				if err != nil {
					return "", err
				}
				switch {
				case utf16.IsSurrogate(r) && r < 0xDC00: // high surrogate
					if p.pos+6 > len(p.data) || p.data[p.pos] != '\\' || p.data[p.pos+1] != 'u' {
						return "", &Error{Offset: escPos, err: ErrLoneSurrogate}
					}
					p.pos += 2
					r2, err := p.hex4()
					if err != nil {
						return "", err
					}
					if r2 < 0xDC00 || r2 > 0xDFFF {
						return "", &Error{Offset: escPos, err: ErrLoneSurrogate}
					}
					r = utf16.DecodeRune(r, r2)
				case utf16.IsSurrogate(r): // low surrogate without a high one
					return "", &Error{Offset: escPos, err: ErrLoneSurrogate}
				}
				sb = utf8.AppendRune(sb, r)
			default:
				return "", &Error{Offset: escPos, Detail: "invalid escape", err: ErrSyntax}
			}
		case c < 0x20:
			return "", p.errf(ErrSyntax, "unescaped control character in string")
		case c < utf8.RuneSelf:
			sb = append(sb, c)
			p.pos++
		default:
			r, size := utf8.DecodeRune(p.data[p.pos:])
			if r == utf8.RuneError && size <= 1 {
				return "", p.errf(ErrInvalidUTF8, "")
			}
			sb = append(sb, p.data[p.pos:p.pos+size]...)
			p.pos += size
		}
	}
}

// ---- serializer ----

func (n *node) write(b *bytes.Buffer) {
	switch n.kind {
	case kNull:
		b.WriteString("null")
	case kTrue:
		b.WriteString("true")
	case kFalse:
		b.WriteString("false")
	case kNumber:
		b.WriteString(strconv.FormatInt(n.num, 10))
	case kString:
		writeString(b, n.str)
	case kArray:
		b.WriteByte('[')
		for i, v := range n.arr {
			if i > 0 {
				b.WriteByte(',')
			}
			v.write(b)
		}
		b.WriteByte(']')
	case kObject:
		ms := make([]member, len(n.members))
		copy(ms, n.members)
		keys := make([][]uint16, len(ms))
		for i := range ms {
			keys[i] = utf16.Encode([]rune(ms[i].name))
		}
		idx := make([]int, len(ms))
		for i := range idx {
			idx[i] = i
		}
		sort.Slice(idx, func(a, c int) bool { return lessUTF16(keys[idx[a]], keys[idx[c]]) })
		b.WriteByte('{')
		for i, j := range idx {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, ms[j].name)
			b.WriteByte(':')
			ms[j].value.write(b)
		}
		b.WriteByte('}')
	}
}

func lessUTF16(a, b []uint16) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

const hexDigits = "0123456789abcdef"

// writeString writes s (valid UTF-8) using the JCS / ECMAScript escaping rules.
func writeString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if c < 0x20 {
				b.WriteString(`\u00`)
				b.WriteByte(hexDigits[c>>4])
				b.WriteByte(hexDigits[c&0xF])
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
}
