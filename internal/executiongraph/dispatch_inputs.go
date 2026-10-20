package executiongraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/rgomids/axiom/internal/strictjson"
)

const MaxDispatchInputsBytes = MaxGraphBytes
const OperationalDispatchInputsVersion = 2

// DispatchInputs retains the exact reviewed dispatch preview bytes so a
// detached supervisor can build each child's task packet without receiving
// paths or content on its command line. It is inert: it grants nothing, and
// consumers accept it only when the preview's own digest equals the committed
// dispatch authority's PreviewDigest.
type DispatchInputs struct {
	FormatVersion int             `json:"formatVersion"`
	ProjectID     string          `json:"projectId"`
	ParentID      string          `json:"parentId"`
	PreviewDigest string          `json:"previewDigest"`
	Preview       json.RawMessage `json:"preview"`
}

func ValidateDispatchInputs(d DispatchInputs) error {
	trimmed := bytes.TrimSpace(d.Preview)
	if d.FormatVersion != 1 && d.FormatVersion != OperationalDispatchInputsVersion || !validToken(d.ProjectID) || !validOpaqueID(d.ParentID) || !successorDigest(d.PreviewDigest) || len(trimmed) < 2 || trimmed[0] != '{' || !json.Valid(trimmed) {
		return ErrInvalidGraph
	}
	return nil
}

func EncodeDispatchInputs(d DispatchInputs) ([]byte, error) {
	if ValidateDispatchInputs(d) != nil {
		return nil, ErrInvalidGraph
	}
	var compact bytes.Buffer
	if json.Compact(&compact, d.Preview) != nil {
		return nil, ErrInvalidGraph
	}
	d.Preview = compact.Bytes()
	b, err := json.Marshal(d)
	if err != nil || len(b)+1 > MaxDispatchInputsBytes {
		return nil, ErrInvalidGraph
	}
	return append(b, '\n'), nil
}

func DecodeDispatchInputs(b []byte) (DispatchInputs, error) {
	var d DispatchInputs
	if strictjson.Decode(b, &d, MaxDispatchInputsBytes) != nil || ValidateDispatchInputs(d) != nil {
		return DispatchInputs{}, ErrInvalidGraph
	}
	return d, nil
}

func DispatchInputsDigest(d DispatchInputs) (string, error) {
	b, err := EncodeDispatchInputs(d)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
