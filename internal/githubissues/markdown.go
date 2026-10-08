package githubissues

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

// titleLimit keeps a generated title on one scannable line of a GitHub Issue
// list. Specificity comes from choosing the outcome's first line and, when it
// is too long, its leading whole sentences or a word-boundary cut.
const titleLimit = 72

var (
	blockMarkers = regexp.MustCompile(`^[ \t]*(?:(?:>[ \t]*)|(?:(?:#{1,6}|[-+*]|[0-9]{1,9}[.)]|\[[ xX]\])[ \t]+))*`)
	atxHeading   = regexp.MustCompile(`^#{1,6}(?:[ \t]|$)`)
	setextLine   = regexp.MustCompile(`^(?:=+|-+)[ \t]*$`)
	thematic     = regexp.MustCompile(`^(?:(?:-[ \t]*){3,}|(?:\*[ \t]*){3,}|(?:_[ \t]*){3,})$`)
	listMarker   = regexp.MustCompile(`^(?:[-+*]|[0-9]{1,9}[.)])(?:[ \t]|$)`)
	taskBox      = regexp.MustCompile(`^\[[ xX]\](?:[ \t]|$)`)
	plainLabel   = regexp.MustCompile(`^\[[^\]\[\\]*\](?:[^:]|$)`)
	autolink     = regexp.MustCompile(`^<[A-Za-z][A-Za-z0-9+.-]{1,31}:[^\s<>]*>`)
)

// title derives a concise, deterministic Issue title from the desired outcome.
func title(outcome string) string {
	line := ""
	for _, candidate := range strings.Split(outcome, "\n") {
		if line = strings.Join(strings.Fields(blockMarkers.ReplaceAllString(candidate, "")), " "); line != "" {
			break
		}
	}
	if line == "" {
		line = strings.Join(strings.Fields(outcome), " ")
	}
	runes := []rune(line)
	if len(runes) <= titleLimit {
		return withoutFinalPeriod(line)
	}
	// Prefer whole leading sentences when they keep most of the budget.
	for end := titleLimit; end >= titleLimit/2; end-- {
		if strings.ContainsRune(".!?", runes[end-1]) && runes[end] == ' ' {
			return withoutFinalPeriod(string(runes[:end]))
		}
	}
	cut := titleLimit - 1
	for space := cut; space >= cut/2; space-- {
		if runes[space] == ' ' {
			cut = space
			break
		}
	}
	// A hard cut never separates a base rune from its combining marks or a
	// zero-width-joined sequence.
	for cut > 1 && runes[cut] != ' ' && (unicode.Is(unicode.M, runes[cut]) || runes[cut] == '‍' || runes[cut-1] == '‍') {
		cut--
	}
	return strings.TrimRight(string(runes[:cut]), " ,;:-–—") + "…"
}

func withoutFinalPeriod(value string) string {
	if strings.HasSuffix(value, ".") && !strings.HasSuffix(value, "..") && utf8.RuneCountInString(value) > 1 {
		return value[:len(value)-1]
	}
	return value
}

// sectionMarkdown renders declared section content as native Markdown while
// guaranteeing it cannot produce Axiom's reserved structure: headings, thematic
// breaks, raw HTML (comments and markers included), link or footnote
// definitions, or a code fence that swallows later sections. Every change is a
// Markdown backslash escape, so the rendered text stays the declared content.
func sectionMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var out strings.Builder
	for index := 0; index < len(lines); index++ {
		if end := fencedBlockEnd(lines, index); end > index {
			// Indenting the verified block by one space keeps it a root fence,
			// leaves its rendered content unchanged, and keeps verbatim code
			// lines from starting at column 0 for line-based body consumers.
			for _, line := range lines[index:end] {
				if line != "" {
					out.WriteByte(' ')
				}
				out.WriteString(line)
				out.WriteByte('\n')
			}
			out.WriteString(" " + strings.TrimSpace(lines[end]) + "\n")
			index = end
			continue
		}
		out.WriteString(escapeLine(lines[index]))
		out.WriteByte('\n')
	}
	return out.String()
}

// fencedBlockEnd returns the closing line of a code fence opened at column 0
// of lines[start], or -1. Only column-0 fences are trusted because they cannot
// belong to a list item or block quote; every other fence-like line is escaped.
func fencedBlockEnd(lines []string, start int) int {
	opener := lines[start]
	if len(opener) < 3 || (opener[0] != '`' && opener[0] != '~') {
		return -1
	}
	char := opener[0]
	run := len(opener) - len(strings.TrimLeft(opener, string(char)))
	if run < 3 || char == '`' && strings.ContainsRune(opener[run:], '`') {
		return -1
	}
	for index := start + 1; index < len(lines); index++ {
		line := lines[index]
		indent := len(line) - len(strings.TrimLeft(line, " "))
		rest := line[indent:]
		closing := len(rest) - len(strings.TrimLeft(rest, string(char)))
		if indent <= 3 && closing >= run && strings.Trim(rest[closing:], " \t") == "" {
			return index
		}
	}
	return -1
}

func escapeLine(line string) string {
	position := 0
	for {
		start := position
		for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
			start++
		}
		rest := line[start:]
		if structural(rest) {
			return line[:start] + `\` + escapeInline(rest)
		}
		if strings.HasPrefix(rest, ">") {
			position = start + 1
			continue
		}
		if marker := listMarker.FindString(rest); marker != "" {
			position = start + len(strings.TrimRight(marker, " \t"))
			continue
		}
		return line[:start] + escapeInline(rest)
	}
}

func structural(rest string) bool {
	if rest == "" {
		return false
	}
	switch {
	case atxHeading.MatchString(rest), setextLine.MatchString(rest), thematic.MatchString(rest):
		return true
	case strings.HasPrefix(rest, "```"), strings.HasPrefix(rest, "~~~"):
		return true
	case rest[0] == '[':
		return !taskBox.MatchString(rest) && !plainLabel.MatchString(rest)
	}
	return false
}

// escapeInline neutralizes every raw HTML start (tags, comments, declarations,
// processing instructions). It also applies inside inline code, where the
// backslash stays visible: inline spans cannot be identified safely without a
// full Markdown parser, and a wrong guess would let hidden HTML through.
func escapeInline(text string) string {
	var out strings.Builder
	for index := 0; index < len(text); index++ {
		current := text[index]
		if current == '\\' && index+1 < len(text) && isASCIIPunctuation(text[index+1]) {
			out.WriteString(text[index : index+2])
			index++
			continue
		}
		if current == '<' {
			if link := autolink.FindString(text[index:]); link != "" {
				out.WriteString(link)
				index += len(link) - 1
				continue
			}
			if index+1 < len(text) && htmlStart(text[index+1]) {
				out.WriteByte('\\')
			}
		}
		out.WriteByte(current)
	}
	return out.String()
}

func htmlStart(next byte) bool {
	return next >= 'a' && next <= 'z' || next >= 'A' && next <= 'Z' || next == '/' || next == '!' || next == '?'
}

func isASCIIPunctuation(value byte) bool {
	return value >= '!' && value <= '/' || value >= ':' && value <= '@' || value >= '[' && value <= '`' || value >= '{' && value <= '~'
}

// provenanceFooter states provenance and per-section authorship once, after
// all section content, instead of repeating it inside every section.
func provenanceFooter(source provenance.Value, sections []workitem.DraftSection) string {
	names := make([]string, 0, len(sections))
	authorship := make(map[string]provenance.Authorship, len(sections))
	for _, current := range sections {
		names = append(names, displayName(current.Name))
		authorship[displayName(current.Name)] = current.Authorship
	}
	return "---\n\n_Structure authored by " + source.Product() + " `" + source.Version() + "` (revision `" + source.Revision() + "`, " + string(source.SourceState()) + " source); section content keeps its declared authorship._\n" + authorshipStatement(names, authorship) + "\n"
}

// authorshipStatement is the single footer statement of per-section
// authorship, in document order; ReviseDocument re-renders it the same way.
func authorshipStatement(names []string, authorship map[string]provenance.Authorship) string {
	var user, axiom []string
	for _, name := range names {
		if authorship[name] == provenance.AxiomAuthored {
			axiom = append(axiom, name)
		} else {
			user = append(user, name)
		}
	}
	switch {
	case len(user) == 0:
		return "_All section content is Axiom-authored._"
	case len(axiom) != 0:
		return "_User-authored: " + strings.Join(user, ", ") + ". Axiom-authored: " + strings.Join(axiom, ", ") + "._"
	}
	return "_All section content is user-authored._"
}

func displayName(name string) string {
	switch name {
	case "beneficiary", "value":
		return "Story " + name
	}
	return heading(name)
}
