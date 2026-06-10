## stageflow project

Manage remote projects and scan them against their baselines

### Synopsis

Manage projects registered on a StageFlow API.

A project stores target URLs, scanner configuration, and a promoted baseline
server-side. `stageflow project scan` runs a scan against those URLs and diffs
the results against the baseline, making regressions visible in CI.

```
stageflow project [flags]
```

### Options

```
  -h, --help   help for project
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI
* [stageflow project create](stageflow_project_create.md)	 - Create a remote project
* [stageflow project delete](stageflow_project_delete.md)	 - Delete a remote project
* [stageflow project list](stageflow_project_list.md)	 - List remote projects
* [stageflow project promote](stageflow_project_promote.md)	 - Promote a scan job to be the project baseline
* [stageflow project scan](stageflow_project_scan.md)	 - Scan a remote project and diff against its baseline
* [stageflow project show](stageflow_project_show.md)	 - Show remote project details
* [stageflow project update](stageflow_project_update.md)	 - Update a remote project

