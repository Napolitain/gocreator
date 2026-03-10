package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"gocreator/internal/interfaces"

	"github.com/stretchr/testify/require"
)

type recordedCommand struct {
	Name string
	Args []string
}

type expectedCommand struct {
	Name     string
	Contains []string
	Result   interfaces.CommandResult
	Err      error
	Run      func(name string, args []string)
}

type fakeCommandExecutor struct {
	mu           sync.Mutex
	expectations []expectedCommand
	calls        []recordedCommand
}

func newFakeCommandExecutor(expectations ...expectedCommand) *fakeCommandExecutor {
	cloned := make([]expectedCommand, len(expectations))
	copy(cloned, expectations)
	return &fakeCommandExecutor{
		expectations: cloned,
	}
}

func newCommandResult(stdout, stderr string) interfaces.CommandResult {
	return interfaces.CommandResult{
		Stdout: []byte(stdout),
		Stderr: []byte(stderr),
	}
}

func (f *fakeCommandExecutor) Run(_ context.Context, name string, args ...string) (interfaces.CommandResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	copiedArgs := append([]string(nil), args...)
	f.calls = append(f.calls, recordedCommand{Name: name, Args: copiedArgs})

	if len(f.expectations) == 0 {
		return interfaces.CommandResult{}, fmt.Errorf("unexpected command: %s", formatCommand(name, args...))
	}

	expectation := f.expectations[0]
	f.expectations = f.expectations[1:]

	if expectation.Name != "" && expectation.Name != name {
		return interfaces.CommandResult{}, fmt.Errorf("expected command %q, got %q", expectation.Name, name)
	}

	joinedArgs := strings.Join(args, " ")
	for _, fragment := range expectation.Contains {
		if !strings.Contains(joinedArgs, fragment) {
			return interfaces.CommandResult{}, fmt.Errorf("command %q missing %q in %q", name, fragment, joinedArgs)
		}
	}

	if expectation.Run != nil {
		expectation.Run(name, copiedArgs)
	}

	return expectation.Result, expectation.Err
}

func (f *fakeCommandExecutor) AssertDone(t *testing.T) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Empty(t, f.expectations)
}

func (f *fakeCommandExecutor) Calls() []recordedCommand {
	f.mu.Lock()
	defer f.mu.Unlock()

	cloned := make([]recordedCommand, len(f.calls))
	for i, call := range f.calls {
		cloned[i] = recordedCommand{
			Name: call.Name,
			Args: append([]string(nil), call.Args...),
		}
	}

	return cloned
}
