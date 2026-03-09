package cli

import (
	"fmt"
	"path/filepath"

	"gocreator/internal/config"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new GoCreator project",
		Long:  `Creates a default gocreator.yaml configuration file, project directories, and example sidecar narration files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing config file")

	return cmd
}

func runInit(force bool) error {
	fs := afero.NewOsFs()
	configPath := "gocreator.yaml"

	// Check if config file already exists
	exists, err := afero.Exists(fs, configPath)
	if err != nil {
		return fmt.Errorf("failed to check for existing config: %w", err)
	}

	if exists && !force {
		return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Save config file
	if err := config.SaveConfig(fs, configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	fmt.Printf("✓ Created config file: %s\n\n", configPath)

	// Create directory structure
	dirs := []string{
		"data/slides",
		"data/out",
		"data/cache",
	}

	fmt.Println("Creating directory structure:")
	for _, dir := range dirs {
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  ✓ %s/\n", dir)
	}

	// Create example sidecar text files
	exampleSidecars := map[string]string{
		filepath.Join("data", "slides", "01-welcome.txt"): "Welcome to GoCreator",
		filepath.Join("data", "slides", "02-details.txt"): "Add a matching slide file such as 02-details.png or 02-details.mp4.",
	}

	for path, content := range exampleSidecars {
		if err := afero.WriteFile(fs, path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create example sidecar file %s: %w", path, err)
		}
		fmt.Printf("  ✓ %s\n", path)
	}

	// Create .gitignore
	gitignorePath := ".gitignore"
	gitignoreExists, err := afero.Exists(fs, gitignorePath)
	if err != nil {
		return fmt.Errorf("failed to check for .gitignore: %w", err)
	}

	if !gitignoreExists {
		gitignoreContent := `# GoCreator outputs and cache
data/out/
data/cache/
*.mp4
*.mp3
*.wav

# OS files
.DS_Store
Thumbs.db

# Editor files
.vscode/
.idea/
*.swp
*.swo
`
		if err := afero.WriteFile(fs, gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		fmt.Printf("  ✓ %s\n", gitignorePath)
	}

	fmt.Println("\n✓ Project initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your slide media to data/slides/ (for example 01-welcome.png)")
	fmt.Println("  2. Keep narration next to each slide as matching .txt, .<lang>.txt, or audio sidecars")
	fmt.Println("  3. Review and edit gocreator.yaml")
	fmt.Println("  4. Run: gocreator create")

	return nil
}
