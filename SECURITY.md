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

The public no-account demo's scan-data handling and 24-hour object lifecycle
are documented in [docs/privacy.md](docs/privacy.md). Do not use the hosted
demo for confidential builds or sensitive authenticated targets.

## Scope

The following areas are in scope for security reports:

- **SSRF protections** in URL intake (`services/platform-api/internal/api/security.go`)
- **Archive extraction safety** (ZIP bomb, path traversal, entry limits in `services/archive-extractor`)
- **API authentication and authorization** (API key middleware, orchestrator token auth)
- **Container isolation** (Podman pod boundaries, scanner runtime sandboxing)
- **Secret handling** (credential exposure in logs, config, or artifacts)
- **Dependency vulnerabilities** in Go modules or npm packages

Current narrowly scoped audit waivers, their owners, and removal triggers are
listed in [docs/dependency-exceptions.md](docs/dependency-exceptions.md).

## Residual DNS Rebinding Risk

StageFlow validates scan targets for SSRF policy at submission time and in scanner-runner runtime policy checks, but the headless browser resolves hostnames independently when it loads a page or follows browser-level network activity. StageFlow does not currently pin DNS TTLs or intercept browser connections to force every request through a connection-level private-network check.

For deployments that scan untrusted live URLs with `allow_private_targets=false`, treat DNS rebinding as a residual risk and apply defense in depth:

- Isolate scanner pods in a private network namespace, for example with `POD_NETNS_MODE`, so scanner egress cannot reach sensitive host or cluster networks by default.
- Use an internal DNS resolver that blocks public names from resolving to RFC1918, loopback, link-local, ULA, and other reserved internal ranges.
- Keep scan job lifetimes short so DNS rebinding windows remain small.

Out of scope:

- The hosted demo at `stageflow.org` (report infrastructure issues separately)
- Rate limiting and WAF at the edge/proxy layer (deployment-specific)
- Social engineering or phishing attacks
