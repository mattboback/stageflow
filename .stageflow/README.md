# StageFlow project setup

This folder configures `stageflow project` for this repository.

## Quick setup

1. Open `config.yaml` in this folder.
2. Set `dev.start.cmd` to the command that starts your app.
3. Set `dev.ready.url` to the URL that returns HTTP 2xx or 3xx when your app is ready.
4. Set `scan.urls` to the page URLs you want scanned.
5. Run `stageflow project` again.

## Example dev commands

- npm: `cmd: ["npm", "run", "dev"]`
- bun: `cmd: ["bun", "run", "dev"]`
- pnpm: `cmd: ["pnpm", "dev"]`
- yarn: `cmd: ["yarn", "dev"]`

## Localhost/private scans

For local targets like `localhost` and `127.0.0.1`:

1. Start the StageFlow local overlay:
   - `just dev up local`
   - `just dev init local`
2. Re-run `stageflow project`.

## Troubleshooting

- If you see "ENOENT" for your dev command, verify `dev.start.cmd` and `dev.start.cwd`.
- If readiness times out, verify `dev.ready.url` responds while your app is running.
