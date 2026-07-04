## stageflow stack down

Stop the local stack (podman compose down)

```
stageflow stack down
```

### Options

```
  -h, --help   help for down
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --env string       Compose overlay: dev or local (default "dev")
      --format string    Output format: text, markdown, or json (default "text")
      --project string   Compose project name (default: $COMPOSE_PROJECT_NAME or stageflow_dev)
```

### SEE ALSO

* [stageflow stack](stageflow_stack.md)	 - Manage the local self-hosted StageFlow compose stack

