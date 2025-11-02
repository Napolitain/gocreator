package main

import (
	"fmt"
	"os"

	"gocreator/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
