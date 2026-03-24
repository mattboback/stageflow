# Security Policy

Thanks for helping keep StageFlow safe.

## Supported versions

StageFlow is still moving quickly. Security fixes land on `main` first and, when applicable, in the latest tagged release. Older releases are not maintained.

| Version        | Supported |
| -------------- | --------- |
| `main`         | ✅        |
| Latest release | ✅        |
| Older releases | ❌        |

## Reporting a vulnerability

Please report suspected vulnerabilities privately by emailing **security@stageflow.org**.

Do **not** open a public GitHub issue for an unpatched vulnerability.

Include as much of the following as you can:

- what the issue is and why it matters
- clear reproduction steps, payloads, or proof-of-concept details
- the affected commit, branch, or release if known
- whether the issue affects local-only setups, self-hosted deployments, or the public demo
- any mitigation ideas or relevant logs

If you encrypt sensitive material, mention that in the email so we can coordinate a safe reply path.

## What to expect

- We will acknowledge receipt as soon as we can.
- If you have not heard back within 3 business days, please send a follow-up.
- After triage, we will let you know whether we could reproduce the issue and how we plan to handle disclosure.

StageFlow does not currently run a public bug bounty program.

## What belongs here

Please use the private security channel for issues such as:

- SSRF or network-boundary bypasses
- sandbox, container, or scanner-isolation escapes
- unsafe file handling, archive extraction, or path traversal
- authentication, authorization, or secret exposure issues
- remote code execution or privilege-escalation bugs

For non-sensitive defects, scanner false positives, and general feature requests, use the public issue tracker instead.

## Current security boundaries

These are important design boundaries today:

- strict URL intake validation and SSRF guardrails in the Platform API
- archive extraction limits to reduce zip-bomb and path-traversal risk
- per-job pod or container isolation for scanner execution

If you find a bypass to one of these boundaries, please report it privately.

## Responsible testing

Please avoid testing against systems you do not own or have explicit permission to assess. If a report involves the public `stageflow.org` deployment, keep testing minimal and non-destructive.
