## stageflow project

Run Project Mode scan using .stageflow/config.yaml

```
stageflow project [path]
```

### Options

```
      --category string       Filter displayed findings by category (comma-separated: accessibility,performance,seo,security,best-practices)
      --fail-on string        Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)
      --group-by string       Group findings by: none, category, scanner (default: category for markdown, none for text)
  -h, --help                  help for project
      --max-issues int        Max issues to include in output (0 = unlimited) (default 200)
      --max-occurrences int   Max occurrences per issue to display (0 = unlimited) (default 3)
      --severity string       Filter displayed findings by severity (comma-separated: critical,serious,moderate,minor,info)
      --summary-only          Only show summary, skip detailed findings
      --timeout duration      Max total time (dev + scan) (default 10m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI
* [stageflow project create](stageflow_project_create.md)	 - Create a remote project
* [stageflow project delete](stageflow_project_delete.md)	 - Delete a remote project
* [stageflow project doctor](stageflow_project_doctor.md)	 - Validate project config and dev readiness without scanning
* [stageflow project hosted](stageflow_project_hosted.md)	 - Run hosted project scan from .stageflow config
* [stageflow project init](stageflow_project_init.md)	 - Create .stageflow config and setup guide
* [stageflow project list](stageflow_project_list.md)	 - List remote projects
* [stageflow project promote](stageflow_project_promote.md)	 - Set a job as the project baseline
* [stageflow project show](stageflow_project_show.md)	 - Show remote project details
* [stageflow project update](stageflow_project_update.md)	 - Update a remote project

