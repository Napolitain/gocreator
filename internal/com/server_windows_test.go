//go:build windows

package com

import (
	"testing"
)

func TestNewGoCreatorCOM(t *testing.T) {
	server := NewGoCreatorCOM()
	if server == nil {
		t.Fatal("NewGoCreatorCOM returned nil")
	}

	if server.logger == nil {
		t.Error("logger should not be nil")
	}

	if server.rootDir == "" {
		t.Error("rootDir should not be empty")
	}
}

func TestSetRootDirectory(t *testing.T) {
	server := NewGoCreatorCOM()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid directory",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "invalid directory",
			path:    "/nonexistent/path/that/does/not/exist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.SetRootDirectory(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRootDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	server := NewGoCreatorCOM()
	version := server.GetVersion()
	if version == "" {
		t.Error("GetVersion should return a non-empty string")
	}
}

func TestGetOutputPath(t *testing.T) {
	server := NewGoCreatorCOM()
	server.SetRootDirectory(".")

	path := server.GetOutputPath("en")
	if path == "" {
		t.Error("GetOutputPath should return a non-empty path")
	}

	// Check that the path contains the language code
	if len(path) < 2 {
		t.Error("GetOutputPath returned an invalid path")
	}
}

func TestParseLanguages(t *testing.T) {
	tests := []struct {
		name        string
		outputLangs string
		inputLang   string
		want        []string
	}{
		{
			name:        "single language",
			outputLangs: "en",
			inputLang:   "en",
			want:        []string{"en"},
		},
		{
			name:        "multiple languages",
			outputLangs: "en,fr,es",
			inputLang:   "en",
			want:        []string{"en", "fr", "es"},
		},
		{
			name:        "input lang not in output",
			outputLangs: "fr,es",
			inputLang:   "en",
			want:        []string{"en", "fr", "es"},
		},
		{
			name:        "duplicate input lang",
			outputLangs: "en,fr,en,es",
			inputLang:   "en",
			want:        []string{"en", "fr", "es"},
		},
		{
			name:        "empty output langs",
			outputLangs: "",
			inputLang:   "en",
			want:        []string{"en"},
		},
		{
			name:        "whitespace in langs",
			outputLangs: " en , fr , es ",
			inputLang:   "en",
			want:        []string{"en", "fr", "es"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLanguages(tt.outputLangs, tt.inputLang)
			if len(got) != len(tt.want) {
				t.Errorf("parseLanguages() got length %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseLanguages() got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitCommaSeparated(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple",
			input: "a,b,c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "with spaces",
			input: "a , b , c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single item",
			input: "a",
			want:  []string{"a"},
		},
		{
			name:  "trailing comma",
			input: "a,b,",
			want:  []string{"a", "b"},
		},
		{
			name:  "leading comma",
			input: ",a,b",
			want:  []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCommaSeparated(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitCommaSeparated() got length %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCommaSeparated() got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsCOMAvailable(t *testing.T) {
	// On Windows, COM should be available
	if !IsCOMAvailable() {
		t.Error("IsCOMAvailable() should return true on Windows")
	}
}
