# AGENTS.md — platform/scanner-runner

TypeScript/Bun/Playwright scanner worker. Runs inside per-job pods. Discovers plugins, validates `SCANNER_OPTIONS`, executes scan lifecycle, uploads artifacts to MinIO, publishes events to NATS.

## STRUCTURE

```
src/
├── index.ts               # Process entry point
├── worker.ts              # Plugin resolution, validation, scanner.run() dispatch
├── core/
│   ├── scanner-base.ts    # Abstract ScannerBase (all scanners extend this)
│   ├── types.ts           # ScannerConfig, ScanContext, PageScanResult, StorageProvider
│   ├── browser-manager.ts # Playwright browser/context/navigation lifecycle
│   ├── page-iterator.ts   # Per-page iteration with concurrency + retry
│   ├── event-publisher.ts # NATS: pageCompleted, scanCompleted, scanFailed
│   ├── config-loader.ts   # Load ScannerConfig from env vars
│   ├── plugins/           # Plugin loader, discovery, dynamic import
│   └── storage-provider/  # MinIO StorageProvider implementation
├── scanners/
│   ├── axe/               # axe-core WCAG violations + screenshot integration
│   ├── lighthouse/        # Lighthouse audits
│   ├── seo/               # SEO checks
│   ├── security-headers/  # HTTP security header analysis
│   ├── link-checker/      # Broken link detection
│   └── ai-navigator/      # LLM+vision agent navigator
├── screenshots/axe/       # Axe screenshot strategies (violation capture, page overview)
└── ai/                    # Vision client, page analyzer, action decider
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Plugin resolution + config | `src/worker.ts` |
| Scan lifecycle | `src/core/scanner-base.ts` → `run()` |
| Core types | `src/core/types.ts` |
| Plugin discovery paths | `src/core/plugins/plugin-discovery.ts` |
| Dynamic plugin import | `src/core/plugins/plugin-load.ts` |
| Manifest + schema validation | `src/core/manifest/index.ts` |
| `SCANNER_OPTIONS` validation | `src/worker/worker-validation.ts` |
| Axe screenshot capture | `src/screenshots/axe/violation-capture.ts` |
| AI agent orchestration | `src/scanners/ai-navigator/agent.ts` |

## CONVENTIONS

- All scanners extend `ScannerBase` and implement `scanPage(context: ScanContext): Promise<PageScanResult>`.
- `bun run prepare:contracts` **must** run before build/test (copies generated contract types).
- Built-in manifests: `packages/shared-go/scannercatalog/manifests/*/manifest.json` → copied to `dist/scanners/` at build.
- Plugin search order: `dist/scanners` → `/plugins` (volume) → `$HOME/.stageflow/plugins` → `PLUGIN_PATHS`.
- `NODE_ENV=production` enables strict manifest/configSchema validation.
- Never edit `dist/` — compiled output only. All edits in `src/`.

## NOTES

- `SCANNER_TYPE` env → plugin id or alias. `SCANNER_OPTIONS` JSON → validated against manifest `configSchema`.
- ai-navigator requires `OPENROUTER_API_KEY` env. Never pass `vision.apiKey` inside `SCANNER_OPTIONS`.
- Screenshot policies: `never`, `on-violation`, `always` — configured in `src/config/rule-behaviors.ts`.
- Tests: `bun run test` (auto-runs prepare:contracts). Vitest; coverage thresholds enforced in `vitest.config.ts`.
