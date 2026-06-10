## stageflow dev

Scan your local dev server from .stageflow/config.yaml

### Synopsis

Manage the local dev-server scan loop configured in .stageflow/config.yaml.

Run `stageflow dev init` once to scaffold the config, `stageflow dev doctor` to
validate it, and `stageflow dev scan` to start your dev server, scan it, and
report the results.

```
stageflow dev [flags]
```

### Options

```
  -h, --help   help for dev
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI
* [stageflow dev doctor](stageflow_dev_doctor.md)	 - Validate the config and dev-server readiness without scanning
* [stageflow dev init](stageflow_dev_init.md)	 - Scaffold .stageflow/config.yaml and a setup guide
* [stageflow dev scan](stageflow_dev_scan.md)	 - Start the dev server, scan it, and report results

