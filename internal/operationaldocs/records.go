package operationaldocs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	MaxOutputs = 64
	MaxContent = 8 << 20
	MaxFile    = 1 << 20
	MaxControl = 256 << 10
)

var hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var fullCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)
var outputName = regexp.MustCompile(`^(docs/text/[a-z0-9][a-z0-9._-]*\.md|docs/man/man1/[a-z0-9][a-z0-9._-]*\.1)$`)
var sourceName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)
var commandName = regexp.MustCompile(`^axiom( [a-z0-9][a-z0-9-]*)*$`)
var sectionName = regexp.MustCompile(`^[A-Z][A-Z0-9_-]{0,63}$`)

type Entry struct {
	Path   string
	Kind   string
	Bytes  int
	SHA256 string
}

type Bundle struct {
	Files            map[string][]byte
	Entries          []Entry
	Generation       string
	InventorySHA256  string
	ProvenanceSHA256 string
	ManMapSHA256     string
	Bytes            int
}

type Descriptor struct {
	Generation   string
	Identity     string
	SourceCommit string
	Platform     string
	Architecture string
	SkillsSHA256 string
}

func Hash(wire []byte) string {
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}

func Generation(inventory, provenance, manMap string) string {
	return Hash([]byte("axiom-docs-generation-v1\n" + inventory + "\n" + provenance + "\n" + manMap + "\n"))
}

func Association(generation, commit, platform, arch, skills string) string {
	return Hash([]byte("axiom-docs-association-v1\n" + generation + "\n" + commit + "\n" + platform + "\n" + arch + "\n" + skills + "\n"))
}

func SkillsIdentity(skills map[string]string) (string, error) {
	names := []string{"axiom-project", "axiom-runtime", "axiom-work-item", "axiom-workflow"}
	if len(skills) != len(names) {
		return "", errors.New("inventory_mismatch")
	}
	var text strings.Builder
	for _, name := range names {
		if !hexDigest.MatchString(skills[name]) {
			return "", errors.New("malformed_record")
		}
		text.WriteString(skills[name] + "  skills/" + name + "/SKILL.md\n")
	}
	return Hash([]byte(text.String())), nil
}

func (d Descriptor) Valid() bool {
	return hexDigest.MatchString(d.Generation) && hexDigest.MatchString(d.SkillsSHA256) && fullCommit.MatchString(d.SourceCommit) && (d.Platform == "macos" && d.Architecture == "arm64" || d.Platform == "linux" && (d.Architecture == "amd64" || d.Architecture == "arm64")) && d.Identity == Association(d.Generation, d.SourceCommit, d.Platform, d.Architecture, d.SkillsSHA256)
}

func rows(wire []byte, header string, fields, limit int) ([][]string, error) {
	if len(wire) > MaxControl {
		return nil, errors.New("quota_exceeded")
	}
	if !utf8.Valid(wire) || bytes.ContainsAny(wire, "\r\x00") || !bytes.HasSuffix(wire, []byte("\n")) {
		return nil, errors.New("malformed_record")
	}
	lines := strings.Split(string(wire[:len(wire)-1]), "\n")
	if len(lines) < 3 || lines[0] != "formatVersion=1" || lines[1] != header {
		return nil, errors.New("unsupported_format")
	}
	if len(lines)-2 > limit {
		return nil, errors.New("quota_exceeded")
	}
	result := make([][]string, 0, len(lines)-2)
	previous := ""
	for _, line := range lines[2:] {
		cells := strings.Split(line, "\t")
		if len(cells) != fields || line <= previous {
			return nil, errors.New("malformed_record")
		}
		for _, c := range cells {
			if c == "" || strings.ContainsFunc(c, func(r rune) bool { return r < 32 || r == 127 }) {
				return nil, errors.New("malformed_record")
			}
		}
		previous = line
		result = append(result, cells)
	}
	return result, nil
}

func Validate(files map[string][]byte, commit string) (Bundle, error) {
	b := Bundle{Files: map[string][]byte{}}
	if !fullCommit.MatchString(commit) {
		return b, errors.New("malformed_record")
	}
	inventory, err := rows(files["docs/inventory.tsv"], "path\tkind\tbytes\tsha256", 4, MaxOutputs)
	if err != nil {
		return b, err
	}
	for _, r := range inventory {
		if !outputName.MatchString(r[0]) || len(r[0]) > 240 || len(path.Base(r[0])) > 128 {
			return b, errors.New("unsafe_path")
		}
		kind := "markdown"
		if strings.HasPrefix(r[0], "docs/man/") {
			kind = "man1"
		}
		n, err := strconv.Atoi(r[2])
		if err != nil || strconv.Itoa(n) != r[2] || n <= 0 || n > MaxFile || b.Bytes > MaxContent-n {
			return b, errors.New("quota_exceeded")
		}
		if _, exists := b.Files[r[0]]; exists {
			return b, errors.New("inventory_mismatch")
		}
		wire, present := files[r[0]]
		if !present || kind != r[1] || !hexDigest.MatchString(r[3]) || len(wire) != n || Hash(wire) != r[3] || !utf8.Valid(wire) {
			return b, errors.New("hash_mismatch")
		}
		b.Files[r[0]] = bytes.Clone(wire)
		b.Entries = append(b.Entries, Entry{r[0], r[1], n, r[3]})
		b.Bytes += n
	}
	for _, name := range []string{"index", "concepts", "installation", "project", "runtime", "workflow", "work-item", "configuration", "troubleshooting", "cli", "notices"} {
		if _, ok := b.Files["docs/text/"+name+".md"]; !ok {
			return b, errors.New("inventory_mismatch")
		}
	}
	provenance, err := rows(files["docs/provenance.tsv"], "output\tsource\tsourceCommit\tsourceSha256\ttransform", 5, 1024)
	if err != nil {
		return b, err
	}
	mapped := map[string]bool{}
	previous := ""
	for _, r := range provenance {
		key := r[0] + "\t" + r[1]
		if key == previous || b.Files[r[0]] == nil || !sourceName.MatchString(r[1]) || len(r[1]) > 240 || path.Clean(r[1]) != r[1] || strings.Contains(r[1], "..") || r[2] != commit || !hexDigest.MatchString(r[3]) || (r[4] != "copy-v1" && r[4] != "operational-synthesis-v1" && r[4] != "cli-render-v1") {
			return b, errors.New("inventory_mismatch")
		}
		previous = key
		mapped[r[0]] = true
	}
	if len(mapped) != len(b.Entries) {
		return b, errors.New("inventory_mismatch")
	}
	manMap, err := rows(files["docs/man-map.tsv"], "command\tpage\tanchor", 3, 256)
	if err != nil {
		return b, err
	}
	commands := map[string]bool{}
	for _, r := range manMap {
		if commands[r[0]] || !commandName.MatchString(r[0]) || len(r[0]) > 256 || !strings.HasPrefix(r[1], "docs/man/man1/") || b.Files[r[1]] == nil || !sectionName.MatchString(r[2]) || !bytes.Contains(b.Files[r[1]], []byte(".SH "+r[2]+"\n")) {
			return b, errors.New("inventory_mismatch")
		}
		commands[r[0]] = true
	}
	if !commands["axiom"] {
		return b, errors.New("inventory_mismatch")
	}
	for name := range files {
		if strings.HasPrefix(name, "docs/") && b.Files[name] == nil && name != "docs/inventory.tsv" && name != "docs/provenance.tsv" && name != "docs/man-map.tsv" {
			return b, errors.New("inventory_mismatch")
		}
	}
	for _, name := range []string{"docs/inventory.tsv", "docs/provenance.tsv", "docs/man-map.tsv"} {
		b.Files[name] = bytes.Clone(files[name])
	}
	b.InventorySHA256 = Hash(files["docs/inventory.tsv"])
	b.ProvenanceSHA256 = Hash(files["docs/provenance.tsv"])
	b.ManMapSHA256 = Hash(files["docs/man-map.tsv"])
	b.Generation = Generation(b.InventorySHA256, b.ProvenanceSHA256, b.ManMapSHA256)
	return b, nil
}

func DecodeClosed(wire []byte, target any, fields []string) error {
	if len(wire) > 16<<10 || !utf8.Valid(wire) {
		return errors.New("malformed_record")
	}
	decoder := json.NewDecoder(bytes.NewReader(wire))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return errors.New("malformed_record")
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return errors.New("malformed_record")
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return errors.New("malformed_record")
		}
		seen[key] = true
		var value json.RawMessage
		if decoder.Decode(&value) != nil || bytes.Equal(value, []byte("null")) {
			return errors.New("malformed_record")
		}
	}
	if _, err := decoder.Token(); err != nil {
		return errors.New("malformed_record")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return errors.New("malformed_record")
	}
	if len(seen) != len(fields) {
		return errors.New("malformed_record")
	}
	for _, field := range fields {
		if !seen[field] {
			return errors.New("malformed_record")
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(wire))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return errors.New("malformed_record")
	}
	return nil
}

func SortedNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
