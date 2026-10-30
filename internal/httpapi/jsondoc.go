package httpapi

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
)

// errJSONDoc is the syntax failure checkJSONDocument maps to HTTP 400.
// The walker applies the canonjson subset to a body that is already inside
// its class cap and above canonjson.MaxBytes. It stores only the depth and
// the keys of objects that are still open.
var errJSONDoc = errors.New("json: invalid document")

// errSchemaMismatch is a structural schema failure found while walking a
// body above canonjson.MaxBytes. The caller maps it to HTTP 422.
var errSchemaMismatch = errors.New("json: schema mismatch")

// jsonCheck is the optional work done in the same walk as the syntax check.
// onIdentity sees each node_id, node_incarnation_id, and incarnation_id.
// schema, when set, is a streamable request schema; a mismatch is reported
// as errSchemaMismatch after the document has been scanned.
type jsonCheck struct {
	onIdentity func(key, value string, isString bool)
	schema     *openapi3.Schema
	patterns   map[string]*regexp.Regexp
	schemaErr  error
}

// walkJSONDocument checks one JSON value with the signed-document rules:
// UTF-8, no byte order mark, no unescaped control character, no lone
// UTF-16 surrogate, no duplicate key at any object level, depth at most
// canonjson.MaxDepth, integers within ±(2^53−1) with no fraction or
// exponent, and no trailing data.
func walkJSONDocument(raw []byte) error {
	return walkChecked(raw, nil)
}

func walkChecked(raw []byte, chk *jsonCheck) error {
	if bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return errJSONDoc
	}
	if !utf8.Valid(raw) {
		return errJSONDoc
	}
	w := &jsonWalk{data: raw, chk: chk}
	w.skipWS()
	var schema *openapi3.Schema
	if chk != nil {
		schema = chk.schema
	}
	if err := w.valueUnder(schema); err != nil {
		return err
	}
	w.skipWS()
	if w.pos != len(w.data) {
		return errJSONDoc
	}
	if chk != nil && chk.schemaErr != nil {
		return errSchemaMismatch
	}
	return nil
}

type jsonWalk struct {
	data  []byte
	pos   int
	depth int
	chk   *jsonCheck
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

func (w *jsonWalk) valueUnder(s *openapi3.Schema) error {
	if w.pos >= len(w.data) {
		return errJSONDoc
	}
	c := w.data[w.pos]
	if s != nil {
		switch c {
		case 'n':
			if s.Nullable || s.Type.Permits("null") {
				return w.literal("null")
			}
			w.badSchema()
			s = nil
		case '{':
			if !s.Type.Permits("object") {
				w.badSchema()
				s = nil
			}
		case '[':
			if !s.Type.Permits("array") {
				w.badSchema()
				s = nil
			}
		case '"':
			if !s.Type.Permits("string") {
				w.badSchema()
				s = nil
			}
		case 't', 'f':
			if !s.Type.Permits("boolean") {
				w.badSchema()
				s = nil
			}
		default:
			if c == '-' || (c >= '0' && c <= '9') {
				if !s.Type.Permits("integer") && !s.Type.Permits("number") {
					w.badSchema()
					s = nil
				}
			}
		}
	}
	switch c {
	case '{':
		return w.objectUnder(s)
	case '[':
		return w.arrayUnder(s)
	case '"':
		return w.stringValue(s)
	case 't':
		if err := w.literal("true"); err != nil {
			return err
		}
		w.checkEnumScalar(s, true)
		return nil
	case 'f':
		if err := w.literal("false"); err != nil {
			return err
		}
		w.checkEnumScalar(s, false)
		return nil
	case 'n':
		return w.literal("null")
	case '-':
		fallthrough
	default:
		if c == '-' || (c >= '0' && c <= '9') {
			n, err := w.number()
			if err != nil {
				return err
			}
			w.checkInt(s, n)
			return nil
		}
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

func (w *jsonWalk) objectUnder(s *openapi3.Schema) error {
	if err := w.enter(); err != nil {
		return err
	}
	w.pos++ // '{'
	w.skipWS()
	if w.pos < len(w.data) && w.data[w.pos] == '}' {
		w.pos++
		w.depth--
		w.checkObjectClose(s, nil)
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
		prop, extra := propertySchema(s, key)
		if extra {
			w.badSchema()
		}
		if w.chk != nil && w.chk.onIdentity != nil && isIdentityKey(key) {
			if err = w.identityValue(key, prop); err != nil {
				return err
			}
		} else if err = w.valueUnder(prop); err != nil {
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
			w.checkObjectClose(s, seen)
			return nil
		default:
			return errJSONDoc
		}
	}
}

func (w *jsonWalk) arrayUnder(s *openapi3.Schema) error {
	if err := w.enter(); err != nil {
		return err
	}
	w.pos++ // '['
	w.skipWS()
	var item *openapi3.Schema
	if s != nil && s.Items != nil {
		item = s.Items.Value
	}
	if w.pos < len(w.data) && w.data[w.pos] == ']' {
		w.pos++
		w.depth--
		w.checkArrayClose(s, 0)
		return nil
	}
	n := 0
	for {
		w.skipWS()
		n++
		if s != nil && s.MaxItems != nil && uint64(n) > *s.MaxItems {
			w.badSchema()
		}
		if err := w.valueUnder(item); err != nil {
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
			w.checkArrayClose(s, n)
			return nil
		default:
			return errJSONDoc
		}
	}
}

func (w *jsonWalk) number() (int64, error) {
	neg := false
	if w.data[w.pos] == '-' {
		neg = true
		w.pos++
		if w.pos >= len(w.data) {
			return 0, errJSONDoc
		}
	}
	digitsStart := w.pos
	for w.pos < len(w.data) && w.data[w.pos] >= '0' && w.data[w.pos] <= '9' {
		w.pos++
	}
	digits := w.data[digitsStart:w.pos]
	if len(digits) == 0 {
		return 0, errJSONDoc
	}
	if w.pos < len(w.data) {
		if c := w.data[w.pos]; c == '.' || c == 'e' || c == 'E' {
			return 0, errJSONDoc
		}
	}
	if len(digits) > 1 && digits[0] == '0' {
		return 0, errJSONDoc
	}
	if len(digits) > 16 { // 2^53-1 has 16 digits
		return 0, errJSONDoc
	}
	v, err := strconv.ParseInt(string(digits), 10, 64)
	if err != nil || v > canonjson.MaxSafeInteger {
		return 0, errJSONDoc
	}
	if neg && v == 0 {
		return 0, errJSONDoc
	}
	if neg {
		v = -v
	}
	return v, nil
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

func isIdentityKey(key string) bool {
	return key == "node_id" || key == "node_incarnation_id" || key == "incarnation_id"
}

func (w *jsonWalk) identityValue(key string, s *openapi3.Schema) error {
	if w.pos < len(w.data) && w.data[w.pos] == '"' {
		text, err := w.scanString(true)
		if err != nil {
			return err
		}
		w.chk.onIdentity(key, text, true)
		if s != nil {
			w.checkString(s, text)
		}
		return nil
	}
	w.chk.onIdentity(key, "", false)
	return w.valueUnder(s)
}

func (w *jsonWalk) badSchema() {
	if w.chk != nil && w.chk.schemaErr == nil {
		w.chk.schemaErr = errSchemaMismatch
	}
}

func propertySchema(parent *openapi3.Schema, key string) (*openapi3.Schema, bool) {
	if parent == nil {
		return nil, false
	}
	if parent.Properties != nil {
		if ref := parent.Properties[key]; ref != nil && ref.Value != nil {
			return ref.Value, false
		}
	}
	if ref := parent.AdditionalProperties.Schema; ref != nil && ref.Value != nil {
		return ref.Value, false
	}
	if allowed := parent.AdditionalProperties.Has; allowed != nil && !*allowed {
		return nil, true
	}
	return nil, false
}

func (w *jsonWalk) checkObjectClose(s *openapi3.Schema, seen map[string]struct{}) {
	if s == nil {
		return
	}
	for _, req := range s.Required {
		if _, ok := seen[req]; !ok {
			w.badSchema()
			break
		}
	}
	n := uint64(len(seen))
	if s.MinProps != 0 && n < s.MinProps {
		w.badSchema()
	}
	if s.MaxProps != nil && n > *s.MaxProps {
		w.badSchema()
	}
}

func (w *jsonWalk) checkArrayClose(s *openapi3.Schema, n int) {
	if s == nil {
		return
	}
	if uint64(n) < s.MinItems {
		w.badSchema()
	}
	if s.MaxItems != nil && uint64(n) > *s.MaxItems {
		w.badSchema()
	}
}

// stringValue scans one JSON string. A schema check uses the raw bytes when
// the string has no escape, so a multi-megabyte usage batch does not copy
// every value onto the heap.
func (w *jsonWalk) stringValue(s *openapi3.Schema) error {
	if w.pos >= len(w.data) || w.data[w.pos] != '"' {
		return errJSONDoc
	}
	start := w.pos
	w.pos++
	rawStart := w.pos
	for w.pos < len(w.data) {
		c := w.data[w.pos]
		if c == '"' {
			raw := w.data[rawStart:w.pos]
			w.pos++
			if s != nil {
				w.checkStringBytes(s, raw)
			}
			return nil
		}
		if c == '\\' || c < 0x20 {
			w.pos = start
			text, err := w.scanString(s != nil)
			if err != nil {
				return err
			}
			if s != nil {
				w.checkString(s, text)
			}
			return nil
		}
		w.pos++
	}
	return errJSONDoc
}

func (w *jsonWalk) checkStringBytes(s *openapi3.Schema, v []byte) {
	if s.MinLength != 0 || s.MaxLength != nil {
		length := int64(utf8.RuneCount(v))
		if s.MinLength != 0 && length < int64(s.MinLength) {
			w.badSchema()
		}
		if s.MaxLength != nil && length > int64(*s.MaxLength) {
			w.badSchema()
		}
	}
	if s.Pattern != "" && !w.matchPatternBytes(s.Pattern, v) {
		w.badSchema()
	}
	switch s.Format {
	case "uuid":
		if !uuidBytes(v) {
			w.badSchema()
		}
	case "date-time":
		if _, err := time.Parse(time.RFC3339Nano, string(v)); err != nil {
			w.badSchema()
		}
	}
	if len(s.Enum) > 0 && !enumBytes(s, v) {
		w.badSchema()
	}
}

func uuidBytes(v []byte) bool {
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

func enumBytes(s *openapi3.Schema, v []byte) bool {
	for _, e := range s.Enum {
		text, ok := e.(string)
		if ok && text == string(v) {
			return true
		}
	}
	return false
}

func (w *jsonWalk) matchPatternBytes(pattern string, v []byte) bool {
	switch pattern {
	case "^[1-9][0-9]*$":
		return decimalBytes(v, false)
	case "^(0|[1-9][0-9]*)$":
		return decimalBytes(v, true)
	case "^[0-9a-f]{64}$":
		if len(v) != 64 {
			return false
		}
		for _, c := range v {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
		return true
	case "^[0-9]{4}-(0[1-9]|1[0-2])$":
		return len(v) == 7 && v[4] == '-' && v[0] >= '0' && v[0] <= '9' && v[1] >= '0' && v[1] <= '9' && v[2] >= '0' && v[2] <= '9' && v[3] >= '0' && v[3] <= '9' && ((v[5] == '0' && v[6] >= '1' && v[6] <= '9') || (v[5] == '1' && v[6] >= '0' && v[6] <= '2'))
	case "^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?Z$":
		return dateTimeBytes(v)
	default:
		return w.matchPattern(pattern, string(v))
	}
}

func decimalBytes(v []byte, zeroOK bool) bool {
	if len(v) == 0 || len(v) > 20 {
		return false
	}
	if v[0] == '0' {
		return zeroOK && len(v) == 1
	}
	for _, c := range v {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func dateTimeBytes(v []byte) bool {
	// YYYY-MM-DDTHH:MM:SSZ or with a 1..9 digit fraction.
	if len(v) < 20 || v[4] != '-' || v[7] != '-' || v[10] != 'T' || v[13] != ':' || v[16] != ':' {
		return false
	}
	for _, i := range []int{0, 1, 2, 3, 5, 6, 8, 9, 11, 12, 14, 15, 17, 18} {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
	}
	if v[len(v)-1] != 'Z' {
		return false
	}
	if len(v) == 20 {
		return true
	}
	if v[19] != '.' || len(v) < 22 || len(v) > 30 {
		return false
	}
	for _, c := range v[20 : len(v)-1] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (w *jsonWalk) checkString(s *openapi3.Schema, v string) {
	if s.MinLength != 0 || s.MaxLength != nil {
		var length int64
		for _, r := range v {
			if utf16.IsSurrogate(r) {
				length += 2
			} else {
				length++
			}
		}
		if s.MinLength != 0 && length < int64(s.MinLength) {
			w.badSchema()
		}
		if s.MaxLength != nil && length > int64(*s.MaxLength) {
			w.badSchema()
		}
	}
	if s.Pattern != "" && !w.matchPattern(s.Pattern, v) {
		w.badSchema()
	}
	switch s.Format {
	case "uuid":
		if _, err := uuid.Parse(v); err != nil {
			w.badSchema()
		}
	case "date-time":
		if _, err := time.Parse(time.RFC3339Nano, v); err != nil {
			w.badSchema()
		}
	}
	if len(s.Enum) > 0 && !enumString(s, v) {
		w.badSchema()
	}
}

func (w *jsonWalk) matchPattern(pattern, v string) bool {
	if w.chk == nil {
		return false
	}
	if w.chk.patterns == nil {
		w.chk.patterns = map[string]*regexp.Regexp{}
	}
	re, ok := w.chk.patterns[pattern]
	if !ok {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return false
		}
		w.chk.patterns[pattern] = re
	}
	return re.MatchString(v)
}

func enumString(s *openapi3.Schema, v string) bool {
	for _, e := range s.Enum {
		if text, ok := e.(string); ok && text == v {
			return true
		}
	}
	return false
}

func (w *jsonWalk) checkInt(s *openapi3.Schema, n int64) {
	if s == nil {
		return
	}
	if s.Min != nil && float64(n) < *s.Min {
		w.badSchema()
	}
	if s.Max != nil && float64(n) > *s.Max {
		w.badSchema()
	}
	if len(s.Enum) == 0 {
		return
	}
	for _, e := range s.Enum {
		switch x := e.(type) {
		case float64:
			if x == float64(n) {
				return
			}
		case int:
			if int64(x) == n {
				return
			}
		case int64:
			if x == n {
				return
			}
		}
	}
	w.badSchema()
}

func (w *jsonWalk) checkEnumScalar(s *openapi3.Schema, v bool) {
	if s == nil || len(s.Enum) == 0 {
		return
	}
	for _, e := range s.Enum {
		if b, ok := e.(bool); ok && b == v {
			return
		}
	}
	w.badSchema()
}

// schemaStreamable reports whether the streaming checker covers every keyword
// that would change acceptance. A schema that uses a keyword it does not
// cover stays on openapi3filter.
func schemaStreamable(s *openapi3.Schema) bool {
	return schemaNodeStreamable(s, map[*openapi3.Schema]struct{}{})
}

func schemaNodeStreamable(s *openapi3.Schema, seen map[*openapi3.Schema]struct{}) bool {
	if s == nil {
		return true
	}
	if _, ok := seen[s]; ok {
		return true
	}
	seen[s] = struct{}{}
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 || len(s.AllOf) > 0 || s.Not != nil || s.Discriminator != nil {
		return false
	}
	if s.UniqueItems || s.MultipleOf != nil || len(s.PatternProperties) > 0 || s.If != nil || s.PrefixItems != nil || s.ExclusiveMin.IsSet() || s.ExclusiveMax.IsSet() || s.Const != nil {
		return false
	}
	if s.Format != "" && s.Format != "uuid" && s.Format != "date-time" {
		return false
	}
	if len(s.Enum) > 0 && s.Type != nil && !s.Type.Permits("string") && !s.Type.Permits("integer") && !s.Type.Permits("number") && !s.Type.Permits("boolean") {
		return false
	}
	for _, p := range s.Properties {
		if p == nil || p.Value == nil || !schemaNodeStreamable(p.Value, seen) {
			return false
		}
	}
	if s.Items != nil && (s.Items.Value == nil || !schemaNodeStreamable(s.Items.Value, seen)) {
		return false
	}
	if s.AdditionalProperties.Schema != nil && (s.AdditionalProperties.Schema.Value == nil || !schemaNodeStreamable(s.AdditionalProperties.Schema.Value, seen)) {
		return false
	}
	return true
}
