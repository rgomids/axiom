package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workitem"
)

type WorkItemStore struct {
	root  string
	hooks publicationHooks
}

type workItemDTO struct {
	FormatVersion int    `json:"formatVersion"`
	ProjectID     string `json:"projectId"`
	RepositoryKey string `json:"repositoryKey"`
	Provider      string `json:"provider"`
	Resource      string `json:"resource"`
	ExternalID    string `json:"externalId"`
	URL           string `json:"url"`
	State         string `json:"state"`
}

func NewWorkItemStore(root string) (WorkItemStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return WorkItemStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return WorkItemStore{}, err
	}
	return WorkItemStore{root: canonical}, nil
}

func (s WorkItemStore) Save(ctx context.Context, link workitem.Link) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validWorkItemLink(link) {
		return ErrUnsafe
	}
	wire, err := json.Marshal(workItemDTO{1, link.ProjectID, link.RepositoryKey, link.Provider, link.Resource, link.ExternalID, link.URL, link.State})
	if err != nil {
		return err
	}
	wire = append(wire, '\n')
	root, items, projectRoot, err := s.openProject(link.ProjectID, true)
	if err != nil {
		return err
	}
	defer root.Close()
	defer items.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(true, root, items, projectRoot)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	name := workItemName(link.RepositoryKey, link.ExternalID)
	create := link.Revision == ([32]byte{})
	var expected []byte
	if !create {
		expected, err = readPublishedFile(projectRoot, name)
		if err != nil {
			return err
		}
		if sha256.Sum256(expected) != link.Revision {
			return workItemStoreError(ErrConflict)
		}
	}
	err = publishFile(ctx, projectRoot, name, expected, wire, create, s.hooks)
	return workItemStoreError(err)
}

func (s WorkItemStore) Load(ctx context.Context, projectID, repositoryKey, externalID string) (workitem.Link, error) {
	if err := ctx.Err(); err != nil {
		return workitem.Link{}, err
	}
	probe := workitem.Link{ProjectID: projectID, RepositoryKey: repositoryKey, Provider: "provider", Resource: "resource", ExternalID: externalID, URL: "https://example.invalid/item", State: "OPEN"}
	if !validWorkItemLink(probe) {
		return workitem.Link{}, ErrUnsafe
	}
	root, items, projectRoot, err := s.openProject(projectID, false)
	if err != nil {
		return workitem.Link{}, workItemStoreError(err)
	}
	defer root.Close()
	defer items.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(false, root, items, projectRoot)
	if err != nil {
		return workitem.Link{}, workItemStoreError(err)
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(projectRoot, workItemName(repositoryKey, externalID))
	if err != nil {
		return workitem.Link{}, workItemStoreError(err)
	}
	link, err := decodeWorkItem(wire)
	if err != nil {
		return workitem.Link{}, err
	}
	if link.ProjectID != projectID || link.RepositoryKey != repositoryKey || link.ExternalID != externalID {
		return workitem.Link{}, ErrUnsafe
	}
	link.Revision = sha256.Sum256(wire)
	return link, nil
}

func workItemStoreError(err error) error {
	if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
		return errors.Join(err, workitem.ErrNotFound)
	}
	if errors.Is(err, ErrRecoveryRequired) {
		return errors.Join(err, workitem.ErrRecoveryRequired)
	}
	if errors.Is(err, ErrConflict) {
		return errors.Join(err, workitem.ErrConflict)
	}
	return err
}

func (s WorkItemStore) openProject(projectID string, create bool) (*os.Root, *os.Root, *os.Root, error) {
	openRoot := existingPrivateRoot
	openChild := existingPrivateChild
	if create {
		openRoot = privateRoot
		openChild = privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, nil, err
	}
	items, err := openChild(root, "work-items")
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	projectRoot, err := openChild(items, projectID)
	if err != nil {
		items.Close()
		root.Close()
		return nil, nil, nil, err
	}
	return root, items, projectRoot, nil
}

func decodeWorkItem(wire []byte) (workitem.Link, error) {
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(wire), MaxRecordBytes+1))
	decoder.DisallowUnknownFields()
	var dto workItemDTO
	if err := decoder.Decode(&dto); err != nil {
		return workitem.Link{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return workitem.Link{}, ErrUnsafe
	}
	link := workitem.Link{ProjectID: dto.ProjectID, RepositoryKey: dto.RepositoryKey, Provider: dto.Provider, Resource: dto.Resource, ExternalID: dto.ExternalID, URL: dto.URL, State: dto.State}
	if dto.FormatVersion != 1 || !validWorkItemLink(link) {
		return workitem.Link{}, ErrUnsafe
	}
	return link, nil
}

func validWorkItemLink(link workitem.Link) bool {
	if len(project.ValidateIdentity(link.ProjectID, "work-item")) != 0 || !project.ValidSlug(link.RepositoryKey) || !workitem.ValidProviderID(link.Provider) || !validProviderResource(link.Resource) || !workitem.ValidExternalID(link.ExternalID) {
		return false
	}
	parsed, err := url.Parse(link.URL)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" && (link.State == "OPEN" || link.State == "CLOSED")
}

func validProviderResource(value string) bool {
	return value != "" && len(value) <= 256 && utf8.ValidString(value) && !strings.ContainsAny(value, "\\\x00\n\r")
}

func workItemName(repositoryKey, externalID string) string {
	return fmt.Sprintf("%s-%s.json", repositoryKey, externalID)
}
