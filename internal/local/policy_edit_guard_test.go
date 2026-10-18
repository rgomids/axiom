package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

func (f editPublicationFixture) policyRequest(t *testing.T) projectapp.EditPublicationRequest {
	t.Helper()
	ctx := context.Background()
	ports := f.policyPorts(t)
	intent := policyIntent(t, `{"runtimes":{"mode":"members","add":[{"id":"another"}]}}`)
	proposal, failure := projectapp.PreviewEdit(ctx, ports.Edit, intent)
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	selection, failure := ports.Edit.Source.SelectForEdit(ctx, "sample")
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	facts, failure := ports.Facts.ObservePolicyEdit(ctx, selection, proposal.Project())
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	return projectapp.EditPublicationRequest{ProjectID: editTestProjectID, Slug: "sample", PortableDestination: f.source, LocalDestination: filepath.Dir(f.recordPath()), PortableExpected: selection.Portable.Manifest(), PortableNext: proposal.Manifest(), LocalExpected: selection.LocalWire, LocalNext: proposal.LocalWire(), PolicyDependencies: &facts.Dependencies}
}

func TestPolicyPublicationGuardsAndPreservesLocalAbsence(t *testing.T) {
	f := newPolicyEditFixture(t)
	request := f.policyRequest(t)
	result := f.publisher(nil).PublishEdit(context.Background(), request)
	if result.Status != projectapp.EditPublicationApplied {
		t.Fatalf("guarded publication: %+v", result)
	}
	if !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableNext) || !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalNext) {
		t.Fatal("wrong generations published")
	}
	for _, path := range []string{filepath.Join(f.state, "runtime-profiles"), filepath.Join(filepath.Dir(f.recordPath()), operationalRecordName)} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("publication created dependency", err)
		}
	}
	if len(f.editStates(t)) != 0 {
		t.Fatal("successful publication retained recovery")
	}
}

func TestPolicyPublicationRejectsEachStaleDependency(t *testing.T) {
	for _, name := range []string{"installation", "source", "workflow", "operational", "runtime", "readiness", "locations"} {
		t.Run(name, func(t *testing.T) {
			f := newPolicyEditFixture(t)
			r := f.policyRequest(t)
			d := r.PolicyDependencies
			stale := hexDigest([]byte("stale"))
			switch name {
			case "installation":
				d.Installation = stale
			case "source":
				d.Source = stale
			case "workflow":
				d.Workflow = stale
			case "operational":
				d.Operational = stale
			case "runtime":
				d.Runtime = stale
			case "readiness":
				d.Readiness = stale
			case "locations":
				d.Locations = stale
			}
			result := f.publisher(nil).PublishEdit(context.Background(), r)
			if result.Status != projectapp.EditPublicationConflict || !bytes.Equal(fileBytes(t, f.manifestPath()), r.PortableExpected) || !bytes.Equal(fileBytes(t, f.recordPath()), r.LocalExpected) || len(f.editStates(t)) != 0 {
				t.Fatalf("stale dependency: %+v", result)
			}
		})
	}
}

func TestPolicyPublicationRechecksAtCommit(t *testing.T) {
	for _, name := range []string{"installation", "source", "workflow", "operational", "runtime"} {
		t.Run(name, func(t *testing.T) {
			f := newPolicyEditFixture(t)
			r := f.policyRequest(t)
			publisher := f.publisher(nil)
			publisher.Portable.beforeUpdatePublication = func() {
				switch name {
				case "installation":
					path := f.recordPath()
					if err := os.Rename(path, path+".moved"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, r.LocalExpected, 0600); err != nil {
						t.Fatal(err)
					}
				case "source":
					path := f.manifestPath()
					if err := os.Rename(path, path+".moved"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, r.PortableExpected, 0600); err != nil {
						t.Fatal(err)
					}
				case "workflow":
					path := filepath.Join(f.source, "workflows")
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(path, "index.json"), []byte("unsafe"), 0600); err != nil {
						t.Fatal(err)
					}
				case "operational":
					wire, err := EncodeOperational(editTestProjectID, projectapp.DefaultOperationalState())
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(filepath.Dir(f.recordPath()), operationalRecordName), wire, 0600); err != nil {
						t.Fatal(err)
					}
				case "runtime":
					if err := os.Mkdir(filepath.Join(f.state, "runtime-profiles"), 0700); err != nil {
						t.Fatal(err)
					}
				}
			}
			result := publisher.PublishEdit(context.Background(), r)
			if result.Status == projectapp.EditPublicationApplied || result.Status == projectapp.EditPublicationPartial || !bytes.Equal(fileBytes(t, f.manifestPath()), r.PortableExpected) {
				t.Fatalf("commit guard missed drift: %+v", result)
			}
		})
	}
}

func TestPolicyPublicationRejectsChangesOutsidePolicy(t *testing.T) {
	f := newPolicyEditFixture(t)
	r := f.policyRequest(t)
	r.PortableNext = bytes.ReplaceAll(r.PortableNext, []byte("SENTINEL-NAME"), []byte("changed-name"))
	result := f.publisher(nil).PublishEdit(context.Background(), r)
	if result.Status != projectapp.EditPublicationFailed || !bytes.Equal(fileBytes(t, f.manifestPath()), r.PortableExpected) || len(f.editStates(t)) != 0 {
		t.Fatal("unrelated mutation accepted", result)
	}
}

func TestPolicyPublicationProcessHelper(t *testing.T) {
	state := os.Getenv("AXIOM_POLICY_TEST_STATE")
	if state == "" {
		return
	}
	var request projectapp.EditPublicationRequest
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		t.Fatal(err)
	}
	installation, err := NewInstallationStore(state)
	if err != nil {
		t.Fatal(err)
	}
	portable, err := NewPortableStore(os.Getenv("AXIOM_POLICY_TEST_PROJECTS"))
	if err != nil {
		t.Fatal(err)
	}
	result := (ProjectEditPublisher{Installation: installation, Portable: portable}).PublishEdit(context.Background(), request)
	fmt.Println("policy-result:" + string(result.Status))
}

func TestPolicyPublicationConflictingProcesses(t *testing.T) {
	f := newPolicyEditFixture(t)
	request := f.policyRequest(t)
	wire, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	commands := make([]*exec.Cmd, 2)
	outputs := make([]bytes.Buffer, 2)
	for i := range commands {
		commands[i] = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPolicyPublicationProcessHelper$")
		commands[i].Env = append(os.Environ(), "AXIOM_POLICY_TEST_STATE="+f.state, "AXIOM_POLICY_TEST_PROJECTS="+f.projects)
		commands[i].Stdin = bytes.NewReader(wire)
		commands[i].Stdout = &outputs[i]
		commands[i].Stderr = &outputs[i]
		if err := commands[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	applied, conflicts := 0, 0
	for i, command := range commands {
		if err := command.Wait(); err != nil {
			t.Fatalf("publisher process: %v %s", err, outputs[i].String())
		}
		if strings.Contains(outputs[i].String(), "policy-result:applied") {
			applied++
		}
		if strings.Contains(outputs[i].String(), "policy-result:conflict") {
			conflicts++
		}
	}
	if applied != 1 || conflicts != 1 || !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableNext) || !bytes.Equal(fileBytes(t, f.recordPath()), request.LocalNext) {
		t.Fatalf("conflicting publishers: applied=%d conflict=%d", applied, conflicts)
	}
}

func TestPolicyPublicationDetectsEqualRevisionContentAndIdentityDrift(t *testing.T) {
	for _, name := range []string{"runtime_content", "operational_identity", "source_relocation", "manifest_hardlink"} {
		t.Run(name, func(t *testing.T) {
			f := newPolicyEditFixture(t)
			ctx := context.Background()
			store, err := NewRuntimeProfileStore(f.state)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := store.Inspect(ctx)
			if err != nil {
				t.Fatal(err)
			}
			cfg := fx.CodexOnly()
			wire, err := runtimeprofile.EncodeV3(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Publish(ctx, fresh.Token, wire); err != nil {
				t.Fatal(err)
			}
			operational, err := EncodeOperational(editTestProjectID, projectapp.DefaultOperationalState())
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(filepath.Dir(f.recordPath()), operationalRecordName)
			if err := os.WriteFile(path, operational, 0600); err != nil {
				t.Fatal(err)
			}
			request := f.policyRequest(t)
			publisher := f.publisher(nil)
			publisher.Portable.beforeUpdatePublication = func() {
				switch name {
				case "runtime_content":
					cfg.Runtimes[0].Enabled = false
					next, err := runtimeprofile.EncodeV3(cfg)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(f.state, "runtime-profiles", "v1", runtimeProfileObject), next, 0600); err != nil {
						t.Fatal(err)
					}
				case "operational_identity":
					if err := os.Rename(path, path+".moved"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, operational, 0600); err != nil {
						t.Fatal(err)
					}
				case "source_relocation":
					if err := os.Rename(f.source, f.source+"-moved"); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(f.source, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(f.manifestPath(), request.PortableExpected, 0600); err != nil {
						t.Fatal(err)
					}
				case "manifest_hardlink":
					if err := os.Link(f.manifestPath(), filepath.Join(f.projects, "linked")); err != nil {
						t.Fatal(err)
					}
				}
			}
			result := publisher.PublishEdit(ctx, request)
			if result.Status == projectapp.EditPublicationApplied || result.Status == projectapp.EditPublicationPartial || !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableExpected) {
				t.Fatal("drift committed", result)
			}
		})
	}
}

func TestPolicyPublicationGuardsObservedRepositoryReadiness(t *testing.T) {
	f := newPolicyEditFixture(t)
	repo := privateTestRoot(t)
	ctx := context.Background()
	intent := projectapp.EditIntent{Selector: "sample", RepositoryUpserts: []projectapp.RepositoryUpsert{{Key: "api", Path: repo}}}
	ports := f.policyPorts(t)
	proposal, failure := projectapp.PreviewEdit(ctx, ports.Edit, intent)
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	preview := proposal.Preview()
	applied := projectapp.ApplyEdit(ctx, ports.Edit, f.publisher(nil), intent, projectapp.EditAuthority{ProjectID: preview.ProjectID, PreviewDigest: preview.Digest, AuthorizeLocal: true})
	if applied.Outcome != projectapp.EditPublished {
		t.Fatal(applied.Outcome, applied.Failure)
	}
	request := f.policyRequest(t)
	publisher := f.publisher(nil)
	publisher.Portable.beforeUpdatePublication = func() {
		if err := os.Rename(repo, repo+"-moved"); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(repo, 0700); err != nil {
			t.Fatal(err)
		}
	}
	result := publisher.PublishEdit(ctx, request)
	if result.Status != projectapp.EditPublicationConflict || !bytes.Equal(fileBytes(t, f.manifestPath()), request.PortableExpected) {
		t.Fatal("observed Repository drift committed", result)
	}
}
