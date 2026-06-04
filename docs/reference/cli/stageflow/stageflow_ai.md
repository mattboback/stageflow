## stageflow ai

Run the experimental AI navigator

### Synopsis

Run the experimental AI navigator scanner against one URL.

This command requires the target StageFlow API to be configured with an AI provider such as OpenRouter.

```
stageflow ai <url> <objective> [flags]
```

### Options

```
      --allow-private-targets   Allow private/loopback targets
      --expand-provenance       Show full provenance JSON
  -h, --help                    help for ai
      --timeout duration        Max wait time (default 10m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI

