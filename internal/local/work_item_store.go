package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workitem"
)

type WorkItemStore struct {
	root  string
	hooks publicationHooks
}

type workItemDTO struct {
	FormatVersion      int    `json:"formatVersion"`
	ProjectID          string `json:"projectId"`
	RepositoryKey      string `json:"repositoryKey"`
	ProviderRepository string `json:"providerRepository"`
	Number             int    `json:"number"`
	URL                string `json:"url"`
	State              string `json:"state"`
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
	wire, err := json.Marshal(workItemDTO{1, link.ProjectID, link.RepositoryKey, link.ProviderRepository, link.Number, link.URL, link.State})
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
	name := workItemName(link.RepositoryKey, link.Number)
	create := link.Revision == ([32]byte{})
	var expected []byte
	if !create {
		expected, err = readPublishedFile(projectRoot, name)
		if err != nil {
			return err
		}
		if sha256.Sum256(expected) != link.Revision {
			return ErrConflict
		}
	}
	err = publishFile(ctx, projectRoot, name, expected, wire, create, s.hooks)
	return workItemStoreError(err)
}

func (s WorkItemStore) Load(ctx context.Context, projectID, repositoryKey string, number int) (workitem.Link, error) {
	if err := ctx.Err(); err != nil {
		return workitem.Link{}, err
	}
	probe := workitem.Link{ProjectID: projectID, RepositoryKey: repositoryKey, ProviderRepository: "x/y", Number: number, URL: fmt.Sprintf("https://github.com/x/y/issues/%d", number), State: "OPEN"}
	if !validWorkItemLink(probe) {
		return workitem.Link{}, ErrUnsafe
	}
	root, items, projectRoot, err := s.openProject(projectID, false)
	if err != nil {
		return workitem.Link{}, err
	}
	defer root.Close()
	defer items.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(false, root, items, projectRoot)
	if err != nil {
		return workitem.Link{}, err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(projectRoot, workItemName(repositoryKey, number))
	if err != nil {
		return workitem.Link{}, workItemStoreError(err)
	}
	link, err := decodeWorkItem(wire)
	if err == nil {
		link.Revision = sha256.Sum256(wire)
	}
	return link, err
}

func workItemStoreError(err error) error {
	if errors.Is(err, ErrRecoveryRequired) {
		return errors.Join(err, workitem.ErrRecoveryRequired)
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
	link := workitem.Link{ProjectID: dto.ProjectID, RepositoryKey: dto.RepositoryKey, ProviderRepository: dto.ProviderRepository, Number: dto.Number, URL: dto.URL, State: dto.State}
	if dto.FormatVersion != 1 || !validWorkItemLink(link) {
		return workitem.Link{}, ErrUnsafe
	}
	return link, nil
}

func validWorkItemLink(link workitem.Link) bool {
	if len(project.ValidateIdentity(link.ProjectID, "work-item")) != 0 || !project.ValidSlug(link.RepositoryKey) || !workitem.ValidGitHubRepository(link.ProviderRepository) || link.Number <= 0 {
		return false
	}
	expected := fmt.Sprintf("https://github.com/%s/issues/%d", link.ProviderRepository, link.Number)
	return link.URL == expected && (link.State == "OPEN" || link.State == "CLOSED")
}

func workItemName(repositoryKey string, number int) string {
	return fmt.Sprintf("%s-%d.json", repositoryKey, number)
}
