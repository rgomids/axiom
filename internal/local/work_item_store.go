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
	"regexp"
	"strconv"
	"strings"

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
	number, valid := validV1WorkItemLink(link)
	if !valid {
		return ErrUnsafe
	}
	wire, err := json.Marshal(workItemDTO{1, link.ProjectID, link.RepositoryKey, link.Resource, number, link.URL, link.State})
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
	if len(project.ValidateIdentity(projectID, "work-item")) != 0 || !project.ValidSlug(repositoryKey) || !validV1GitHubNumber(externalID) {
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
	link := workitem.Link{ProjectID: dto.ProjectID, RepositoryKey: dto.RepositoryKey, Provider: "github", Resource: dto.ProviderRepository, ExternalID: strconv.Itoa(dto.Number), URL: dto.URL, State: dto.State}
	if dto.FormatVersion != 1 {
		return workitem.Link{}, ErrUnsafe
	}
	if _, valid := validV1WorkItemLink(link); !valid {
		return workitem.Link{}, ErrUnsafe
	}
	return link, nil
}

func validV1WorkItemLink(link workitem.Link) (int, bool) {
	if len(project.ValidateIdentity(link.ProjectID, "work-item")) != 0 || !project.ValidSlug(link.RepositoryKey) || link.Provider != "github" || !validV1GitHubRepository(link.Resource) || !validV1GitHubNumber(link.ExternalID) {
		return 0, false
	}
	number, _ := strconv.Atoi(link.ExternalID)
	expectedURL := fmt.Sprintf("https://github.com/%s/issues/%d", link.Resource, number)
	return number, link.URL == expectedURL && (link.State == "OPEN" || link.State == "CLOSED")
}

func validV1GitHubNumber(value string) bool {
	number, err := strconv.Atoi(value)
	return err == nil && number > 0 && strconv.Itoa(number) == value
}

var v1GitHubSegment = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func validV1GitHubRepository(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && parts[0] != "." && parts[0] != ".." && parts[1] != "." && parts[1] != ".." && v1GitHubSegment.MatchString(parts[0]) && v1GitHubSegment.MatchString(parts[1])
}

func workItemName(repositoryKey, externalID string) string {
	return fmt.Sprintf("%s-%s.json", repositoryKey, externalID)
}
