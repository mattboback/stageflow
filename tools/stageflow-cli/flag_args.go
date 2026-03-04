package main

import "strings"

// The standard library flag parser stops at the first non-flag argument.
// For a nicer UX, we allow a single positional argument (job ID / project path)
// to appear anywhere by reordering arguments so flags parse correctly.
func reorderArgsForFlagParsing(args []string, valueFlags map[string]bool) []string {
	if len(args) == 0 {
		return args
	}

	flagArgs := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Preserve explicit end-of-flags behavior.
		if arg == "--" {
			flagArgs = append(flagArgs, arg)
			positionals = append(positionals, args[i+1:]...)

			break
		}

		if isFlagToken(arg) {
			flagArgs = append(flagArgs, arg)

			// If this flag expects a separate value, treat the next token as its value even
			// if it begins with '-' (for example: --threshold-total -1).
			if flagConsumesValue(arg, valueFlags) && !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}

			continue
		}

		positionals = append(positionals, arg)
	}

	if len(positionals) == 0 {
		return flagArgs
	}

	if len(flagArgs) == 0 {
		return positionals
	}

	out := make([]string, 0, len(args))
	out = append(out, flagArgs...)
	out = append(out, positionals...)

	return out
}

func isFlagToken(token string) bool {
	if token == "" || token == "-" {
		return false
	}

	return strings.HasPrefix(token, "-")
}

func flagConsumesValue(token string, valueFlags map[string]bool) bool {
	name, ok := flagName(token)
	if !ok {
		return false
	}

	return valueFlags[name]
}

func flagName(token string) (string, bool) {
	if !strings.HasPrefix(token, "-") {
		return "", false
	}

	trimmed := strings.TrimLeft(token, "-")
	if trimmed == "" {
		return "", false
	}

	name, _, _ := strings.Cut(trimmed, "=")

	return name, true
}
