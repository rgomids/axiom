#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEST_ROOT"' EXIT
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go build -o "$TEST_ROOT/check-domain" "$ROOT/scripts/check-project-domain.go"
mkdir -p "$TEST_ROOT/internal/project"

check_case() {
  local expected="$1"
  local source="$2"
  printf '%s\n' "$source" > "$TEST_ROOT/internal/project/fixture_test.go"
  if (cd "$TEST_ROOT" && ./check-domain > result.txt 2>&1); then
    [[ "$expected" == "pass" ]] || { printf 'FAIL: forbidden fixture accepted\n' >&2; exit 1; }
    return
  fi
  [[ "$expected" == "fail" ]] || { printf 'FAIL: pure fixture rejected\n' >&2; exit 1; }
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
