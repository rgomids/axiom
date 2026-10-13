package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/testfs"
	"github.com/rgomids/axiom/internal/validatorpolicy"
)

func storePolicy() validatorpolicy.Policy {
	d := validatorpolicy.Definition{FormatVersion: 1, ProjectID: "project-1", Commands: []validatorpolicy.Command{{Reference: "tests", Argv: []string{testfs.Path("/synthetic/validator"), "test"}, Env: []string{"LC_ALL=C"}, OutputMax: 4096}}}
	preview, _ := validatorpolicy.PreviewDigest(d)
	return validatorpolicy.Policy{Definition: d, Review: validatorpolicy.Review{PreviewDigest: preview, ActorKind: "human-operator", Actor: "operator", ReviewedAt: time.Unix(10, 0).UTC(), AuthorityRef: strings.Repeat("a", 64)}}
}
func TestValidatorPolicyReviewStoreReference(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewValidatorPolicyStore(root)
	if err != nil {
		t.Fatal(err)
	}
	service := stageexecution.ValidatorPolicyService{Store: store, Now: func() time.Time { return time.Unix(10, 0) }}
	p := storePolicy()
	preview, err := service.Preview(p.Definition)
	if err != nil {
		t.Fatal(err)
	}
	request := stageexecution.ValidatorPolicyApply{Definition: p.Definition, PreviewDigest: preview, AuthorizeLocal: true, Source: "human-operator", Actor: "operator", AuthorityRef: p.Review.AuthorityRef}
	for _, source := range []string{"agent-record", "plan-document", ""} {
		r := request
		r.Source = source
		if _, e := service.Apply(ctx, r); !errors.Is(e, validatorpolicy.ErrAuthority) {
			t.Fatal(source, e)
		}
	}
	r := request
	r.AuthorizeLocal = false
	if _, e := service.Apply(ctx, r); !errors.Is(e, validatorpolicy.ErrAuthority) {
		t.Fatal(e)
	}
	r = request
	r.Definition.Commands = append([]validatorpolicy.Command(nil), r.Definition.Commands...)
	r.Definition.Commands[0].Argv = []string{testfs.Path("/synthetic/changed")}
	if _, e := service.Apply(ctx, r); !errors.Is(e, validatorpolicy.ErrDigest) {
		t.Fatal("unreviewed change", e)
	}
	id, err := service.Apply(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := service.Commands(ctx, "project-1", id)
	if err != nil || commands[0].Argv[0] != testfs.Path("/synthetic/validator") {
		t.Fatal(commands, err)
	}
	commands[0].Argv[0] = "changed"
	if loaded.Commands[0].Argv[0] != testfs.Path("/synthetic/validator") {
		t.Fatal("alias")
	}
	if _, err := service.Commands(ctx, "other-project", id); !errors.Is(err, validatorpolicy.ErrAuthority) {
		t.Fatal(err)
	}
	if _, err := validatorpolicy.Commands(loaded, strings.Repeat("b", 64)); !errors.Is(err, validatorpolicy.ErrDigest) {
		t.Fatal(err)
	}
	inventory, err := InspectStateInventory(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Entries) != 1 || inventory.Entries[0].Kind != InventoryValidatorPolicy {
		t.Fatal(inventory)
	}
	loaded.Commands[0].Argv[0] = testfs.Path("/synthetic/tampered")
	wire, _ := validatorpolicy.Encode(loaded)
	if len(wire) == 0 {
		wire = []byte(`{}`)
	}
	if err := os.WriteFile(filepath.Join(root, "validator-policies", "v1", id+".json"), wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(ctx, id); !errors.Is(err, ErrUnsafe) {
		t.Fatal("tamper", err)
	}
	if _, err := store.Load(ctx, "../escape"); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
}
func TestValidatorPolicyStoreCanonicalAndSymlink(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, _ := NewValidatorPolicyStore(root)
	p := storePolicy()
	id, err := store.Create(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, "validator-policies", "v1", id+".json")
	wire, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, append([]byte(" "), wire...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), id); !errors.Is(err, ErrUnsafe) {
		t.Fatal("byte drift", err)
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, name); err != nil {
		t.Skip(err)
	}
	if _, err := store.Load(context.Background(), id); err == nil {
		t.Fatal("followed symlink")
	}
}

func TestValidatorPolicyPublicationRecovery(t *testing.T) {
	for _, stage := range []FaultStage{FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "state")
			store, err := NewValidatorPolicyStore(root)
			if err != nil {
				t.Fatal(err)
			}
			store.hooks.fault = func(at FaultStage) error {
				if at == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			_, err = store.Create(context.Background(), storePolicy())
			if !errors.Is(err, ErrSimulatedInterruption) {
				t.Fatal(err)
			}
			report, err := InspectRecovery(context.Background(), RecoveryRoots{State: root})
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Plans) != 1 || report.Plans[0].Directory != "validator-policies/v1" {
				t.Fatal(report)
			}
			inventory, err := InspectStateInventory(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, entry := range inventory.Entries {
				if entry.Kind == InventoryRecovery {
					found = true
				}
			}
			if !found {
				t.Fatal("missing recovery inventory")
			}
		})
	}
}
