package netbird

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type object map[string]any

func stringField(o object, k string) string { s, _ := o[k].(string); return s }
func stableName(kind, id string) string {
	sum := sha256.Sum256([]byte(id))
	return "fvpn-" + kind + "-" + hex.EncodeToString(sum[:12])
}
func decodeObject(v any) object {
	b, _ := json.Marshal(v)
	var o object
	_ = json.Unmarshal(b, &o)
	return o
}
func sorted(ids []string) []string {
	v := append([]string{}, ids...)
	sort.Strings(v)
	out := v[:0]
	for _, id := range v {
		if len(out) == 0 || id != out[len(out)-1] {
			out = append(out, id)
		}
	}
	return out
}
func ids(v any) []string {
	out := []string{}
	a, _ := v.([]any)
	for _, x := range a {
		switch t := x.(type) {
		case string:
			out = append(out, t)
		case map[string]any:
			out = append(out, stringField(t, "id"))
		}
	}
	return sorted(out)
}

// canonical is RFC 8785 for this adapter's deliberately limited schema: ASCII
// object keys, valid Unicode strings, booleans and safe integers. It rejects
// fractional numbers and non-ASCII keys; no signatures use this helper. Replace
// with pkg/canonjson when that shared package lands without changing hashes.
func canonical(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "null", nil
	case bool:
		if x {
			return "true", nil
		}
		return "false", nil
	case string:
		if !utf8.ValidString(x) {
			return "", failure(Invalid)
		}
		var b strings.Builder
		b.WriteByte('"')
		for _, r := range x {
			switch r {
			case '"', '\\':
				b.WriteByte('\\')
				b.WriteRune(r)
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
				if r < 32 {
					b.WriteString(`\u00`)
					b.WriteString(hex.EncodeToString([]byte{byte(r)}))
				} else {
					b.WriteRune(r)
				}
			}
		}
		b.WriteByte('"')
		return b.String(), nil
	case float64:
		if x != float64(int64(x)) || x > 9007199254740991 || x < -9007199254740991 {
			return "", failure(Invalid)
		}
		return strconv.FormatInt(int64(x), 10), nil
	case []any:
		a := []string{}
		for _, e := range x {
			s, err := canonical(e)
			if err != nil {
				return "", err
			}
			a = append(a, s)
		}
		return "[" + strings.Join(a, ",") + "]", nil
	case map[string]any:
		keys := []string{}
		for k := range x {
			for _, r := range k {
				if r > 127 {
					return "", failure(Invalid)
				}
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		a := []string{}
		for _, k := range keys {
			s, e := canonical(x[k])
			if e != nil {
				return "", e
			}
			q, _ := canonical(k)
			a = append(a, q+":"+s)
		}
		return "{" + strings.Join(a, ",") + "}", nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", failure(Invalid)
		}
		var y any
		if json.Unmarshal(b, &y) != nil {
			return "", failure(Invalid)
		}
		return canonical(y)
	}
}
func hash(v any) (string, error) {
	s, e := canonical(v)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:]), nil
}
func (c *Client) list(ctx context.Context, kind string) ([]object, error) {
	var a []object
	e := c.call(ctx, "GET", endpoint(kind, ""), nil, &a)
	return a, e
}
func (c *Client) get(ctx context.Context, kind, id string) (object, error) {
	var o object
	e := c.call(ctx, "GET", endpoint(kind, id), nil, &o)
	if e == nil && (id == "" || stringField(o, "id") != id) {
		return nil, failure(Unconfirmed)
	}
	return o, e
}
func nameField(kind string) string {
	if kind == "routes" {
		return "network_id"
	}
	return "name"
}

// locate checks both stored IDs and stable names. Multiple matches, renames, or
// an occupied binding fail closed; a name alone never authorizes a member peer.
func (c *Client) locate(ctx context.Context, kind, key, name string) (object, error) {
	b, ok, e := c.store.Get(ctx, c.deployment, kind, key)
	if e != nil {
		return nil, e
	}
	var known object
	if ok {
		known, e = c.get(ctx, kind, b.ExternalID)
		if e != nil && !IsKind(e, NotFound) {
			return nil, e
		}
		if e == nil && stringField(known, nameField(kind)) != name {
			return nil, failure(Conflict)
		}
	}
	all, e := c.list(ctx, kind)
	if e != nil {
		return nil, e
	}
	var found object
	for _, o := range all {
		if stringField(o, nameField(kind)) == name {
			if found != nil {
				return nil, failure(Conflict)
			}
			found = o
		}
	}
	if known != nil && (found == nil || stringField(known, "id") != stringField(found, "id")) {
		return nil, failure(Conflict)
	}
	if ok && known == nil {
		// Only an observed absence releases an old external binding.
		if found != nil {
			return nil, failure(Conflict)
		}
		if e = c.store.Delete(ctx, c.deployment, kind, key); e != nil {
			return nil, e
		}
	}
	return found, nil
}
func projection(kind string, o object) object {
	o = decodeObject(o) // Normalize typed request slices to the same JSON tree as responses.
	// Only API request fields are hashed; response-only IDs, counters and times
	// are omitted. Normalize response group objects to request group IDs.
	var fields []string
	switch kind {
	case "groups":
		fields = []string{"name", "peers", "resources"}
	case "routes":
		fields = []string{"network_id", "description", "network", "peer", "groups", "enabled", "masquerade", "metric", "keep_route", "skip_auto_apply", "access_control_groups", "peer_groups", "domains"}
	case "policies":
		fields = []string{"name", "description", "enabled", "rules", "source_posture_checks"}
	case "dns/nameservers":
		fields = []string{"name", "description", "domains", "enabled", "groups", "nameservers", "primary", "search_domains_enabled"}
	}
	p := object{}
	for _, f := range fields {
		if v, ok := o[f]; ok {
			p[f] = v
		}
	}
	switch kind {
	case "groups":
		p["peers"] = ids(o["peers"])
		if p["resources"] == nil {
			p["resources"] = []any{}
		}
	case "routes":
		p["groups"] = ids(o["groups"])
		p["access_control_groups"] = ids(o["access_control_groups"])
		p["peer_groups"] = ids(o["peer_groups"])
		p["domains"] = ids(o["domains"])
	case "policies":
		p["source_posture_checks"] = ids(o["source_posture_checks"])
		rules := []any{}
		a, _ := o["rules"].([]any)
		for _, r := range a {
			m, _ := r.(map[string]any)
			q := object{}
			for _, f := range []string{"name", "description", "enabled", "action", "bidirectional", "protocol", "ports", "port_ranges", "sourceResource", "destinationResource", "authorized_groups"} {
				if v, ok := m[f]; ok {
					q[f] = v
				}
			}
			q["sources"] = ids(m["sources"])
			q["destinations"] = ids(m["destinations"])
			q["ports"] = ids(m["ports"])
			rules = append(rules, q)
		}
		p["rules"] = rules
	case "dns/nameservers":
		p["groups"] = ids(o["groups"])
		p["domains"] = ids(o["domains"])
	}
	return p
}
func (c *Client) bind(ctx context.Context, kind, key, id, h string) error {
	b := Binding{DeploymentID: c.deployment, Kind: kind, InternalID: key, ExternalID: id, DesiredHash: h}
	if h != "" {
		b.LastObservedAt = time.Now().UTC()
	}
	return c.store.Put(ctx, b)
}
func (c *Client) ensure(ctx context.Context, kind, key string, want object) (object, error) {
	name := stringField(want, nameField(kind))
	found, e := c.locate(ctx, kind, key, name)
	if e != nil {
		return nil, e
	}
	wh, e := hash(projection(kind, want))
	if e != nil {
		return nil, e
	}
	id := stringField(found, "id")
	oh, _ := hash(projection(kind, found))
	if found == nil || wh != oh {
		method := "PUT"
		if found == nil {
			method = "POST"
		}
		var reply object
		if e = c.call(ctx, method, endpoint(kind, id), want, &reply); e != nil {
			return nil, e
		}
		if found == nil {
			id = stringField(reply, "id")
		}
		if id == "" {
			return nil, failure(Unconfirmed)
		}
		if e = c.bind(ctx, kind, key, id, ""); e != nil {
			return nil, e
		}
	}
	// Even an apparent no-op requires a fresh GET, never just a cached binding.
	observed, e := c.get(ctx, kind, id)
	if e != nil {
		return nil, e
	}
	h, e := hash(projection(kind, observed))
	if e != nil {
		return nil, e
	}
	if h != wh {
		return nil, failure(Unconfirmed)
	}
	if e = c.bind(ctx, kind, key, id, wh); e != nil {
		return nil, e
	}
	return observed, nil
}
func (c *Client) remove(ctx context.Context, kind, id string) error {
	e := c.call(ctx, "DELETE", endpoint(kind, id), nil, nil)
	if e != nil && !IsKind(e, NotFound) {
		return e
	}
	_, e = c.get(ctx, kind, id)
	if IsKind(e, NotFound) {
		return nil
	}
	if e != nil {
		return e
	}
	return failure(Unconfirmed)
}
