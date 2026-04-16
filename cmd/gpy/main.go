// Command gpy is the entry point for the go-package-yourself tool.
package main

import (
	"fmt"
	"os"

	"go-package-yourself/internal/cli"
)

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X main.Version=$(cat VERSION)" ./cmd/gpy
//
// Falls back to "dev" for local builds without ldflags.
var Version = "dev"

func main() {
	if err := cli.Execute(os.Args[1:], Version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
