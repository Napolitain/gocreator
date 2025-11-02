package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for gocreator
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gocreator",
		Short: "GoCreator - A video creation tool",
		Long:  `GoCreator is a CLI tool for creating videos with translations and audio.`,
	}

	// Add subcommands
	rootCmd.AddCommand(NewCreateCommand())

	return rootCmd
}
