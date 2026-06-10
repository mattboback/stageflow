## stageflow auth capture

Launch a non-headless Chromium and write Playwright storage state on exit

### Synopsis

Opens a non-headless Chromium pointed at <url> using `npx playwright open --save-storage`. Log in by hand, then close the browser; the resulting storage-state JSON is written to --output with file mode 0600.

Pass that file to `stageflow scan --auth-state <path>` to attach the captured session to a job.

Requires Node.js and Playwright to be installed locally (npm install -g playwright, or run from a project that already has playwright as a dependency). The CLI never sees the password; only the resulting cookies + localStorage do.

```
stageflow auth capture <url> [flags]
```

### Options

```
  -h, --help                             help for capture
  -o, --output string                    Path to write the storage-state JSON (required). Created with mode 0600.
      --playwright-arg playwright open   Extra argument to forward to playwright open (repeatable, e.g. --playwright-arg=--browser=chromium)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow auth](stageflow_auth.md)	 - Manage authentication artifacts for authenticated scans

