## stageflow completion

Generate shell completion scripts

### Synopsis

To load completions:

Bash:

  $ source <(stageflow completion bash)

Zsh:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  $ stageflow completion zsh > "${fpath[1]}/_stageflow"

fish:

  $ stageflow completion fish | source

PowerShell:

  PS> stageflow completion powershell | Out-String | Invoke-Expression


```
stageflow completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --api string       API base URL (default "http://localhost:8080")
      --api-key string   API key
      --format string    Output format: text, markdown, or json (default "text")
```

### SEE ALSO

* [stageflow](stageflow.md)	 - StageFlow CLI

