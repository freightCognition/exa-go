package exa

// Spec coverage: these tests compare the SDK against Exa's OpenAPI specs in
// testdata/ and fail on any endpoint or field the SDK doesn't model.
//
// Each area registers, from its own *_test.go init():
//   - registerEndpoint: a call that hits one spec operation
//   - registerSchema:   the Go type for one spec component schema
//   - exemptSchema:     a component schema deliberately not modelled, with why

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
)

type endpointCase struct {
	method, path string // spec operation, e.g. "GET", "/v0/websets/{id}"
	admin        bool   // team management API (admin base URL)
	call         func(context.Context, *Client) error
}

type schemaCase struct {
	name string
	typ  reflect.Type
	skip map[string]bool
}

var (
	regMu         sync.Mutex
	endpointCases []endpointCase
	schemaCases   = map[string]schemaCase{}
	schemaExempt  = map[string]string{}
)

// registerEndpoint records that call exercises the spec operation method+path.
func registerEndpoint(method, path string, call func(context.Context, *Client) error) {
	regMu.Lock()
	defer regMu.Unlock()
	endpointCases = append(endpointCases, endpointCase{method: method, path: path, call: call})
}

// registerAdminEndpoint is registerEndpoint for the team management spec.
func registerAdminEndpoint(method, path string, call func(context.Context, *Client) error) {
	regMu.Lock()
	defer regMu.Unlock()
	endpointCases = append(endpointCases, endpointCase{method: method, path: path, admin: true, call: call})
}

// registerSchema maps a spec component schema to its Go type. skip lists
// names exempt from comparison in either direction: spec properties left out
// on purpose, or Go fields a shared type carries that this schema lacks
// (e.g. Page.Object on a list without "object"). Keep it empty unless needed.
func registerSchema(name string, v any, skip ...string) {
	regMu.Lock()
	defer regMu.Unlock()
	if _, dup := schemaCases[name]; dup {
		panic("schema registered twice: " + name)
	}
	s := schemaCase{name: name, typ: reflect.TypeOf(v), skip: map[string]bool{}}
	for _, k := range skip {
		s.skip[k] = true
	}
	schemaCases[name] = s
}

// exemptSchema marks a component schema as intentionally not having its own Go type.
func exemptSchema(name, reason string) {
	regMu.Lock()
	defer regMu.Unlock()
	schemaExempt[name] = reason
}

// ---- spec loading ----

type specDoc struct {
	Paths      map[string]map[string]json.RawMessage `json:"paths"`
	Components struct {
		Schemas map[string]map[string]any `json:"schemas"`
	} `json:"components"`
}

var (
	specOnce            sync.Once
	mainSpec, adminSpec specDoc
	specErr             error
	httpMethods         = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
)

func loadSpecs(t *testing.T) (specDoc, specDoc) {
	t.Helper()
	specOnce.Do(func() {
		for file, dst := range map[string]*specDoc{"testdata/exa-spec.json": &mainSpec, "testdata/team-management-spec.json": &adminSpec} {
			b, err := os.ReadFile(file)
			if err != nil {
				specErr = err
				return
			}
			if err := json.Unmarshal(b, dst); err != nil {
				specErr = fmt.Errorf("%s: %w", file, err)
				return
			}
		}
	})
	if specErr != nil {
		t.Fatal(specErr)
	}
	return mainSpec, adminSpec
}

func specOps(d specDoc) []string {
	var out []string
	for p, ops := range d.Paths {
		for m := range ops {
			if httpMethods[m] {
				out = append(out, strings.ToUpper(m)+" "+p)
			}
		}
	}
	sort.Strings(out)
	return out
}

var paramRe = regexp.MustCompile(`\{[^}]+\}`)

func pathMatches(template, got string) bool {
	// QuoteMeta escapes the braces; undo that before substituting params.
	quoted := strings.NewReplacer(`\{`, "{", `\}`, "}").Replace(regexp.QuoteMeta(template))
	re := "^" + paramRe.ReplaceAllString(quoted, `[^/]+`) + "$"
	return regexp.MustCompile(re).MatchString(got)
}

// ---- endpoint coverage ----

func TestSpecEndpointsCovered(t *testing.T) {
	main, admin := loadSpecs(t)

	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c, err := New(WithAPIKey("k"), WithBaseURL(srv.URL), WithAdminBaseURL(srv.URL+"/team-management"))
	if err != nil {
		t.Fatal(err)
	}

	covered := map[string]bool{}
	for _, ec := range endpointCases {
		mu.Lock()
		seen = nil
		mu.Unlock()
		if err := ec.call(context.Background(), c); err != nil {
			t.Errorf("%s %s: call failed: %v", ec.method, ec.path, err)
			continue
		}
		mu.Lock()
		got := append([]string(nil), seen...)
		mu.Unlock()
		want := ec.method + " " + ec.path
		ok := false
		for _, g := range got {
			m, p, _ := strings.Cut(g, " ")
			if ec.admin {
				p = strings.TrimPrefix(p, "/team-management")
			} else if strings.HasPrefix(ec.path, "/v0/") {
				// Websets paths are served under https://api.exa.ai/websets (the
				// spec sets that per path; /v0/teams/me omits it but is served there too).
				p = strings.TrimPrefix(p, "/websets")
			}
			if m == ec.method && pathMatches(ec.path, p) {
				ok = true
			}
		}
		if !ok {
			t.Errorf("%s: registered call sent %v", want, got)
			continue
		}
		key := want
		if ec.admin {
			key = "admin " + want
		}
		covered[key] = true
	}

	var missing []string
	for _, op := range specOps(main) {
		if !covered[op] {
			missing = append(missing, op)
		}
	}
	for _, op := range specOps(admin) {
		if !covered["admin "+op] {
			missing = append(missing, "admin "+op)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d spec operations have no SDK method:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// ---- schema coverage ----

func resolve(d specDoc, s map[string]any) map[string]any {
	for i := 0; i < 10; i++ {
		ref, ok := s["$ref"].(string)
		if !ok {
			return s
		}
		next := d.Components.Schemas[strings.TrimPrefix(ref, "#/components/schemas/")]
		if next == nil {
			return s
		}
		s = next
	}
	return s
}

// objectShape returns the merged properties of s, or nil when s is not a
// single object shape (a scalar, or a union of several object shapes).
func objectShape(d specDoc, s map[string]any) map[string]any {
	s = resolve(d, s)
	props := map[string]any{}
	found := false
	if p, ok := s["properties"].(map[string]any); ok {
		for k, v := range p {
			props[k] = v
		}
		found = true
	}
	if all, ok := s["allOf"].([]any); ok {
		for _, m := range all {
			if mm, ok := m.(map[string]any); ok {
				if sub := objectShape(d, mm); sub != nil {
					for k, v := range sub {
						props[k] = v
					}
					found = true
				}
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		alts, ok := s[key].([]any)
		if !ok {
			continue
		}
		var objs []map[string]any
		for _, a := range alts {
			if am, ok := a.(map[string]any); ok {
				if sub := objectShape(d, am); sub != nil {
					objs = append(objs, sub)
				}
			}
		}
		if len(objs) == 1 { // nullable object
			for k, v := range objs[0] {
				props[k] = v
			}
			found = true
		} else if len(objs) > 1 && !found {
			return nil // a real union: modelled by hand, checked per variant
		}
	}
	if !found {
		return nil
	}
	return props
}

// itemsOf unwraps arrays (possibly nullable) to their item schema.
func itemsOf(d specDoc, s map[string]any) map[string]any {
	s = resolve(d, s)
	if it, ok := s["items"].(map[string]any); ok {
		return it
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		if alts, ok := s[key].([]any); ok {
			for _, a := range alts {
				if am, ok := a.(map[string]any); ok {
					if it, ok := resolve(d, am)["items"].(map[string]any); ok {
						return it
					}
				}
			}
		}
	}
	return nil
}

func jsonFields(t reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return out
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" {
			for k, v := range jsonFields(f.Type) {
				out[k] = v
			}
			continue
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		out[name] = f.Type
	}
	return out
}

func structElem(t reflect.Type) reflect.Type {
	for {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			t = t.Elem()
		case reflect.Struct:
			return t
		default:
			return nil
		}
	}
}

// customJSON reports whether t decodes itself (a hand-written union), so its
// fields can't be compared one to one. Types that only customise encoding
// (e.g. "send true when empty") are still compared field by field.
func customJSON(t reflect.Type) bool {
	u := reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	b := reflect.TypeOf((*interface{ acceptsBool() })(nil)).Elem()
	if t.Implements(b) {
		return false // true|false|object options: fields still map onto the spec
	}
	return t.Implements(u) || reflect.PointerTo(t).Implements(u)
}

func compareShape(d specDoc, where string, spec map[string]any, goT reflect.Type, skip map[string]bool, depth int, problems *[]string) {
	if depth > 8 {
		return
	}
	props := objectShape(d, spec)
	st := structElem(goT)
	if props == nil || st == nil || customJSON(st) {
		return
	}
	fields := jsonFields(st)
	for name, ps := range props {
		if skip[name] {
			continue
		}
		ft, ok := fields[name]
		if !ok {
			*problems = append(*problems, fmt.Sprintf("%s: spec field %q missing from %s", where, name, st.Name()))
			continue
		}
		psm, _ := ps.(map[string]any)
		if psm == nil {
			continue
		}
		child := psm
		if it := itemsOf(d, psm); it != nil {
			child = it
		}
		compareShape(d, where+"."+name, child, ft, nil, depth+1, problems)
	}
	for name := range fields {
		if _, ok := props[name]; !ok && !skip[name] {
			*problems = append(*problems, fmt.Sprintf("%s: %s has field %q that the spec does not define", where, st.Name(), name))
		}
	}
}

func TestSpecSchemasCovered(t *testing.T) {
	main, admin := loadSpecs(t)
	var problems []string
	for _, doc := range []specDoc{main, admin} {
		names := make([]string, 0, len(doc.Components.Schemas))
		for n := range doc.Components.Schemas {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, name := range names {
			s := doc.Components.Schemas[name]
			sc, registered := schemaCases[name]
			if !registered {
				if _, ok := schemaExempt[name]; !ok && objectShape(doc, s) != nil {
					problems = append(problems, fmt.Sprintf("%s: object schema has no registered Go type", name))
				}
				continue
			}
			compareShape(doc, name, s, sc.typ, sc.skip, 0, &problems)
		}
	}
	for name := range schemaCases {
		if main.Components.Schemas[name] == nil && admin.Components.Schemas[name] == nil {
			problems = append(problems, fmt.Sprintf("%s: registered but not in either spec", name))
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Errorf("%d schema gaps:\n  %s", len(problems), strings.Join(problems, "\n  "))
	}
}

func TestPathMatches(t *testing.T) {
	cases := []struct {
		tmpl, got string
		want      bool
	}{
		{"/v0/websets/{id}", "/v0/websets/ws_1", true},
		{"/v0/websets/{id}", "/v0/websets/ws_1/cancel", false},
		{"/v0/websets/{webset}/items/{id}", "/v0/websets/a/items/b", true},
		{"/search", "/search", true},
		{"/search", "/searchx", false},
	}
	for _, c := range cases {
		if got := pathMatches(c.tmpl, c.got); got != c.want {
			t.Errorf("pathMatches(%q,%q) = %v", c.tmpl, c.got, got)
		}
	}
}
