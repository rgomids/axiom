package cli

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// The executable catalog owns routing; prose and the versioned reconciliation
// must describe exactly its operations, commands, effects and authority inputs.
// Argument requirements continue to come from the parser's FlagSets.
func TestCanonicalRoutingContract(t *testing.T) {
	matrix, err := os.ReadFile("../../docs/specifications/004-mvp-v1-baseline/issue-230-resource-lifecycle-matrix.md")
	if err != nil {
		t.Fatal(err)
	}
	start := "<!-- canonical-runtime-catalog:start -->"
	end := "<!-- canonical-runtime-catalog:end -->"
	_, block, found := strings.Cut(string(matrix), start)
	if !found {
		t.Fatal("missing current catalog reconciliation")
	}
	block, _, found = strings.Cut(block, end)
	if !found {
		t.Fatal("unterminated current catalog reconciliation")
	}
	actual := []string{}
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "| axiom-") {
			actual = append(actual, line)
		}
	}
	expected := []string{}
	for _, name := range []string{"axiom-project", "axiom-work-item"} {
		source, err := os.ReadFile("../codexruntime/skills/" + name + "/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		rows := [][]string{}
		for _, line := range strings.Split(string(source), "\n") {
			if !strings.HasPrefix(line, "| `") {
				continue
			}
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) != 6 {
				t.Fatalf("malformed routing row %s", line)
			}
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			rows = append(rows, cells)
		}
		rowIndex := 0
		for _, spec := range skillOperationSpecs(name) {
			covered := map[action]bool{}
			declared := map[action]bool{}
			for _, operation := range spec.actions {
				declared[operation] = true
			}
			for _, mode := range spec.modes {
				for _, operation := range mode.actions {
					if !declared[operation] {
						t.Fatalf("%s/%s mode routes undeclared action %s", name, spec.name, operation)
					}
					covered[operation] = true
				}
			}
			if !reflect.DeepEqual(covered, declared) {
				t.Fatalf("%s/%s has action outside routing modes", name, spec.name)
			}
			for _, mode := range spec.modes {
				if rowIndex >= len(rows) {
					t.Fatalf("%s missing mode %s/%s", name, spec.name, mode.name)
				}
				row := rows[rowIndex]
				rowIndex++
				proseMode := mode.name
				if proseMode == "default" {
					proseMode = "-"
				}
				if strings.Trim(row[0], "`") != spec.name || strings.Trim(row[1], "`") != proseMode || strings.ReplaceAll(row[3], " ", "-") != mode.effect {
					t.Fatalf("routing metadata drift %s: %v", name, row)
				}
				for _, input := range mode.authorityInputs {
					if !strings.Contains(row[4], "`"+input+"`") {
						t.Fatalf("%s authority drift: %s", name, input)
					}
				}
				if mode.effect == effectReadOnly && row[5] != "allowed" {
					t.Fatalf("%s read-only semantic drift", name)
				}
				if mode.effect != effectReadOnly && !strings.HasPrefix(row[5], "only for unambiguous") {
					t.Fatalf("%s mutation semantic drift", name)
				}
				commands := []string{}
				for _, action := range mode.actions {
					command := skillCommandName(action)
					commands = append(commands, command)
					proseCommand := strings.Replace(command, "axiom ", "axiom --json ", 1)
					if !strings.Contains(row[2], "`"+proseCommand+"`") && !strings.Contains(row[2], "`"+proseCommand+" ") {
						t.Fatalf("%s command drift: %s", name, command)
					}
				}
				// No unrelated command may be smuggled into a prose route.
				if strings.Count(row[2], "axiom --json ") != len(mode.actions) {
					t.Fatalf("extra command in %s %s/%s", name, spec.name, mode.name)
				}
				expected = append(expected, "| "+name+" | "+spec.name+" | "+mode.name+" | "+strings.Join(commands, "; ")+" | "+mode.effect+" | "+strings.Join(mode.authorityInputs, " ")+" |")
			}
		}
		if rowIndex != len(rows) {
			t.Fatalf("%s extra prose mode", name)
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("versioned catalog drift\nactual:\n%s\nexpected:\n%s", strings.Join(actual, "\n"), strings.Join(expected, "\n"))
	}
}
