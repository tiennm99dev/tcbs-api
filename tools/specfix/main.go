// Command specfix turns the TCBS developer-portal OpenAPI spec into one that
// code generators can use. The portal spec has three defects:
//
//   - nested response fields are written as flat property names such as
//     "assets[].symbol" or "data.items[].price";
//   - some operations declare path parameters that their path does not have;
//   - no operation has an operationId, so generators name methods after paths.
//
// specfix rebuilds the nested schemas, drops the stray path parameters, and
// sets each operationId from a "METHOD /path" -> "Name" JSON map.
//
// The spec belongs to TCBS and is not stored in this repository: -in is
// normally the portal URL, and -sha256 pins the exact file the committed
// client was generated from, so a changed portal spec fails loudly instead of
// silently producing a different client.
//
// Usage:
//
//	specfix -in https://.../openapi-v1.0.0.json -sha256 <hex> -ids operation-ids.json -out fixed.json
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

func main() {
	in := flag.String("in", "", "portal OpenAPI spec: an https URL or a local JSON file")
	sum := flag.String("sha256", "", "expected SHA-256 of the -in content (hex); empty skips the check")
	ids := flag.String("ids", "", `JSON map of "METHOD /path" to operationId`)
	out := flag.String("out", "", "fixed spec output path")
	flag.Parse()
	if *in == "" || *ids == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*in, *sum, *ids, *out); err != nil {
		fmt.Fprintln(os.Stderr, "specfix:", err)
		os.Exit(1)
	}
}

func run(in, wantSum, idsPath, outPath string) error {
	raw, err := load(in)
	if err != nil {
		return err
	}
	if err := checkSum(raw, wantSum); err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	var spec map[string]any
	if err := json.Unmarshal(raw, &spec); err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	var ids map[string]string
	if err := readJSON(idsPath, &ids); err != nil {
		return err
	}
	rep, err := Fix(spec, ids)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "specfix: %d flattened fields nested, %d stray path params dropped, %d operationIds set\n",
		rep.Nested, rep.DroppedParams, rep.OperationIDs)
	return nil
}

// load reads an https URL or a local file.
func load(in string) ([]byte, error) {
	if !strings.HasPrefix(in, "https://") {
		return os.ReadFile(in)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(in)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", in, resp.Status)
	}
	const maxSpec = 32 << 20
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxSpec+1))
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", in, err)
	}
	if len(b) > maxSpec {
		return nil, fmt.Errorf("GET %s: spec larger than %d bytes", in, maxSpec)
	}
	return b, nil
}

func checkSum(b []byte, want string) error {
	if want == "" {
		return nil
	}
	sum := sha256.Sum256(b)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, want) {
		return fmt.Errorf("sha256 %s, want %s: the spec changed; review it, then update the pinned checksum", got, want)
	}
	return nil
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// Report counts what Fix changed.
type Report struct {
	Nested        int
	DroppedParams int
	OperationIDs  int
}

var methods = []string{"get", "put", "post", "delete", "patch", "head", "options"}

var pathParam = regexp.MustCompile(`{(\w+)}`)

// Fix repairs spec in place. Every operation must have an entry in ids, and
// every ids entry must match an operation, so a portal update that adds or
// renames an endpoint fails loudly instead of generating a path-named method.
func Fix(spec map[string]any, ids map[string]string) (Report, error) {
	var rep Report
	paths, _ := spec["paths"].(map[string]any)
	used := map[string]bool{}
	var missing []string
	for path, rawItem := range paths {
		item, _ := rawItem.(map[string]any)
		names := map[string]bool{}
		for _, m := range pathParam.FindAllStringSubmatch(path, -1) {
			names[m[1]] = true
		}
		rep.DroppedParams += dropStrayParams(item, names)
		for _, method := range methods {
			op, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			rep.DroppedParams += dropStrayParams(op, names)
			key := strings.ToUpper(method) + " " + path
			id, ok := ids[key]
			if !ok {
				missing = append(missing, key)
				continue
			}
			op["operationId"] = id
			used[key] = true
			rep.OperationIDs++
		}
	}
	var stale []string
	for key := range ids {
		if !used[key] {
			stale = append(stale, key)
		}
	}
	if len(missing) > 0 || len(stale) > 0 {
		sort.Strings(missing)
		sort.Strings(stale)
		return rep, fmt.Errorf("operation-id map out of date: missing %v, unknown %v", missing, stale)
	}

	walkSchemas(spec["paths"], &rep)
	if comps, ok := spec["components"].(map[string]any); ok {
		if schemas, ok := comps["schemas"].(map[string]any); ok {
			for name, s := range schemas {
				schemas[name] = nest(s, &rep)
			}
		}
	}
	return rep, nil
}

// dropStrayParams removes path parameters whose name is not in the path.
func dropStrayParams(scope map[string]any, names map[string]bool) int {
	params, ok := scope["parameters"].([]any)
	if !ok {
		return 0
	}
	keep := params[:0:0]
	for _, raw := range params {
		p, _ := raw.(map[string]any)
		if p["in"] == "path" && !names[fmt.Sprint(p["name"])] {
			continue
		}
		keep = append(keep, raw)
	}
	scope["parameters"] = keep
	return len(params) - len(keep)
}

// walkSchemas nests every "schema" value found under v.
func walkSchemas(v any, rep *Report) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if k == "schema" {
				t[k] = nest(child, rep)
				continue
			}
			walkSchemas(child, rep)
		}
	case []any:
		for _, child := range t {
			walkSchemas(child, rep)
		}
	}
}

type segment struct {
	name  string
	array bool
}

func splitFlat(name string) []segment {
	parts := strings.Split(name, ".")
	segs := make([]segment, len(parts))
	for i, p := range parts {
		segs[i] = segment{name: strings.TrimSuffix(p, "[]"), array: strings.HasSuffix(p, "[]")}
	}
	return segs
}

func isFlat(name string) bool {
	return strings.Contains(name, ".") || strings.HasSuffix(name, "[]")
}

// nest rebuilds the object tree of a schema whose property names are flattened
// paths, then recurses into the result.
func nest(raw any, rep *Report) any {
	s, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	for _, k := range []string{"items", "additionalProperties"} {
		if child, ok := s[k].(map[string]any); ok {
			s[k] = nest(child, rep)
		}
	}
	for _, k := range []string{"allOf", "oneOf", "anyOf"} {
		if list, ok := s[k].([]any); ok {
			for i := range list {
				list[i] = nest(list[i], rep)
			}
		}
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		return s
	}
	if arr := topLevelArray(s, props); arr != nil {
		rep.Nested += len(props)
		return nest(arr, rep)
	}

	// Plain names first, then flattened ones from shallow to deep, so a parent
	// declared as its own property is in place before its children attach.
	names := make([]string, 0, len(props))
	for n := range props {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		di, dj := strings.Count(names[i], "."), strings.Count(names[j], ".")
		if isFlat(names[i]) != isFlat(names[j]) {
			return !isFlat(names[i])
		}
		if di != dj {
			return di < dj
		}
		return names[i] < names[j]
	})

	root := map[string]any{}
	for _, name := range names {
		sub, _ := props[name].(map[string]any)
		if !isFlat(name) {
			root[name] = sub
			continue
		}
		rep.Nested++
		place(root, splitFlat(name), sub)
	}
	s["properties"] = root

	if req, ok := s["required"].([]any); ok {
		kept := req[:0:0]
		for _, r := range req {
			if _, ok := root[fmt.Sprint(r)]; ok {
				kept = append(kept, r)
			}
		}
		if len(kept) == 0 {
			delete(s, "required")
		} else {
			s["required"] = kept
		}
	}
	for n, child := range root {
		root[n] = nest(child, rep)
	}
	return s
}

// topLevelArray handles a response that is itself an array: the portal writes
// its item fields as "[].name". It returns the equivalent array schema, or nil
// when the properties are not all of that form.
func topLevelArray(s, props map[string]any) map[string]any {
	items := map[string]any{}
	for name, sub := range props {
		rest, ok := strings.CutPrefix(name, "[].")
		if !ok {
			return nil
		}
		items[rest] = sub
	}
	arr := map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "object", "properties": items},
	}
	copyDescription(arr, s)
	return arr
}

// place puts leaf at the flattened path segs under props, creating the
// intermediate objects and arrays of objects on the way.
func place(props map[string]any, segs []segment, leaf map[string]any) {
	seg := segs[0]
	if len(segs) == 1 {
		if existing, ok := props[seg.name].(map[string]any); ok && isContainer(existing) {
			copyDescription(existing, leaf)
			return
		}
		if seg.array {
			props[seg.name] = map[string]any{"type": "array", "items": leaf}
		} else {
			props[seg.name] = leaf
		}
		return
	}

	node, _ := props[seg.name].(map[string]any)
	var obj map[string]any
	if seg.array {
		items, _ := node["items"].(map[string]any)
		if node == nil || node["type"] != "array" || items == nil || !objectLike(items) {
			node = replaceKeepingDescription(node, map[string]any{"type": "array", "items": map[string]any{}})
			items = node["items"].(map[string]any)
		}
		obj = items
	} else {
		if node == nil || !objectLike(node) {
			node = replaceKeepingDescription(node, map[string]any{})
		}
		obj = node
	}
	obj["type"] = "object"
	child, _ := obj["properties"].(map[string]any)
	if child == nil {
		child = map[string]any{}
		obj["properties"] = child
	}
	props[seg.name] = node
	place(child, segs[1:], leaf)
}

func objectLike(s map[string]any) bool {
	t, has := s["type"]
	return !has || t == "object"
}

func isContainer(s map[string]any) bool {
	if _, ok := s["properties"]; ok {
		return true
	}
	items, ok := s["items"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = items["properties"]
	return ok
}

func copyDescription(dst, src map[string]any) {
	if d, ok := src["description"]; ok {
		if _, has := dst["description"]; !has {
			dst["description"] = d
		}
	}
}

func replaceKeepingDescription(old, fresh map[string]any) map[string]any {
	if old != nil {
		copyDescription(fresh, old)
	}
	return fresh
}
