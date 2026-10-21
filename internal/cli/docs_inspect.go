package cli

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/operationaldocs"
	"github.com/rgomids/axiom/internal/provenance"
)

const docsInspectAction action = "docs_inspect"

const docsInspectLimit = 64 << 10

type docsBinary struct {
	Path       *string `json:"path"`
	SHA256     *string `json:"sha256"`
	Commit     *string `json:"sourceCommit"`
	Version    *string `json:"version"`
	ImageProof string  `json:"imageProof"`
}

type docsInstallation struct {
	BindingSHA256 *string `json:"bindingSha256"`
	Generation    *string `json:"generation"`
	Compatibility string  `json:"compatibility"`
}

type docsPaths struct {
	Root       *string `json:"root"`
	Index      *string `json:"index"`
	Inventory  *string `json:"inventory"`
	Provenance *string `json:"provenance"`
	ManRoot    *string `json:"manRoot"`
	ManMap     *string `json:"manMap"`
}

type docsIntegrity struct {
	Inventory  *string `json:"inventorySha256"`
	Provenance *string `json:"provenanceSha256"`
	ManMap     *string `json:"manMapSha256"`
	Files      int     `json:"files"`
	Bytes      int     `json:"bytes"`
}

type docsSkills struct {
	ExpectedManifest *string          `json:"expectedManifestSha256"`
	Expected         []map[string]any `json:"expected"`
	Observed         []map[string]any `json:"observed"`
}

type docsMan struct {
	State          string  `json:"state"`
	Page           *string `json:"page"`
	ReaderSelected string  `json:"readerSelection"`
}

type docsDiagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type docsReport struct {
	SchemaVersion int              `json:"schemaVersion"`
	State         string           `json:"state"`
	Reason        string           `json:"reason"`
	Binary        docsBinary       `json:"binary"`
	Installation  docsInstallation `json:"installation"`
	Paths         docsPaths        `json:"paths"`
	Integrity     docsIntegrity    `json:"integrity"`
	Skills        docsSkills       `json:"skills"`
	Man           docsMan          `json:"man"`
	Diagnostics   []docsDiagnostic `json:"diagnostics"`
}

var docsReasonText = map[string]string{
	"ok":                      "Installed documentation verified",
	"source_unqualified":      "This build carries no documentation descriptor",
	"binding_absent":          "No documentation binding matches this executable",
	"generation_absent":       "The bound documentation generation is absent",
	"legacy_installation":     "This installation predates installed documentation",
	"identity_mismatch":       "Installed documentation does not belong to this build",
	"hash_mismatch":           "Installed documentation bytes differ from their inventory",
	"inventory_mismatch":      "Installed documentation inventory is inconsistent",
	"malformed_record":        "An installed documentation record is malformed",
	"unsupported_format":      "An installed documentation record has an unsupported format",
	"unsafe_path":             "Installed documentation contains an unsafe path or type",
	"unsafe_permissions":      "Installed documentation has unsafe ownership or permissions",
	"binding_conflict":        "More than one documentation binding matches this executable",
	"permission_denied":       "Permission denied while inspecting installed documentation",
	"concurrent_change":       "Installed documentation changed during inspection",
	"quota_exceeded":          "Installed documentation exceeds an inspection bound",
	"io_unavailable":          "Installed documentation could not be read",
	"image_proof_unavailable": "The executing image could not be proven",
}

func ptr(value string) *string { return &value }

// SkillObservation is one Runtime's read-only skill observation. Loaded is
// never observed by this inspector and is always reported as unknown.
type SkillObservation struct{ Runtime, Receipt, Files, ManifestSHA256 string }

// SkillExpectation is the binary's own skill set, used only when it reproduces
// the verified installation's skill manifest digest.
type SkillExpectation struct {
	Digests  map[string]string
	Manifest string
}

func docsReportFor(r operationaldocs.Resolution, expectation SkillExpectation, observed []SkillObservation) docsReport {
	report := docsReport{SchemaVersion: 1, State: r.State, Reason: r.Reason, Binary: docsBinary{ImageProof: "unavailable"}, Installation: docsInstallation{Compatibility: "unknown"}, Skills: docsSkills{Expected: []map[string]any{}, Observed: []map[string]any{}}, Man: docsMan{State: "unknown", ReaderSelected: "unknown"}, Diagnostics: []docsDiagnostic{}}
	switch r.Reason {
	case "source_unqualified":
		report.Installation.Compatibility = "unqualified"
	case "identity_mismatch":
		report.Installation.Compatibility = "mismatch"
	case "legacy_installation":
		report.Installation.Compatibility = "legacy"
	}
	if r.State == operationaldocs.StateAbsent {
		report.Man.State = "absent"
	}
	if r.State == operationaldocs.StateVerified {
		b := r.Binding
		report.Binary = docsBinary{ptr(r.Binary), ptr(r.BinarySHA256), ptr(b.SourceCommit), ptr(b.Version), "unavailable"}
		report.Installation = docsInstallation{ptr(r.BindingSHA256), ptr(b.Generation), "compatible"}
		report.Paths = docsPaths{ptr(r.Locator("docs")), ptr(r.Locator("docs/text/index.md")), ptr(r.Locator("docs/inventory.tsv")), ptr(r.Locator("docs/provenance.tsv")), ptr(r.Locator("docs/man")), ptr(r.Locator("docs/man-map.tsv"))}
		report.Integrity = docsIntegrity{ptr(r.Bundle.InventorySHA256), ptr(r.Bundle.ProvenanceSHA256), ptr(r.Bundle.ManMapSHA256), len(r.Bundle.Entries) + 3, r.Bundle.Bytes}
		report.Skills.ExpectedManifest = ptr(b.SkillManifestSHA256)
		if expectation.Manifest == b.SkillManifestSHA256 {
			names := make([]string, 0, len(expectation.Digests))
			for name := range expectation.Digests {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				report.Skills.Expected = append(report.Skills.Expected, map[string]any{"name": name, "sha256": expectation.Digests[name]})
			}
		} else {
			report.Diagnostics = append(report.Diagnostics, docsDiagnostic{"identity_mismatch", "The binary's skills differ from the installed skill manifest"})
		}
		report.Man = docsMan{"available", ptr(r.Locator("docs/man/man1/axiom.1")), "unknown"}
		appendObserved(&report, observed)
		return report
	}
	report.Diagnostics = append(report.Diagnostics, docsDiagnostic{r.Reason, docsReasonText[r.Reason]})
	appendObserved(&report, observed)
	return report
}

func appendObserved(report *docsReport, observed []SkillObservation) {
	for _, o := range observed {
		entry := map[string]any{"runtime": o.Runtime, "receipt": o.Receipt, "files": o.Files, "manifestSha256": nil, "loaded": "unknown"}
		if o.ManifestSHA256 != "" {
			entry["manifestSha256"] = o.ManifestSHA256
		}
		report.Skills.Observed = append(report.Skills.Observed, entry)
	}
}

func docsFacts(state string) completion.Facts {
	switch state {
	case operationaldocs.StateVerified:
		return completion.Facts{Completed: true}
	case operationaldocs.StateInvalid:
		return completion.Facts{ValidationFailed: true}
	}
	return completion.Facts{Failed: true}
}

// InspectDocs answers `docs inspect` from the supplied read-only resolution and
// never starts a reader, a Runtime, or any write.
func InspectDocs(args []string, source provenance.Value, output io.Writer, resolve func() operationaldocs.Resolution, expectation SkillExpectation, observe func() []SkillObservation) (bool, int) {
	output = withHelpGuidance(output, args)
	mode, args := parseOutputMode(args)
	if len(args) == 0 || args[0] != "docs" {
		return false, 0
	}
	if len(args) != 2 || !commandMatches(args, docsInspectAction) {
		return true, emitParserFailure(output, mode, docsInspectAction, "invalid_input", source)
	}
	var observed []SkillObservation
	if observe != nil {
		observed = observe()
	}
	report := docsReportFor(resolve(), expectation, observed)
	text, next := docsReasonText[report.Reason], ""
	if report.State != operationaldocs.StateVerified {
		next = "Read axiom --help; this command never repairs or downloads documentation"
	}
	message, err := provenance.NewText(text, provenance.AxiomAuthored)
	if err != nil {
		return true, ExitFailure
	}
	var nextText provenance.Text
	if next != "" {
		if nextText, err = provenance.NewText(next, provenance.AxiomAuthored); err != nil {
			return true, ExitFailure
		}
	}
	result, err := completion.New(docsFacts(report.State), message, nil, nextText, "", source)
	if err != nil || output == nil {
		return true, ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), Next: result.Next().String(), Provenance: provenanceEvent{Product: source.Product(), Version: source.Version(), Revision: source.Revision(), SourceState: source.SourceState()}}
	wire, err := json.Marshal(struct {
		completionEvent
		Docs docsReport `json:"docs"`
	}{base, report})
	if err != nil {
		return true, ExitFailure
	}
	return true, presentEvent(output, mode, result.Status(), append(wire, '\n'), docsInspectLimit)
}
