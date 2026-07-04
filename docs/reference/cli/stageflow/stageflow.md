## stageflow

StageFlow CLI

### Synopsis

StageFlow CLI — scan web pages for accessibility, performance, and SEO issues.

There are three ways to scan:

  stageflow scan <url>        one-off scan of any URL
  stageflow dev scan          start your local dev server, then scan it
  stageflow project scan      scan a registered project and diff its baseline

Exit codes: 0 success, 1 policy failure (--fail-on threshold or regression),
2 usage or API error.

```
stageflow [flags]
```

### Options

```
      --api string       API base URL (env: STAGEFLOW_API_URL) (default "http://localhost:8080")
      --api-key string   API key (env: STAGEFLOW_API_KEY)
      --format string    Output format: text, markdown, or json (default "text")
  -h, --help             help for stageflow
```

### SEE ALSO

* [stageflow auth](stageflow_auth.md)	 - Manage authentication artifacts for authenticated scans
* [stageflow completion](stageflow_completion.md)	 - Generate shell completion scripts
* [stageflow dev](stageflow_dev.md)	 - Scan your local dev server from .stageflow/config.yaml
* [stageflow diff](stageflow_diff.md)	 - Compare a current scan against a saved baseline
* [stageflow docs](stageflow_docs.md)	 - Generate CLI documentation (Markdown)
* [stageflow project](stageflow_project.md)	 - Manage remote projects and scan them against their baselines
* [stageflow report](stageflow_report.md)	 - Fetch and display results for an existing job
* [stageflow scan](stageflow_scan.md)	 - Scan URLs, or upload a local build directory / ZIP archive and scan it
* [stageflow scanners](stageflow_scanners.md)	 - List available scanners
* [stageflow version](stageflow_version.md)	 - Print version information

