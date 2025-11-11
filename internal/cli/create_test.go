package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreateCommand(t *testing.T) {
	cmd := NewCreateCommand()
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "create", cmd.Use)
	assert.Contains(t, cmd.Short, "Create")
	assert.NotEmpty(t, cmd.Long)
	
	// Check that flags exist
	langFlag := cmd.Flags().Lookup("lang")
	assert.NotNil(t, langFlag)
	assert.Equal(t, "l", langFlag.Shorthand)
	
	langsOutFlag := cmd.Flags().Lookup("langs-out")
	assert.NotNil(t, langsOutFlag)
	assert.Equal(t, "o", langsOutFlag.Shorthand)
	
	googleSlidesFlag := cmd.Flags().Lookup("google-slides")
	assert.NotNil(t, googleSlidesFlag)
	
	configFlag := cmd.Flags().Lookup("config")
	assert.NotNil(t, configFlag)
	assert.Equal(t, "c", configFlag.Shorthand)
	
	noProgressFlag := cmd.Flags().Lookup("no-progress")
	assert.NotNil(t, noProgressFlag)
	assert.Equal(t, "false", noProgressFlag.DefValue)
}

func TestCreateCommand_Help(t *testing.T) {
	cmd := NewCreateCommand()
	
	// Test that help flag works
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	
	// Help should not return an error
	assert.NoError(t, err)
}

func TestCreateCommand_FlagsParsing(t *testing.T) {
	cmd := NewCreateCommand()
	
	// Test parsing multiple flags
	err := cmd.Flags().Parse([]string{
		"--lang", "en",
		"--langs-out", "es,fr,de",
		"--google-slides", "test-id-123",
		"--config", "/path/to/config.yaml",
		"--no-progress",
	})
	
	assert.NoError(t, err)
	
	langFlag := cmd.Flags().Lookup("lang")
	assert.Equal(t, "en", langFlag.Value.String())
	
	langsOutFlag := cmd.Flags().Lookup("langs-out")
	assert.Equal(t, "es,fr,de", langsOutFlag.Value.String())
	
	googleSlidesFlag := cmd.Flags().Lookup("google-slides")
	assert.Equal(t, "test-id-123", googleSlidesFlag.Value.String())
	
	configFlag := cmd.Flags().Lookup("config")
	assert.Equal(t, "/path/to/config.yaml", configFlag.Value.String())
	
	noProgressFlag := cmd.Flags().Lookup("no-progress")
	assert.Equal(t, "true", noProgressFlag.Value.String())
}

func TestParseLanguages(t *testing.T) {
tests := []struct {
name        string
outputLangs string
inputLang   string
expected    []string
}{
{
name:        "basic case",
outputLangs: "es,fr,de",
inputLang:   "en",
expected:    []string{"en", "es", "fr", "de"},
},
{
name:        "input lang in output langs",
outputLangs: "en,es,fr",
inputLang:   "en",
expected:    []string{"en", "es", "fr"},
},
{
name:        "with spaces",
outputLangs: " es , fr , de ",
inputLang:   "en",
expected:    []string{"en", "es", "fr", "de"},
},
{
name:        "empty langs filtered",
outputLangs: "es,,fr",
inputLang:   "en",
expected:    []string{"en", "es", "fr"},
},
{
name:        "single language",
outputLangs: "es",
inputLang:   "en",
expected:    []string{"en", "es"},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := parseLanguages(tt.outputLangs, tt.inputLang)
assert.Equal(t, tt.expected, result)
})
}
}

func TestEnsureInputLanguageFirst(t *testing.T) {
tests := []struct {
name      string
languages []string
inputLang string
expected  []string
}{
{
name:      "input lang not in list",
languages: []string{"es", "fr", "de"},
inputLang: "en",
expected:  []string{"en", "es", "fr", "de"},
},
{
name:      "input lang at end",
languages: []string{"es", "fr", "en"},
inputLang: "en",
expected:  []string{"en", "es", "fr"},
},
{
name:      "input lang in middle",
languages: []string{"es", "en", "fr"},
inputLang: "en",
expected:  []string{"en", "es", "fr"},
},
{
name:      "input lang already first",
languages: []string{"en", "es", "fr"},
inputLang: "en",
expected:  []string{"en", "es", "fr"},
},
{
name:      "input lang duplicated",
languages: []string{"en", "es", "en", "fr"},
inputLang: "en",
expected:  []string{"en", "es", "fr"},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := ensureInputLanguageFirst(tt.languages, tt.inputLang)
assert.Equal(t, tt.expected, result)
})
}
}
