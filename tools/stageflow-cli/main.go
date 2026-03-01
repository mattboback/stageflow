package main

import (
	"os"
)

func main() {
	os.Exit(run(os.Args, os.Getenv, os.Stdout, os.Stderr))
}
