//go:build unix && !darwin

package executiongraph

func observeAdoptedProcesses() (map[int]string, error) { return nil, nil }
