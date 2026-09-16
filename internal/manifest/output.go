package manifest

import "io"

// boundedOutput retains at most MaxBytes, including backing-array capacity.
// Overflow rejects the entire write before allocation/copy and stays failed.
// It bounds our output storage, not the YAML encoder's own working memory.
type boundedOutput struct {
	data     []byte
	exceeded bool
}

var _ io.Writer = (*boundedOutput)(nil)

func (w *boundedOutput) Write(p []byte) (int, error) {
	if w.exceeded || len(p) > MaxBytes-len(w.data) {
		w.exceeded = true
		return 0, io.ErrShortBuffer
	}
	size := len(w.data) + len(p)
	if size > cap(w.data) {
		capacity := min(MaxBytes, max(size, 2*cap(w.data)))
		grown := make([]byte, len(w.data), capacity)
		copy(grown, w.data)
		w.data = grown
	}
	w.data = append(w.data, p...)
	return len(p), nil
}
