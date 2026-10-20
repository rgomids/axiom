package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

type DocumentationPage struct {
	Command  string
	Page     string
	Anchor   string
	Text     []byte
	Mappings []DocumentationMapping
}

type DocumentationMapping struct{ Command, Page, Anchor string }

func DocumentationPages(version, commit string) ([]DocumentationPage, error) {
	var pages []DocumentationPage
	byPath := map[string]*DocumentationPage{}
	documentationWalk(&publicCommands, nil, func(command *commandDefinition, path []string) {
		if len(command.children) != 0 && len(path) == 0 {
			return
		}
		full := append([]string{"axiom"}, path...)
		page, anchor := documentationLocation(path)
		var help bytes.Buffer
		if renderHelp(&help, command, path) != ExitSuccess {
			return
		}
		help.WriteString(renderFlagDefaults(command))
		entry := byPath[page]
		if entry == nil {
			entry = &DocumentationPage{Page: page}
			byPath[page] = entry
		}
		entry.Command = strings.Join(full, " ")
		entry.Anchor = anchor
		entry.Mappings = append(entry.Mappings, DocumentationMapping{entry.Command, page, anchor})
		entry.Text = append(entry.Text, renderManSection(anchor, strings.Join(full, " "), version, commit, help.String())...)
	})
	for _, page := range byPath {
		pages = append(pages, *page)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no public CLI documentation pages")
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Page < pages[j].Page })
	root := DocumentationPage{Command: "axiom", Page: "docs/man/man1/axiom.1", Anchor: "ROOT"}
	root.Mappings = append(root.Mappings, DocumentationMapping{"axiom", root.Page, "ROOT"})
	var help bytes.Buffer
	if renderHelp(&help, &publicCommands, nil) != ExitSuccess {
		return nil, fmt.Errorf("render root help")
	}
	root.Text = renderManSection("ROOT", "axiom", version, commit, help.String())
	for i := range pages {
		if pages[i].Page == root.Page {
			root.Text = append(root.Text, pages[i].Text...)
			root.Mappings = append(root.Mappings, pages[i].Mappings...)
		} else {
			pages[i].Text = renderManPage(manName(pages[i].Page), version, commit, pages[i].Text)
		}
	}
	filtered := pages[:0]
	for _, page := range pages {
		if page.Page != root.Page {
			filtered = append(filtered, page)
		}
	}
	pages = append(filtered, root)
	rootIndex := len(pages) - 1
	pages[rootIndex].Text = renderManPage("axiom", version, commit, pages[rootIndex].Text)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Page < pages[j].Page })
	return pages, nil
}

func documentationLocation(path []string) (string, string) {
	if len(path) == 0 || path[0] == "help" || path[0] == "version" {
		return "docs/man/man1/axiom.1", strings.ToUpper(strings.ReplaceAll(strings.Join(path, "_"), "-", "_"))
	}
	if path[0] == "runtime" && len(path) > 1 && (path[1] == "binding" || path[1] == "profile") {
		return "docs/man/man1/axiom-runtime-" + path[1] + ".1", strings.ToUpper(strings.ReplaceAll(strings.Join(path[1:], "_"), "-", "_"))
	}
	if path[0] == "docs" {
		return "docs/man/man1/axiom-docs.1", strings.ToUpper(strings.ReplaceAll(strings.Join(path[1:], "_"), "-", "_"))
	}
	return "docs/man/man1/axiom-" + path[0] + ".1", strings.ToUpper(strings.ReplaceAll(strings.Join(path, "_"), "-", "_"))
}

func renderManPage(name, version, commit string, sections []byte) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, ".TH %s 1 \"%s\" \"%s\" \"Axiom\"\n.SH SOURCE\n.nf\nVersion: %s\nCommit: %s\nTransform: cli-render-v1\n.fi\n", roffText(name), roffText(version), roffText(commit[:min(12, len(commit))]), printableText(version), printableText(commit))
	out.Write(sections)
	return out.Bytes()
}

func renderManSection(anchor, command, version, commit, help string) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, ".SH %s\n.nf\nCommand: %s\nVersion: %s\nCommit: %s\nTransform: cli-render-v1\n", anchor, roffText(command), roffText(version), roffText(commit))
	for _, line := range strings.Split(strings.TrimSuffix(help, "\n"), "\n") {
		out.WriteString(roffText(line))
		out.WriteByte('\n')
	}
	out.WriteString(".fi\n")
	return out.Bytes()
}

func roffText(value string) string {
	value = printableText(value)
	value = strings.ReplaceAll(value, "\\", "\\e")
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "'") {
			lines[i] = "\\&" + line
		}
	}
	return strings.Join(lines, "\n")
}

func printableText(value string) string {
	var out strings.Builder
	for _, r := range value {
		if r == '\n' || r == '\t' || r >= 0x20 && r != 0x7f && !(r >= 0x80 && r <= 0x9f) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func RenderDocumentationCLI(writer io.Writer, version, commit string) error {
	if writer == nil {
		return fmt.Errorf("nil documentation writer")
	}
	var out strings.Builder
	fmt.Fprintf(&out, "# Axiom CLI reference\n\nVersion: %s  \nCommit: %s  \nTransform: cli-render-v1\n\nGenerated from the public command tree and parser registrations.\n\n", version, commit)
	documentationWalk(&publicCommands, nil, func(command *commandDefinition, path []string) {
		var help bytes.Buffer
		if renderHelp(&help, command, path) == ExitSuccess {
			help.WriteString(renderFlagDefaults(command))
			name := strings.Join(append([]string{"axiom"}, path...), " ")
			out.WriteString("<a id=\"" + markdownAnchor(name) + "\"></a>\n\n## `" + name + "`\n\n```text\n")
			out.WriteString(help.String())
			out.WriteString("```\n\n")
		}
	})
	_, err := io.WriteString(writer, out.String())
	return err
}

func renderFlagDefaults(command *commandDefinition) string {
	if len(command.children) != 0 {
		return ""
	}
	var out strings.Builder
	set := commandFlagSet(command.operation)
	set.VisitAll(func(f *flag.Flag) { fmt.Fprintf(&out, "\nDefault values:\n  --%s: %s\n", f.Name, f.DefValue) })
	return out.String()
}

func markdownAnchor(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, " ", "-"), "_", "-")
}

func documentationWalk(node *commandDefinition, path []string, visit func(*commandDefinition, []string)) {
	visit(node, path)
	for i := range node.children {
		child := &node.children[i]
		documentationWalk(child, append(append([]string(nil), path...), child.name), visit)
	}
}

func manName(path string) string {
	base := strings.TrimPrefix(path, "docs/man/man1/")
	return strings.TrimSuffix(base, ".1")
}
