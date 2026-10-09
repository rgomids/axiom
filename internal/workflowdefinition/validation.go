package workflowdefinition

import (
	"fmt"
	"github.com/rgomids/axiom/internal/portableconfig"
	"sort"
)

func validate(d Definition) []Diagnostic {
	var issues []Diagnostic
	add := func(field, code string) { issues = append(issues, Diagnostic{field, code}) }
	unique := func(ids []string, field string) map[string]bool {
		m := map[string]bool{}
		for _, id := range ids {
			if m[id] {
				add(field, "duplicate_id")
			}
			m[id] = true
		}
		return m
	}
	phases := map[string]int{"intake": 0, "specification": 1, "planning": 2, "implementation": 3, "review": 4, "completion": 5}
	stages, covered, prior, gates := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	last := -1
	for i, s := range d.Stages {
		field := fmt.Sprintf("stages[%d]", i)
		if stages[s.ID] {
			add(field, "duplicate_stage")
		}
		stages[s.ID] = true
		if phases[s.Phase] < last {
			add(field, "phase_order")
		}
		last = phases[s.Phase]
		covered[s.Phase] = true
		if s.Checkpoint != (i == len(d.Stages)-1 || d.Stages[i+1].Phase != s.Phase) {
			add(field, "phase_checkpoint")
		}
		for _, text := range []string{s.Purpose, s.Instructions} {
			if !portableconfig.SafeProse(text) {
				add(field, "unsafe_portable_text")
			}
		}
		inputs, outputs, validators, criteria := []string{}, []string{}, []string{}, []string{}
		for _, in := range s.Inputs {
			inputs = append(inputs, in.ID)
			if !logicalSource(in.Source) {
				add(field+".inputs", "invalid_logical_reference")
			}
			if in.Kind == "stage-output" && !prior[in.Source] {
				add(field+".inputs", "dangling_stage_output")
			}
		}
		for _, out := range s.Outputs {
			outputs = append(outputs, out.ID)
		}
		for _, v := range s.Validators {
			validators = append(validators, v.ID)
		}
		for _, c := range s.CompletionCriteria {
			criteria = append(criteria, c.ID)
			if !portableconfig.SafeProse(c.Description) {
				add(field+".completionCriteria", "unsafe_portable_text")
			}
		}
		ins, outs, vals := unique(inputs, field+".inputs"), unique(outputs, field+".outputs"), unique(validators, field+".validators")
		unique(criteria, field+".completionCriteria")
		for _, c := range s.CompletionCriteria {
			if !outs[c.OutputRef] || !vals[c.ValidatorRef] {
				add(field+".completionCriteria", "dangling_reference")
			}
		}
		for _, g := range s.HumanGates {
			expected := map[string][2]string{"planning_authority": {"planning", "before"}, "implementation_authority": {"implementation", "before"}, "review_started": {"review", "before"}, "human_acceptance": {"completion", "after"}}[g.Kind]
			if gates[g.Kind] {
				add(field+".humanGates", "duplicate_gate")
			}
			gates[g.Kind] = true
			if expected != [2]string{s.Phase, g.Timing} || (g.Kind == "human_acceptance" && i != len(d.Stages)-1) || (g.Kind != "human_acceptance" && i > 0 && d.Stages[i-1].Phase == s.Phase) {
				add(field+".humanGates", "invalid_gate_position")
			}
		}
		nodes := map[string]Agent{}
		owner := ""
		owners := 0
		assigned := map[string]bool{}
		if s.Concurrency > len(s.Agents) || (s.Mode == "sequential" && s.Concurrency != 1) {
			add(field, "invalid_concurrency")
		}
		for j, a := range s.Agents {
			af := fmt.Sprintf("%s.agents[%d]", field, j)
			if _, ok := nodes[a.ID]; ok {
				add(af, "duplicate_agent")
			}
			nodes[a.ID] = a
			if !portableconfig.SafeProse(a.Responsibilities) {
				add(af, "unsafe_portable_text")
			}
			if a.IntegrationOwner {
				owner = a.ID
				owners++
				if !a.ValidationOwner {
					add(af, "validation_owner_required")
				}
			}
			for _, in := range a.Inputs {
				if !ins[in] {
					add(af, "dangling_input")
				}
			}
			for _, out := range a.Outputs {
				if !outs[out] {
					add(af, "dangling_output")
				}
				assigned[out] = true
			}
			unique(a.DependsOn, af+".dependsOn")
			unique(a.RuntimeConstraints, af+".runtimeConstraints")
			unique(a.Capabilities, af+".capabilities")
			unique(a.Inputs, af+".inputs")
			unique(a.Outputs, af+".outputs")
			unique(a.EffectCeilings, af+".effectCeilings")
			if (a.Effort.Mode == "runtime-default") != (a.Effort.Value == "default") {
				add(af, "invalid_effort")
			}
			if s.Mode == "sequential" && j > 0 {
				found := false
				for _, dep := range a.DependsOn {
					found = found || dep == s.Agents[j-1].ID
				}
				if !found {
					add(af, "sequential_chain_required")
				}
			}
		}
		// Memoized DFS bounds traversal even for dense 32-node DAGs.
		color := map[string]int{}
		ancestors := map[string]map[string]bool{}
		var visit func(string) map[string]bool
		visit = func(id string) map[string]bool {
			if color[id] == 1 {
				add(field+".agents", "dependency_cycle")
				return map[string]bool{}
			}
			if color[id] == 2 {
				return ancestors[id]
			}
			a, ok := nodes[id]
			if !ok {
				add(field+".agents", "dangling_dependency")
				return map[string]bool{}
			}
			color[id] = 1
			m := map[string]bool{}
			for _, dep := range a.DependsOn {
				m[dep] = true
				for k := range visit(dep) {
					m[k] = true
				}
			}
			color[id] = 2
			ancestors[id] = m
			return m
		}
		for _, a := range s.Agents {
			visit(a.ID)
		}
		if owners != 1 || len(ancestors[owner]) != len(nodes)-1 {
			add(field+".agents", "integration_owner_required")
		}
		for _, out := range s.Outputs {
			if out.Required && !assigned[out.ID] {
				add(field+".outputs", "unowned_output")
			}
			prior[s.ID+"/"+out.ID] = true
		}
	}
	if len(covered) != 6 {
		add("stages", "phase_coverage_required")
	}
	if len(gates) != 4 {
		add("stages", "protected_gates_required")
	}
	if !portableconfig.SafeProse(d.Name) {
		add("name", "unsafe_portable_text")
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Field != issues[j].Field {
			return issues[i].Field < issues[j].Field
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}
