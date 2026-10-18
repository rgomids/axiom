package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/rgomids/axiom/internal/completion"
)

var statusLabels = map[completion.Status]string{
	completion.Success:           "Succeeded",
	completion.Failure:           "Failed",
	completion.ValidationFailure: "Validation failed",
	completion.DeniedAuthority:   "Authority denied",
	completion.Partial:           "Partially completed — recovery required",
	completion.Interrupted:       "Interrupted — outcome not confirmed",
	completion.RetryableFailure:  "Retryable failure",
}

var canonicalFields = map[string]bool{"status": true, "result": true, "references": true, "next": true, "details": true, "provenance": true}

func humanOutputLimit(limit int) int { return 4 * limit }

func presentEvent(writer io.Writer, mode outputMode, status completion.Status, wire []byte, limit int) int {
	if writer == nil || len(wire) == 0 || len(wire) > limit {
		return ExitFailure
	}
	content := wire
	if mode != jsonOutput {
		rendered, err := renderTerminal(wire)
		if err != nil || len(rendered) > humanOutputLimit(limit) {
			return ExitFailure
		}
		content = rendered
	}
	if written, err := writer.Write(content); err != nil || written != len(content) {
		return ExitFailure
	}
	return completionExitCode(status)
}

type jsonKind int

const (
	jsonScalar jsonKind = iota
	jsonString
	jsonObject
	jsonArray
)

// jsonNode keeps the canonical JSON field order, so rendering is stable.
type jsonNode struct {
	kind     jsonKind
	text     string
	keys     []string
	children []*jsonNode
}

func (n *jsonNode) field(name string) *jsonNode {
	for index, key := range n.keys {
		if key == name {
			return n.children[index]
		}
	}
	return nil
}

func decodeOrdered(wire []byte) (*jsonNode, error) {
	decoder := json.NewDecoder(bytes.NewReader(wire))
	decoder.UseNumber()
	root, err := decodeNode(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("trailing canonical event content")
	}
	return root, nil
}

func decodeNode(decoder *json.Decoder) (*jsonNode, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch value := token.(type) {
	case json.Delim:
		node := &jsonNode{kind: jsonArray}
		if value == '{' {
			node.kind = jsonObject
		}
		for decoder.More() {
			if node.kind == jsonObject {
				key, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				name, ok := key.(string)
				if !ok {
					return nil, errors.New("invalid canonical event key")
				}
				node.keys = append(node.keys, name)
			}
			child, err := decodeNode(decoder)
			if err != nil {
				return nil, err
			}
			node.children = append(node.children, child)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return node, nil
	case string:
		return &jsonNode{kind: jsonString, text: value}, nil
	case json.Number:
		return &jsonNode{kind: jsonScalar, text: value.String()}, nil
	case bool:
		return &jsonNode{kind: jsonScalar, text: fmt.Sprint(value)}, nil
	case nil:
		return &jsonNode{kind: jsonScalar, text: "null"}, nil
	}
	return nil, errors.New("unsupported canonical event token")
}

func renderTerminal(wire []byte) ([]byte, error) {
	root, err := decodeOrdered(wire)
	if err != nil {
		return nil, err
	}
	status, result, provenance := root.field("status"), root.field("result"), root.field("provenance")
	if root.kind != jsonObject || status == nil || status.kind != jsonString || result == nil || result.kind != jsonString || provenance == nil || provenance.kind != jsonObject {
		return nil, errors.New("canonical completion event is incomplete")
	}
	label, known := statusLabels[completion.Status(status.text)]
	if !known {
		label = "Unrecognized status"
	}
	var output bytes.Buffer
	fmt.Fprintf(&output, "%s (%s)\n%s\n", label, terminalText(status.text), terminalText(result.text))
	for _, name := range []string{"next", "references", "details"} {
		if value := root.field(name); value != nil {
			writeTerminalField(&output, "", name, value)
		}
	}
	for index, key := range root.keys {
		if !canonicalFields[key] {
			writeTerminalField(&output, "", key, root.children[index])
		}
	}
	writeTerminalField(&output, "", "provenance", provenance)
	return output.Bytes(), nil
}

func writeTerminalField(output *bytes.Buffer, indent, key string, node *jsonNode) {
	fmt.Fprintf(output, "%s%s:", indent, terminalText(key))
	if (node.kind == jsonObject || node.kind == jsonArray) && len(node.children) != 0 && len(indent) < 6 && !flatObject(node) {
		output.WriteByte('\n')
		for index, child := range node.children {
			name := fmt.Sprintf("[%d]", index+1)
			if node.kind == jsonObject {
				name = node.keys[index]
			}
			writeTerminalField(output, indent+"  ", name, child)
		}
		return
	}
	output.WriteByte(' ')
	writeTerminalValue(output, node)
	output.WriteByte('\n')
}

func flatObject(node *jsonNode) bool {
	if node.kind != jsonObject || len(node.children) > 6 {
		return false
	}
	for _, child := range node.children {
		if child.kind == jsonObject || child.kind == jsonArray {
			return false
		}
	}
	return true
}

func writeTerminalValue(output *bytes.Buffer, node *jsonNode) {
	if node.kind != jsonObject && node.kind != jsonArray {
		output.WriteString(terminalText(node.text))
		return
	}
	open, close := "[", "]"
	if node.kind == jsonObject {
		open, close = "{", "}"
	}
	output.WriteString(open)
	for index, child := range node.children {
		if index != 0 {
			output.WriteString(", ")
		}
		if node.kind == jsonObject {
			fmt.Fprintf(output, "%s: ", terminalText(node.keys[index]))
		}
		writeTerminalValue(output, child)
	}
	output.WriteString(close)
}

func terminalText(text string) string {
	if text == "" {
		return `""`
	}
	return escapeControls(text)
}

func escapeControls(text string) string {
	var output strings.Builder
	for _, value := range text {
		if unicode.IsControl(value) || bidiControl(value) {
			fmt.Fprintf(&output, "\\u%04X", value)
			continue
		}
		output.WriteRune(value)
	}
	return output.String()
}

func bidiControl(value rune) bool {
	return value == 0x061C || value == 0x200E || value == 0x200F || (value >= 0x202A && value <= 0x202E) || (value >= 0x2066 && value <= 0x2069)
}
