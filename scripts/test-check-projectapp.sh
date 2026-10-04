#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEST_ROOT"' EXIT
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go build -o "$TEST_ROOT/check-architecture" "$ROOT/scripts/check-architecture.go"
mkdir -p "$TEST_ROOT/internal/projectapp"

check_case() {
  local expected="$1"
  local source="$2"
  local file="${3:-fixture_test.go}"
  rm -f "$TEST_ROOT/internal/projectapp/"*.go
  printf '%s\n' "$source" > "$TEST_ROOT/internal/projectapp/$file"
  if (cd "$TEST_ROOT" && ./check-architecture application > result.txt 2>&1); then
    [[ "$expected" == "pass" ]] || { printf 'FAIL: forbidden fixture accepted\n' >&2; exit 1; }
    return
  fi
  [[ "$expected" == "fail" ]] || { printf 'FAIL: pure fixture rejected\n' >&2; exit 1; }
}

# Application allowances must not leak into the independent domain policy.
check_domain_rejects() {
  local source="$1"
  local message="$2"
  local file="${3:-fixture_test.go}"
  rm -rf "$TEST_ROOT/other"
  mkdir -p "$TEST_ROOT/other/internal/project"
  printf '%s\n' "$source" > "$TEST_ROOT/other/internal/project/$file"
  if (cd "$TEST_ROOT/other" && ../check-architecture domain > ../other.txt 2>&1); then
    printf 'FAIL: domain profile accepted an application-only allowance\n' >&2; exit 1
  fi
  grep -qxF "FAIL: $message" "$TEST_ROOT/other.txt" || { printf 'FAIL: unexpected domain diagnostic\n' >&2; exit 1; }
}

check_usage() {
  local status=0
  (cd "$TEST_ROOT" && ./check-architecture "$@" > usage.txt 2>&1) || status=$?
  [[ "$status" == 2 ]] && grep -q '^usage: ' "$TEST_ROOT/usage.txt" || { printf 'FAIL: invalid profile not rejected with usage\n' >&2; exit 1; }
}

check_case pass 'package projectapp_test; import "github.com/rgomids/axiom/internal/project"; var p project.Project'
check_case pass 'package projectapp_test; import "unicode/utf8"; func pure() { _ = utf8.ValidString("context/api�legacy.md") }'
check_case fail 'package projectapp_test; import "unicode/utf8"; func unreviewed() { _ = utf8.RuneCountInString("text") }'
check_case fail 'package projectapp_test; import "os"; func impure() { _ = os.Getenv("SYNTHETIC") }'
check_case fail 'package projectapp_test; import "net/http"; func impure() { _, _ = http.Get("https://example.invalid") }'
check_case fail 'package projectapp_test; import "os/exec"; func impure() { _ = exec.Command("synthetic") }'
check_case fail 'package projectapp_test; import "os"; func impure() { _ = os.WriteFile("synthetic", nil, 0600) }'
check_case fail 'package projectapp_test; import "testing"; func TestImpure(t *testing.T) { _ = t.TempDir() }'
check_case fail 'package projectapp_test; import "time"; func impure() { _ = time.Now() }'
check_case fail 'package projectapp_test; import "path/filepath"; func impure() { _, _ = filepath.Abs("synthetic") }'
check_case fail 'package projectapp_test; import "encoding/json"; func unreviewed() { _ = json.Valid(nil) }'
check_case fail 'package projectapp_test; import "reflect"; func impure() { _ = reflect.ValueOf(1) }'
check_case fail 'package projectapp_test; import alias "strings"; func pure() { _ = alias.TrimSpace("text") }'
check_case fail 'package projectapp_test; import "github.com/rgomids/axiom/internal/local"; var _ = local.Store'
check_case fail 'package projectapp_test; import "unsafe"; var _ unsafe.Pointer'
check_case fail 'package projectapp_test; //go:linkname escape runtime.escape'

printf 'PASS: application checker accepts inward domain dependency and pure UTF-8 validation; rejects fourteen unreviewed symbol, ambient I/O, outward dependency and bypass fixtures without executing them\n'

check_case pass 'package projectapp; import "github.com/rgomids/axiom/internal/project"; var _ project.Project' fixture.go
check_case pass 'package projectapp; import "time"; var _ = time.Unix(0, 0)' fixture.go
check_case fail 'package projectapp; import "reflect"; var _ = reflect.DeepEqual(1, 1)' fixture.go
check_case fail 'package projectapp; import "github.com/rgomids/axiom/internal/manifest"; var _ manifest.Codec' fixture.go
check_domain_rejects 'package project_test; import "unicode/utf8"; var _ = utf8.ValidString("text")' 'non-allowlisted domain import: unicode/utf8'
check_domain_rejects 'package project; import "github.com/rgomids/axiom/internal/project"; var _ project.Project' 'non-allowlisted domain import: github.com/rgomids/axiom/internal/project' fixture.go
check_usage
check_usage Application
check_usage application domain

printf 'PASS: application profile keeps inward production dependency and test-only helpers; its allowances are rejected by the domain profile; missing, unknown and extra profiles are refused with usage\n'
