// Command gpy is the entry point for the go-package-yourself tool.
package main

import (
	"fmt"
	"os"

	"go-package-yourself/internal/cli"
)

func main() {
	if err := cli.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
