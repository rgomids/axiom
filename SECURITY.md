# Security Policy

## Reporting a vulnerability

Report suspected vulnerabilities privately through GitHub's **Security** tab and its private vulnerability reporting form.

Do not publish unpatched vulnerabilities, reproduction details, exploit code, credentials, customer data, or other sensitive evidence in public issues, discussions, pull requests, or commits.

If private vulnerability reporting is unavailable, contact the repository owner through an established private channel and request reporting instructions without including vulnerability details in a public message.

Include only the information needed to investigate safely:

- affected artifact or workflow;
- impact and preconditions;
- minimal reproduction steps;
- suggested mitigation, when known;
- whether any secret or private data may have been exposed.

Never include live secrets or credentials. Revoke or rotate an exposed credential through its provider before sharing sanitized evidence.

## Initial scope

This policy currently covers:

- the public `rgomids/axiom` repository;
- the Codex agent harness under `.agents/`;
- repository validation scripts;
- versioned product, architecture, decision, research, security, and development documentation;
- repository security configuration on GitHub.

Axiom does not yet contain an application runtime, CLI implementation, deployment, database, or production service. Scope will be revised when those assets exist.

## Disclosure

Allow maintainers reasonable time to validate and remediate a report before public disclosure. Coordinate disclosure timing and redact secrets, private provider data, and identifying customer information from any eventual advisory.

Operational repository rules live in [docs/security/repository-security.md](docs/security/repository-security.md).
