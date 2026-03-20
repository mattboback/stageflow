# Security Policy

## Supported Versions

StageFlow is currently in early active development. We apply security patches to the `main` branch and the latest tagged release.

| Version | Supported          |
| ------- | ------------------ |
| `main`  | :white_check_mark: |
| Latest release | :white_check_mark: |
| Older releases | :x:                |

## Reporting a Vulnerability

We take the security of StageFlow seriously. If you discover a security vulnerability, please report it privately rather than opening a public issue.

### How to Report

Please send an email to **security@stageflow.org** with the following details:

- A description of the vulnerability and its potential impact.
- Steps to reproduce the issue (including any necessary configuration or payloads).
- The version(s) of StageFlow affected.
- Any potential mitigation or remediation steps you are aware of.

### Response Timeline

We aim to respond to all security reports within 48 hours. After initial triage, we will keep you updated on the progress of our investigation and the timeline for releasing a fix.

## Security Features

StageFlow is designed with several security boundaries:
- Strict SSRF protections for URL intake.
- Safe archive extraction limits to prevent zip bombs and directory traversal.
- Isolated container execution for scanner plugins.

If you find a bypass to any of these boundaries, please report it immediately.
