package capabilityproof

import "time"

type EnvelopePolicy struct {
	Variant         string
	StateRoot       string
	RepositoryRoots []string
	VendorHomes     []string
}

type EnvelopeLimits struct {
	Timeout  time.Duration `json:"timeout"`
	MaxFiles int           `json:"maxFiles"`
	MaxBytes int64         `json:"maxBytes"`
}

type EnvelopeManifest struct {
	Variant         string            `json:"variant"`
	NegativeTargets []string          `json:"negativeTargets"`
	Phase           string            `json:"phase"`
	Version         int               `json:"version"`
	ProbeID         string            `json:"probeId"`
	Nonce           string            `json:"nonce"`
	Revision        string            `json:"creationRevision"`
	CreatedAt       time.Time         `json:"createdAt"`
	AllowedRoots    []string          `json:"allowedRoots"`
	Sentinel        string            `json:"sentinel"`
	Limits          EnvelopeLimits    `json:"limits"`
	Identities      map[string]string `json:"identities"`
}
