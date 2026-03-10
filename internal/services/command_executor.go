package services

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"

	"gocreator/internal/interfaces"
)

type realCommandExecutor struct{}

func newCommandExecutor() interfaces.CommandExecutor {
	return realCommandExecutor{}
}

func (realCommandExecutor) Run(ctx context.Context, name string, args ...string) (interfaces.CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return interfaces.CommandResult{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}, err
}

func formatCommand(name string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteCommandArg(name))
	for _, arg := range args {
		parts = append(parts, quoteCommandArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteCommandArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if strings.ContainsAny(arg, " \t\"'") {
		return strconv.Quote(arg)
	}
	return arg
}
