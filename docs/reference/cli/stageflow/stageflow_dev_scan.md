## stageflow dev scan

Start the dev server, scan it, and report results

```
stageflow dev scan [path]
```

### Options

```
      --fail-on string     Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)
  -h, --help               help for scan
      --max-issues int     Max issues to include in output (0 = unlimited) (default 200)
      --timeout duration   Max total time (dev + scan) (default 10m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow dev](stageflow_dev.md)	 - Scan your local dev server from .stageflow/config.yaml

