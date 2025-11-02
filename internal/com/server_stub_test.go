//go:build !windows

package com

import (
	"testing"
)

func TestNewGoCreatorCOM_Stub(t *testing.T) {
	server := NewGoCreatorCOM()
	if server == nil {
		t.Fatal("NewGoCreatorCOM returned nil")
	}
}

func TestSetRootDirectory_Stub(t *testing.T) {
	server := NewGoCreatorCOM()
	err := server.SetRootDirectory(".")
	if err == nil {
		t.Error("SetRootDirectory should return an error on non-Windows platforms")
	}
}

func TestCreateVideo_Stub(t *testing.T) {
	server := NewGoCreatorCOM()
	err := server.CreateVideo("en", "en,fr", "")
	if err == nil {
		t.Error("CreateVideo should return an error on non-Windows platforms")
	}
}

func TestGetVersion_Stub(t *testing.T) {
	server := NewGoCreatorCOM()
	version := server.GetVersion()
	if version == "" {
		t.Error("GetVersion should return a non-empty string")
	}
}

func TestGetOutputPath_Stub(t *testing.T) {
	server := NewGoCreatorCOM()
	path := server.GetOutputPath("en")
	if path != "" {
		t.Error("GetOutputPath should return empty string on non-Windows platforms")
	}
}

func TestRegisterCOMServer_Stub(t *testing.T) {
	err := RegisterCOMServer("/path/to/exe")
	if err == nil {
		t.Error("RegisterCOMServer should return an error on non-Windows platforms")
	}
}

func TestUnregisterCOMServer_Stub(t *testing.T) {
	err := UnregisterCOMServer()
	if err == nil {
		t.Error("UnregisterCOMServer should return an error on non-Windows platforms")
	}
}

func TestIsCOMAvailable_Stub(t *testing.T) {
	if IsCOMAvailable() {
		t.Error("IsCOMAvailable() should return false on non-Windows platforms")
	}
}
