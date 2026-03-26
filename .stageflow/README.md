# StageFlow self-scan for this repo

This config is already wired for `clients/web`.

- It starts the frontend with `just run clients/web`.
- It overrides `VITE_API_URL` so the dev server talks to the local StageFlow API.
- It waits for `http://127.0.0.1:5173`.
- It scans that local app with `axe,lighthouse,seo,link-checker`.

## Quick path

```bash
cp .env.example .env
just setup
just images
just dev up local
just dev init local
just cli-install
stageflow project doctor .
stageflow project .
```

Use `stageflow project doctor .` first if you only want to verify the dev-server wiring without running a scan yet.

For the general project-mode reference, see [`docs/PROJECT_MODE.md`](../docs/PROJECT_MODE.md).
