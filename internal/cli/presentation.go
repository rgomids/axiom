package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/rgomids/axiom/internal/completion"
)

// Issue #232: the human-first presentation of a canonical completion event.
// Every human view is rendered deterministically from the exact canonical JSON
// event the same command emits with --json, so the two views cannot diverge
// and no field can disappear between them. No status, effect or authority rule
// lives here, and no model call is made to format a result.

// statusLabels names each canonical status for people. The canonical status
// code is always printed beside its label, so outcomes stay distinguishable
// without parsing prose.
var statusLabels = map[completion.Status]string{
	completion.Success:           "Succeeded",
	completion.Failure:           "Failed",
	completion.ValidationFailure: "Validation failed",
	completion.DeniedAuthority:   "Authority denied",
	completion.Partial:           "Partially completed — recovery required",
	completion.Interrupted:       "Interrupted — outcome not confirmed",
	completion.RetryableFailure:  "Retryable failure",
}

// canonicalFields are the completion fields rendered in the summary block;
// every other top-level field is operation-specific payload.
var canonicalFields = map[string]bool{"status": true, "result": true, "references": true, "next": true, "details": true, "provenance": true}

// humanOutputLimit bounds the Markdown view of a canonical event whose JSON is
// bounded by limit. Nested list structure has no fixed expansion bound, so a
// readable view beyond this limit is replaced by the literal view: the same
// summary with each payload as its exact canonical JSON. That view is at most
// three times the JSON (only control runes escape, to six bytes each) plus the
// summary, whose fields internal/completion bounds below 8 KiB, so it always
// fits four times a bound of at least 16 KiB. Nothing is dropped or truncated.
func humanOutputLimit(limit int) int { return 4 * limit }

// presentEvent writes one canonical completion event: the canonical JSON wire
// value with --json, or its Markdown rendering by default.
func presentEvent(writer io.Writer, mode outputMode, status completion.Status, wire []byte, limit int) int {
	if writer == nil || len(wire) == 0 || len(wire) > limit {
		return ExitFailure
	}
	content := wire
	if mode != jsonOutput {
		rendered, err := renderMarkdown(wire)
		if err == nil && len(rendered) > humanOutputLimit(limit) {
			rendered, err = renderMarkdownView(wire, false)
		}
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

// renderMarkdown renders one canonical completion event: the status, result
// and remaining canonical completion fields, every operation-specific field in
// its canonical order, and the provenance footer.
func renderMarkdown(wire []byte) ([]byte, error) { return renderMarkdownView(wire, true) }

// renderMarkdownView renders the payload as readable lists, or, when readable
// is false, as the literal view: each payload field as its exact canonical JSON.
func renderMarkdownView(wire []byte, readable bool) ([]byte, error) {
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
	var output, summary bytes.Buffer
	fmt.Fprintf(&output, "### %s (%s)\n\n%s\n", label, codeSpan(status.text), plainText(result.text))
	if next := root.field("next"); next != nil {
		fmt.Fprintf(&summary, "- **Next:** %s\n", plainText(next.text))
	}
	if references := root.field("references"); references != nil {
		writeField(&summary, "", "References", references)
	}
	if details := root.field("details"); details != nil {
		writeField(&summary, "", "Details", details)
	}
	var sections bytes.Buffer
	if !readable {
		if err := writeLiteralPayload(&sections, wire); err != nil {
			return nil, err
		}
	}
	for index, key := range root.keys {
		value := root.children[index]
		if canonicalFields[key] || !readable {
			continue
		}
		if composite(value) {
			fmt.Fprintf(&sections, "\n#### %s\n\n", markdownKey(key))
			if key == "projects" && value.kind == jsonArray {
				writeProjectRows(&sections, value)
			} else {
				writeChildren(&sections, "", value)
			}
			continue
		}
		writeField(&summary, "", key, value)
	}
	if summary.Len() != 0 {
		output.WriteString("\n")
		output.Write(summary.Bytes())
	}
	output.Write(sections.Bytes())
	fmt.Fprintf(&output, "\n**Provenance:** %s\n", inlineObject(provenance))
	return output.Bytes(), nil
}

// maxInlineFields bounds the fields of one object rendered on a single line.
const maxInlineFields = 6

// flatObject reports an object of single-line scalars, rendered on one line.
func flatObject(node *jsonNode) bool {
	if node.kind != jsonObject || len(node.children) == 0 || len(node.children) > maxInlineFields {
		return false
	}
	for _, child := range node.children {
		if child.kind == jsonObject || child.kind == jsonArray || strings.ContainsAny(child.text, "\n\r") {
			return false
		}
	}
	return true
}

func inlineObject(node *jsonNode) string {
	fields := make([]string, len(node.children))
	for index, child := range node.children {
		fields[index] = markdownKey(node.keys[index]) + " " + inlineValue(child)
	}
	return strings.Join(fields, " · ")
}

// writeLiteralPayload prints each operation-specific field as its exact
// canonical JSON. Encoded JSON has no line breaks and never starts with a
// backtick, so a three-backtick fence always holds it.
func writeLiteralPayload(output *bytes.Buffer, wire []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(wire))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return errors.New("canonical completion event is not an object")
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, _ := token.(string)
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		if canonicalFields[key] {
			continue
		}
		fmt.Fprintf(output, "\n#### %s (canonical JSON)\n\n```json\n%s\n```\n", markdownKey(key), escapeControls(string(value), false))
	}
	return nil
}

func composite(node *jsonNode) bool {
	return (node.kind == jsonObject || node.kind == jsonArray) && len(node.children) != 0
}

func writeChildren(output *bytes.Buffer, indent string, node *jsonNode) {
	for index, child := range node.children {
		if node.kind == jsonObject {
			writeField(output, indent, node.keys[index], child)
			continue
		}
		if child.kind == jsonArray || (child.kind == jsonObject && !flatObject(child)) {
			writeField(output, indent, fmt.Sprintf("[%d]", index+1), child)
			continue
		}
		writeItem(output, indent, "", child)
	}
}

func writeField(output *bytes.Buffer, indent, key string, node *jsonNode) {
	writeItem(output, indent, "**"+markdownKey(key)+":**", node)
}

func writeItem(output *bytes.Buffer, indent, label string, node *jsonNode) {
	prefix := indent + "- "
	if label != "" {
		prefix += label + " "
	}
	switch {
	case node.kind == jsonString && strings.ContainsAny(node.text, "\n\r"):
		fmt.Fprintf(output, "%s\n\n", strings.TrimRight(prefix, " "))
		writeBlock(output, indent+"  ", node.text)
	case node.kind == jsonObject && len(node.children) == 0:
		fmt.Fprintf(output, "%s_empty_\n", prefix)
	case node.kind == jsonArray && len(node.children) == 0:
		fmt.Fprintf(output, "%s_none_\n", prefix)
	case flatObject(node):
		fmt.Fprintf(output, "%s%s\n", prefix, inlineObject(node))
	case node.kind == jsonObject || node.kind == jsonArray:
		fmt.Fprintf(output, "%s\n", strings.TrimRight(prefix, " "))
		writeChildren(output, indent+"  ", node)
	default:
		fmt.Fprintf(output, "%s%s\n", prefix, inlineValue(node))
	}
}

func writeBlock(output *bytes.Buffer, indent, text string) {
	text = escapeControls(strings.ReplaceAll(text, "\r\n", "\n"), true)
	fence := strings.Repeat("`", max(3, longestFenceLine(text)+1))
	fmt.Fprintf(output, "%s%s\n", indent, fence)
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		fmt.Fprintf(output, "%s%s\n", indent, line)
	}
	fmt.Fprintf(output, "%s%s\n\n", indent, fence)
}

func inlineValue(node *jsonNode) string {
	if node.kind == jsonString && node.text == "" {
		return "`\"\"`"
	}
	if node.kind == jsonObject || node.kind == jsonArray {
		return "_empty_"
	}
	return codeSpan(node.text)
}

// codeSpan renders a value literally: Markdown in Provider-authored or local
// values cannot change the structure of the view.
func codeSpan(text string) string {
	text = escapeControls(text, false)
	fence := strings.Repeat("`", shortestAbsentRun(text))
	if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") || strings.HasPrefix(text, " ") || strings.HasSuffix(text, " ") {
		return fence + " " + text + " " + fence
	}
	return fence + text + fence
}

// plainText renders Axiom-authored completion prose on one line.
func plainText(text string) string {
	return escapeControls(strings.Join(strings.Fields(text), " "), false)
}

var plainKey = regexp.MustCompile(`^[A-Za-z0-9_.\-\[\]]+$`)

func markdownKey(key string) string {
	if plainKey.MatchString(key) {
		return key
	}
	return codeSpan(key)
}

// shortestAbsentRun is the shortest backtick fence that no backtick run inside
// text matches, so it delimits the span. Runs of every shorter length need at
// least n(n-1)/2 backticks, so the fence stays near sqrt(2*len(text)).
func shortestAbsentRun(text string) int {
	runs := map[int]bool{}
	current := 0
	for _, value := range text + " " {
		if value == '`' {
			current++
			continue
		}
		runs[current] = true
		current = 0
	}
	fence := 1
	for runs[fence] {
		fence++
	}
	return fence
}

// longestFenceLine is the longest line made only of backticks, the only
// content that could close a fenced block.
func longestFenceLine(text string) int {
	longest := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.Trim(line, "`") == "" {
			longest = max(longest, len(line))
		}
	}
	return longest
}

// escapeControls keeps terminal control and bidirectional override characters
// visible instead of interpreting them; JSON mode escapes the same values.
func escapeControls(text string, keepLayout bool) string {
	var output strings.Builder
	for _, value := range text {
		if keepLayout && (value == '\n' || value == '\t') {
			output.WriteRune(value)
			continue
		}
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

// R10 preserves existing compact Project rows while adding the selection below.
func writeProjectRows(output *bytes.Buffer, node *jsonNode) {
	for _, row := range node.children {
		if row.kind != jsonObject {
			writeItem(output, "", "", row)
			continue
		}
		prior := &jsonNode{kind: jsonObject}
		var active *jsonNode
		for i, key := range row.keys {
			if key == "activeWorkflow" {
				active = row.children[i]
				continue
			}
			prior.keys = append(prior.keys, key)
			prior.children = append(prior.children, row.children[i])
		}
		if active == nil || !flatObject(prior) {
			writeItem(output, "", "", row)
			continue
		}
		writeItem(output, "", "", prior)
		writeField(output, "  ", "activeWorkflow", active)
	}
}
