package local

import (
	"encoding/json"
	"reflect"
	"strings"
)

// DecodeRecord accepts exactly one existing versioned JSON object. No partial
// state escapes any stage. Missing records must use DecodeObservedRecord.
func DecodeRecord(input []byte) (Record, []Issue) {
	tree, issues := parseRecord(input)
	if len(issues) > 0 {
		return Record{}, issues
	}
	object, ok := tree.(map[string]any)
	if !ok {
		return Record{}, problem("installation", "invalid_type")
	}
	version, ok := object["formatVersion"].(json.Number)
	if !ok || strings.ContainsAny(string(version), ".eE") {
		return Record{}, problem("installation.formatVersion", "invalid_version")
	}
	if version != "1" {
		return Record{}, problem("installation.formatVersion", "unsupported_local_format")
	}
	if issues = checkShape(tree, reflect.TypeFor[recordDTO](), "installation"); len(issues) > 0 {
		return Record{}, issues
	}
	var dto recordDTO
	if err := json.Unmarshal(input, &dto); err != nil {
		return Record{}, problem("installation", "invalid_json")
	}
	state, issues := fromDTO(dto)
	if len(issues) > 0 {
		return Record{}, issues
	}
	return NewRecord(state)
}

// EncodeRecord emits only the closed DTO and checks the same complete decoder
// before returning bytes. It never returns partial or invalid output.
func EncodeRecord(r Record) ([]byte, []Issue) {
	if !r.valid {
		return nil, problem("installation", "invalid_record")
	}
	dto := toDTO(r.state)
	output, err := json.Marshal(dto)
	if err != nil {
		return nil, problem("installation", "invalid_encoding")
	}
	output = append(output, '\n')
	if _, issues := DecodeRecord(output); len(issues) > 0 {
		return nil, issues
	}
	return output, nil
}
