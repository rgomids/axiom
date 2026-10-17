package gitworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/rgomids/axiom/internal/workflow"
)

type DeliveryCommitPreview struct {
	PacketDigest string                  `json:"packetDigest"`
	Target       workflow.DeliveryTarget `json:"target"`
	PreviousHead string                  `json:"previousHead"`
	Date         string                  `json:"date"`
}

func (p DeliveryCommitPreview) Digest() string {
	wire, _ := json.Marshal(p)
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}
func (g DeliveryGit) PreviewDeliveryCommit(ctx context.Context, p workflow.DeliveryPacket, t workflow.DeliveryTarget) (DeliveryCommitPreview, error) {
	result := DeliveryCommitPreview{Target: t, Date: "2000-01-01T00:00:00Z"}
	ref, err := workflow.DeliveryPacketDigest(p)
	if err != nil || !p.Ready {
		return result, workflow.ErrDelivery
	}
	result.PacketDigest = ref
	var tree string
	for _, s := range p.Stages {
		if s.ResultTree != "" {
			tree = s.ResultTree
		}
	}
	if tree == "" || t.Tree != tree || !validObjectID(t.BaseCommit) || t.HeadRef == t.BaseRef || !strings.HasPrefix(t.HeadRef, "refs/heads/") {
		return result, workflow.ErrDelivery
	}
	m := g.manager
	if _, err = m.gitOutput(ctx, m.repository, nil, nil, "check-ref-format", t.HeadRef); err != nil {
		return result, err
	}
	if _, err = m.revision(ctx, m.repository, t.BaseCommit); err != nil {
		return result, err
	}
	output, err := m.gitOutput(ctx, m.repository, nil, nil, "for-each-ref", "--format=%(objectname)", "--", t.HeadRef)
	if err != nil {
		return result, err
	}
	result.PreviousHead = strings.TrimSpace(string(output))
	if result.PreviousHead != "" && !validObjectID(result.PreviousHead) {
		return result, workflow.ErrDelivery
	}
	return result, nil
}
func (g DeliveryGit) ApplyDeliveryCommit(ctx context.Context, p workflow.DeliveryPacket, preview DeliveryCommitPreview, digest string, local bool) (workflow.DeliveryTarget, error) {
	current, err := g.PreviewDeliveryCommit(ctx, p, preview.Target)
	if err != nil || !local || current.Digest() != digest || preview.Digest() != digest {
		return preview.Target, workflow.ErrDelivery
	}
	m, t := g.manager, preview.Target
	commit, err := m.gitObjectWithEnv(ctx, m.repository, []string{"GIT_AUTHOR_NAME=Axiom", "GIT_AUTHOR_EMAIL=axiom@example.invalid", "GIT_COMMITTER_NAME=Axiom", "GIT_COMMITTER_EMAIL=axiom@example.invalid", "GIT_AUTHOR_DATE=" + preview.Date, "GIT_COMMITTER_DATE=" + preview.Date}, "commit-tree", t.Tree, "-p", t.BaseCommit, "-m", "Axiom delivery "+preview.PacketDigest)
	if err != nil {
		return t, err
	}
	previous := preview.PreviousHead
	if previous == "" {
		previous = strings.Repeat("0", len(commit))
	}
	if _, err = m.gitOutput(ctx, m.repository, nil, nil, "update-ref", t.HeadRef, commit, previous); err != nil {
		return t, err
	}
	confirmed, err := m.revision(ctx, m.repository, t.HeadRef)
	if err != nil || confirmed != commit {
		return t, workflow.ErrRecoveryRequired
	}
	t.HeadCommit = commit
	return t, nil
}
