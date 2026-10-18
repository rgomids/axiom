package local

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
)

func validPolicyPublication(request projectapp.EditPublicationRequest) bool {
	if request.PolicyDependencies == nil || len(request.PortableNext) == 0 {
		return false
	}
	before, issues := projectapp.ReadSnapshot(manifest.Codec{}, request.PortableExpected, nil)
	if len(issues) != 0 {
		return false
	}
	after, issues := projectapp.ReadSnapshot(manifest.Codec{}, request.PortableNext, nil)
	if len(issues) != 0 || before.Project().State().SchemaVersion < 2 || before.Project().State().SchemaVersion > 3 {
		return false
	}
	a, b := before.Project().State(), after.Project().State()
	b.Runtimes, b.ModelProfiles, b.RuntimePreferences = a.Runtimes, a.ModelProfiles, a.RuntimePreferences
	if !reflect.DeepEqual(a, b) || a.ID != request.ProjectID || a.Slug != request.Slug {
		return false
	}
	prior, _, localIssues := DecodeObservedRecord(request.LocalExpected, true)
	if len(localIssues) != 0 {
		return false
	}
	next, _, localIssues := DecodeObservedRecord(request.LocalNext, true)
	if len(localIssues) != 0 {
		return false
	}
	oldLocal, newLocal := prior.State(), next.State()
	if oldLocal.ProjectID != request.ProjectID || oldLocal.ObservedSlug != request.Slug || oldLocal.SourceLocation != request.PortableDestination || oldLocal.PortableRevision != before.Revision() || !reflect.DeepEqual(oldLocal.ArtifactDigests, before.Digests()) {
		return false
	}
	if newLocal.PortableRevision != after.Revision() || !reflect.DeepEqual(newLocal.ArtifactDigests, after.Digests()) {
		return false
	}
	newLocal.PortableRevision, newLocal.ArtifactDigests = oldLocal.PortableRevision, oldLocal.ArtifactDigests
	return reflect.DeepEqual(oldLocal, newLocal)
}

func (p ProjectEditPublisher) policyGuard(request projectapp.EditPublicationRequest, target *os.Root) func(*os.Root) error {
	expected := *request.PolicyDependencies
	return func(portableProject *os.Root) error {
		current, err := readPrivateFile(target, installationRecord)
		if err != nil || !bytes.Equal(current, request.LocalExpected) {
			return ErrConflict
		}
		installation, err := policyFileToken(filepath.Join(request.LocalDestination, installationRecord))
		if err != nil || installation != expected.Installation {
			return ErrConflict
		}
		operational, _, err := observeOperationalRecord(target, request.ProjectID)
		if err != nil || hexDigest([]byte("operational:"+operational.Revision)) != expected.Operational {
			return ErrConflict
		}
		runtimeStore, err := NewRuntimeProfileStore(p.Installation.root)
		if err != nil {
			return err
		}
		runtime, err := runtimeStore.inspectLocked()
		if err != nil || runtime.Token != expected.Runtime {
			return ErrConflict
		}
		catalog, err := readWorkflowCatalog(portableProject)
		if err != nil {
			return err
		}
		workflowToken, err := policyWorkflowToken(catalog)
		if err != nil || workflowToken != expected.Workflow {
			return ErrConflict
		}
		source, err := policySourceToken(request.PortableDestination)
		if err != nil || source != expected.Source {
			return ErrConflict
		}
		before, issues := manifest.Decode(request.PortableExpected)
		if len(issues) != 0 {
			return ErrUnsafe
		}
		after, issues := manifest.Decode(request.PortableNext)
		if len(issues) != 0 || !projectapp.CheckPolicyWorkflow(before, after, catalog, "").Compatible {
			return ErrUnsafe
		}
		return nil
	}
}
