## stageflow project scan

Scan a remote project and diff against its baseline

### Synopsis

Run a scan of a remote project's configured URLs and compare the results
against its promoted baseline.

Pass the project slug as an argument, or omit it to use `stageflow.project`
from .stageflow/config.yaml in the current repository.

```
stageflow project scan [slug]
```

### Options

```
      --fail-on string     Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)
  -h, --help               help for scan
      --max-issues int     Max issues to include in output (0 = unlimited) (default 200)
      --timeout duration   Max wait time (default 5m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow project](stageflow_project.md)	 - Manage remote projects and scan them against their baselines

