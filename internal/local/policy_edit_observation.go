package local

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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
	source, sourceErr := trustedCanonical(selection.PortableDestination)
	installation, installationErr := trustedCanonical(selection.LocalDestination)
	if sourceErr != nil || installationErr != nil || candidate.State().ID != state.ID || source != filepath.Join(o.Portable.root, state.Slug) || installation != filepath.Join(o.Installation.root, "projects", state.ID) {
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
	facts.Dependencies.Operational, err = policyOperationalToken(selection.LocalDestination, facts.Operational.Revision)
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
	facts.Dependencies.Locations, err = policyLocationToken(selection.PortableDestination, selection.LocalDestination)
	if err != nil {
		return facts, projectapp.EditStateUnsafe
	}
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
	metadata, token, err := observePolicyBindings(ctx, candidate, selection.Local)
	if err != nil {
		return projectapp.PolicyEditFacts{}, projectapp.EditStateUnsafe
	}
	facts.Repositories, facts.Documentation = metadata.Repositories, metadata.Documentation
	facts.Dependencies.Readiness = token
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
	root, err := existingPrivateRoot(source)
	if err != nil {
		return "", err
	}
	defer root.Close()
	catalog, err := readWorkflowCatalog(root)
	if err != nil {
		return "", err
	}
	names := []string{manifestName}
	if catalog.Wire != nil {
		names = append(names, "workflows/index.json")
		for name := range catalog.Documents {
			names = append(names, "workflows/"+name)
		}
	}
	sort.Strings(names)
	var facts strings.Builder
	for _, name := range names {
		token, err := policyFileToken(filepath.Join(source, name))
		if err != nil {
			return "", err
		}
		facts.WriteString(name + "\x00" + token + "\x00")
	}
	return hexDigest([]byte(facts.String())), nil
}

func policyFileToken(target string) (string, error) { return policyObjectToken(target, false) }

func policyObjectToken(target string, directory bool) (string, error) {
	if !filepath.IsAbs(target) {
		return "", ErrUnsafe
	}
	canonical, err := trustedCanonical(target)
	if err != nil {
		return "", err
	}
	var facts strings.Builder
	current := volumeRoot(canonical)
	parts := pathComponents(canonical)
	for index, part := range parts {
		current = filepath.Join(current, part)
		var identity string
		var err error
		if index == len(parts)-1 && !directory {
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

func policyOperationalToken(directory, revision string) (string, error) {
	path := filepath.Join(directory, operationalRecordName)
	identity := "absent"
	if _, err := os.Lstat(path); err == nil {
		identity, err = policyFileToken(path)
		if err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return hexDigest([]byte("operational:" + revision + "\x00" + identity)), nil
}

type policyBindingFacts struct {
	Repositories  map[string]bool
	Documentation map[string]projectapp.DocumentationStatus
}

func observePolicyBindings(ctx context.Context, p project.Project, record projectapp.LocalRecordState) (policyBindingFacts, string, error) {
	facts := policyBindingFacts{Repositories: map[string]bool{}, Documentation: map[string]projectapp.DocumentationStatus{}}
	paths := map[string]string{}
	for _, binding := range record.Repositories {
		available := (RepositoryProbe{}).RepositoryAvailable(ctx, binding)
		facts.Repositories[binding.RepositoryKey] = available
		if available {
			paths[binding.RepositoryKey] = binding.ExplicitPath
		}
	}
	bindings := map[string]*projectapp.DocumentationBinding{}
	for i := range record.Documentation {
		bindings[record.Documentation[i].SourceKey] = &record.Documentation[i]
	}
	sources, _ := p.State().DocumentationSources.Value()
	for _, source := range sources {
		request := projectapp.DocumentationRequest{Source: source, Binding: bindings[source.Key]}
		if ref, ok := source.RepositoryRef.Value(); ok {
			request.RepositoryPath = paths[ref]
		}
		facts.Documentation[source.Key] = (DocumentationResolver{}).ResolveDocumentation(ctx, request)
	}
	wire, err := json.Marshal(facts)
	if ctx.Err() != nil {
		return facts, "", ctx.Err()
	}
	return facts, hexDigest(wire), err
}

func policyLocationToken(source, installation string) (string, error) {
	a, err := policyObjectToken(source, true)
	if err != nil {
		return "", err
	}
	b, err := policyObjectToken(installation, true)
	if err != nil {
		return "", err
	}
	return hexDigest([]byte(a + "\x00" + b)), nil
}
