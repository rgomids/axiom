package cli

import (
	"flag"
	"io"
	"runtime"
	"strings"
)

// commandDefinition is the public presentation tree used by routing and help.
// It holds no application service, Project state or domain behavior.
type commandDefinition struct {
	name, summary string
	operation     action
	children      []commandDefinition
}

func leaf(name, summary string, operation action) commandDefinition {
	return commandDefinition{name: name, summary: summary, operation: operation}
}
func group(name, summary string, children ...commandDefinition) commandDefinition {
	return commandDefinition{name: name, summary: summary, children: children}
}

var publicCommands = commandTree()

func commandTree() commandDefinition {
	root := group("axiom", "Axiom — Lingo local control plane",
		leaf("help", "Show root help", "help"),
		leaf("version", "Inspect build information", "version"),
		leaf("first-run", "Install Axiom skills for detected Runtimes", firstRunAction),
		group("skill", "Inspect embedded Runtime skills", leaf("inspect", "Discover skill arguments", skillInspectAction)),
		group("runtime", "Manage Runtime integrations and profile policy",
			group("codex", "Codex integration", leaf("install", "Install user-global skills", codexInstallAction), leaf("status", "Inspect integration", codexStatusAction), leaf("auth", "Inspect subscription authentication (read-only)", codexAuthAction)),
			group("claude", "Claude integration", leaf("install", "Install user-global skills", claudeInstallAction), leaf("status", "Inspect integration", claudeStatusAction), leaf("auth", "Inspect subscription authentication (read-only)", claudeAuthAction)),
			group("profile", "Inspect local profile policy", leaf("validate", "Validate local configuration", runtimeProfileValidateAction), leaf("preview", "Preview Runtime resolution", runtimeProfilePreviewAction))),
		group("project", "Configure and inspect Projects",
			leaf("configure", "Preview CREATE or EDIT; publish with exact authority", configureAction),
			leaf("list", "List configured Projects", listAction),
			leaf("show", "Inspect configured Project", showAction),
			leaf("resolve", "Resolve Project identity", resolveAction),
			leaf("validate", "Validate portable or installed Project", validateAction),
			leaf("archive", "Preview or apply machine-local archive", projectArchiveAction),
			leaf("reactivate", "Preview or apply machine-local reactivation", projectReactivateAction),
			leaf("init", "Initialize authored Project (historical)", initAction),
			leaf("reopen", "Reopen authored Project (historical)", reopenAction),
			leaf("update", "Update authored Project (historical)", updateAction),
			leaf("install", "Install authored manifest (historical)", installAction),
			group("context", "Inspect or select local Project context",
				leaf("show", "Inspect effective context", "context_show"),
				leaf("default-set", "Set local default", "context_default-set"),
				leaf("default-clear", "Clear local default", "context_default-clear"),
				leaf("session-set", "Set caller session override", "context_session-set"),
				leaf("session-clear", "Clear caller session override", "context_session-clear"),
				leaf("session-end", "End caller session selection", "context_session-end"))),
		group("integration", "Inspect and administer declared Integrations",
			leaf("list", "List Integrations", integrationListAction), leaf("show", "Inspect Integration", integrationShowAction),
			leaf("validate", "Validate static configuration", integrationValidateAction), leaf("disable", "Preview or disable local use", integrationDisableAction),
			leaf("enable", "Preview or enable local use", integrationEnableAction), leaf("remove", "Preview or remove portable declaration", integrationRemoveAction)),
		group("work-item", "Manage linked Work Items",
			leaf("create", "Preview or publish Work Item", workItemCreateAction), leaf("select", "Preview or link existing Work Item", workItemSelectAction),
			leaf("list", "List local links", workItemListAction), leaf("show", "Inspect linked Work Item", workItemShowAction),
			leaf("update", "Preview or update Provider content", workItemUpdateAction), leaf("comment", "Preview or publish comment", workItemCommentAction),
			leaf("close", "Preview or close Provider Work Item", workItemCloseAction), leaf("reopen", "Preview or reopen Provider Work Item", workItemReopenAction),
			leaf("complete", "Compatibility spelling of close", workItemCompleteAction)),
		group("workflow", "Inspect and drive Executions with explicit authority",
			leaf("start", "Preview Runtime resolution or start Execution", workflowStartAction), leaf("advance", "Record technical result or evaluate Intake", workflowAdvanceAction),
			leaf("fact", "Record explicit lifecycle fact", workflowFactAction), leaf("resume", "Resume committed Execution", workflowResumeAction),
			leaf("status", "Inspect Execution and next gate action", workflowStatusAction), leaf("evidence", "Inspect Execution Evidence", workflowEvidenceAction),
			leaf("list", "List local Executions", workflowListAction), leaf("reconcile", "Preview or project lifecycle metadata", workflowReconcileAction)),
		group("compatibility", "Inspect and preserve local compatibility",
			leaf("inspect", "Inspect compatibility", compatibilityInspectAction), leaf("backup", "Preview or back up state", compatibilityBackupAction), leaf("export", "Preview or export state", compatibilityExportAction)),
		group("artifact", "Maintain owned artifacts", leaf("cleanup", "Preview or clean artifacts", artifactCleanupAction), leaf("retire", "Preview or retire one artifact", artifactRetireAction)),
		group("recovery", "Inspect and apply exact recovery plans", leaf("inspect", "Inspect recovery", recoveryInspectAction), leaf("apply", "Apply digest-selected plan", recoveryApplyAction)),
		leaf("upgrade", "Preview or upgrade owned release installation", upgradeAction),
	)

	if runtime.GOOS == "windows" {
		root.children = append(root.children, windowsPermissionsCommand())
	}
	return root
}

func commandChild(parent *commandDefinition, name string) *commandDefinition {
	for i := range parent.children {
		if parent.children[i].name == name {
			return &parent.children[i]
		}
	}
	return nil
}

// resolveCommand consumes command words only, leaving flag/value parsing to the
// established parser. An incomplete or unknown path returns its closest group.
func resolveCommand(args []string) (*commandDefinition, []string, []string, bool) {
	current := &publicCommands
	var path []string
	for len(current.children) != 0 && len(args) != 0 {
		child := commandChild(current, args[0])
		if child == nil {
			return current, path, args, false
		}
		path = append(path, child.name)
		current, args = child, args[1:]
	}
	return current, path, args, current.operation != ""
}

func commandByAction(operation action) (*commandDefinition, []string) {
	var visit func(*commandDefinition, []string) (*commandDefinition, []string)
	visit = func(c *commandDefinition, path []string) (*commandDefinition, []string) {
		if c.operation == operation {
			return c, path
		}
		for i := range c.children {
			child := &c.children[i]
			if found, result := visit(child, append(append([]string(nil), path...), child.name)); found != nil {
				return found, result
			}
		}
		return nil, nil
	}
	return visit(&publicCommands, nil)
}

func commandFlagSet(operation action) *flag.FlagSet {
	switch {
	case operation == windowsPermissionsRestoreAction:
		return windowsPermissionsFlagSet()
	case strings.HasPrefix(string(operation), "context_"):
		return projectContextFlagSet(&ProjectContextInput{Action: strings.TrimPrefix(string(operation), "context_")})
	case operation == runtimeProfilePreviewAction:
		return runtimePreviewFlagSet(&RuntimeProfilePreviewInput{}, new(string))
	case maintenanceOperation(operation):
		return maintenanceFlagSet(operation, &MaintenanceInput{})
	case operation == firstRunAction || operation == codexInstallAction || operation == codexStatusAction || operation == claudeInstallAction || operation == claudeStatusAction || operation == codexAuthAction || operation == claudeAuthAction || operation == runtimeProfileValidateAction || operation == skillInspectAction || operation == "help" || operation == "version":
		set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
		set.SetOutput(io.Discard)
		return set
	default:
		return skillFlagSet(operation, &requestInput{})
	}
}

func maintenanceOperation(operation action) bool {
	return operation == upgradeAction || strings.HasPrefix(string(operation), "compatibility_") || strings.HasPrefix(string(operation), "artifact_") || strings.HasPrefix(string(operation), "recovery_")
}

func operationInGroup(operation action, name string) bool {
	c, path := commandByAction(operation)
	return c != nil && len(path) == 2 && path[0] == name
}

func commandMatches(args []string, operation action) bool {
	command, _, _, ok := resolveCommand(args)
	return ok && command.operation == operation
}

// VersionFormat keeps the executable's fixed build-info forms bound to the
// same public tree, without introducing application composition.
func VersionFormat(args []string) (CompletionFormat, bool) {
	mode, rest := parseOutputMode(args)
	if len(rest) != 1 || !commandMatches(rest, "version") {
		return "", false
	}
	if mode == jsonOutput {
		return CompletionJSON, true
	}
	return CompletionHuman, true
}
