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
	if version != "1" && version != "2" {
		return Record{}, problem("installation.formatVersion", "unsupported_local_format")
	}
	// Format 2 is format 1 plus documentation bindings; format 1 carrying them
	// is an unknown field. Reading never converts one format into the other.
	bindings, hasBindings := object["documentationBindings"]
	if version == "2" {
		if !hasBindings {
			return Record{}, problem("installation.documentationBindings", "required")
		}
		delete(object, "documentationBindings")
		if issues = checkShape(bindings, reflect.TypeFor[[]documentationDTO](), "installation.documentationBindings"); len(issues) > 0 {
			return Record{}, issues
		}
	}
	if issues = checkShape(object, reflect.TypeFor[recordDTO](), "installation"); len(issues) > 0 {
		return Record{}, issues
	}
	var dto recordV2DTO
	if err := json.Unmarshal(input, &dto); err != nil {
		return Record{}, problem("installation", "invalid_json")
	}
	if version == "2" && len(dto.DocumentationBindings) == 0 {
		return Record{}, problem("installation.documentationBindings", "noncanonical_format")
	}
	state, issues := fromDTO(dto.recordDTO)
	if len(issues) > 0 {
		return Record{}, issues
	}
	if state.Documentation, issues = fromDocumentationDTO(dto.DocumentationBindings); len(issues) > 0 {
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
	// Format 1 stays byte-identical to the previous writer; format 2 is written
	// only when machine-local documentation bindings exist.
	var dto any = toDTO(r.state)
	if len(r.state.Documentation) != 0 {
		v2 := recordV2DTO{recordDTO: toDTO(r.state), DocumentationBindings: toDocumentationDTO(r.state.Documentation)}
		v2.FormatVersion = 2
		dto = v2
	}
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
