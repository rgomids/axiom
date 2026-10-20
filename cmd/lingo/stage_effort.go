package main

import (
	"context"
	"errors"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
)

var errConfigurationUnavailable = errors.New("configuration_unavailable")

func productionEffortInspector(stateRoot, repository string, children []stageexecution.ChildRights) (runtimeadapter.StageEffortInspector, error) {
	profiles, err := local.NewRuntimeProfileStore(stateRoot)
	if err != nil {
		return runtimeadapter.StageEffortInspector{}, err
	}
	inspector := runtimeadapter.StageEffortInspector{Configuration: canonicalConfiguration(profiles), Repository: repository, Consented: true}
	for _, child := range children {
		for _, effect := range child.Effects {
			if effect.Kind == "repository-write" {
				inspector.Workspaces = append(inspector.Workspaces, child.Workspace)
			}
		}
	}
	return inspector, nil
}

func canonicalConfiguration(store local.RuntimeProfileStore) func(context.Context) (runtimeprofile.ConfigurationV4, error) {
	return func(ctx context.Context) (runtimeprofile.ConfigurationV4, error) {
		snapshot, err := store.Inspect(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return runtimeprofile.ConfigurationV4{}, ctx.Err()
			}
			return runtimeprofile.ConfigurationV4{}, errConfigurationUnavailable
		}
		if snapshot.State != "active-v4" || snapshot.V4 == nil {
			return runtimeprofile.ConfigurationV4{}, errConfigurationUnavailable
		}
		return *snapshot.V4, nil
	}
}
