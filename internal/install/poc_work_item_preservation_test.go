package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workitem"
)

// Save through the current writer: its resource-qualified filename is not a
// historical filename and must not be mistaken for POC workflow membership.
func saveTransitionLink(t *testing.T, installed installation, number int, resource string) (workitem.Link, string) {
	t.Helper()
	store, err := local.NewWorkItemStore(installed.target.State.State)
	if err != nil {
		t.Fatal(err)
	}
	link := workitem.Link{ProjectID: pocProjectID, RepositoryKey: "main", Provider: "github", Resource: resource, ExternalID: strconv.Itoa(number), URL: "https://github.com/" + resource + "/issues/" + strconv.Itoa(number), State: "OPEN"}
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(link.Provider + "\x00" + link.Resource + "\x00" + link.ExternalID))
	path := filepath.Join(installed.target.State.State, "work-items", pocProjectID, "main-"+hex.EncodeToString(digest[:])+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	return link, path
}

func transitionSourceBytes(t *testing.T, installed installation) map[string]string {
	t.Helper()
	objects := map[string]string{}
	for category, root := range map[string]string{"state": installed.target.State.State, "projects": installed.target.State.Projects} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			wire, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			objects[category+"/"+filepath.ToSlash(relative)] = string(wire)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return objects
}

func assertTransitionObjectSets(t *testing.T, installed installation, expected map[string]string, retirePaths []string) {
	t.Helper()
	archive := installed.archivePath(t)
	document := readManifest(t, archive)
	seen := map[string]string{}
	retired := map[string]bool{}
	for _, path := range retirePaths {
		retired[path] = true
	}
	wantKept := []string{}
	for path := range expected {
		if !retired[path] {
			wantKept = append(wantKept, path)
		}
	}
	for _, object := range document.Objects {
		key := object.Category + "/" + object.Relative
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate manifest object %s", key)
		}
		wire, err := os.ReadFile(filepath.Join(archive, "objects", object.Digest))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(wire)
		if hex.EncodeToString(digest[:]) != object.Digest || int64(len(wire)) != object.Bytes {
			t.Fatalf("archive object %s does not verify", key)
		}
		seen[key] = string(wire)
	}
	if !reflect.DeepEqual(seen, expected) {
		t.Fatalf("archive object set/bytes differ: got %d, want %d", len(seen), len(expected))
	}
	sorted := func(paths []string) []string {
		result := append([]string{}, paths...)
		sort.Strings(result)
		return result
	}
	if !reflect.DeepEqual(sorted(document.Retired), sorted(retirePaths)) || !reflect.DeepEqual(sorted(document.Kept), sorted(wantKept)) {
		t.Fatalf("manifest retired=%v kept=%v, want retired=%v kept=%v", document.Retired, document.Kept, retirePaths, wantKept)
	}
	for path, wire := range expected {
		parts := strings.SplitN(path, "/", 2)
		root := installed.target.State.State
		if parts[0] == "projects" {
			root = installed.target.State.Projects
		}
		active := filepath.Join(root, filepath.FromSlash(parts[1]))
		if retired[path] {
			if _, err := os.Lstat(active); !os.IsNotExist(err) {
				t.Fatalf("retired %s present: %v", path, err)
			}
			continue
		}
		if got := read(t, active); got != wire {
			t.Fatalf("kept %s changed", path)
		}
	}
}

func TestRecognizedPOCUpgradeKeepsUnrelatedWorkItemLinks(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		t.Run(strconv.FormatBool(multiple), func(t *testing.T) {
			installed := pocInstallation(t)
			kept, _ := saveTransitionLink(t, installed, 42, "owner/repo")
			retired := []string{"state/work-items/" + pocProjectID + "/main-7.json", "state/workflows/" + pocProjectID + "/main-7.json"}
			if multiple {
				wire := strings.Replace(read(t, installed.pocWorkflow()), `"workItem":7`, `"workItem":8`, 1)
				workflowPath := filepath.Join(filepath.Dir(installed.pocWorkflow()), "main-8.json")
				writeFile(t, workflowPath, []byte(wire), 0o600)
				_, linkPath := saveTransitionLink(t, installed, 8, "owner/repo")
				relative, err := filepath.Rel(installed.target.State.State, linkPath)
				if err != nil {
					t.Fatal(err)
				}
				retired = append(retired, "state/"+filepath.ToSlash(relative), "state/workflows/"+pocProjectID+"/main-8.json")
			}
			expected := transitionSourceBytes(t, installed)
			report, err := compatibility.Inspect(context.Background(), installed.target.State)
			if err != nil || report.Classification != compatibility.RecognizedPOC {
				t.Fatalf("classification=%+v err=%v", report, err)
			}
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil {
				t.Fatal(err)
			}
			if countKind(preview.Effects, "retire") != len(retired) {
				t.Fatalf("retirement count=%d, want %d", countKind(preview.Effects, "retire"), len(retired))
			}
			result, err := applyPreview(t, NewService(), preview)
			if err != nil || result.Status != "success" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			assertTransitionObjectSets(t, installed, expected, retired)
			store, err := local.NewWorkItemStore(installed.target.State.State)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := store.Load(context.Background(), kept.ProjectID, kept.RepositoryKey, kept.Provider, kept.Resource, kept.ExternalID)
			if err != nil || loaded.Resource != kept.Resource || loaded.ExternalID != kept.ExternalID || loaded.URL != kept.URL {
				t.Fatalf("kept link=%+v err=%v", loaded, err)
			}
			report, err = compatibility.Inspect(context.Background(), installed.target.State)
			if err != nil || report.Classification != compatibility.ValidV1 {
				t.Fatalf("rebuilt=%+v err=%v", report, err)
			}
		})
	}
}

func TestRecognizedPOCUpgradeRefusesUnprovedWorkItemMembership(t *testing.T) {
	for _, name := range []string{"missing matching link", "ambiguous provider resources"} {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			if name == "missing matching link" {
				if err := os.Remove(installed.workItem()); err != nil {
					t.Fatal(err)
				}
			} else {
				saveTransitionLink(t, installed, 7, "other/repo")
			}
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			before := snapshot(t, filepath.Dir(installed.target.ReceiptDir))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err == nil || preview.Digest != "" || len(preview.Effects) != 0 {
				t.Fatalf("unproved membership authorized: effects=%v digest=%q err=%v", effectKinds(preview.Effects), preview.Digest, err)
			}
			if snapshot(t, filepath.Dir(installed.target.ReceiptDir)) != before {
				t.Fatal("refusal mutated installation")
			}
		})
	}
}

func TestRecognizedPOCResumeRefusesMissingKeptWorkItemLink(t *testing.T) {
	installed := pocInstallation(t)
	_, keptPath := saveTransitionLink(t, installed, 42, "owner/repo")
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	preview, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	service := Service{afterEffect: func(label string) error {
		if strings.HasPrefix(label, "retire:state/work-items/") {
			return errors.New("stop after first confirmed retirement")
		}
		return nil
	}}
	result, err := applyPreview(t, service, preview)
	if err == nil || result.Status != "partial" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if err := os.Remove(keptPath); err != nil {
		t.Fatalf("unrelated link retired before interruption: %v", err)
	}
	before := snapshot(t, filepath.Dir(installed.target.ReceiptDir))
	resume, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err == nil || resume.Digest != "" || len(resume.Effects) != 0 {
		t.Fatalf("missing kept link accepted: effects=%v digest=%q err=%v", effectKinds(resume.Effects), resume.Digest, err)
	}
	if snapshot(t, filepath.Dir(installed.target.ReceiptDir)) != before {
		t.Fatal("refused resume mutated installation")
	}
}

func TestRecognizedPOCUpgradeRefusesIdentityLocationMismatch(t *testing.T) {
	for _, name := range []string{"workflow filename", "workflow project", "link filename", "link project"} {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			source := installed.pocWorkflow()
			if strings.HasPrefix(name, "link") {
				source = installed.workItem()
			}
			target := filepath.Join(filepath.Dir(source), "main-8.json")
			if strings.HasSuffix(name, "project") {
				target = filepath.Join(filepath.Dir(filepath.Dir(source)), "1b8dfadf-0e89-46da-93b6-fb6a486d9142", filepath.Base(source))
				if err := os.Mkdir(filepath.Dir(target), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Rename(source, target); err != nil {
				t.Fatal(err)
			}
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			before := snapshot(t, filepath.Dir(installed.target.ReceiptDir))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err == nil || preview.Digest != "" || len(preview.Effects) != 0 {
				t.Fatalf("identity location mismatch accepted: effects=%v digest=%q err=%v", effectKinds(preview.Effects), preview.Digest, err)
			}
			if snapshot(t, filepath.Dir(installed.target.ReceiptDir)) != before {
				t.Fatal("refusal mutated installation")
			}
		})
	}
}

func TestRecognizedPOCResumeRefusesKeptWorkItemReclassifiedAsRetired(t *testing.T) {
	installed := pocInstallation(t)
	_, keptPath := saveTransitionLink(t, installed, 42, "owner/repo")
	relative, err := filepath.Rel(installed.target.State.State, keptPath)
	if err != nil {
		t.Fatal(err)
	}
	keptKey := "state/" + filepath.ToSlash(relative)
	retiredKey := "state/work-items/" + pocProjectID + "/main-7.json"
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	preview, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	interrupted := Service{afterEffect: func(label string) error {
		if label == "preservation_manifest" {
			return errors.New("stop before first retirement")
		}
		return nil
	}}
	if _, err := applyPreview(t, interrupted, preview); err == nil {
		t.Fatal("interruption missing")
	}
	archive := installed.archivePath(t)
	document := readManifest(t, archive)
	foundKept, foundRetired := false, false
	for i, key := range document.Kept {
		if key == keptKey {
			foundKept = true
			document.Kept[i] = retiredKey
		}
	}
	for i, key := range document.Retired {
		if key == retiredKey {
			foundRetired = true
			document.Retired[i] = keptKey
		}
	}
	if !foundKept || !foundRetired {
		t.Fatal("manifest lacks expected kept #42 or retired #7 before tampering")
	}
	rewriteManifest(t, archive, document)
	before := snapshot(t, filepath.Dir(installed.target.ReceiptDir))
	resume, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err == nil || resume.Digest != "" || len(resume.Effects) != 0 {
		t.Fatalf("kept/retired swap accepted: effects=%v digest=%q err=%v", effectKinds(resume.Effects), resume.Digest, err)
	}
	if snapshot(t, filepath.Dir(installed.target.ReceiptDir)) != before {
		t.Fatal("refused resume mutated installation")
	}
	if _, err := os.Stat(keptPath); err != nil {
		t.Fatalf("kept link removed: %v", err)
	}
}
