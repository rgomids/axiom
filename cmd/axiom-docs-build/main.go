package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/operationaldocs"
)

const maxOutputs = 64
const maxOutputBytes = 1 << 20
const maxTotalBytes = 8 << 20
const maxControlBytes = 256 << 10
const maxProvenanceRows = 1024
const maxManRows = 256

var versionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+_-]{0,127}$`)
var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var sourceNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*\.md$`)

type sourceRecord struct {
	output  string
	sources []sourceOrigin
	data    []byte
	kind    string
}

type sourceOrigin struct{ path, digest, transform string }

func main() {
	root := flag.String("source-root", ".", "source checkout root")
	output := flag.String("output", "", "new output directory")
	version := flag.String("version", "", "documented Axiom version")
	commit := flag.String("commit", "", "full source commit")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected arguments"))
	}
	result, err := generate(*root, *output, *version, *commit)
	if err != nil {
		fail(err)
	}
	encoded, _ := json.Marshal(struct {
		Generation   string            `json:"generation"`
		Outputs      int               `json:"outputs"`
		Bytes        int               `json:"bytes"`
		SourceHashes map[string]string `json:"sourceHashes"`
	}{result.generation, result.outputs, result.bytes, result.sourceHashes})
	fmt.Println(string(encoded))
}

type buildResult struct {
	generation   string
	outputs      int
	bytes        int
	sources      int
	sourceHashes map[string]string
}

func generate(root, output, version, commit string) (buildResult, error) {
	if !versionPattern.MatchString(version) || !commitPattern.MatchString(commit) {
		return buildResult{}, fmt.Errorf("invalid version or source commit")
	}
	if output == "" {
		return buildResult{}, fmt.Errorf("output directory is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return buildResult{}, err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return buildResult{}, err
	}
	if _, err := os.Lstat(output); err == nil {
		return buildResult{}, fmt.Errorf("output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return buildResult{}, err
	}
	allowlistPath := filepath.Join(root, "docs/operational/allowlist.json")
	allowData, err := readRegular(allowlistPath, maxControlBytes)
	if err != nil {
		return buildResult{}, err
	}
	var names []string
	if err := json.Unmarshal(allowData, &names); err != nil || names == nil {
		return buildResult{}, fmt.Errorf("invalid operational allowlist")
	}
	sort.Strings(names)
	for i, name := range names {
		if !sourceNamePattern.MatchString(name) || name == "allowlist.json" || i > 0 && names[i-1] == name {
			return buildResult{}, fmt.Errorf("invalid or duplicate operational source")
		}
	}
	if !contains(names, "notices.md") {
		return buildResult{}, fmt.Errorf("operational notices are required")
	}
	records := make([]sourceRecord, 0, len(names)+16)
	sourceHashes := map[string]string{}
	for _, name := range names {
		source := "docs/operational/" + name
		data, err := readRegular(filepath.Join(root, filepath.FromSlash(source)), maxOutputBytes)
		if err != nil {
			return buildResult{}, err
		}
		sourceHashes[source] = digest(data)
		records = append(records, record("docs/text/"+name, source, data, "markdown"))
	}
	var markdown strings.Builder
	if err := cli.RenderDocumentationCLI(&markdown, version, commit); err != nil {
		return buildResult{}, err
	}
	records = append(records, record("docs/text/cli.md", "internal/cli/commands.go", []byte(markdown.String()), "markdown"))
	manPages, err := cli.DocumentationPages(version, commit)
	if err != nil {
		return buildResult{}, err
	}
	mapRows := make([]string, 0, len(manPages))
	for _, page := range manPages {
		records = append(records, record(page.Page, "internal/cli/commands.go", page.Text, "man1"))
		for _, mapping := range page.Mappings {
			mapRows = append(mapRows, strings.Join([]string{mapping.Command, mapping.Page, mapping.Anchor}, "\t"))
		}
	}
	if len(records) > maxOutputs {
		return buildResult{}, fmt.Errorf("documentation output count exceeds limit")
	}
	for _, required := range []string{"docs/text/index.md", "docs/text/concepts.md", "docs/text/installation.md", "docs/text/project.md", "docs/text/runtime.md", "docs/text/workflow.md", "docs/text/work-item.md", "docs/text/configuration.md", "docs/text/troubleshooting.md", "docs/text/notices.md", "docs/text/cli.md"} {
		if !hasOutput(records, required) {
			return buildResult{}, fmt.Errorf("required output missing: %s", required)
		}
	}
	cliSources, err := filepath.Glob(filepath.Join(root, "internal/cli/*.go"))
	if err != nil {
		return buildResult{}, err
	}
	cliOrigins := make([]sourceOrigin, 0, len(cliSources))
	for _, absolute := range cliSources {
		if strings.HasSuffix(absolute, "_test.go") {
			continue
		}
		source, err := filepath.Rel(root, absolute)
		if err != nil {
			return buildResult{}, err
		}
		source = filepath.ToSlash(source)
		data, err := readRegular(absolute, maxOutputBytes)
		if err != nil {
			return buildResult{}, err
		}
		sourceHashes[source] = digest(data)
		cliOrigins = append(cliOrigins, sourceOrigin{source, digest(data), "cli-render-v1"})
	}
	sort.Slice(cliOrigins, func(i, j int) bool { return cliOrigins[i].path < cliOrigins[j].path })
	for i := range records {
		if strings.HasPrefix(records[i].output, "docs/man/man1/") || records[i].output == "docs/text/cli.md" {
			records[i].sources = cliOrigins
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].output < records[j].output })
	if len(mapRows) > maxManRows {
		return buildResult{}, fmt.Errorf("documentary row limit exceeded")
	}
	var inventory, provenance strings.Builder
	inventory.WriteString("formatVersion=1\npath\tkind\tbytes\tsha256\n")
	provenance.WriteString("formatVersion=1\noutput\tsource\tsourceCommit\tsourceSha256\ttransform\n")
	total := 0
	sourceSet := map[string]bool{}
	for _, entry := range records {
		if len(entry.data) == 0 || len(entry.data) > maxOutputBytes || !validOutput(entry.output) || !utf8.Valid(entry.data) || strings.ContainsRune(string(entry.data), '\r') {
			return buildResult{}, fmt.Errorf("invalid output or output exceeds limit")
		}
		total += len(entry.data)
		if total > maxTotalBytes {
			return buildResult{}, fmt.Errorf("aggregate output limit exceeded")
		}
		kind := "markdown"
		if entry.kind == "man1" {
			kind = "man1"
		}
		fmt.Fprintf(&inventory, "%s\t%s\t%d\t%s\n", entry.output, kind, len(entry.data), digest(entry.data))
		for _, origin := range entry.sources {
			fmt.Fprintf(&provenance, "%s\t%s\t%s\t%s\t%s\n", entry.output, origin.path, commit, origin.digest, origin.transform)
		}
		for _, origin := range entry.sources {
			sourceSet[origin.path] = true
		}
	}
	sort.Strings(mapRows)
	var manMap strings.Builder
	manMap.WriteString("formatVersion=1\ncommand\tpage\tanchor\n")
	for _, row := range mapRows {
		manMap.WriteString(row + "\n")
	}
	controls := []struct{ name, text string }{{"docs/inventory.tsv", inventory.String()}, {"docs/provenance.tsv", provenance.String()}, {"docs/man-map.tsv", manMap.String()}}
	controlBytes := 0
	for _, item := range controls {
		controlBytes += len(item.text)
		if len(item.text) > maxControlBytes {
			return buildResult{}, fmt.Errorf("documentary control exceeds limit")
		}
	}
	if controlBytes > 3*maxControlBytes {
		return buildResult{}, fmt.Errorf("documentary controls exceed limit")
	}
	if strings.Count(provenance.String(), "\n")-2 > maxProvenanceRows {
		return buildResult{}, fmt.Errorf("provenance row limit exceeded")
	}
	files := map[string][]byte{}
	for _, entry := range records {
		files[entry.output] = entry.data
	}
	for _, item := range controls {
		files[item.name] = []byte(item.text)
	}
	bundle, err := operationaldocs.Validate(files, commit)
	if err != nil {
		return buildResult{}, fmt.Errorf("validate documentation corpus: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return buildResult{}, err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(output), ".axiom-docs-build-")
	if err != nil {
		return buildResult{}, err
	}
	defer os.RemoveAll(tmp)
	for _, entry := range records {
		if err := writeNew(tmp, entry.output, entry.data); err != nil {
			return buildResult{}, err
		}
	}
	for _, item := range controls {
		if err := writeNew(tmp, item.name, []byte(item.text)); err != nil {
			return buildResult{}, err
		}
	}
	if err := os.Rename(tmp, output); err != nil {
		return buildResult{}, err
	}
	return buildResult{generation: bundle.Generation, outputs: len(records), bytes: total, sources: len(sourceSet), sourceHashes: sourceHashes}, nil
}

func readRegular(path string, limit int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return nil, fmt.Errorf("unsafe or oversized source: %s", filepath.Base(path))
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, fmt.Errorf("cannot read bounded source: %s", filepath.Base(path))
	}
	return data, nil
}

func record(output, source string, data []byte, kind string) sourceRecord {
	transform := "copy-v1"
	if source == "internal/cli/commands.go" && kind == "markdown" {
		transform = "cli-render-v1"
	}
	return sourceRecord{output: output, sources: []sourceOrigin{{source, digest(data), transform}}, data: data, kind: kind}
}

func writeNew(root, relative string, data []byte) error {
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validOutput(path string) bool {
	return !strings.HasPrefix(path, "/") && filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) == path && !strings.Contains(path, "..")
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func hasOutput(records []sourceRecord, name string) bool {
	for _, entry := range records {
		if entry.output == name {
			return true
		}
	}
	return false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
