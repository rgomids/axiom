package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

const (
	MaxRecordBytes = 256 * 1024
	MaxRecordDepth = 16
	MaxRecordNodes = 16384
)

// Inspect tokens before decoding structs: encoding/json alone accepts duplicate
// and case-insensitive keys, null scalars, and invalid Unicode replacement.
func parseRecord(input []byte) (any, []Issue) {
	if len(input) > MaxRecordBytes {
		return nil, problem("installation", "byte_limit")
	}
	if !utf8.Valid(input) {
		return nil, problem("installation", "invalid_utf8")
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	nodes := 0
	value, issues := readValue(decoder, 1, &nodes)
	if len(issues) > 0 {
		return nil, issues
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, problem("installation", "invalid_json")
	}
	return value, nil
}
func readValue(d *json.Decoder, depth int, nodes *int) (any, []Issue) {
	*nodes++
	if depth > MaxRecordDepth {
		return nil, problem("installation", "depth_limit")
	}
	if *nodes > MaxRecordNodes {
		return nil, problem("installation", "node_limit")
	}
	token, err := d.Token()
	if err != nil {
		return nil, problem("installation", "invalid_json")
	}
	switch token {
	case json.Delim('{'):
		return readObject(d, depth, nodes)
	case json.Delim('['):
		return readArray(d, depth, nodes)
	}
	if s, ok := token.(string); ok && strings.ContainsRune(s, '\ufffd') {
		return nil, problem("installation", "invalid_unicode")
	}
	return token, nil
}
func readObject(d *json.Decoder, depth int, nodes *int) (any, []Issue) {
	result := map[string]any{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, problem("installation", "invalid_json")
		}
		name, ok := key.(string)
		if !ok {
			return nil, problem("installation", "invalid_json")
		}
		*nodes++
		if _, exists := result[name]; exists {
			return nil, problem("installation", "duplicate_field")
		}
		value, issues := readValue(d, depth+1, nodes)
		if len(issues) > 0 {
			return nil, issues
		}
		result[name] = value
	}
	if token, err := d.Token(); err != nil || token != json.Delim('}') {
		return nil, problem("installation", "invalid_json")
	}
	return result, nil
}
func readArray(d *json.Decoder, depth int, nodes *int) (any, []Issue) {
	result := []any{}
	for d.More() {
		value, issues := readValue(d, depth+1, nodes)
		if len(issues) > 0 {
			return nil, issues
		}
		result = append(result, value)
	}
	if token, err := d.Token(); err != nil || token != json.Delim(']') {
		return nil, problem("installation", "invalid_json")
	}
	return result, nil
}

// Reflection reads only the private, fixed DTO schema. Input cannot add fields,
// extension bags, coercions or serializers. Schema order fixes first-error order.
func checkShape(value any, t reflect.Type, field string) []Issue {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok {
			return problem(field, "invalid_type")
		}
		known := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			known[strings.Split(t.Field(i).Tag.Get("json"), ",")[0]] = true
		}
		for key := range object {
			if !known[key] {
				return problem(field, "unknown_field")
			}
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			name := tag[0]
			child, exists := object[name]
			if !exists && len(tag) > 1 {
				continue
			}
			if !exists {
				return problem(field+"."+name, "required")
			}
			if issues := checkShape(child, f.Type, field+"."+name); len(issues) > 0 {
				return issues
			}
		}
	case reflect.Slice:
		array, ok := value.([]any)
		if !ok {
			return problem(field, "invalid_type")
		}
		for i, child := range array {
			if issues := checkShape(child, t.Elem(), fmt.Sprintf("%s[%d]", field, i)); len(issues) > 0 {
				return issues
			}
		}
	case reflect.String:
		if _, ok := value.(string); !ok {
			return problem(field, "invalid_type")
		}
	case reflect.Int:
		n, ok := value.(json.Number)
		if !ok || strings.ContainsAny(string(n), ".eE") {
			return problem(field, "invalid_type")
		}
	default:
		return problem(field, "invalid_type")
	}
	return nil
}
