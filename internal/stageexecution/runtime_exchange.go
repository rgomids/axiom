package stageexecution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
)

// The message areas live inside the attempt's private inbox. '@' is not a
// valid token character, so neither name can collide with a declared output.
// The child writes each candidate as <send>/<id>.json (atomically, e.g. by
// rename) and reads delivered messages from <receive>/<id>.json. Both Codex
// and Claude children receive the same two paths in their stdin packet.
const (
	sendArea                 = "@send"
	receiveArea              = "@recv"
	maxMessageCandidateBytes = workflow.MaxMessageBytes + 1024
)

// deliveredMessage is the recipient's view. The sender is the agent recorded
// by the engine from the sending attempt, never a value the sender supplied.
type deliveredMessage struct {
	ID      string               `json:"id"`
	Sender  string               `json:"sender"`
	Kind    workflow.MessageKind `json:"kind"`
	ReplyTo string               `json:"replyTo,omitempty"`
	Content string               `json:"content"`
}

func (l RuntimeLauncher) messageArea(r LaunchRequest, name string) (*os.Root, error) {
	inbox, err := l.inbox(r)
	if err != nil || !runtimeOutputSupported {
		return nil, ErrRuntimeOutput
	}
	path := filepath.Join(inbox, name)
	if !cleanRuntimePath(path) {
		return nil, ErrRuntimeOutput
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return nil, ErrRuntimeOutput
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, ErrRuntimeOutput
	}
	if opened, err := root.Stat("."); err != nil || !os.SameFile(info, opened) {
		root.Close()
		return nil, ErrRuntimeOutput
	}
	return root, nil
}

// Poll reads bounded regular candidate files from this attempt's send area.
// Unsafe, oversized, malformed or misnamed files are skipped and never
// reach ingress; an unsafe area or entry flood refuses the whole poll.
func (l RuntimeLauncher) Poll(ctx context.Context, r LaunchRequest) ([]workflow.MessageCandidate, error) {
	root, err := l.messageArea(r, sendArea)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dir, err := root.Open(".")
	if err != nil {
		return nil, ErrRuntimeOutput
	}
	names, err := dir.Readdirnames(workflow.MaxExecutionMessages + 1)
	dir.Close()
	if (err != nil && !errors.Is(err, io.EOF)) || len(names) > workflow.MaxExecutionMessages {
		return nil, ErrRuntimeOutput
	}
	sort.Strings(names)
	var candidates []workflow.MessageCandidate
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		id, ok := strings.CutSuffix(name, ".json")
		if !ok || !workflow.ValidToken(id) {
			continue
		}
		data, ok := readBoundedRegular(root, name, maxMessageCandidateBytes)
		if !ok {
			continue
		}
		var c workflow.MessageCandidate
		if strictjson.Decode(data, &c, maxMessageCandidateBytes) != nil || c.ID != id {
			continue
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

// Deliver publishes one accepted message into this attempt's receive area by
// exclusive create and rename. An identical existing delivery is idempotent;
// a different one at the same ID refuses.
func (l RuntimeLauncher) Deliver(ctx context.Context, r LaunchRequest, m workflow.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !workflow.ValidToken(m.ID) || m.Recipient != r.Ref.AgentID {
		return ErrRuntimeOutput
	}
	wire, err := json.Marshal(deliveredMessage{ID: m.ID, Sender: m.Sender, Kind: m.Kind, ReplyTo: m.ReplyTo, Content: m.Content})
	if err != nil {
		return ErrRuntimeOutput
	}
	root, err := l.messageArea(r, receiveArea)
	if err != nil {
		return err
	}
	defer root.Close()
	name := m.ID + ".json"
	if _, err := root.Lstat(name); err == nil {
		existing, ok := readBoundedRegular(root, name, maxMessageCandidateBytes)
		if ok && bytes.Equal(existing, wire) {
			return nil
		}
		return ErrRuntimeOutput
	}
	staging := "@" + m.ID + ".tmp"
	if err := root.Remove(staging); err != nil && !errors.Is(err, os.ErrNotExist) {
		return ErrRuntimeOutput
	}
	f, err := root.OpenFile(staging, os.O_WRONLY|os.O_CREATE|os.O_EXCL|noFollowRuntimeOutput, 0o600)
	if err != nil {
		return ErrRuntimeOutput
	}
	_, writeErr := f.Write(wire)
	syncErr := f.Sync()
	if closeErr := f.Close(); writeErr != nil || syncErr != nil || closeErr != nil {
		root.Remove(staging)
		return ErrRuntimeOutput
	}
	if err := root.Rename(staging, name); err != nil {
		root.Remove(staging)
		return ErrRuntimeOutput
	}
	return nil
}

// readBoundedRegular opens name without following links and returns at most
// limit bytes of a regular file that was not replaced between stat and open.
func readBoundedRegular(root *os.Root, name string, limit int) ([]byte, bool) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return nil, false
	}
	f, err := root.OpenFile(name, os.O_RDONLY|noFollowRuntimeOutput, 0)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	if opened, err := f.Stat(); err != nil || !os.SameFile(info, opened) {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, false
	}
	return data, true
}
