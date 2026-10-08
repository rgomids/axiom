package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/projectapp"
)

type projectContextWire struct {
	Version   int    `json:"formatVersion"`
	ProjectID string `json:"projectId"`
}

func contextFile(session string) string {
	if session == "" {
		return "default.json"
	}
	digest := sha256.Sum256([]byte(session))
	return "session-" + hex.EncodeToString(digest[:]) + ".json"
}

func decodeProjectContext(wire []byte) (string, error) {
	value, issues := parseRecord(wire)
	object, ok := value.(map[string]any)
	if len(issues) != 0 || !ok || len(object) != 2 {
		return "", ErrUnsafe
	}
	version, ok := object["formatVersion"].(json.Number)
	if !ok || version.String() != "1" {
		return "", ErrUnsafe
	}
	id, ok := object["projectId"].(string)
	if !ok || id != "" && !validUUID(id) {
		return "", ErrUnsafe
	}
	return id, nil
}

func (s InstallationStore) contextRoots(create bool) (*os.Root, *os.Root, error) {
	openRoot, openChild := existingPrivateRoot, existingPrivateChild
	if create {
		openRoot, openChild = privateRoot, privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, err
	}
	child, err := openChild(root, "project-context")
	if err != nil {
		root.Close()
		return nil, nil, err
	}
	defer child.Close()
	version, err := openChild(child, "v1")
	if err != nil {
		root.Close()
		return nil, nil, err
	}
	return root, version, nil
}

func (s InstallationStore) ReadProjectContext(ctx context.Context, session string) (string, error) {
	if !projectapp.ValidProjectSession(session) {
		return "", ErrUnsafe
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root, child, err := s.contextRoots(false)
	if errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer root.Close()
	defer child.Close()
	locks, err := lockRoots(false, root, child)
	if err != nil {
		return "", err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(child, contextFile(session))
	if errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return decodeProjectContext(wire)
}

func (s InstallationStore) WriteProjectContext(ctx context.Context, session, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !projectapp.ValidProjectSession(session) || id != "" && !validUUID(id) {
		return ErrUnsafe
	}
	root, child, err := s.contextRoots(true)
	if err != nil {
		return err
	}
	defer root.Close()
	defer child.Close()
	locks, err := lockRoots(true, root, child)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	name := contextFile(session)
	previous, err := readPublishedFile(child, name)
	create := errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist)
	if err != nil && !create {
		return err
	}
	if !create {
		if _, err := decodeProjectContext(previous); err != nil {
			return err
		}
	}
	wire, _ := json.Marshal(projectContextWire{Version: 1, ProjectID: id})
	return publishFile(ctx, child, name, previous, append(wire, '\n'), create, publicationHooks{})
}

func (s InstallationStore) ResolveContextProject(ctx context.Context, selector string) (string, string) {
	result := s.Resolve(ctx, selector)
	if result.Status != ResolutionFound {
		return "", result.Category
	}
	return result.Project.ID, ""
}
