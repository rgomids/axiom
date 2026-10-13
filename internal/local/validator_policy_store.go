package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/validatorpolicy"
)

type ValidatorPolicyStore struct {
	root  string
	hooks publicationHooks
}

func NewValidatorPolicyStore(root string) (ValidatorPolicyStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return ValidatorPolicyStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return ValidatorPolicyStore{}, err
	}
	return ValidatorPolicyStore{root: canonical}, nil
}
func (s ValidatorPolicyStore) open(create bool) (*os.Root, *os.Root, *os.Root, error) {
	openRoot, openChild := existingPrivateRoot, existingPrivateChild
	if create {
		openRoot, openChild = privateRoot, privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, nil, err
	}
	parent, err := openChild(root, "validator-policies")
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	version, err := openChild(parent, "v1")
	if err != nil {
		parent.Close()
		root.Close()
		return nil, nil, nil, err
	}
	return root, parent, version, nil
}
func (s ValidatorPolicyStore) Create(ctx context.Context, p validatorpolicy.Policy) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	wire, err := validatorpolicy.Encode(p)
	if err != nil {
		return "", err
	}
	id, _ := validatorpolicy.Digest(p)
	root, parent, version, err := s.open(true)
	if err != nil {
		return "", err
	}
	defer root.Close()
	defer parent.Close()
	defer version.Close()
	locks, err := lockRoots(true, root, parent, version)
	if err != nil {
		return "", err
	}
	defer closeFiles(locks)
	if err := publishFile(ctx, version, id+".json", nil, wire, true, s.hooks); err != nil {
		return "", err
	}
	return id, nil
}
func (s ValidatorPolicyStore) Load(ctx context.Context, id string) (validatorpolicy.Policy, error) {
	if err := ctx.Err(); err != nil {
		return validatorpolicy.Policy{}, err
	}
	if !hexDigestName.MatchString(id + ".json") {
		return validatorpolicy.Policy{}, ErrUnsafe
	}
	root, parent, version, err := s.open(false)
	if err != nil {
		return validatorpolicy.Policy{}, err
	}
	defer root.Close()
	defer parent.Close()
	defer version.Close()
	locks, err := lockRoots(false, root, parent, version)
	if err != nil {
		return validatorpolicy.Policy{}, err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(version, id+".json")
	if err != nil {
		return validatorpolicy.Policy{}, err
	}
	e, err := validatorpolicy.Decode(wire)
	if err != nil {
		return e, ErrUnsafe
	}
	actual, err := validatorpolicy.Digest(e)
	sum := sha256.Sum256(wire)
	if err != nil || actual != id || hex.EncodeToString(sum[:]) != id {
		return validatorpolicy.Policy{}, ErrUnsafe
	}
	return e, nil
}
