package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "gocreator", cmd.Use)
	assert.Contains(t, cmd.Short, "video creation tool")
	assert.NotEmpty(t, cmd.Long)
	
	// Check that subcommands are added
	commands := cmd.Commands()
	assert.NotEmpty(t, commands)
	
	// Verify the command has the expected subcommands
	var hasInit, hasCreate bool
	for _, c := range commands {
		if c.Use == "init" {
			hasInit = true
		}
		if c.Use == "create" {
			hasCreate = true
		}
	}
	
	assert.True(t, hasInit, "init command should be present")
	assert.True(t, hasCreate, "create command should be present")
}

func TestRootCommandHelp(t *testing.T) {
	cmd := NewRootCommand()
	
	// Test that help flag works
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	
	// Help should not return an error
	assert.NoError(t, err)
}
