## stageflow auth

Manage authentication artifacts for authenticated scans

### Synopsis

Subcommands that capture browser session state for use with `stageflow scan --auth-state`.

See docs/architecture.md#authenticated-scanning for the design and trust boundaries.

### Options

```
  -h, --help   help for auth
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI
* [stageflow auth capture](stageflow_auth_capture.md)	 - Launch a non-headless Chromium and write Playwright storage state on exit

