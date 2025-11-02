package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gocreator/internal/com"

	"github.com/spf13/cobra"
)

// NewCOMCommand creates the com command for COM server management
func NewCOMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "com",
		Short: "Manage COM server (Windows only)",
		Long:  `Manage the GoCreator COM server registration and configuration. COM support is only available on Windows.`,
	}

	// Add subcommands
	cmd.AddCommand(newCOMRegisterCommand())
	cmd.AddCommand(newCOMUnregisterCommand())
	cmd.AddCommand(newCOMInfoCommand())

	return cmd
}

// newCOMRegisterCommand creates the register subcommand
func newCOMRegisterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register the COM server (Windows only)",
		Long:  `Register the GoCreator COM server in the Windows registry. This allows other applications to use GoCreator via COM.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !com.IsCOMAvailable() {
				fmt.Fprintln(os.Stderr, "Error: COM support is only available on Windows")
				return fmt.Errorf("COM not available on this platform")
			}

			// Get the executable path
			exePath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}

			// Resolve to absolute path
			exePath, err = filepath.Abs(exePath)
			if err != nil {
				return fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			fmt.Printf("Registering COM server...\n")
			fmt.Printf("Executable: %s\n\n", exePath)

			if err := com.RegisterCOMServer(exePath); err != nil {
				return fmt.Errorf("failed to register COM server: %w", err)
			}

			fmt.Println("\nCOM server registration information printed above.")
			fmt.Println("Note: Manual registry configuration may be required.")
			return nil
		},
	}
}

// newCOMUnregisterCommand creates the unregister subcommand
func newCOMUnregisterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unregister",
		Short: "Unregister the COM server (Windows only)",
		Long:  `Unregister the GoCreator COM server from the Windows registry.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !com.IsCOMAvailable() {
				fmt.Fprintln(os.Stderr, "Error: COM support is only available on Windows")
				return fmt.Errorf("COM not available on this platform")
			}

			fmt.Println("Unregistering COM server...")

			if err := com.UnregisterCOMServer(); err != nil {
				return fmt.Errorf("failed to unregister COM server: %w", err)
			}

			fmt.Println("\nCOM server unregistration information printed above.")
			fmt.Println("Note: Manual registry configuration may be required.")
			return nil
		},
	}
}

// newCOMInfoCommand creates the info subcommand
func newCOMInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Display COM server information",
		Long:  `Display information about the GoCreator COM server, including availability and version.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("GoCreator COM Server Information")
			fmt.Println("=================================")
			fmt.Println()

			if com.IsCOMAvailable() {
				fmt.Println("Status: COM support is AVAILABLE")
				fmt.Println("Platform: Windows")
				
				server := com.NewGoCreatorCOM()
				fmt.Printf("Version: %s\n", server.GetVersion())
				fmt.Println()
				fmt.Println("Usage:")
				fmt.Println("  1. Register the COM server:")
				fmt.Println("     gocreator com register")
				fmt.Println()
				fmt.Println("  2. Use from other applications via COM interface")
				fmt.Println()
				fmt.Println("  3. Unregister when no longer needed:")
				fmt.Println("     gocreator com unregister")
			} else {
				fmt.Println("Status: COM support is NOT AVAILABLE")
				fmt.Println("Platform: Non-Windows")
				fmt.Println()
				fmt.Println("COM (Component Object Model) is a Windows-specific technology.")
				fmt.Println("On this platform, please use the CLI commands directly:")
				fmt.Println("  gocreator create --lang en --langs-out en,fr,es")
			}
		},
	}
}
