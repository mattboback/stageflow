# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `open-graph` scanner: social preview and metadata validation (og:title, og:image, Twitter cards)
- `spelling-grammar` scanner: AI-assisted content quality analysis
- Standalone HTML report generation in scanner runtime (self-contained, portable output)

### Changed
- Svelte 5 runes migration: state and props use `$state`, `$derived`, `$effect` throughout
- Monorepo orchestration consolidated; import paths and module boundaries clarified
- Public-facing docs tightened around CLI setup, project mode, and operational references

### Chore
- Pre-commit framework replaces lint-staged/Husky for formatting and secret scanning

## [0.1.0] - 2026-02-03
- Initial open-source release of StageFlow
- Implementation of axe, lighthouse, seo, security-headers, link-checker, and ai-navigator scanners
- Full Svelte 5 frontend with realtime SSE integration
- Core NATS JetStream and Podman orchestration engine
