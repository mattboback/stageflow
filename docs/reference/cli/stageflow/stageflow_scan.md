## stageflow scan

Scan one or more URLs and report the results

```
stageflow scan <url>...
```

### Options

```
      --allow-private-targets   Allow private/loopback targets (requires API instance to permit it)
      --auth-recipe string      Path to a YAML/JSON form-auth recipe (Provenance.auth.form shape). Step values may be literal strings or {from_env: NAME} references.
      --auth-state string       Path to a Playwright storage-state JSON captured via stageflow auth capture. Uploaded under the job's MinIO prefix and referenced from Provenance.auth.
      --fail-on string          Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)
  -h, --help                    help for scan
      --max-issues int          Max issues to include in output (0 = unlimited) (default 200)
      --scanner strings         Scanner module (repeatable or comma-separated) (default [axe,lighthouse,seo,link-checker])
      --screenshot              Capture screenshots (default true)
      --timeout duration        Max wait time (default 5m0s)
```

### Options inherited from parent commands

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI

