package local

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

type PolicyEditObserver struct {
	Installation InstallationStore
	Portable     PortableStore
}

func (o PolicyEditObserver) ObservePolicyEdit(ctx context.Context, selection projectapp.EditSelection, candidate project.Project) (projectapp.PolicyEditFacts, projectapp.EditFailure) {
	facts := projectapp.PolicyEditFacts{}
	if ctx.Err() != nil {
		return facts, projectapp.EditCancelled
	}
	state := selection.Portable.Project().State()
	if candidate.State().ID != state.ID || selection.PortableDestination != filepath.Join(o.Portable.root, state.Slug) || selection.LocalDestination != filepath.Join(o.Installation.root, "projects", state.ID) {
		return facts, projectapp.EditStateUnsafe
	}
	root, projects, target, err := openInstallationChain(o.Installation.root, state.ID)
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
	defer root.Close()
	defer projects.Close()
	defer target.Close()
	locks, err := lockRoots(false, root, projects, target)
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
	defer closeFiles(locks)
	current, err := readPrivateFile(target, installationRecord)
	if err != nil || !bytes.Equal(current, selection.LocalWire) {
		return facts, projectapp.EditStateUnsafe
	}
	facts.Operational, _, err = observeOperational(target, state.ID)
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
	facts.Dependencies.Operational = hexDigest([]byte("operational:" + facts.Operational.Revision))
	facts.Dependencies.Installation, err = policyFileToken(filepath.Join(selection.LocalDestination, installationRecord))
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
	runtimeStore, err := NewRuntimeProfileStore(o.Installation.root)
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
	runtime, err := runtimeStore.inspectLocked()
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
	facts.Dependencies.Runtime = runtime.Token
	err = o.Portable.withLock(state.Slug, false, false, func(portableRoot *os.Root) error {
		portableProject, err := openManifestProject(portableRoot, state.Slug)
		if err != nil {
			return err
		}
		defer portableProject.Close()
		wire, err := readPrivateFile(portableProject, manifestName)
		if err != nil || !bytes.Equal(wire, selection.Portable.Manifest()) {
			return ErrConflict
		}
		facts.Catalog, err = readWorkflowCatalog(portableProject)
		if err != nil {
			return err
		}
		facts.Dependencies.Workflow, err = policyWorkflowToken(facts.Catalog)
		if err != nil {
			return err
		}
		facts.Dependencies.Source, err = policySourceToken(selection.PortableDestination)
		return err
	})
	if err != nil {
		return projectapp.PolicyEditFacts{}, projectapp.EditStateUnsafe
	}
	workflow := projectapp.CheckPolicyWorkflow(selection.Portable.Project(), candidate, facts.Catalog, "")
	facts.Runtime = policyRuntimeReadiness(candidate, workflow, runtime)
	return facts, projectapp.EditOK
}

func policyWorkflowToken(catalog WorkflowCatalog) (string, error) {
	wire, err := json.Marshal(catalog)
	if err != nil {
		return "", ErrUnsafe
	}
	return hexDigest(wire), nil
}

func policySourceToken(source string) (string, error) {
	return policyFileToken(filepath.Join(source, manifestName))
}

func policyFileToken(target string) (string, error) {
	var facts strings.Builder
	current := volumeRoot(target)
	parts := pathComponents(target)
	for index, part := range parts {
		current = filepath.Join(current, part)
		var identity string
		var err error
		if index == len(parts)-1 {
			identity, err = FileIdentity(current)
		} else {
			identity, err = DirectoryIdentity(current)
		}
		if err != nil {
			return "", err
		}
		facts.WriteString(current)
		facts.WriteByte(0)
		facts.WriteString(identity)
		facts.WriteByte(0)
	}
	return hexDigest([]byte(facts.String())), nil
}
