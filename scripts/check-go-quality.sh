#!/usr/bin/env bash
# Read-only structural checks. CI bounds installation + execution to 10 minutes.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"
export GOTOOLCHAIN=local GOFLAGS= GOWORK=off
STATICCHECK_VERSION=v0.8.1
# Safety/correctness plus unreachable private code; no style/performance policy.
STATICCHECK_CHECKS='SA1*,SA2*,SA3*,SA4*,SA5*,SA9*,U1000,-SA1019'

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
printf 'Reproduce: ./scripts/check-go-quality.sh %s\n' "${1:-all}"
go version

format() {
  local files=() path output
  # Include new files; do not inspect ignored build output or mutate sources.
  while IFS= read -r -d '' path; do
    [[ -f "$path" ]] && files+=("$path")
  done < <(git ls-files -z --cached --others --exclude-standard -- '*.go')
  [[ ${#files[@]} -gt 0 ]] || fail 'no Go files found'
  output=$(gofmt -l "${files[@]}") || fail 'gofmt could not parse sources'
  if [[ -n "$output" ]]; then
    printf '%s\n' "$output" >&2
    fail 'gofmt: format the listed files with gofmt -w'
  fi
}

tidy() {
  printf 'Command: go mod tidy -diff\n'
  go mod tidy -diff || fail 'module tidiness: run go mod tidy and review go.mod/go.sum'
}

analyze() {
  local tool="${STATICCHECK:-staticcheck}" version
  version=$("$tool" -version) || fail "install: go install honnef.co/go/tools/cmd/staticcheck@$STATICCHECK_VERSION"
  printf '%s\n' "$version"
  [[ "$version" == 'staticcheck 2026.2.1 (0.8.1)' ]] || fail "expected Staticcheck $STATICCHECK_VERSION"
  printf 'Command: staticcheck -checks=%s ./... (including tests)\n' "$STATICCHECK_CHECKS"
  "$tool" -checks="$STATICCHECK_CHECKS" ./... || fail 'Staticcheck: resolve the file:line diagnostic; see docs/development/go-quality.md'
}

case "${1:-all}" in
  all) format; tidy; analyze ;;
  format) format ;;
  tidy) tidy ;;
  staticcheck) analyze ;;
  *) fail 'usage: check-go-quality.sh [all|format|tidy|staticcheck]' ;;
esac
printf 'PASS: Go quality (%s)\n' "${1:-all}"
