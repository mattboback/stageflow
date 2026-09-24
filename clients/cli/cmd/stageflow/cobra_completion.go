package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion scripts",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			out := cmd.OutOrStdout()

			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}

	cmd.Long = fmt.Sprintf(`To load completions:

Bash:

  $ source <(%[1]s completion bash)

Zsh:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  $ %[1]s completion zsh > "${fpath[1]}/_%[1]s"

fish:

  $ %[1]s completion fish | source

PowerShell:

  PS> %[1]s completion powershell | Out-String | Invoke-Expression
`, "stageflow")

	return cmd
}
