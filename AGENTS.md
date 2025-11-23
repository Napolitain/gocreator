# GoCreator Development Workflow

This document outlines the standard development workflow for AI agents and developers working on GoCreator.

---

## 🔄 Standard Development Workflow

When making changes to the codebase, follow this workflow:

### 1. **Tidy Dependencies**
```bash
go mod tidy
```
Ensures `go.mod` and `go.sum` are up to date with the code.

### 2. **Run Linter** (if available)
```bash
golangci-lint run
```
**Note**: If `golangci-lint` is not installed, skip this step or install it:
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 3. **Run Tests**
```bash
# Run all tests with verbose output
go test -v ./...

# Run tests with coverage report
go test -v -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in browser
go tool cover -html=coverage.out
```

### 4. **Build the Application**
```bash
# Build main application
go build -o gocreator.exe ./cmd/gocreator

# Build performance test tool
go build -o perftest.exe ./cmd/perftest

# Build cache performance test tool
go build -o cache-perf-test.exe ./cmd/cache-perf-test
```

---

## 📁 Project Structure

```
gocreator/
├── cmd/
│   ├── gocreator/         # Main application
│   ├── perftest/          # Performance testing tool
│   └── cache-perf-test/   # Cache performance testing tool
├── internal/              # Internal packages
├── examples/              # Example configurations and demos
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── AGENTS.md              # This file
└── README.md              # Project documentation
```

---

## 🛠️ Common Commands

### **Development**
```bash
# Install dependencies
go mod download

# Update dependencies
go get -u ./...
go mod tidy

# Format code
go fmt ./...

# Vet code for issues
go vet ./...
```

### **Testing**
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests in a specific package
go test ./internal/video

# Run specific test
go test -v -run TestMultiView ./internal/video

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### **Building**
```bash
# Build main application
go build ./cmd/gocreator

# Build with specific output name
go build -o gocreator.exe ./cmd/gocreator

# Build for production (optimized)
go build -ldflags="-s -w" -o gocreator.exe ./cmd/gocreator

# Build all binaries
go build ./cmd/gocreator && go build ./cmd/perftest && go build ./cmd/cache-perf-test
```

### **Running**
```bash
# Run without building
go run ./cmd/gocreator

# Run with arguments
go run ./cmd/gocreator create --config examples/demo-multiview/config.yaml

# Run built executable
./gocreator.exe create --config examples/demo-multiview/config.yaml
```

---

## ✅ Pre-Commit Checklist

Before committing changes, ensure:

1. ✅ **Dependencies are tidy**
   ```bash
   go mod tidy
   ```

2. ✅ **Code is formatted**
   ```bash
   go fmt ./...
   ```

3. ✅ **No vet issues**
   ```bash
   go vet ./...
   ```

4. ✅ **Tests pass**
   ```bash
   go test ./...
   ```

5. ✅ **Builds successfully**
   ```bash
   go build ./cmd/gocreator
   ```

6. ✅ **Linter passes** (if installed)
   ```bash
   golangci-lint run
   ```

---

## 🧪 Testing Guidelines

### **Unit Tests**
- Test files should be named `*_test.go`
- Place tests in the same package as the code
- Use table-driven tests for multiple cases

### **Running Specific Tests**
```bash
# Run tests in specific package
go test ./internal/video

# Run specific test function
go test -v -run TestMultiView

# Run tests matching pattern
go test -v -run "Multi.*"
```

### **Coverage**
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage summary
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

---

## 🚀 Build Targets

### **Main Application** (`cmd/gocreator`)
```bash
go build -o gocreator.exe ./cmd/gocreator
```
Creates the main GoCreator video generation tool.

### **Performance Test** (`cmd/perftest`)
```bash
go build -o perftest.exe ./cmd/perftest
```
Tool for performance testing and benchmarking.

### **Cache Performance Test** (`cmd/cache-perf-test`)
```bash
go build -o cache-perf-test.exe ./cmd/cache-perf-test
```
Tool for testing cache performance specifically.

---

## 🔍 Debugging

### **Enable Verbose Output**
```bash
go build -v ./cmd/gocreator
go test -v ./...
```

### **Print Build Information**
```bash
go version -m gocreator.exe
```

### **Race Detection**
```bash
go test -race ./...
go build -race ./cmd/gocreator
```

---

## 📝 Code Quality

### **Formatting**
```bash
# Format all Go files
go fmt ./...

# Use goimports (if installed)
goimports -w .
```

### **Vetting**
```bash
# Check for common issues
go vet ./...
```

### **Linting** (optional, requires golangci-lint)
```bash
# Run linter
golangci-lint run

# Run with auto-fix
golangci-lint run --fix
```

---

## 🎯 Workflow Summary

**Standard workflow for making changes:**

```bash
# 1. Make your changes
# 2. Tidy dependencies
go mod tidy

# 3. Format code
go fmt ./...

# 4. Check for issues
go vet ./...

# 5. Run tests
go test ./...

# 6. Build
go build ./cmd/gocreator

# 7. Test the built binary
./gocreator.exe create --config examples/demo-multiview/config.yaml

# 8. Commit if everything passes
git add .
git commit -m "your commit message"
git push origin main
```

---

## 🛑 Common Issues

### **"golangci-lint not found"**
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### **"package not found"**
```bash
# Download dependencies
go mod download
go mod tidy
```

### **"build failed"**
```bash
# Clean build cache
go clean -cache
go build ./cmd/gocreator
```

---

## 📚 Additional Resources

- **Go Documentation**: https://go.dev/doc/
- **Go Testing**: https://go.dev/doc/tutorial/add-a-test
- **golangci-lint**: https://golangci-lint.run/

---

**Last Updated**: 2025-11-23
        