package projectapp

import "testing"

func TestPolicyDraftTreeBounds(t *testing.T) {
	for _, size := range []int{16383, 16384} {
		tree := make([]any, size)
		n := 0
		if policyTreeBounded(tree, 1, &n) != (size == 16383) {
			t.Fatalf("node boundary %d", size+1)
		}
	}
	for _, depth := range []int{32, 33} {
		var tree any = "leaf"
		for i := 1; i < depth; i++ {
			tree = []any{tree}
		}
		n := 0
		if policyTreeBounded(tree, 1, &n) != (depth == 32) {
			t.Fatalf("depth boundary %d", depth)
		}
	}
}
