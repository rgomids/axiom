package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workitem"
)

type WorkItemStore struct{ root string }

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
	root, err := privateRoot(s.root)
	if err != nil {
		return err
	}
	defer root.Close()
	items, err := privateChild(root, "work-items")
	if err != nil {
		return err
	}
	defer items.Close()
	projectRoot, err := privateChild(items, link.ProjectID)
	if err != nil {
		return err
	}
	defer projectRoot.Close()
	lock, err := lockDirectory(projectRoot, true)
	if err != nil {
		return err
	}
	defer lock.Close()
	name := workItemName(link.RepositoryKey, link.Number)
	if current, readErr := readPrivateFile(projectRoot, name); readErr == nil {
		if _, decodeErr := decodeWorkItem(current); decodeErr != nil {
			return ErrUnsafe
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	temporary, err := temporaryName(".lingo-work-item-")
	if err != nil {
		return err
	}
	defer projectRoot.Remove(temporary)
	if err := writePrivateFile(projectRoot, temporary, wire); err != nil {
		return err
	}
	if err := verifyPreparedFile(projectRoot, temporary, wire); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return projectRoot.Rename(temporary, name)
}

func (s WorkItemStore) Load(ctx context.Context, projectID, repositoryKey string, number int) (workitem.Link, error) {
	if err := ctx.Err(); err != nil {
		return workitem.Link{}, err
	}
	probe := workitem.Link{ProjectID: projectID, RepositoryKey: repositoryKey, ProviderRepository: "x/y", Number: number, URL: fmt.Sprintf("https://github.com/x/y/issues/%d", number), State: "OPEN"}
	if !validWorkItemLink(probe) {
		return workitem.Link{}, ErrUnsafe
	}
	root, err := existingPrivateRoot(s.root)
	if err != nil {
		return workitem.Link{}, err
	}
	defer root.Close()
	items, err := existingPrivateChild(root, "work-items")
	if err != nil {
		return workitem.Link{}, err
	}
	defer items.Close()
	projectRoot, err := existingPrivateChild(items, projectID)
	if err != nil {
		return workitem.Link{}, err
	}
	defer projectRoot.Close()
	lock, err := lockDirectory(projectRoot, false)
	if err != nil {
		return workitem.Link{}, err
	}
	defer lock.Close()
	wire, err := readPrivateFile(projectRoot, workItemName(repositoryKey, number))
	if err != nil {
		return workitem.Link{}, err
	}
	return decodeWorkItem(wire)
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
