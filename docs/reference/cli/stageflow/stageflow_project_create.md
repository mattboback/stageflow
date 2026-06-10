## stageflow project create

Create a remote project

```
stageflow project create <slug> [flags]
```

### Options

```
  -h, --help              help for create
      --name string       Display name (defaults to slug)
      --scanner strings   Scanner module (repeatable; omit for all)
      --url strings       Target URL (repeatable)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow project](stageflow_project.md)	 - Manage remote projects and scan them against their baselines

