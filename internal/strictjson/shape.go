// Package strictjson adds exact member names and required wire members to the
// shared bounded decoder. Go's decoder alone accepts case-insensitive aliases
// and null for scalar fields.
package strictjson

import (
	"bytes"
	"encoding/json"

	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func Decode(wire []byte, target any, limit int) error {
	if err := workflowdefinition.StrictJSONBounded(wire, target, limit); err != nil {
		return err
	}
	canonical, err := json.Marshal(target)
	if err != nil {
		return workflowdefinition.ErrInvalid
	}
	var input, output any
	read := func(b []byte, v *any) error {
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		return d.Decode(v)
	}
	if read(wire, &input) != nil || read(canonical, &output) != nil || !sameShape(input, output) {
		return workflowdefinition.ErrInvalid
	}
	return nil
}
func sameShape(a, b any) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			w, ok := y[k]
			if !ok || !sameShape(v, w) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i, v := range x {
			if !sameShape(v, y[i]) {
				return false
			}
		}
		return true
	case nil:
		return b == nil
	case string:
		_, ok := b.(string)
		return ok
	case bool:
		_, ok := b.(bool)
		return ok
	case json.Number:
		_, ok := b.(json.Number)
		return ok
	default:
		return false
	}
}
