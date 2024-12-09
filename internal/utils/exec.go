package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Output standard output and standard error
func RunCommand(logger *Logger, name string, args ...string) error {
	cmdLine := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	logger.Info("Executing command: %s", cmdLine)

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %v, command: %s", err, cmdLine)
	}

	return nil
}

// Execute commands that accept standard input
func RunCommandWithInput(logger *Logger, input string, name string, args ...string) error {
	cmdLine := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	logger.Info("Executing command: %s with input: %s", cmdLine, input)

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(input)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %v, command: %s", err, cmdLine)
	}

	return nil
}

// Command and returns output.
func RunCommandWithOutput(logger *Logger, name string, args ...string) ([]byte, error) {
	cmdLine := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	logger.Info("Executing command: %s", cmdLine)

	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %v, command: %s", err, cmdLine)
	}

	return output, nil
}
