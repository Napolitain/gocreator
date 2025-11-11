package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInitCommand(t *testing.T) {
	cmd := NewInitCommand()
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "init", cmd.Use)
	assert.Contains(t, cmd.Short, "Initialize")
	assert.NotEmpty(t, cmd.Long)
	
	// Check that the force flag exists
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestInitCommand_Help(t *testing.T) {
	cmd := NewInitCommand()
	
	// Test that help flag works
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	
	// Help should not return an error
	assert.NoError(t, err)
}

func TestInitCommand_Flags(t *testing.T) {
	cmd := NewInitCommand()
	
	// Test force flag parsing
	cmd.SetArgs([]string{"--force"})
	err := cmd.Flags().Parse([]string{"--force"})
	
	assert.NoError(t, err)
	
	forceFlag := cmd.Flags().Lookup("force")
	assert.Equal(t, "true", forceFlag.Value.String())
}
