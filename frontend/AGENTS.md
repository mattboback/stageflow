# AGENTS.md — frontend

SvelteKit 5 SPA. Runes-based reactivity. Tailwind v4. Static adapter (nginx/Caddy in prod).

## STRUCTURE

```
src/
├── routes/
│   ├── +layout.svelte          # Site layout, global head
│   ├── +page.svelte            # Landing page
│   ├── playground/             # Job submission UI
│   └── scan/[id]/
│       ├── +page.svelte        # Live scan status (SSE)
│       └── report/+page.svelte # Aggregated report view
└── lib/
    ├── api/                    # client.ts (submit/fetch), utils.ts (buildApiUrl)
    ├── stores/                 # Factory stores (.svelte.ts) for job lifecycle
    ├── components/
    │   ├── ui/                 # Primitives: Button, Modal, Tabs, Panel, PageSection…
    │   ├── report/             # ReportShell, IssuesView, IssueDetailModal…
    │   ├── scan-status/        # ScanTerminal, ScanStatusHeader…
    │   └── playground/         # Submission UI components
    ├── report/                 # Report transform helpers (grouping, filters, sorting)
    ├── types/                  # scan.ts, unified-report.ts, report-audience.ts
    └── utils/                  # cn.ts, date.ts, wcag helpers
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Job submission | `src/lib/api/client.ts` → `submitScanJob()` |
| API base URL | `src/lib/api/utils.ts` → `buildApiUrl()` (uses `VITE_API_URL`) |
| Live scan state + SSE | `src/lib/stores/scan-status.svelte.ts` |
| Report data store | `src/lib/stores/scan-report.svelte.ts` |
| SSE constants / retry | `src/lib/stores/scan-status/constants.ts` |
| Scan route (store wiring) | `src/routes/scan/[id]/+page.svelte` |
| Report page composition | `src/lib/components/report/ReportShell.svelte` |
| UI primitives | `src/lib/components/ui/` |

## CONVENTIONS

### Runes
- `$state` — local mutable state; DOM refs; internal store state.
- `$derived` / `$derived.by()` — computed values; `.by()` for heavier computations.
- `$effect` — lifecycle (create/start store on mount, return cleanup fn); DOM side effects.
- Factory stores expose getters + control methods (`start()`, `cleanup()`). Internal state uses `$state`.

### Canonical store wiring pattern (routes)
```ts
let scanStore = $state<ReturnType<typeof createScanStatusStore> | null>(null);
$effect(() => {
  const store = createScanStatusStore(scanId);
  scanStore = store;
  store.start();
  return () => store.cleanup();
});
```

### API errors
- `client.ts` throws `Error` with user-friendly message — catch in UI and surface as alert.
- `readApiErrorMessage` maps 400/413/422/5xx to specific messages.

## ANTI-PATTERNS

- No Svelte writable stores in new code — runes or factory stores only.
- No `any` — extend types or narrow with `unknown`.
- SSE reconnect behavior is intentional; `constants.ts` controls retry limits + polling fallback. Do not remove.

## COMMANDS

```bash
bun run dev           # Dev server (or: just run frontend)
bun run storybook     # Storybook dev server
bun run test-storybook # Storybook interaction + a11y tests
bun run ci            # lint + type-check + test:coverage
bun run lint:fix      # Auto-fix lint issues
bun run test:watch    # Vitest watch mode
```

## NOTES

- `VITE_API_URL` → API base URL (empty = same origin).
- SSE stream at `/api/v1/jobs/:id/stream`. Falls back to polling if `EventSource` fails.
- `ReportShell` uses `goto()` with `replaceState: true` to update query params without navigation.
- Tailwind v4 — uses `@tailwindcss/vite` plugin (no `tailwind.config.js`).
- Static build (`adapter-static`) — no SSR. All data fetching is client-side.
