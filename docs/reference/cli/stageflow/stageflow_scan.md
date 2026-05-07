## stageflow scan

Submit a scan job and wait for results

```
stageflow scan [url...] [--project <slug>]
```

### Options

```
      --allow-private-targets   Allow private/loopback targets (requires API instance to permit it)
      --fail-on string          Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)
  -h, --help                    help for scan
      --max-issues int          Max issues to include in output (0 = unlimited) (default 200)
      --project string          Scan using a registered project's URLs and config
      --scanners string         Comma-separated scanner modules (default "axe,lighthouse,seo,link-checker")
      --screenshot              Capture screenshots (default true)
      --timeout duration        Max wait time (default 5m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI

