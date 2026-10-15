//go:build unix && !darwin

package executiongraph

func observeAdoptedProcesses(bool) (map[int]string, error) { return nil, nil }
