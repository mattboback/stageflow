package main

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type getenvFunc func(string) string

func run(args []string, getenv getenvFunc, stdout, stderr io.Writer) int {
	rootCmd := newRootCmd(getenv, stdout, stderr)
	rootCmd.SetArgs(args[1:])

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		code := 2

		var exitErr exitCodeError
		if errors.As(err, &exitErr) {
			code = exitErr.Code
			err = exitErr.Err
		}

		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}

		return code
	}

	return 0
}
