package workitem

// Type describes delivery intent independently of any provider's taxonomy.
type Type string

const (
	Story Type = "story"
	Bug   Type = "bug"
	Task  Type = "task"
)

func (t Type) Valid() bool { return t == Story || t == Bug || t == Task }
