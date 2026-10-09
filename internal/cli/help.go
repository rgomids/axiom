package cli

import "io"

const helpText = `Axiom — Lingo local control plane

Usage:
  axiom [--human|--json] <command>
  axiom [--human|--json] [--session <id>] <command>
  axiom help

Commands:
  skill inspect <skill-name>
  first-run
  runtime codex install|status|auth
  runtime claude install|status|auth
  runtime profile validate
  runtime profile preview --project <uuid-or-slug> --role <token> --complexity <token>
    --capabilities <comma-list> [--runtime codex|claude]
  project configure|list|show|resolve|validate|archive|reactivate
  project init|reopen|update|install (historical)
  project context show [--selector <uuid-or-slug>]
  project context default-set|session-set --selector <uuid-or-slug> --authorize-local
  project context default-clear|session-clear|session-end --authorize-local
  integration list|show|validate|disable|enable|remove
  work-item create|select|list|show|update|comment|close|reopen|complete
  workflow start|advance|fact|resume|status|evidence|list|reconcile
  compatibility inspect|backup|export
  artifact cleanup|retire
  recovery inspect|apply
  upgrade --archive <path> --checksums <path> --bin-dir <dir> --receipt-dir <dir>

Stable Runtime skill mapping:
  $axiom-project           -> domain operations configure|list|show|validate|
                              archive|reactivate|integration
  $axiom-work-item         -> domain operations create|run|status|list|show|
                              update|comment|close|reopen

first-run finds Codex and Claude by their executables on PATH and installs or
upgrades Axiom's user-global skills for each one found (Codex:
$HOME/.agents/skills; Claude: <CLAUDE_CONFIG_DIR or ~/.claude>/skills). It never
installs a Runtime or touches credentials; no Runtime found is success.

runtime <codex|claude> auth is the read-only subscription authentication
preflight: it runs only the vendor version and login-status commands with this
shell's environment, inspects API-key/provider overrides in environment and
configuration, and reports names only. It never logs in, runs inference or
reads credential values; a reported login does not prove usability or billing.

Runtime profile validation reads local configuration without changing state,
invoking a runtime, or probing authentication. It accepts no flags or arguments.

Maintenance commands are read-only previews unless repeated with the exact
--preview-digest and --authorize-local. Backup/export targets must be absent
absolute paths. Recovery applies one plan selected by its digest.

Use --json for machine-readable output. Default and --human output are readable
status summaries. Mutation authority remains explicit through
--authorize-external or --authorize-local.

project configure without --project bootstraps a new schema v3 Project from
explicit Repository locations (--repository <key>=<absolute-path> or a bare
absolute path). Lingo reads each location's Git metadata and file names only:
no Git process, network, CWD, or remote-name priority (origin is not special).
Ambiguous remotes block until --repository-remote <key>=<locator>|none.
Optional CREATE intent: --runtime/--model-profile (local candidates only, never
defaulted), --runtime-preference <role>/<complexity>=<profile>, --technology
<key>=<value>, --remove-technology <key>, --documentation <key>=repository:<repo>/
<path>|local-file:<absolute-path>, --business-context, --context-source,
--glossary <key>=<term>:<definition>. An already
configured slug or --project-id fails without changes. project configure
--project <project-uuid-or-slug> previews an edit of an existing Project:
  --name <name>, --work-item-provider <id> | --remove-work-item-provider
  --repository <key>=<absolute-path> (repeatable attach/update)
  --remove-repository <key> (repeatable detach; never deletes a working copy)
Omitted values are preserved. Edit rejects --slug (rename). Publish the exact
reviewed edit by repeating it with the returned --project-id, --preview-digest
and --authorize-local; a partial replay tuple is refused.

project archive|reactivate --project <uuid-or-slug> change only this machine's
Project state (archive blocks operational work, never inspection or
administration); project list hides archived Projects unless
--include-archived. integration disable|enable change only this machine's use
of a declared Integration; integration remove drops the portable declaration
through the reviewed edit. None of them revokes credentials or touches a
Provider. workflow list discovers Executions; Execution cancellation is not
supported.

project validate --slug <slug> (or --project <uuid-or-slug>, reading the
installed Project's recorded source) checks the portable Project and reports
read-only operation readiness (work-item, execution): ready, partial or
blocked, with exact blocker and warning codes. Work Item operations and
workflow start/resume enforce the same blockers before any effect; readiness
never grants authority.

project install --source <dir> records an authored manifest, such as one that
declares a Runtime/Profile policy; each Repository it declares needs exactly one
--repository <key>=<absolute-path>.

Strict selector vocabulary:
  --project <project-uuid-or-slug>
  --repository <project-scoped-key>
  --work-item github:<owner>/<repository>#<number>
  --execution <execution-id> (all workflow operations except start)
  --runtime codex|claude (optional policy constraint for workflow start)
  --role <token> --complexity <token> --capabilities <comma-list> (workflow start policy inputs)
  --runtime-preview <digest> (start only after exact preview and fresh validation)

Workflow start without --runtime-preview previews only. No Runtime is assumed.
Lingo observes each configured Runtime itself: its executable on PATH (identity
only, never run) and Axiom's skill integration, the only capability it proves
(axiom-skills). Any other required capability stays unproven and blocks.

Fully specified selectors require no prompt. Missing selectors may be prompted;
unknown, duplicate, conflicting, or ambiguous selectors fail validation without
CWD, Git, Provider, or Runtime fallback.

Workflow gates:
  workflow status reports workflow.gateAction and workflow.gateCommand.
  workflow advance --expected-revision <revision> --automatic
    evaluates Intake only; cannot combine with --gate/--outcome/--reference/--next.
  workflow advance --expected-revision <revision> --gate <gate> --outcome pass|fail
    records observed technical results; never infers human approval.
  workflow fact --expected-revision <revision> --fact <fact> --active
    --reference <kind>:<reference>:<sha256> --authorize-local
    records explicit planning/implementation authority or human acceptance.
  Always pass exact Project, Repository, Work Item and Execution selectors.
`

func Help(writer io.Writer) int {
	if writer == nil {
		return ExitFailure
	}
	if _, err := io.WriteString(writer, helpText); err != nil {
		return ExitFailure
	}
	return ExitSuccess
}
