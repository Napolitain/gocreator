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
		Long:  `Creates a default gocreator.yaml configuration file and directory structure.`,
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

	// Create example text file
	textsPath := filepath.Join("data", "texts.txt")
	exampleText := `Welcome to GoCreator
-
This is an example narration text for your first slide
-
You can add more slides and narration here
-
Each section is separated by a single dash on its own line`

	if err := afero.WriteFile(fs, textsPath, []byte(exampleText), 0644); err != nil {
		return fmt.Errorf("failed to create example text file: %w", err)
	}
	fmt.Printf("  ✓ %s\n", textsPath)

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
	fmt.Println("  1. Add your slides to data/slides/")
	fmt.Println("  2. Edit data/texts.txt with your narration")
	fmt.Println("  3. Review and edit gocreator.yaml")
	fmt.Println("  4. Run: gocreator create")

	return nil
}
