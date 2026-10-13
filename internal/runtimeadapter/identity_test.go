package runtimeadapter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
)

func identityRequest(runtime string, write, process bool) StageInvocationRequest {
	version := "0.159.1"
	if runtime == "claude" {
		version = "2.1.296"
	}
	return StageInvocationRequest{Profile: CommandProfile{RuntimeID: runtime, Executable: "/bin/" + runtime, Model: "model"}, Version: version, Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom", RepositoryWrite: write, Process: process}
}

func TestAdapterVersionCanonical(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		version, err := AdapterVersion(runtime)
		again, secondErr := AdapterVersion(runtime)
		if err != nil || secondErr != nil || version == "" || version != again {
			t.Fatalf("%s: unstable adapter version", runtime)
		}
		digest, err := ModeDigest(identityRequest(runtime, false, false))
		b := capabilityproof.Binding{RuntimeID: runtime, Version: "vendor-version", AdapterVersion: version, Variant: "read-only", ModeDigest: digest, ExecutableDigest: strings.Repeat("a", 64), AxiomDigest: strings.Repeat("b", 64)}
		if err != nil || !capabilityproof.ValidBinding(b) {
			t.Fatalf("%s: invalid canonical binding", runtime)
		}
		// A caller can mutate its copy of Binding, but cannot supply that copy
		// to AdapterVersion or override the adapter identity in ModeDigest.
		b.AdapterVersion = "caller-version"
		wire, err := encodeModeIdentity(identityRequest(runtime, false, false))
		if err != nil || !bytes.Contains(wire, []byte(version)) || bytes.Contains(wire, []byte(b.AdapterVersion)) {
			t.Fatal("caller overrode adapter identity")
		}
	}
	for _, runtime := range []string{"", "other", "Codex", "codex-adapter-v2"} {
		if version, err := AdapterVersion(runtime); err == nil || version != "" {
			t.Fatalf("accepted unsupported adapter %q", runtime)
		}
	}
}

func TestModeDigestGolden(t *testing.T) {
	// Initial vectors derive from the v1 stage contract, never persisted Evidence.
	for _, tc := range []struct {
		runtime        string
		write, process bool
		want           string
	}{
		{"codex", false, false, "b917cedaaa0aa071099ddd5e28b25806f4ba9717cadae978adc90703dd376ca0"},
		{"codex", false, true, "b917cedaaa0aa071099ddd5e28b25806f4ba9717cadae978adc90703dd376ca0"},
		{"codex", true, false, "3a05324dc6433b785879c07d52b1351ddded938ddface51679a885dafc9abe36"},
		{"codex", true, true, "3a05324dc6433b785879c07d52b1351ddded938ddface51679a885dafc9abe36"},
		{"claude", false, false, "32a45eaafc51f3f6043a550481b4671f5f642818fc0a8d50f7ba076ddd5525a0"},
		{"claude", false, true, "6b42f16a8cd6bdbf1c706b188540451bd78e1b7d29bb12766e4925ec35f8c5ae"},
		{"claude", true, false, "c3fe725b4d4b2610f9a633b47f8c4fc86a65fca8ba44901f788bd34b4681c83f"},
		{"claude", true, true, "e24a5d78beeaf46711058beb3172bc00565b3317f3d872d9d7adc89e6701a375"},
	} {
		request := identityRequest(tc.runtime, tc.write, tc.process)
		for range 2 {
			got, err := ModeDigest(request)
			if err != nil || got != tc.want {
				t.Fatalf("%s write=%v process=%v: got %s (%v), want %s", tc.runtime, tc.write, tc.process, got, err, tc.want)
			}
		}
	}
}

func TestModeDigestVolatileNormalization(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		for _, write := range []bool{false, true} {
			base := identityRequest(runtime, write, false)
			other := base
			other.Profile.Executable = "/other/bin/" + runtime
			other.Profile.Model, other.Profile.ModelProfileID = "other-model", "other-profile"
			other.Workspace, other.Inbox, other.AxiomPath = "/tmp/another-work", "/tmp/another-inbox", "/tmp/another-axiom"
			other.ParentEnvironment = []string{"HOME=/tmp/another-home", "TMPDIR=/tmp/volatile", "PROVIDER_AUTH=redacted-placeholder"}
			other.Correlation = AttemptCorrelation{"project", "parent", "child", "attempt"}
			want, err := ModeDigest(base)
			got, otherErr := ModeDigest(other)
			if err != nil || otherErr != nil || got != want {
				t.Fatalf("%s: equivalent process requests drifted: %v %v", runtime, err, otherErr)
			}
			wire, err := encodeModeIdentity(other)
			if err != nil {
				t.Fatal(err)
			}
			for _, excluded := range []string{"/tmp/", "/other/", "redacted-placeholder", "PROVIDER_AUTH", "other-model", "other-profile", "attempt"} {
				if bytes.Contains(wire, []byte(excluded)) {
					t.Fatalf("serialized excluded input %q", excluded)
				}
			}
		}
	}
}

func TestModeDigestSafetyDifferences(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		read, _ := ModeDigest(identityRequest(runtime, false, false))
		write, _ := ModeDigest(identityRequest(runtime, true, false))
		if read == write {
			t.Fatal("confinement variants collapsed")
		}
	}
	limited, _ := ModeDigest(identityRequest("claude", false, false))
	process, _ := ModeDigest(identityRequest("claude", false, true))
	if limited == process {
		t.Fatal("restricted Axiom Bash and unrestricted Bash collapsed")
	}
	codex, _ := ModeDigest(identityRequest("codex", false, false))
	if codex == limited {
		t.Fatal("sandbox and permission-tool policies collapsed")
	}
	for _, tc := range []struct{ runtime, flags string }{
		{"codex", `"--ignore-user-config","--ephemeral","-","--sandbox","workspace-write"`},
		{"claude", `"--permission-prompts","none","--permission-mode","dontAsk"`},
	} {
		wire, err := encodeModeIdentity(identityRequest(tc.runtime, false, false))
		if err != nil || !bytes.Contains(wire, []byte(tc.flags)) {
			t.Fatal("fixed safety flags missing from identity")
		}
	}
}

func TestModeDigestRejectsUnsupported(t *testing.T) {
	for name, mutate := range map[string]func(*StageInvocationRequest){
		"runtime":              func(r *StageInvocationRequest) { r.Profile.RuntimeID = "other" },
		"version":              func(r *StageInvocationRequest) { r.Version = "unknown" },
		"sandbox override":     func(r *StageInvocationRequest) { r.Profile.Arguments = []string{"--sandbox=danger-full-access"} },
		"permission override":  func(r *StageInvocationRequest) { r.Profile.Arguments = []string{"--permission-mode=bypassPermissions"} },
		"environment override": func(r *StageInvocationRequest) { r.Profile.Environment = []string{"SETTING=override"} },
		"credential override":  func(r *StageInvocationRequest) { r.Profile.CredentialReference = "redacted-placeholder" },
		"overlapping roots":    func(r *StageInvocationRequest) { r.Inbox = r.Workspace },
		"relative path":        func(r *StageInvocationRequest) { r.Workspace = "relative" },
	} {
		t.Run(name, func(t *testing.T) {
			r := identityRequest("claude", false, false)
			mutate(&r)
			if digest, err := ModeDigest(r); err == nil || digest != "" {
				t.Fatal("unsupported mode produced an identity")
			}
		})
	}
}
