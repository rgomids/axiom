#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEST_ROOT"' EXIT
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go build -o "$TEST_ROOT/check-architecture" "$ROOT/scripts/check-architecture.go"
mkdir -p "$TEST_ROOT/internal/project"

check_case() {
  local expected="$1"
  local source="$2"
  local file="${3:-fixture_test.go}"
  rm -f "$TEST_ROOT/internal/project/"*.go
  printf '%s\n' "$source" > "$TEST_ROOT/internal/project/$file"
  if (cd "$TEST_ROOT" && ./check-architecture domain > result.txt 2>&1); then
    [[ "$expected" == "pass" ]] || { printf 'FAIL: forbidden fixture accepted\n' >&2; exit 1; }
    return
  fi
  [[ "$expected" == "fail" ]] || { printf 'FAIL: pure fixture rejected\n' >&2; exit 1; }
}

# Domain allowances must not leak into the independent application policy.
check_application_rejects() {
  local source="$1"
  local message="$2"
  local file="${3:-fixture_test.go}"
  rm -rf "$TEST_ROOT/other"
  mkdir -p "$TEST_ROOT/other/internal/projectapp"
  printf '%s\n' "$source" > "$TEST_ROOT/other/internal/projectapp/$file"
  if (cd "$TEST_ROOT/other" && ../check-architecture application > ../other.txt 2>&1); then
    printf 'FAIL: application profile accepted a domain-only allowance\n' >&2; exit 1
  fi
  grep -qxF "FAIL: $message" "$TEST_ROOT/other.txt" || { printf 'FAIL: unexpected application diagnostic\n' >&2; exit 1; }
}

check_usage() {
  local status=0
  (cd "$TEST_ROOT" && ./check-architecture "$@" > usage.txt 2>&1) || status=$?
  [[ "$status" == 2 ]] && grep -q '^usage: ' "$TEST_ROOT/usage.txt" || { printf 'FAIL: invalid profile not rejected with usage\n' >&2; exit 1; }
}

check_case pass 'package project_test; import "strings"; func pure() { _ = strings.TrimSpace("text") }'
check_case fail 'package project_test; import "os"; func impure() { _ = os.Getenv("SYNTHETIC") }'
check_case fail 'package project_test; import "net/http"; func impure() { _, _ = http.Get("https://example.invalid") }'
check_case fail 'package project_test; import "os/exec"; func impure() { _ = exec.Command("synthetic") }'
check_case fail 'package project_test; import "fmt"; func impure() { fmt.Println("text") }'
check_case fail 'package project_test; import "testing"; func TestImpure(t *testing.T) { _ = t.TempDir() }'
check_case fail 'package project_test; import "reflect"; func impure() { _ = reflect.ValueOf(1) }'
check_case fail 'package project_test; import alias "strings"; func pure() { _ = alias.TrimSpace("text") }'

printf 'PASS: domain boundary checker accepts pure syntax and rejects seven I/O/bypass fixtures; fixtures never executed\n'

check_case pass 'package project; import "fmt"; var _ = fmt.Sprintf("%d", 1)' fixture.go
check_case pass 'package project; import "reflect"; var _ = reflect.DeepEqual(1, 1)' fixture.go
check_case fail 'package project; import "testing"; var _ *testing.T' fixture.go
check_case fail 'package project; import "github.com/rgomids/axiom/internal/project"; var _ project.Project' fixture.go
check_application_rejects 'package projectapp_test; import "fmt"; var _ = fmt.Sprintf("%d", 1)' 'non-allowlisted application import: fmt'
check_application_rejects 'package projectapp; import "reflect"; var _ = reflect.DeepEqual(1, 1)' 'test-only helper imported by production application' fixture.go
check_usage
check_usage projectapp
check_usage domain application

printf 'PASS: domain profile keeps production/test exceptions; its allowances are rejected by the application profile; missing, unknown and extra profiles are refused with usage\n'
