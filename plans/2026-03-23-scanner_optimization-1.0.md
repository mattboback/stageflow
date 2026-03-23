# Scanner Performance Optimization Analysis

## Root Cause: Hardcoded Concurrency
- `services/scanner-runner/src/worker.ts` currently ignores the scanner manifest's `maxConcurrency` capability.
- `loadConfigFromEnv` falls back to a hardcoded concurrency of 4 for all scanners.

## Bottlenecks Caused
1. **Lightweight scanners throttled**: `open-graph` and `security-headers` support 10 concurrent pages but run at 4.
2. **Heavy scanners thrashing**: `lighthouse` supports 1 concurrent page but runs at 4, opening 4 Playwright tabs that go stale in the internal queue, forcing a slow double-navigation penalty.

## Proposed Fix
Update `worker.ts` to pass `manifest.capabilities.maxConcurrency` as the default concurrency to `loadConfigFromEnv`.