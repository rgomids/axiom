package cli

import "io"

const helpText = `Lingo — Axiom local control plane

Usage:
  lingo [--human|--json] <command>
  lingo help

Commands:
  first-run
  runtime codex install|status
  project configure|show|resolve|init|validate|reopen|update|install
  work-item create|select|show|comment|complete
  workflow start|advance|resume|status|evidence

Stable Codex skill mapping:
  $axiom-project-configure -> lingo --json project configure
  $axiom-project-show      -> lingo --json project show
  $axiom-work-item-create  -> lingo --json work-item create|select
  $axiom-work-item-run     -> lingo --json workflow start|advance|resume
  $axiom-work-item-status  -> lingo --json workflow status|evidence

Use --json for machine-readable output. Default and --human output are readable
status summaries. Mutation authority remains explicit through
--authorize-external or --authorize-local.
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
