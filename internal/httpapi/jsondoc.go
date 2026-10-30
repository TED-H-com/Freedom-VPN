package httpapi

import (
	"bytes"
	"errors"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// errJSONDoc is the syntax failure checkJSONDocument maps to HTTP 400.
// The walker applies the canonjson subset to a body that is already inside
// its class cap and above canonjson.MaxBytes. It stores only the depth and
// the keys of objects that are still open.
var errJSONDoc = errors.New("json: invalid document")

// walkJSONDocument checks one JSON value with the signed-document rules:
// UTF-8, no byte order mark, no unescaped control character, no lone
// UTF-16 surrogate, no duplicate key at any object level, depth at most
// canonjson.MaxDepth, integers within ±(2^53−1) with no fraction or
// exponent, and no trailing data.
func walkJSONDocument(raw []byte) error {
	if bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return errJSONDoc
	}
	if !utf8.Valid(raw) {
		return errJSONDoc
	}
	w := &jsonWalk{data: raw}
	w.skipWS()
	if err := w.value(); err != nil {
		return err
	}
	w.skipWS()
	if w.pos != len(w.data) {
		return errJSONDoc
	}
	return nil
}

type jsonWalk struct {
	data  []byte
	pos   int
	depth int
}

func (w *jsonWalk) skipWS() {
	for w.pos < len(w.data) {
		switch w.data[w.pos] {
		case ' ', '\t', '\n', '\r':
			w.pos++
		default:
			return
		}
	}
}

func (w *jsonWalk) value() error {
	if w.pos >= len(w.data) {
		return errJSONDoc
	}
	switch c := w.data[w.pos]; {
	case c == '{':
		return w.object()
	case c == '[':
		return w.array()
	case c == '"':
		_, err := w.scanString(false)
		return err
	case c == 't':
		return w.literal("true")
	case c == 'f':
		return w.literal("false")
	case c == 'n':
		return w.literal("null")
	case c == '-' || (c >= '0' && c <= '9'):
		return w.number()
	default:
		return errJSONDoc
	}
}

func (w *jsonWalk) literal(lit string) error {
	if w.pos+len(lit) > len(w.data) || !bytes.Equal(w.data[w.pos:w.pos+len(lit)], []byte(lit)) {
		return errJSONDoc
	}
	w.pos += len(lit)
	return nil
}

func (w *jsonWalk) enter() error {
	w.depth++
	if w.depth > canonjson.MaxDepth {
		return errJSONDoc
	}
	return nil
}

func (w *jsonWalk) object() error {
	if err := w.enter(); err != nil {
		return err
	}
	w.pos++ // '{'
	w.skipWS()
	if w.pos < len(w.data) && w.data[w.pos] == '}' {
		w.pos++
		w.depth--
		return nil
	}
	seen := make(map[string]struct{})
	for {
		w.skipWS()
		if w.pos >= len(w.data) || w.data[w.pos] != '"' {
			return errJSONDoc
		}
		key, err := w.scanString(true)
		if err != nil {
			return err
		}
		if _, ok := seen[key]; ok {
			return errJSONDoc
		}
		seen[key] = struct{}{}
		w.skipWS()
		if w.pos >= len(w.data) || w.data[w.pos] != ':' {
			return errJSONDoc
		}
		w.pos++
		w.skipWS()
		if err = w.value(); err != nil {
			return err
		}
		w.skipWS()
		if w.pos >= len(w.data) {
			return errJSONDoc
		}
		switch w.data[w.pos] {
		case ',':
			w.pos++
		case '}':
			w.pos++
			w.depth--
			return nil
		default:
			return errJSONDoc
		}
	}
}

func (w *jsonWalk) array() error {
	if err := w.enter(); err != nil {
		return err
	}
	w.pos++ // '['
	w.skipWS()
	if w.pos < len(w.data) && w.data[w.pos] == ']' {
		w.pos++
		w.depth--
		return nil
	}
	for {
		w.skipWS()
		if err := w.value(); err != nil {
			return err
		}
		w.skipWS()
		if w.pos >= len(w.data) {
			return errJSONDoc
		}
		switch w.data[w.pos] {
		case ',':
			w.pos++
		case ']':
			w.pos++
			w.depth--
			return nil
		default:
			return errJSONDoc
		}
	}
}

func (w *jsonWalk) number() error {
	neg := false
	if w.data[w.pos] == '-' {
		neg = true
		w.pos++
		if w.pos >= len(w.data) {
			return errJSONDoc
		}
	}
	digitsStart := w.pos
	for w.pos < len(w.data) && w.data[w.pos] >= '0' && w.data[w.pos] <= '9' {
		w.pos++
	}
	digits := w.data[digitsStart:w.pos]
	if len(digits) == 0 {
		return errJSONDoc
	}
	if w.pos < len(w.data) {
		if c := w.data[w.pos]; c == '.' || c == 'e' || c == 'E' {
			return errJSONDoc
		}
	}
	if len(digits) > 1 && digits[0] == '0' {
		return errJSONDoc
	}
	if len(digits) > 16 { // 2^53-1 has 16 digits
		return errJSONDoc
	}
	v, err := strconv.ParseInt(string(digits), 10, 64)
	if err != nil || v > canonjson.MaxSafeInteger {
		return errJSONDoc
	}
	if neg && v == 0 {
		return errJSONDoc
	}
	return nil
}

func (w *jsonWalk) scanString(keep bool) (string, error) {
	if w.pos >= len(w.data) || w.data[w.pos] != '"' {
		return "", errJSONDoc
	}
	w.pos++
	var decoded []byte
	for {
		if w.pos >= len(w.data) {
			return "", errJSONDoc
		}
		c := w.data[w.pos]
		switch {
		case c == '"':
			w.pos++
			if !keep {
				return "", nil
			}
			return string(decoded), nil
		case c == '\\':
			w.pos++
			if w.pos >= len(w.data) {
				return "", errJSONDoc
			}
			e := w.data[w.pos]
			w.pos++
			var plain byte
			var asRune rune
			runeEsc := false
			switch e {
			case '"', '\\', '/':
				plain = e
			case 'b':
				plain = '\b'
			case 'f':
				plain = '\f'
			case 'n':
				plain = '\n'
			case 'r':
				plain = '\r'
			case 't':
				plain = '\t'
			case 'u':
				r, err := w.u4()
				if err != nil {
					return "", err
				}
				if utf16.IsSurrogate(r) {
					if r >= 0xDC00 || w.pos+6 > len(w.data) || w.data[w.pos] != '\\' || w.data[w.pos+1] != 'u' {
						return "", errJSONDoc
					}
					w.pos += 2
					r2, err := w.u4()
					if err != nil {
						return "", err
					}
					if r2 < 0xDC00 || r2 > 0xDFFF {
						return "", errJSONDoc
					}
					r = utf16.DecodeRune(r, r2)
				}
				asRune = r
				runeEsc = true
			default:
				return "", errJSONDoc
			}
			if keep {
				if runeEsc {
					decoded = utf8.AppendRune(decoded, asRune)
				} else {
					decoded = append(decoded, plain)
				}
			}
		case c < 0x20:
			return "", errJSONDoc
		case c < utf8.RuneSelf:
			if keep {
				decoded = append(decoded, c)
			}
			w.pos++
		default:
			r, size := utf8.DecodeRune(w.data[w.pos:])
			if r == utf8.RuneError && size <= 1 {
				return "", errJSONDoc
			}
			if keep {
				decoded = append(decoded, w.data[w.pos:w.pos+size]...)
			}
			w.pos += size
		}
	}
}

func (w *jsonWalk) u4() (rune, error) {
	if w.pos+4 > len(w.data) {
		return 0, errJSONDoc
	}
	var r rune
	for i := 0; i < 4; i++ {
		h, ok := jsonHex(w.data[w.pos+i])
		if !ok {
			return 0, errJSONDoc
		}
		r = r<<4 | h
	}
	w.pos += 4
	return r, nil
}

func jsonHex(c byte) (rune, bool) {
	switch {
	case c >= '0' && c <= '9':
		return rune(c - '0'), true
	case c >= 'a' && c <= 'f':
		return rune(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return rune(c-'A') + 10, true
	default:
		return 0, false
	}
}
