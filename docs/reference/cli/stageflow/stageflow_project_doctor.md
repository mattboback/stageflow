## stageflow project doctor

Validate project config and dev readiness without scanning

```
stageflow project doctor [path]
```

### Options

```
  -h, --help               help for doctor
      --skip-dev           Skip starting dev server and readiness checks
      --timeout duration   Max total time for doctor checks (default 2m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow project](stageflow_project.md)	 - Run project-mode scan using .stageflow/config.yaml

