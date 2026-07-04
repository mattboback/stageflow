## stageflow stack

Manage the local self-hosted StageFlow compose stack

### Synopsis

Start, stop, and inspect the Podman Compose stack used to self-host StageFlow locally — the same compose files `just dev`/`just demo` drive.

Run from inside a stageflow checkout with `.env` configured and job images already built (`just images`); `stageflow stack` manages the compose lifecycle, it does not build images or scaffold config.

```
stageflow stack [flags]
```

### Options

```
      --env string       Compose overlay: dev or local (default "dev")
  -h, --help             help for stack
      --project string   Compose project name (default: $COMPOSE_PROJECT_NAME or stageflow_dev)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI
* [stageflow stack down](stageflow_stack_down.md)	 - Stop the local stack (podman compose down)
* [stageflow stack status](stageflow_stack_status.md)	 - Show compose service status (podman compose ps)
* [stageflow stack up](stageflow_stack_up.md)	 - Start the local stack (podman compose up -d)

