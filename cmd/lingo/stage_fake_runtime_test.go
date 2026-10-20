package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/runtimeadapter"
)

const fakeStageRuntimeFile = "fake-stage-runtime.json"

type fakeStageScript struct {
	StartupMillis int                        `json:"startupMillis,omitempty"`
	Selectors     []string                   `json:"selectors"`
	Agents        map[string][]fakeStageStep `json:"agents"`
	Attempts      map[string][]fakeStageStep `json:"attempts"`
}

type fakeStageStep struct {
	Op          string   `json:"op"`
	Path        string   `json:"path,omitempty"`
	Content     string   `json:"content,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Fields      []string `json:"fields,omitempty"`
	Environment []string `json:"environment,omitempty"`
	Code        int      `json:"code,omitempty"`
	Millis      int      `json:"millis,omitempty"`
}

type fakeStageInvocation struct {
	Runtime     string   `json:"runtime"`
	Agent       string   `json:"agent"`
	Attempt     string   `json:"attempt"`
	Argv        []string `json:"argv"`
	CWD         string   `json:"cwd"`
	Environment []string `json:"environment"`
}

func stageFakeProcess() (bool, int) {
	name := filepath.Base(os.Args[0])
	if name != "codex" && name != "claude" {
		return false, 0
	}
	wire, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), fakeStageRuntimeFile))
	var script fakeStageScript
	if err != nil || json.Unmarshal(wire, &script) != nil {
		return false, 0
	}
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		if err := recordFakeStageStatus(name, "version"); err != nil {
			return true, 98
		}
		version := "synthetic-unregistered-" + name
		_, err = io.WriteString(os.Stdout, version+"\n")
		return true, boolExit(err)
	}
	if len(os.Args) > 2 && (os.Args[1] == "login" || os.Args[1] == "auth") {
		if err := recordFakeStageStatus(name, "auth"); err != nil {
			return true, 98
		}
		status := map[string]string{"codex": "Logged in using ChatGPT", "claude": `{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}`}[name]
		_, err = io.WriteString(os.Stdout, status+"\n")
		return true, boolExit(err)
	}
	var packet runtimeadapter.TaskPacket
	if json.NewDecoder(os.Stdin).Decode(&packet) != nil {
		return true, 2
	}
	cwd, _ := os.Getwd()
	record := fakeStageInvocation{Runtime: name, Agent: packet.AgentID, Attempt: packet.Coordination.AttemptID, Argv: os.Args, CWD: cwd}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		record.Environment = append(record.Environment, key)
	}
	steps := script.Agents[packet.AgentID]
	if override, ok := script.Attempts[packet.AgentID+"#"+strconv.Itoa(attemptOrdinal(packet.AgentID))]; ok {
		steps = override
	}
	time.Sleep(time.Duration(script.StartupMillis) * time.Millisecond)
	appendFakeStageLog(record)
	return true, runFakeStageSteps(script, packet, steps)
}

func boolExit(err error) int {
	if err != nil {
		return 1
	}
	return 0
}

func attemptOrdinal(agent string) int {
	wire, _ := os.ReadFile(filepath.Join(os.Getenv("HOME"), "fake-stage-invocations.jsonl"))
	n := 1
	for _, line := range bytes.Split(wire, []byte("\n")) {
		var r fakeStageInvocation
		if json.Unmarshal(line, &r) == nil && r.Agent == agent {
			n++
		}
	}
	return n
}

func appendFakeStageLog(r fakeStageInvocation) {
	wire, err := json.Marshal(r)
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), "fake-stage-invocations.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(wire, '\n'))
}

func runFakeStageSteps(script fakeStageScript, packet runtimeadapter.TaskPacket, steps []fakeStageStep) int {
	axiom := packet.Coordination.AxiomPath
	captured := ""
	expand := func(fields []string) []string {
		out := []string{}
		for _, f := range fields {
			out = append(out, "--field", strings.ReplaceAll(f, "{captured}", captured))
		}
		return out
	}
	for _, step := range steps {
		switch step.Op {
		case "write":
			if err := os.MkdirAll(filepath.Dir(step.Path), 0o700); err != nil {
				return 20
			}
			if os.WriteFile(step.Path, []byte(step.Content), 0o600) != nil {
				return 21
			}
		case "publish", "publish-denied":
			cmd := exec.Command(axiom, append([]string{"--json", "workflow", "coordination", "publish", "--kind", step.Kind}, expand(step.Fields)...)...)
			cmd.Env = stageFakeEnvironment(step.Environment)
			out, err := cmd.CombinedOutput()
			if step.Op == "publish-denied" {
				if err == nil || bytes.Contains(out, []byte(`"coordination_candidate_published"`)) || !bytes.Contains(out, []byte(`"child_context_required"`)) {
					return 25
				}
			} else if err != nil || !bytes.Contains(out, []byte(`"coordination_candidate_published"`)) {
				appendFakeStageDiagnostic(packet.AgentID, fmt.Sprintf("publish %s: %v %s", step.Kind, err, out))
				return 22
			}
		case "await":
			captured = ""
			var lastOutput []byte
			var lastError error
			deadline := time.Now().Add(time.Duration(step.Millis) * time.Millisecond)
			for captured == "" {
				if time.Now().After(deadline) {
					appendFakeStageDiagnostic(packet.AgentID, fmt.Sprintf("await %s: %v %s", step.Kind, lastError, lastOutput))
					return 23
				}
				cmd := exec.Command(axiom, append(append([]string{"--json", "workflow", "coordination", "list"}, script.Selectors...), "--kind", step.Kind)...)
				cmd.Env = os.Environ()
				out, err := cmd.CombinedOutput()
				lastOutput, lastError = out, err
				var event struct {
					Stage struct {
						Records []coordination.Record `json:"records"`
					} `json:"stage"`
				}
				if err == nil && json.Unmarshal(bytes.TrimSpace(out), &event) == nil {
					for _, r := range event.Stage.Records {
						if r.ChildID != os.Getenv("AXIOM_GRAPH_ID") {
							captured = r.RecordID
							break
						}
					}
				}
				if captured == "" {
					time.Sleep(100 * time.Millisecond)
				}
			}
		case "sleep":
			time.Sleep(time.Duration(step.Millis) * time.Millisecond)
		case "stderr":
			if _, err := io.WriteString(os.Stderr, step.Content); err != nil {
				return 27
			}
		case "exit":
			return step.Code
		default:
			return 24
		}
	}
	return 0
}

func appendFakeStageDiagnostic(agent, message string) {
	f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), "fake-stage-diagnostics.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s: %s\n", time.Now().UTC().Format(time.RFC3339Nano), agent, message)
}

func stageFakeEnvironment(overrides []string) []string {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	for _, entry := range overrides {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	environment := make([]string, 0, len(keys))
	for _, key := range keys {
		environment = append(environment, key+"="+values[key])
	}
	return environment
}

func recordFakeStageStatus(runtimeID, operation string) error {
	f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), "fake-stage-status.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, runtimeID+" "+operation+"\n")
	return err
}
