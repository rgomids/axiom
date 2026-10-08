package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/testfs"
)

const operationalProjectID = "123e4567-e89b-42d3-a456-426614174000"

// installedOperationalFixture installs one Project into a fresh state root
// and returns that root plus an operational store over it.
func installedOperationalFixture(t *testing.T) (string, OperationalStore) {
	t.Helper()
	source := privateTestRoot(t)
	manifest := []byte("schemaVersion: 1\nproject:\n  id: " + operationalProjectID + "\n  slug: sample\n  name: Sample\n")
	if err := os.WriteFile(filepath.Join(source, manifestName), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	state := privateTestRoot(t)
	installations, err := NewInstallationStore(state)
	if err != nil {
		t.Fatal(err)
	}
	if got := installations.Install(context.Background(), source); got.Status != InstallationApplied {
		t.Fatalf("install = %+v", got)
	}
	store, err := NewOperationalStore(state)
	if err != nil {
		t.Fatal(err)
	}
	return state, store
}

func operationalPath(state string) string {
	return filepath.Join(state, "projects", operationalProjectID, operationalRecordName)
}

func archived(keys ...string) projectapp.OperationalState {
	return projectapp.OperationalState{ProjectStatus: projectapp.ProjectArchived, DisabledIntegrations: append([]string{}, keys...)}
}

func TestOperationalMissingRecordMeansActiveWithNoDisabledIntegration(t *testing.T) {
	_, store := installedOperationalFixture(t)
	observation, err := store.InspectOperational(context.Background(), operationalProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Exists || observation.Revision != projectapp.OperationalRevisionAbsent || !observation.State.Equal(projectapp.DefaultOperationalState()) {
		t.Fatalf("missing record observation = %+v", observation)
	}
}

func TestOperationalRoundTripIsStrictAndCanonical(t *testing.T) {
	state, store := installedOperationalFixture(t)
	next := archived("docs", "work-items")
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, next); err != nil {
		t.Fatal(err)
	}
	wire, err := os.ReadFile(operationalPath(state))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"formatVersion":1,"projectId":"` + operationalProjectID + `","projectStatus":"archived","disabledIntegrations":["docs","work-items"]}` + "\n"
	if string(wire) != want {
		t.Fatalf("wire = %s", wire)
	}
	observation, err := store.InspectOperational(context.Background(), operationalProjectID)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(wire)
	if !observation.Exists || observation.Revision != hex.EncodeToString(digest[:]) || !observation.State.Equal(next) {
		t.Fatalf("observation = %+v", observation)
	}
	if !testfs.PrivateMode(operationalPath(state), 0o600) {
		t.Fatal("operational record must have private permissions")
	}
}

func TestOperationalRejectsSharedRecordPermissions(t *testing.T) {
	state, store := installedOperationalFixture(t)
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived()); err != nil {
		t.Fatal(err)
	}
	observation, err := store.InspectOperational(context.Background(), operationalProjectID)
	if err != nil {
		t.Fatal(err)
	}
	path := operationalPath(state)
	if err := testfs.SharedMode(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if testfs.PrivateMode(path, 0o600) {
		t.Fatal("shared operational record passed private permission check")
	}
	if _, err := store.InspectOperational(context.Background(), operationalProjectID); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("inspect shared record error = %v", err)
	}
	if err := store.CommitOperational(context.Background(), operationalProjectID, observation.Revision, projectapp.DefaultOperationalState()); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("commit shared record error = %v", err)
	}
}

func TestOperationalDecoderFailsClosed(t *testing.T) {
	prefix := `{"formatVersion":1,"projectId":"` + operationalProjectID + `",`
	for name, wire := range map[string]string{
		"empty":                 "",
		"not json":              "archived\n",
		"unknown version":       `{"formatVersion":2,"projectId":"` + operationalProjectID + `","projectStatus":"active","disabledIntegrations":[]}` + "\n",
		"zero version":          `{"formatVersion":0,"projectId":"` + operationalProjectID + `","projectStatus":"active","disabledIntegrations":[]}` + "\n",
		"string version":        `{"formatVersion":"1","projectId":"` + operationalProjectID + `","projectStatus":"active","disabledIntegrations":[]}` + "\n",
		"unknown field":         prefix + `"projectStatus":"active","disabledIntegrations":[],"archivedBy":"x"}` + "\n",
		"duplicate field":       prefix + `"projectStatus":"active","projectStatus":"archived","disabledIntegrations":[]}` + "\n",
		"missing status":        prefix + `"disabledIntegrations":[]}` + "\n",
		"missing integrations":  prefix + `"projectStatus":"active"}` + "\n",
		"null integrations":     prefix + `"projectStatus":"active","disabledIntegrations":null}` + "\n",
		"unknown status":        prefix + `"projectStatus":"deleted","disabledIntegrations":[]}` + "\n",
		"unsorted keys":         prefix + `"projectStatus":"active","disabledIntegrations":["work-items","docs"]}` + "\n",
		"duplicate keys":        prefix + `"projectStatus":"active","disabledIntegrations":["docs","docs"]}` + "\n",
		"unsafe key":            prefix + `"projectStatus":"active","disabledIntegrations":["../docs"]}` + "\n",
		"invalid project id":    `{"formatVersion":1,"projectId":"sample","projectStatus":"active","disabledIntegrations":[]}` + "\n",
		"noncanonical spacing":  `{"formatVersion": 1,"projectId":"` + operationalProjectID + `","projectStatus":"active","disabledIntegrations":[]}` + "\n",
		"missing final newline": prefix + `"projectStatus":"active","disabledIntegrations":[]}`,
		"trailing value":        prefix + `"projectStatus":"active","disabledIntegrations":[]}` + "\n{}\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := DecodeOperational([]byte(wire)); !errors.Is(err, ErrUnsafe) {
				t.Fatalf("decode error = %v", err)
			}
		})
	}
}

func TestOperationalMalformedRecordIsNeverTreatedAsAbsent(t *testing.T) {
	state, store := installedOperationalFixture(t)
	malformed := []byte(`{"formatVersion":2,"projectId":"` + operationalProjectID + `","projectStatus":"active","disabledIntegrations":[]}` + "\n")
	if err := os.WriteFile(operationalPath(state), malformed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InspectOperational(context.Background(), operationalProjectID); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("inspect error = %v", err)
	}
	// Neither an "absent" nor the malformed revision authorizes overwriting it.
	for _, expected := range []string{projectapp.OperationalRevisionAbsent, revisionOf(malformed)} {
		if err := store.CommitOperational(context.Background(), operationalProjectID, expected, archived()); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("commit over malformed with %s = %v", expected, err)
		}
	}
	after, err := os.ReadFile(operationalPath(state))
	if err != nil || !bytes.Equal(after, malformed) {
		t.Fatalf("malformed record changed: %s err=%v", after, err)
	}
}

func TestOperationalRecordForAnotherProjectFailsClosed(t *testing.T) {
	state, store := installedOperationalFixture(t)
	wire, err := EncodeOperational(foreignProjectUUID, archived())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(operationalPath(state), wire, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InspectOperational(context.Background(), operationalProjectID); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("inspect error = %v", err)
	}
}

func TestOperationalStaleRevisionConflictsWithoutWrite(t *testing.T) {
	state, store := installedOperationalFixture(t)
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(operationalPath(state))
	// The absent revision observed before the first commit is now stale.
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, projectapp.DefaultOperationalState()); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale commit = %v", err)
	}
	if err := store.CommitOperational(context.Background(), operationalProjectID, strings.Repeat("0", 64), projectapp.DefaultOperationalState()); !errors.Is(err, ErrConflict) {
		t.Fatalf("foreign revision commit = %v", err)
	}
	after, _ := os.ReadFile(operationalPath(state))
	if !bytes.Equal(before, after) {
		t.Fatalf("stale commit changed the record")
	}
}

func TestOperationalCommitRequiresAnInstalledProject(t *testing.T) {
	state := privateTestRoot(t)
	store, err := NewOperationalStore(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("commit without installation = %v", err)
	}
	if _, err := store.InspectOperational(context.Background(), operationalProjectID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("inspect without installation = %v", err)
	}
	if entries, _ := os.ReadDir(state); len(entries) != 0 {
		t.Fatalf("commit created state: %v", entries)
	}
	// An installation directory without its record is not an installed Project.
	directory := filepath.Join(state, "projects", operationalProjectID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(state, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("commit into empty installation directory = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(directory, operationalRecordName)); !os.IsNotExist(err) {
		t.Fatalf("record written without installation: %v", err)
	}
}

func TestOperationalOrphanRecordBlocksNewInstallation(t *testing.T) {
	state, _ := installedOperationalFixture(t)
	directory := filepath.Join(state, "projects", operationalProjectID)
	wire, err := EncodeOperational(operationalProjectID, archived())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(directory, "installation.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, operationalRecordName), wire, 0o600); err != nil {
		t.Fatal(err)
	}
	source := privateTestRoot(t)
	manifest := []byte("schemaVersion: 1\nproject:\n  id: " + operationalProjectID + "\n  slug: sample\n  name: Sample\n")
	if err := os.WriteFile(filepath.Join(source, manifestName), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	installations, err := NewInstallationStore(state)
	if err != nil {
		t.Fatal(err)
	}
	if got := installations.Install(context.Background(), source); got.Status != InstallationFailed || got.Category != "invalid_existing_local_state" {
		t.Fatalf("install adopting orphan operational state = %+v", got)
	}
}

func TestOperationalRecordDoesNotDisturbInstalledProjectResolution(t *testing.T) {
	state, store := installedOperationalFixture(t)
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived("work-items")); err != nil {
		t.Fatal(err)
	}
	installations, err := NewInstallationStore(state)
	if err != nil {
		t.Fatal(err)
	}
	if got := installations.Select(context.Background(), "sample"); got.Status != ResolutionFound || got.Project.ID != operationalProjectID {
		t.Fatalf("select with operational state = %+v", got)
	}
	if observation, category := installations.Inspect(context.Background(), operationalProjectID); category != "" || !observation.Exists {
		t.Fatalf("inspect with operational state = %+v %s", observation, category)
	}
	listed, err := installations.ListInstalled(context.Background())
	if err != nil || len(listed) != 1 {
		t.Fatalf("list with operational state = %+v err=%v", listed, err)
	}
}

func TestOperationalInterruptedPublicationFailsClosedUntilRecovery(t *testing.T) {
	for _, stage := range []FaultStage{FaultF3, FaultF5, FaultF6, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			state, store := installedOperationalFixture(t)
			store.hooks.fault = func(current FaultStage) error {
				if current == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived())
			var publication *PublicationError
			if !errors.As(err, &publication) {
				t.Fatalf("interrupted commit = %v", err)
			}
			store.hooks = publicationHooks{}
			if _, err := store.InspectOperational(context.Background(), operationalProjectID); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("inspect after interruption = %v", err)
			}
			if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived()); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("commit after interruption = %v", err)
			}
			installations, err := NewInstallationStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if got := installations.Select(context.Background(), operationalProjectID); got.Category != "recovery_required" {
				t.Fatalf("select after interruption = %+v", got)
			}
			roots := RecoveryRoots{State: state}
			plan := singlePlan(t, roots)
			if plan.Directory != "projects/"+operationalProjectID || plan.Protocol != protocolFile {
				t.Fatalf("recovery plan = %+v", plan)
			}
			want := RestorePrior
			if stage >= FaultF6 {
				want = FinalizeCommitted
			}
			if plan.Action != want || publication.Committed != (want == FinalizeCommitted) {
				t.Fatalf("stage %s plan=%s committed=%t", stage, plan.Action, publication.Committed)
			}
			authority, err := AuthorizeRecovery(plan, plan.Digest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ApplyRecovery(context.Background(), roots, plan, authority); err != nil {
				t.Fatal(err)
			}
			observation, err := store.InspectOperational(context.Background(), operationalProjectID)
			if err != nil {
				t.Fatal(err)
			}
			if observation.State.Archived() != (want == FinalizeCommitted) {
				t.Fatalf("recovered state = %+v", observation)
			}
		})
	}
}

func TestOperationalRejectsLinkedRecordAndDirectory(t *testing.T) {
	t.Run("record symlink", func(t *testing.T) {
		state, store := installedOperationalFixture(t)
		outside := filepath.Join(t.TempDir(), "outside.json")
		wire, _ := EncodeOperational(operationalProjectID, archived())
		if err := os.WriteFile(outside, wire, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := testfs.Symlink(t, outside, operationalPath(state)); err != nil {
			t.Fatal(err)
		}
		if _, err := store.InspectOperational(context.Background(), operationalProjectID); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("inspect through symlink = %v", err)
		}
		if err := store.CommitOperational(context.Background(), operationalProjectID, revisionOf(wire), projectapp.DefaultOperationalState()); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("commit through symlink = %v", err)
		}
		if after, _ := os.ReadFile(outside); !bytes.Equal(after, wire) {
			t.Fatalf("symlink target changed")
		}
	})
	t.Run("record hard link", func(t *testing.T) {
		state, store := installedOperationalFixture(t)
		wire, _ := EncodeOperational(operationalProjectID, archived())
		if err := os.WriteFile(operationalPath(state), wire, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(operationalPath(state), filepath.Join(t.TempDir(), "alias.json")); err != nil {
			t.Skip("hard links unavailable")
		}
		if _, err := store.InspectOperational(context.Background(), operationalProjectID); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("inspect hard-linked record = %v", err)
		}
	})
	t.Run("installation directory symlink", func(t *testing.T) {
		state, store := installedOperationalFixture(t)
		directory := filepath.Join(state, "projects", operationalProjectID)
		moved := filepath.Join(privateTestRoot(t), operationalProjectID)
		if err := os.Rename(directory, moved); err != nil {
			t.Fatal(err)
		}
		if err := testfs.Symlink(t, moved, directory); err != nil {
			t.Fatal(err)
		}
		if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived()); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("commit through linked directory = %v", err)
		}
		if _, err := os.Lstat(filepath.Join(moved, operationalRecordName)); !os.IsNotExist(err) {
			t.Fatalf("record written through link: %v", err)
		}
	})
}

func TestOperationalRefusesInstallationDirectoryReplacementBeforeCommit(t *testing.T) {
	state, store := installedOperationalFixture(t)
	directory := filepath.Join(state, "projects", operationalProjectID)
	displaced := filepath.Join(state, "projects", "displaced")
	store.hooks.beforeCommit = func() error {
		if err := os.Rename(directory, displaced); err != nil {
			return err
		}
		return os.Mkdir(directory, 0o700)
	}
	err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived())
	if !errors.Is(err, ErrUnsafe) {
		t.Fatalf("commit after replacement = %v", err)
	}
	var publication *PublicationError
	if errors.As(err, &publication) && publication.Committed {
		t.Fatalf("replacement reported a commit")
	}
	for _, candidate := range []string{directory, displaced} {
		if _, err := os.Lstat(filepath.Join(candidate, operationalRecordName)); !os.IsNotExist(err) {
			t.Fatalf("record published into %s: %v", candidate, err)
		}
	}
}

func TestOperationalStateIsIsolatedBetweenStateRoots(t *testing.T) {
	first, firstStore := installedOperationalFixture(t)
	second, secondStore := installedOperationalFixture(t)
	if err := firstStore.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived("work-items")); err != nil {
		t.Fatal(err)
	}
	other, err := secondStore.InspectOperational(context.Background(), operationalProjectID)
	if err != nil || other.Exists || !other.State.Equal(projectapp.DefaultOperationalState()) {
		t.Fatalf("second machine observed %+v err=%v", other, err)
	}
	if _, err := os.Lstat(operationalPath(second)); !os.IsNotExist(err) {
		t.Fatalf("second machine has a record: %v", err)
	}
	if _, err := os.Lstat(operationalPath(first)); err != nil {
		t.Fatalf("first machine record missing: %v", err)
	}
}

func TestOperationalInventoryClassification(t *testing.T) {
	state, store := installedOperationalFixture(t)
	if err := store.CommitOperational(context.Background(), operationalProjectID, projectapp.OperationalRevisionAbsent, archived()); err != nil {
		t.Fatal(err)
	}
	relative := "projects/" + operationalProjectID + "/" + operationalRecordName
	if kind := kindOf(t, state, relative); kind != InventoryOperational || !kind.Supported() || !kind.V1Only() {
		t.Fatalf("operational record classified %s", kind)
	}
	newer := []byte(`{"formatVersion":2,"projectId":"` + operationalProjectID + `","projectStatus":"active","disabledIntegrations":[]}` + "\n")
	if err := os.WriteFile(operationalPath(state), newer, 0o600); err != nil {
		t.Fatal(err)
	}
	if kind := kindOf(t, state, relative); kind != InventoryNewer {
		t.Fatalf("newer operational record classified %s", kind)
	}
	if err := os.WriteFile(operationalPath(state), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if kind := kindOf(t, state, relative); kind != InventoryMalformed {
		t.Fatalf("malformed operational record classified %s", kind)
	}
}

func revisionOf(wire []byte) string {
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:])
}
