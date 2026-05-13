## stageflow project hosted

Run hosted project scan from .stageflow config

```
stageflow project hosted [path]
```

### Options

```
      --category string       Filter displayed findings by category (comma-separated: accessibility,performance,seo,security,best-practices)
      --fail-on string        Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)
      --group-by string       Group findings by: none, category, scanner (default: category for markdown, none for text)
  -h, --help                  help for hosted
      --max-issues int        Max issues to include in output (0 = unlimited) (default 200)
      --max-occurrences int   Max occurrences per issue to display (0 = unlimited) (default 3)
      --severity string       Filter displayed findings by severity (comma-separated: critical,serious,moderate,minor,info)
      --summary-only          Only show summary, skip detailed findings
      --timeout duration      Max wait time (default 5m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow project](stageflow_project.md)	 - Run Project Mode scan using .stageflow/config.yaml

