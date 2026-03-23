## stageflow diff

Compare a current scan against a saved baseline

```
stageflow diff <baseline.json> <current.json | url>
```

### Options

```
      --fail-on-new string[="any"]   Exit 1 if any NEW issue meets threshold (critical, serious, moderate, minor, info) or 'any'
      --fail-on-regression           Exit 1 if score dropped or new issues appeared
  -h, --help                         help for diff
      --timeout duration             Max wait time for live scan (default 5m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI

