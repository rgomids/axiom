package cli

// commandRequirements describes presence requirements for command discovery.
// It reuses skill requirements without changing their executable validation.
// Conditions labelled application describe effect admission, not parser checks.
func commandRequirements(operation action) []inputRequirement {
	rules := skillRequirements(operation, requestInput{})
	switch operation {
	case windowsPermissionsRestoreAction:
		return []inputRequirement{{name: windowsBackupFlag, required: true}, {name: windowsApproveFlag, required: true}}
	case initAction:
		return []inputRequirement{{name: "slug", required: true}, {name: "name", required: true}}
	case reopenAction:
		return []inputRequirement{{name: "slug", required: true}}
	case updateAction:
		return []inputRequirement{{name: "slug", required: true}, {name: "name", required: true}}
	case installAction:
		return []inputRequirement{
			{name: "source", required: true},
			{name: "repository", when: "application: one binding for each Repository declared by the manifest"},
		}
	case validateAction:
		return []inputRequirement{
			{name: "slug", when: "--project is absent (historical form)"},
			{name: "project", when: "--slug is absent (installed Project form)"},
		}
	case configureAction:
		return append(rules,
			inputRequirement{name: "project-id", when: "editing and any replay field (--project-id, --preview-digest, --authorize-local) is supplied"},
			inputRequirement{name: "preview-digest", when: "editing and any replay field is supplied; application: publishing a reviewed CREATE"},
			inputRequirement{name: "authorize-local", when: "editing and any replay field is supplied (must be true); application: publishing a reviewed CREATE"},
		)
	case compatibilityBackupAction, compatibilityExportAction:
		rules = append(rules, inputRequirement{name: "target", required: true})
	case artifactRetireAction:
		rules = append(rules, inputRequirement{name: "artifact", required: true})
	case upgradeAction:
		rules = append(rules,
			inputRequirement{name: "archive", required: true},
			inputRequirement{name: "checksums", required: true},
			inputRequirement{name: "bin-dir", required: true},
			inputRequirement{name: "receipt-dir", required: true},
		)
	case recoveryApplyAction:
		return []inputRequirement{{name: "preview-digest", required: true}, {name: "authorize-local", required: true, when: "must be true"}}
	case runtimeProfilePreviewAction:
		return []inputRequirement{
			{name: "project", when: "no effective Project context is available"},
			{name: "role", required: true},
			{name: "complexity", required: true},
			{name: "capabilities", required: true},
		}
	case "context_default-set", "context_session-set":
		return []inputRequirement{
			{name: "selector", required: true},
			{name: "authorize-local", when: "application: applying the preference mutation (must be true)"},
		}
	case "context_default-clear", "context_session-clear", "context_session-end":
		return []inputRequirement{{name: "authorize-local", when: "application: applying the preference mutation (must be true)"}}
	}
	switch operation {
	case compatibilityBackupAction, compatibilityExportAction, artifactCleanupAction, artifactRetireAction, upgradeAction:
		rules = append(rules,
			inputRequirement{name: "preview-digest", when: "--authorize-local is true"},
			inputRequirement{name: "authorize-local", when: "application: applying the reviewed local effect (must be true)"},
		)
	}
	return rules
}
