## stageflow project

Run project-mode scan using .stageflow/config.yaml

```
stageflow project [path]
```

### Options

```
  -h, --help               help for project
      --max-issues int     Max issues to include in output (0 = unlimited) (default 200)
      --timeout duration   Max total time (dev + scan) (default 10m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI
* [stageflow project doctor](stageflow_project_doctor.md)	 - Validate project config and dev readiness without scanning
* [stageflow project init](stageflow_project_init.md)	 - Create .stageflow config and setup guide

