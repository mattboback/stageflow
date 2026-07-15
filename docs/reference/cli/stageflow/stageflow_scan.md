## stageflow scan

Scan URLs, or upload a local build directory / ZIP archive and scan it

### Synopsis

Scan one or more URLs and report the results.

When the argument is a local directory or .zip file, it is uploaded to the
API's ZIP intake and served from an isolated static server for scanning —
no dev server or public URL required (e.g. `stageflow scan ./dist`).

```
stageflow scan <url>... | scan <dir|zip>
```

### Options

```
      --allow-private-targets   Allow private/loopback targets (requires API instance to permit it)
      --auth-recipe string      Path to a YAML/JSON form-auth recipe (Provenance.auth.form shape). Step values must use {from_env: NAME} references; literal secrets are rejected.
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

