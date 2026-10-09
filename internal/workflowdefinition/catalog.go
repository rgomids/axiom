package workflowdefinition

type Entry struct {
	WorkflowID string `json:"workflowId"`
	Revision   int    `json:"revision"`
	Digest     string `json:"digest"`
	State      string `json:"state"`
}

func (e Entry) Ref() Ref {
	return Ref{WorkflowID: e.WorkflowID, Revision: e.Revision, Digest: e.Digest, Source: "project"}
}

type Index struct {
	SchemaVersion int     `json:"schemaVersion"`
	Revisions     []Entry `json:"revisions"`
}
type Catalog struct {
	Index     Index
	Wire      []byte
	Documents map[string]Document
	Unindexed []Ref
}
