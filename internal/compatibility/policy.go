package compatibility

import "github.com/rgomids/axiom/internal/codexruntime"

// Strategy is the closed forward-transition decision for inspected persisted
// state. Selecting a strategy is a decision only: it grants no authority and
// performs no effect.
type Strategy string

const (
	StrategyDirect                     Strategy = "direct"
	StrategyMigrate                    Strategy = "migrate"
	StrategyPreserveRebuildReconfigure Strategy = "preserve_rebuild_reconfigure"
	StrategyRefuse                     Strategy = "refuse"
)

// Ownership is the installation-ownership fact established by the caller's
// own ownership rules, such as the installer's receipt and binary checks. The
// policy never derives it from persisted state; owned is necessary but never
// sufficient for a strategy other than refuse.
type Ownership string

const (
	Owned     Ownership = "owned"
	Foreign   Ownership = "foreign"
	Ambiguous Ownership = "ambiguous"
)

// Format names a stable persisted-state format.
type Format string

const FormatV1 Format = "v1"

// StableWindow is this release's exact stable persisted-state compatibility
// window: every retained stable format with its one declared forward strategy.
// It is {v1: direct}. A future stable release that introduces a new persisted
// format must declare here, before publication, each earlier stable format it
// retains and that format's strategy; any format not listed is out of window
// and refused. RecognizedPOC is a separate historical transition and is never
// a member of this window.
func StableWindow() map[Format]Strategy {
	return map[Format]Strategy{FormatV1: StrategyDirect}
}

// stableFormat maps a classification to the stable format it positively
// identifies. Only classifications listed here can enter the window.
func stableFormat(classification Classification) (Format, bool) {
	if classification == ValidV1 {
		return FormatV1, true
	}
	return "", false
}

// Outcome is the bounded product-level meaning of a Decision, independent of
// the internal classification taxonomy.
type Outcome string

const (
	OutcomeCompatible          Outcome = "compatible"
	OutcomeAutomaticTransition Outcome = "automatic_transition"
	OutcomeUnsupported         Outcome = "unsupported_state"
	OutcomeUnsafe              Outcome = "unsafe_state"
	OutcomeRecoveryRequired    Outcome = "recovery_required"
)

// Decision is the deterministic forward-transition resolution for exactly one
// inspected report. Callers bind it, with StateDigest, into their own exact
// authority so a change in observed state invalidates the decision.
type Decision struct {
	Strategy       Strategy       `json:"strategy"`
	Outcome        Outcome        `json:"outcome"`
	Reason         string         `json:"reason"`
	Classification Classification `json:"classification"`
	StateDigest    string         `json:"stateDigest"`
}

// Resolve maps installation ownership and one inspected report to exactly one
// strategy. Ownership and schema compatibility are independent inputs, and any
// input the policy does not positively recognize resolves to refuse.
func Resolve(ownership Ownership, report Report) Decision {
	decision := Decision{Strategy: StrategyRefuse, Outcome: OutcomeUnsafe, Classification: report.Classification, StateDigest: report.Digest}
	if ownership != Owned {
		decision.Reason = "installation_not_owned"
		return decision
	}
	if report.Digest == "" {
		decision.Reason = "unverifiable_inspection"
		return decision
	}
	if format, ok := stableFormat(report.Classification); ok {
		strategy, retained := StableWindow()[format]
		if !retained || strategy == StrategyRefuse {
			decision.Outcome, decision.Reason = OutcomeUnsupported, "state_format_outside_window"
			return decision
		}
		decision.Strategy, decision.Reason = strategy, "stable_format_"+string(format)
		decision.Outcome = OutcomeAutomaticTransition
		if strategy == StrategyDirect {
			decision.Outcome = OutcomeCompatible
		}
		return decision
	}
	switch report.Classification {
	case AbsentV1:
		decision.Strategy, decision.Outcome, decision.Reason = StrategyDirect, OutcomeCompatible, "no_persisted_state"
	case RecognizedPOC:
		if !historicalPOCPreconditions(report) {
			decision.Reason = "historical_poc_preconditions_unmet"
			return decision
		}
		decision.Strategy, decision.Outcome, decision.Reason = StrategyPreserveRebuildReconfigure, OutcomeAutomaticTransition, "historical_poc_transition"
	case UnsupportedNewer:
		decision.Outcome, decision.Reason = OutcomeUnsupported, "newer_state_format"
	case UnsupportedOlder:
		decision.Outcome, decision.Reason = OutcomeUnsupported, "state_format_outside_window"
	case RecoveryRequired:
		decision.Outcome, decision.Reason = OutcomeRecoveryRequired, "interrupted_operation"
	case Malformed:
		decision.Reason = "unrecognized_ambiguous_or_unsafe_state"
	default:
		decision.Outcome, decision.Reason = OutcomeUnsupported, "unclassified_state"
	}
	return decision
}

// historicalPOCPreconditions positively re-checks the frozen POC signature and
// a skill set that is neither foreign nor interrupted. Exact authority is not
// decided here: the caller binds the Decision into its own exact preview.
func historicalPOCPreconditions(report Report) bool {
	if report.Reason != "complete_poc_workflow_signature" || report.POCTag != HistoricalPOCTag || report.POCRevision != HistoricalPOCRevision {
		return false
	}
	switch report.SkillSet.State {
	case codexruntime.SkillSetAbsent, codexruntime.SkillSetCurrent, codexruntime.SkillSetUpgradable:
		return true
	}
	return false
}
