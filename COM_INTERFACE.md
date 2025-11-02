# COM Interface for GoCreator

This document describes the COM (Component Object Model) interface for GoCreator, which allows Windows applications to programmatically create videos using Go bindings.

## Overview

The COM interface provides a Windows-native way to integrate GoCreator functionality into other applications like VBA macros (Excel, Word), PowerShell scripts, C++, C#, or any COM-compatible language.

## Platform Support

- **Windows**: Full COM support available
- **Linux/macOS**: COM interface not available (use CLI instead)

The implementation uses Go build tags to ensure:
- COM code only compiles on Windows (`//go:build windows`)
- Cross-platform CLI builds remain unaffected
- No Windows dependencies are required for non-Windows builds

## Architecture

### Files

- `internal/com/server_windows.go` - Windows COM server implementation
- `internal/com/server_stub.go` - Non-Windows stub (returns errors)
- `internal/com/server_windows_test.go` - Windows-specific tests
- `internal/com/server_stub_test.go` - Non-Windows stub tests
- `internal/cli/com.go` - CLI commands for COM management

### Build Tags

The COM interface uses Go build tags to conditionally compile platform-specific code:

```go
//go:build windows      // Compiles only on Windows
//go:build !windows     // Compiles on all non-Windows platforms
```

This ensures that:
1. Windows builds include full COM support with `go-ole` dependency
2. Non-Windows builds include stub functions that return "not available" errors
3. Cross-compilation works without platform-specific dependencies

## COM Interface

### CLSID and IID

```
CLSID_GoCreator: {8B9C5A3E-1234-5678-9ABC-DEF012345678}
IID_IGoCreator:  {9C8D6B4F-2345-6789-ABCD-EF0123456789}
```

### Methods

#### SetRootDirectory(path string) error
Sets the working directory for video creation operations.

**Parameters:**
- `path` - Absolute path to the working directory

**Returns:**
- Error if the path is invalid or not a directory

#### CreateVideo(inputLang, outputLangs, googleSlidesID string) error
Creates videos with the specified configuration.

**Parameters:**
- `inputLang` - Source language code (e.g., "en", "fr", "es")
- `outputLangs` - Comma-separated list of output languages (e.g., "en,fr,es")
- `googleSlidesID` - Google Slides presentation ID (empty string for local slides)

**Returns:**
- Error if video creation fails

**Example:**
```go
err := server.CreateVideo("en", "en,fr,es", "")
```

#### GetVersion() string
Returns the version of the COM server.

**Returns:**
- Version string (e.g., "1.0.0")

#### GetOutputPath(lang string) string
Returns the expected output path for a video in the specified language.

**Parameters:**
- `lang` - Language code

**Returns:**
- Full path to the output video file

## CLI Usage

### Check COM Availability

```bash
gocreator com info
```

Output on Windows:
```
GoCreator COM Server Information
=================================

Status: COM support is AVAILABLE
Platform: Windows
Version: 1.0.0
```

Output on Linux/macOS:
```
GoCreator COM Server Information
=================================

Status: COM support is NOT AVAILABLE
Platform: Non-Windows

COM (Component Object Model) is a Windows-specific technology.
On this platform, please use the CLI commands directly:
  gocreator create --lang en --langs-out en,fr,es
```

### Register COM Server (Windows only)

```bash
gocreator com register
```

This displays the CLSID and registration information. Manual registry configuration may be required.

### Unregister COM Server (Windows only)

```bash
gocreator com unregister
```

## Programming Examples

### PowerShell

```powershell
# Create COM object (after registration)
$creator = New-Object -ComObject "GoCreator.VideoCreator"

# Set working directory
$creator.SetRootDirectory("C:\Projects\MyVideo")

# Create videos
$creator.CreateVideo("en", "en,fr,es", "")

# Get output path
$path = $creator.GetOutputPath("en")
Write-Host "Video created at: $path"
```

### VBA (Excel/Word)

```vba
Sub CreateVideo()
    Dim creator As Object
    Set creator = CreateObject("GoCreator.VideoCreator")
    
    ' Set working directory
    creator.SetRootDirectory "C:\Projects\MyVideo"
    
    ' Create videos
    creator.CreateVideo "en", "en,fr,es", ""
    
    ' Get output path
    Dim outputPath As String
    outputPath = creator.GetOutputPath("en")
    MsgBox "Video created at: " & outputPath
End Sub
```

### C# (.NET)

```csharp
using System;
using System.Runtime.InteropServices;

// After adding COM reference
var creator = new GoCreator.VideoCreator();

// Set working directory
creator.SetRootDirectory(@"C:\Projects\MyVideo");

// Create videos
creator.CreateVideo("en", "en,fr,es", "");

// Get output path
string path = creator.GetOutputPath("en");
Console.WriteLine($"Video created at: {path}");
```

## Building

### Build for Windows (with COM support)

```bash
GOOS=windows GOARCH=amd64 go build -o gocreator.exe ./cmd/gocreator
```

### Build for Linux/macOS (without COM)

```bash
go build -o gocreator ./cmd/gocreator
```

### Cross-platform Build

The CI/CD pipeline builds for all platforms automatically:

```bash
# GitHub Actions automatically builds:
# - Windows (amd64) - with COM support
# - Linux (amd64, arm64) - without COM
# - macOS (amd64, arm64) - without COM
```

## Testing

### Run All Tests (current platform)

```bash
go test ./...
```

### Run COM Tests on Windows

```bash
go test -v ./internal/com/
```

### Run Stub Tests on Linux/macOS

```bash
go test -v ./internal/com/
```

The test suite automatically runs the appropriate tests based on the build platform.

## Dependencies

### Windows Build
- `github.com/go-ole/go-ole` - COM support for Go

### All Platforms
- Standard GoCreator dependencies (see `go.mod`)

## Limitations

### Current Implementation

The current implementation provides:
- ✅ Cross-platform build support with build tags
- ✅ Complete COM interface structure
- ✅ CLI commands for COM management
- ✅ Comprehensive tests
- ⚠️ Manual registry configuration required

### Future Enhancements

Potential improvements:
1. **Automatic Registry Management**: Implement automated registration/unregistration
2. **COM Type Library**: Generate .tlb file for better IDE support
3. **COM Events**: Add event notifications for progress updates
4. **COM+ Integration**: Support COM+ services for better scalability

## Troubleshooting

### "COM support is only available on Windows"

This error appears when trying to use COM commands on non-Windows platforms. Use the CLI commands directly instead:

```bash
gocreator create --lang en --langs-out en,fr,es
```

### Registration Issues

If COM registration fails:
1. Run as Administrator
2. Manually create registry keys as shown in registration output
3. Verify the executable path is correct

### Build Issues

If the Windows build fails:
```bash
# Ensure go-ole is available
go get github.com/go-ole/go-ole

# Build with verbose output
go build -v ./cmd/gocreator
```

## Security Considerations

1. **Registry Access**: COM registration requires elevated privileges
2. **Path Validation**: Always validate paths before setting root directory
3. **Input Sanitization**: Validate language codes and parameters
4. **Thread Safety**: The COM server uses mutex locks for thread-safe operations

## Contributing

When contributing to the COM interface:

1. **Use Build Tags**: Always use appropriate build tags (`//go:build windows` or `//go:build !windows`)
2. **Test Both Platforms**: Ensure tests pass on both Windows and non-Windows platforms
3. **Document Changes**: Update this README with any interface changes
4. **Follow Patterns**: Maintain consistency with existing COM patterns

## References

- [Go Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [go-ole Documentation](https://github.com/go-ole/go-ole)
- [COM Programming in Go](https://github.com/go-ole/go-ole/tree/master/example)
- [Windows COM Documentation](https://docs.microsoft.com/en-us/windows/win32/com/component-object-model--com--portal)
