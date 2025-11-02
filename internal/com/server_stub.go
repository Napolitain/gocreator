//go:build !windows

package com

import "fmt"

// GoCreatorCOM is a stub for non-Windows platforms
type GoCreatorCOM struct{}

// NewGoCreatorCOM returns an error on non-Windows platforms
func NewGoCreatorCOM() *GoCreatorCOM {
	return &GoCreatorCOM{}
}

// SetRootDirectory returns an error on non-Windows platforms
func (gc *GoCreatorCOM) SetRootDirectory(path string) error {
	return fmt.Errorf("COM support is only available on Windows")
}

// CreateVideo returns an error on non-Windows platforms
func (gc *GoCreatorCOM) CreateVideo(inputLang, outputLangs, googleSlidesID string) error {
	return fmt.Errorf("COM support is only available on Windows")
}

// GetVersion returns the version string
func (gc *GoCreatorCOM) GetVersion() string {
	return "1.0.0 (COM not available)"
}

// GetOutputPath returns an error on non-Windows platforms
func (gc *GoCreatorCOM) GetOutputPath(lang string) string {
	return ""
}

// RegisterCOMServer returns an error on non-Windows platforms
func RegisterCOMServer(exePath string) error {
	return fmt.Errorf("COM server registration is only available on Windows")
}

// UnregisterCOMServer returns an error on non-Windows platforms
func UnregisterCOMServer() error {
	return fmt.Errorf("COM server unregistration is only available on Windows")
}

// IsCOMAvailable returns false on non-Windows platforms
func IsCOMAvailable() bool {
	return false
}
