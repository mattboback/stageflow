# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| main    | Yes       |

## Reporting a Vulnerability

If you discover a security vulnerability in StageFlow, please report it responsibly.

**Do not open a public issue.** Instead, use one of these methods:

1. **GitHub Security Advisories**: Use the [private vulnerability reporting](https://github.com/mattboback/stageflow/security/advisories/new) feature.
2. **Email**: Contact the maintainer directly through the email on the [GitHub profile](https://github.com/mattboback).

Please include:

- Description of the vulnerability
- Steps to reproduce
- Affected component (platform-api, orchestrator, scanner-runner, archive-extractor, web app, CLI)
- Potential impact

## Response

You can expect an acknowledgement within 72 hours. Fixes for confirmed vulnerabilities will be committed directly to `main`.

## Scope

The following areas are in scope for security reports:

- **SSRF protections** in URL intake (`services/platform-api/internal/api/security.go`)
- **Archive extraction safety** (ZIP bomb, path traversal, entry limits in `services/archive-extractor`)
- **API authentication and authorization** (API key middleware, orchestrator token auth)
- **Container isolation** (Podman pod boundaries, scanner runtime sandboxing)
- **Secret handling** (credential exposure in logs, config, or artifacts)
- **Dependency vulnerabilities** in Go modules or npm packages

Out of scope:

- The hosted demo at `stageflow.org` (report infrastructure issues separately)
- Rate limiting and WAF at the edge/proxy layer (deployment-specific)
- Social engineering or phishing attacks
