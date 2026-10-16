//go:build unix && !darwin

package executiongraph

func observeAdoptedProcesses(bool, int) (map[int]string, error) { return nil, nil }
