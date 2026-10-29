// Package linearissues adapts the Linear Issues capability without leaking
// Linear transport or GraphQL into Work Item application semantics.
//
// Resource format: team key only (e.g. "AXM"). ExternalID is the Linear
// issue identifier (e.g. "AXM-39") and must start with resource+"-".
// URLs must be https://linear.app/<workspace>/issue/<identifier>[/...].
//
// Transport: GraphQL over HTTPS to api.linear.app. Auth is LINEAR_API_KEY
// from the process environment at call time (never hardcoded). Optional
// AXIOM_LINEAR_BIN points at an absolute helper binary that speaks the same
// request/response contract used in tests (stdin JSON in, stdout JSON out).
package linearissues
