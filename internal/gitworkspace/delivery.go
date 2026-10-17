package gitworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os/exec"
	"strings"

	"github.com/rgomids/axiom/internal/workflow"
)

type DeliveryGit struct {
	manager        *Manager
	allowFile      bool
	authentication []string
}

func NewDeliveryGit(ctx context.Context, repository string) (DeliveryGit, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return DeliveryGit{}, err
	}
	repository, err = cleanExistingDirectory(repository)
	if err != nil {
		return DeliveryGit{}, err
	}
	m := &Manager{git: git, repository: repository}
	if err = m.rejectExecutableConfiguration(ctx, repository); err != nil {
		return DeliveryGit{}, err
	}
	return DeliveryGit{manager: m}, nil
}
func (g DeliveryGit) ObserveDeliveryGit(ctx context.Context, p workflow.DeliveryPreview) (workflow.DeliveryObservation, error) {
	m, t := g.manager, p.Target
	o := workflow.DeliveryObservation{Candidates: []workflow.DeliveryPR{}}
	if err := g.safeRemote(ctx, t); err != nil {
		return o, err
	}
	var err error
	o.BaseCommit, err = m.revision(ctx, m.repository, t.BaseRef)
	if err != nil {
		return o, err
	}
	o.HeadCommit, err = m.revision(ctx, m.repository, t.HeadRef)
	if err != nil {
		return o, err
	}
	o.Tree, err = m.tree(ctx, m.repository, t.HeadCommit)
	if err != nil {
		return o, err
	}
	patch, err := m.diffTrees(ctx, t.BaseCommit, t.HeadCommit)
	if err != nil {
		return o, err
	}
	sum := sha256.Sum256(patch)
	o.DiffDigest = hex.EncodeToString(sum[:])
	output, err := m.gitOutput(ctx, m.repository, nil, g.authentication, "-c", "protocol.allow=never", "-c", "protocol.https.allow=always", "-c", "protocol.file.allow=always", "ls-remote", "--refs", t.Remote, t.BaseRef, t.HeadRef)
	if err != nil {
		return o, ErrWorkspaceInvalid
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !validObjectID(fields[0]) {
			return o, ErrWorkspaceInvalid
		}
		switch fields[1] {
		case t.BaseRef:
			o.RemoteBase = fields[0]
		case t.HeadRef:
			o.RemoteHead = fields[0]
		default:
			return o, ErrWorkspaceInvalid
		}
	}
	o.Conclusive = true
	return o, nil
}
func (g DeliveryGit) safeRemote(ctx context.Context, t workflow.DeliveryTarget) error {
	m := g.manager
	if err := m.rejectExecutableConfiguration(ctx, m.repository); err != nil {
		return err
	}
	if t.Remote != "https://github.com/"+t.Repository+".git" && !(g.allowFile && strings.HasPrefix(t.Remote, "/")) {
		return ErrInvalidConfiguration
	}
	output, err := m.runGit(ctx, m.repository, nil, nil, "config", "--list", "--includes", "-z")
	if err != nil {
		return err
	}
	for _, entry := range splitNUL(output) {
		key, value, _ := strings.Cut(entry, "\n")
		key = strings.ToLower(key)
		if value != "" && (strings.HasPrefix(key, "url.") || strings.HasPrefix(key, "http.") || strings.HasPrefix(key, "credential.") || strings.HasPrefix(key, "core.ssh")) {
			return ErrExecutableGitConfiguration
		}
	}
	return nil
}
func (g DeliveryGit) PushDelivery(ctx context.Context, p workflow.DeliveryPreview) error {
	if p.Observation.RemoteHead != "" && p.Observation.RemoteHead != p.Target.HeadCommit {
		return ErrWorkspaceInvalid
	}
	if err := g.safeRemote(ctx, p.Target); err != nil {
		return err
	}
	_, err := g.manager.gitOutput(ctx, g.manager.repository, nil, g.authentication, "-c", "protocol.allow=never", "-c", "protocol.file.allow="+map[bool]string{true: "always", false: "never"}[g.allowFile], "-c", "protocol.https.allow=always", "push", "--porcelain", "--force-with-lease="+p.Target.HeadRef+":"+p.Observation.RemoteHead, "--", p.Target.Remote, p.Target.HeadCommit+":"+p.Target.HeadRef)
	if err != nil {
		return ErrWorkspaceInvalid
	}
	return nil
}
func (g DeliveryGit) WithGitHubCredential(token string) (DeliveryGit, error) {
	if token == "" || len(token) > 16384 || strings.ContainsAny(token, " \r\n\x00") {
		return g, ErrInvalidConfiguration
	}
	value := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	g.authentication = []string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.https://github.com/.extraheader", "GIT_CONFIG_VALUE_0=AUTHORIZATION: basic " + value}
	return g, nil
}
