## stageflow dev doctor

Validate the config and dev-server readiness without scanning

```
stageflow dev doctor [path]
```

### Options

```
  -h, --help               help for doctor
      --skip-dev           Skip starting dev server and readiness checks
      --timeout duration   Max total time for doctor checks (default 2m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow dev](stageflow_dev.md)	 - Scan your local dev server from .stageflow/config.yaml

