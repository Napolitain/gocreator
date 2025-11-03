package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"gocreator/internal/cli"
)

func main() {
	// load .env if present (non-fatal)
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to load .env: %v\n", err)
		}
	}

	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
