package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
)

// Test-only objective oracle for captured host prose, NOT a product renderer or
// proof that a host follows a skill. Synthetic probes qualify this oracle only.
type headerProject struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}
type headerDocument struct {
	completionEvent
	Project   *headerProject  `json:"project"`
	Projects  []headerProject `json:"projects"`
	Readiness struct {
		ProjectID string `json:"projectId"`
		Structure string `json:"structure"`
		Blockers  []struct {
			Code   string `json:"code"`
			Detail string `json:"detail"`
		} `json:"blockers"`
		Authority struct {
			GrantsAuthority bool   `json:"grantsAuthority"`
			LocalEffects    string `json:"localEffects"`
			ProviderEffects string `json:"providerEffects"`
		} `json:"authority"`
	} `json:"readiness"`
}
type headerCase struct {
	ID           string            `json:"id"`
	Skill        string            `json:"skill"`
	Operation    string            `json:"operation"`
	Documents    []json.RawMessage `json:"documents"`
	Raw          bool              `json:"raw"`
	Fault        string            `json:"fault"`
	PreviousCase string            `json:"previousCase"`
}

func headerCases(t *testing.T) []headerCase {
	t.Helper()
	wire, err := os.ReadFile("testdata/presentation/response-context/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []headerCase
	if err = json.Unmarshal(wire, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}
func headerDecode(c headerCase) ([]headerDocument, error) {
	docs := make([]headerDocument, len(c.Documents))
	if len(docs) == 0 {
		return nil, fmt.Errorf("missing documents")
	}
	for i, wire := range c.Documents {
		if err := json.Unmarshal(wire, &docs[i]); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

// Bounded normalization convention for these objective probes: collapse controls
// and whitespace, escape HTML and Markdown punctuation. Never interpret labels.
func headerInert(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	var out strings.Builder
	for _, r := range value {
		if strings.ContainsRune("\\`*_{}[]()#|", r) {
			out.WriteRune('\\')
		}
		out.WriteRune(r)
	}
	return html.EscapeString(out.String())
}
func headerObligations(c headerCase) (string, string, []string, error) {
	docs, err := headerDecode(c)
	if err != nil {
		return "", "", nil, err
	}
	last := docs[len(docs)-1]
	heading := "### Axiom"
	if last.Project == nil && last.Readiness.ProjectID != "" {
		last.Project = &headerProject{ID: last.Readiness.ProjectID}
	}
	if c.Operation != "project.list" && last.Project != nil && last.Project.ID != "" {
		label := last.Project.Slug
		names := map[string]bool{}
		for _, d := range docs {
			for _, p := range d.Projects {
				if p.ID == last.Project.ID && p.Name != "" {
					names[p.Name] = true
				}
			}
		}
		if len(names) == 1 {
			for name := range names {
				label = name
			}
		}
		if len(names) <= 1 && label != "" {
			heading += " · " + headerInert(label)
		}
	}
	versions := map[string]bool{}
	for _, d := range docs {
		if d.Provenance.Version != "" {
			versions[d.Provenance.Version] = true
		}
	}
	subtitle := "*Versão não observada*"
	required := []string{string(last.Status), last.Result, last.Next}
	if len(versions) == 1 && last.Provenance.Version != "" {
		subtitle = "*Versão " + headerInert(last.Provenance.Version) + "*"
	}
	if len(versions) > 1 {
		subtitle = "*Versão conflitante*"
		for v := range versions {
			required = append(required, v)
		}
	}
	required = append(required, last.References...)
	var auth struct {
		RuntimeAuth map[string]string `json:"runtimeAuth"`
	}
	if err := json.Unmarshal(c.Documents[len(c.Documents)-1], &auth); err != nil {
		return "", "", nil, err
	}
	for _, key := range []string{"status", "reason", "method", "usability", "revalidation"} {
		if fact := auth.RuntimeAuth[key]; fact != "" {
			required = append(required, fact)
		}
	}
	if last.Provenance.SourceState != "clean" {
		required = append(required, string(last.Provenance.SourceState))
	}
	if len(last.Readiness.Blockers) > 0 {
		required = append(required, "blocked", last.Readiness.Structure)
		for _, b := range last.Readiness.Blockers {
			required = append(required, b.Code, b.Detail)
		}
		required = append(required, "grantsAuthority=false", last.Readiness.Authority.LocalEffects, last.Readiness.Authority.ProviderEffects)
	}
	sort.Strings(required)
	return heading, subtitle, required, nil
}
func checkResponseContext(c headerCase, answer string) error {
	if c.Raw {
		if !bytes.Equal([]byte(answer), c.Documents[len(c.Documents)-1]) {
			return fmt.Errorf("raw JSON bytes changed")
		}
		return nil
	}
	heading, subtitle, required, err := headerObligations(c)
	if err != nil {
		return err
	}
	lines := strings.Split(answer, "\n")
	if len(lines) < 3 || lines[0] != heading || lines[1] != subtitle {
		return fmt.Errorf("incorrect heading/subtitle")
	}
	body := strings.Join(lines[2:], "\n")
	for _, fact := range required {
		if fact != "" && !strings.Contains(body, fact) {
			return fmt.Errorf("missing canonical fact %q", fact)
		}
	}
	// Objective contradiction probes are bounded, not a general semantic grader.
	lower := strings.ToLower(body)
	for _, bad := range []string{"pronto para executar", "ready to execute", "autoridade concedida", "authority granted", "dispatch realizado", "dispatch performed"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("unsupported readiness/authority claim")
		}
	}
	return nil
}
func TestResponseContextOracleMatrix(t *testing.T) {
	results := []map[string]any{}
	for _, c := range headerCases(t) {
		t.Run(c.ID, func(t *testing.T) {
			heading, subtitle, required, err := headerObligations(c)
			if err != nil {
				t.Fatal(err)
			}
			// Deliberately authored probes: never stored/advertised as host observations.
			probe := heading + "\n" + subtitle + "\n\n" + strings.Join(required, "; ")
			if c.Raw {
				probe = string(c.Documents[len(c.Documents)-1])
			}
			if err = checkResponseContext(c, probe); err != nil {
				t.Fatal(err)
			}
			negatives := []string{"### Axiom · invented\n*Versão 9.9.9*\n" + probe, "```json\n" + probe + "\n```", "\n" + probe}
			if !c.Raw {
				negatives = append(negatives, strings.Replace(probe, subtitle, "*Versão 9.9.9*", 1), strings.Replace(probe, heading, "### Axiom · stale-project", 1))
				for _, fact := range required {
					if fact != "" {
						negatives = append(negatives, strings.ReplaceAll(probe, fact, "[omitted]"))
					}
				}
				for _, bad := range []string{"Pronto para executar", "Authority granted", "Dispatch performed"} {
					negatives = append(negatives, probe+"\n"+bad)
				}
			} else {
				negatives = append(negatives, probe+" ")
			}
			for i, bad := range negatives {
				if checkResponseContext(c, bad) == nil {
					t.Fatalf("accepted negative probe %d", i)
				}
			}
			results = append(results, map[string]any{"case": c.ID, "status": "PASS", "origin": "synthetic_oracle_probe", "inputSha256": headerDigest(c.Documents), "negativeAssertions": len(negatives), "assertions": []string{"heading", "version", "canonical material facts", "negative mutations", "raw JSON when requested"}})
		})
	}
	if path := os.Getenv("AXIOM_HEADER_ORACLE_REPORT"); path != "" {
		head, err := exec.Command("git", "rev-parse", "HEAD").Output()
		if err != nil {
			t.Fatal(err)
		}
		skills := map[string]string{}
		for _, name := range []string{"axiom-project", "axiom-runtime", "axiom-work-item", "axiom-workflow"} {
			wire, err := os.ReadFile("../codexruntime/skills/" + name + "/SKILL.md")
			if err != nil {
				t.Fatal(err)
			}
			skills[name] = headerDigest(wire)
		}
		report := map[string]any{"sourceHead": strings.TrimSpace(string(head)), "sourceState": "dirty", "skillSha256": skills, "category": "deterministic_evaluator_self_test", "results": results, "behavioralEvaluation": "NOT_RUN", "nativeCodex": "NOT_RUN", "nativeClaude": "NOT_RUN", "reason": "no separate inference authority", "limitations": []string{"synthetic probes are not observed agent responses", "lexical fact preservation is not full semantic correctness"}}
		wire, _ := json.MarshalIndent(report, "", "  ")
		if err = os.WriteFile(path, append(wire, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func headerDigest(value any) string {
	wire, ok := value.([]byte)
	if !ok {
		wire, _ = json.Marshal(value)
	}
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}
func TestResponseContextIdentityAndVersionOracle(t *testing.T) {
	expected := map[string][2]string{
		"H01-name":                {"### Axiom · Órbita", "*Versão 0.22.1*"},
		"H02-slug":                {"### Axiom · orbita", "*Versão 0.22.1*"},
		"H03-list":                {"### Axiom", "*Versão 0.22.1*"},
		"H04-ambiguous":           {"### Axiom", "*Versão 0.22.1*"},
		"H05-unobserved":          {"### Axiom", "*Versão 0.22.1*"},
		"H06-missing-version":     {"### Axiom · orbita", "*Versão não observada*"},
		"H07-conflicting-version": {"### Axiom · Órbita", "*Versão conflitante*"},
		"H08-hostile-name":        {"### Axiom · \\*\\*Nome\\*\\* \\#\\#\\# falso &lt;script&gt;&amp;", "*Versão 0.22.1*"},
		"H11-successor":           {"### Axiom · segundo", "*Versão 0.23.0*"},
		"H12-vendor-version":      {"### Axiom", "*Versão 0.22.1*"},
		"H13-unicode":             {"### Axiom · 宇宙 · Órbita", "*Versão 0.22.1*"},
	}
	for _, c := range headerCases(t) {
		if want, ok := expected[c.ID]; ok {
			h, v, _, err := headerObligations(c)
			if err != nil || h != want[0] || v != want[1] {
				t.Fatalf("%s: got %q %q, want %q", c.ID, h, v, want)
			}
		}
	}
	cases := headerCases(t)
	h, v, _, _ := headerObligations(cases[0])
	_, _, req, _ := headerObligations(cases[10])
	if checkResponseContext(cases[10], h+"\n"+v+"\n"+strings.Join(req, "; ")) == nil {
		t.Fatal("accepted previous operation identity/version")
	}
}
func TestResponseContextFixtureProvenance(t *testing.T) {
	for _, c := range headerCases(t) {
		docs, err := headerDecode(c)
		if err != nil {
			t.Fatal(err)
		}
		for i, d := range docs {
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(c.Documents[i], &raw); err != nil {
				t.Fatal(err)
			}
			for key, target := range map[string]any{"project": &ProjectView{}, "projects": &[]ProjectListView{}} {
				if payload, ok := raw[key]; ok {
					decoder := json.NewDecoder(bytes.NewReader(payload))
					decoder.DisallowUnknownFields()
					if err := decoder.Decode(target); err != nil {
						t.Fatalf("%s invalid %s shape: %v", c.ID, key, err)
					}
				}
			}
			if d.Status != completion.Success || d.Provenance.Product != "Axiom" {
				t.Fatalf("invalid envelope %s", c.ID)
			}
			if c.Fault == "missing-version" {
				continue
			}
			source, err := provenance.FromBuild(provenance.Build{Release: d.Provenance.Version != provenance.Development, Version: d.Provenance.Version, Revision: d.Provenance.Revision, SourceState: d.Provenance.SourceState}, nil)
			if err != nil {
				t.Fatalf("invalid provenance %s: %v", c.ID, err)
			}
			statement, err := provenance.NewText(d.Result, provenance.AxiomAuthored)
			if err != nil {
				t.Fatal(err)
			}
			next, err := provenance.NewText(d.Next, provenance.AxiomAuthored)
			if err != nil {
				t.Fatal(err)
			}
			result, err := completion.New(completion.Facts{Completed: true}, statement, d.References, next, "", source)
			if err != nil {
				t.Fatal(err)
			}
			wire, err := renderCompletionJSON(result)
			if err != nil {
				t.Fatal(err)
			}
			var produced completionEvent
			if err = json.Unmarshal(wire, &produced); err != nil {
				t.Fatal(err)
			}
			if produced.Provenance != d.Provenance || produced.Status != d.Status || produced.Result != d.Result || produced.Next != d.Next {
				t.Fatalf("fixture diverges from real completion producer: %s", c.ID)
			}
		}
	}
}

// Optional offline grading of externally captured responses; never invokes a
// Runtime. Capture origin and authorization remain auditable human evidence.
func TestResponseContextNativeCaptures(t *testing.T) {
	root := os.Getenv("AXIOM_HEADER_CAPTURE_DIR")
	if root == "" {
		t.Skip("NOT_RUN: no authorized native captures supplied")
	}
	var manifest struct {
		Runtime     string
		Model       string
		Version     string
		Head        string
		SkillSha256 map[string]string
		Records     []struct {
			Case        string
			InputSha256 string
			Response    string
			ExitCode    int
			Command     []string
		}
	}
	wire, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(wire, &manifest); err != nil {
		t.Fatal(err)
	}
	pin := map[string]string{"codex": "gpt-6.1-luna", "claude": "claude-haiku-5-5"}[manifest.Runtime]
	if pin == "" || manifest.Model != pin || manifest.Version == "" || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(manifest.Head) {
		t.Fatal("missing or incompatible capture provenance")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || manifest.Head != strings.TrimSpace(string(head)) {
		t.Fatal("capture HEAD differs from current source")
	}
	cases := map[string]headerCase{}
	for _, c := range headerCases(t) {
		cases[c.ID] = c
	}
	seen := map[string]bool{}
	for _, r := range manifest.Records {
		c, ok := cases[r.Case]
		if !ok || seen[r.Case] || r.ExitCode != 0 || r.InputSha256 != headerDigest(c.Documents) {
			t.Fatal("invalid capture record", r.Case)
		}
		seen[r.Case] = true
		skill, err := os.ReadFile("../codexruntime/skills/" + c.Skill + "/SKILL.md")
		if err != nil || manifest.SkillSha256[c.Skill] != headerDigest(skill) {
			t.Fatal("skill drift", r.Case)
		}
		flag := "-m"
		if manifest.Runtime == "claude" {
			flag = "--model"
		}
		if len(r.Command) < 3 || r.Command[0] != manifest.Runtime || r.Command[1] != flag || r.Command[2] != pin {
			t.Fatal("model not explicitly pinned", r.Case)
		}
		if filepath.Base(r.Response) != r.Response || r.Response == "." {
			t.Fatal("invalid capture filename")
		}
		answer, err := os.ReadFile(filepath.Join(root, r.Response))
		if err != nil {
			t.Fatal(err)
		}
		if err = checkResponseContext(c, string(answer)); err != nil {
			t.Errorf("%s: %v", r.Case, err)
		}
	}
	if len(seen) != len(cases) {
		t.Fatalf("incomplete capture matrix: %d/%d", len(seen), len(cases))
	}
}

// Independently authored, explicitly synthetic probes keep material-fact tests
// independent from the obligation extractor. They are not agent observations.
func TestResponseContextIndependentMaterialProbes(t *testing.T) {
	probes := map[string]string{
		"H09-blocker":        "### Axiom\n*Versão 0.22.1*\n\nsuccess; Project is valid; Inspect configuration; no dispatch authorized; project:11111111-1111-4111-8111-111111111111; structure valid; execution blocked; runtime_resolution_blocked / inventory_unavailable; grantsAuthority=false; explicit_local_authorization; explicit_external_authorization",
		"H12-vendor-version": "### Axiom\n*Versão 0.22.1*\n\nsuccess; Claude subscription login observed; usability stays unproven until a real dispatch; Child dispatch repeats this preflight on its exact invocation; execution needs separate Runtime authorization; runtime-auth:claude:subscription_observed; subscription_observed; claude.ai; unproven_until_real_dispatch; before_each_dispatch",
	}
	for _, c := range headerCases(t) {
		if answer, ok := probes[c.ID]; ok {
			if err := checkResponseContext(c, answer); err != nil {
				t.Fatalf("independent %s: %v", c.ID, err)
			}
			docs, _ := headerDecode(c)
			for _, bad := range []string{"### Axiom\n*Versão 0.22.1*\nsuccess", strings.ReplaceAll(answer, docs[len(docs)-1].Next, "[omitted]")} {
				if checkResponseContext(c, bad) == nil {
					t.Fatalf("accepted material omission %s", c.ID)
				}
			}
		}
	}
}

func TestResponseContextQuotedLabelsRemainInert(t *testing.T) {
	if got := headerInert("\"O'Brien\""); got != "&#34;O&#39;Brien&#34;" {
		t.Fatalf("broken HTML entity: %q", got)
	}
}
