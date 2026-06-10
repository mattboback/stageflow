## stageflow project update

Update a remote project

```
stageflow project update <slug> [flags]
```

### Options

```
  -h, --help              help for update
      --name string       Display name
      --scanner strings   Scanner module (repeatable; replaces all scanners)
      --url strings       Target URL (repeatable; replaces all URLs)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow project](stageflow_project.md)	 - Manage remote projects and scan them against their baselines

