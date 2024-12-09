package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunCommand はコマンドを実行し、標準出力と標準エラー出力を表示します
func RunCommand(logger *Logger, name string, args ...string) error {
	logger.Info("Executing command: %s %v", name, args)

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %v", err)
	}

	return nil
}

// RunCommandWithInput は標準入力を受け付けるコマンドを実行します
func RunCommandWithInput(logger *Logger, input string, name string, args ...string) error {
	logger.Info("Executing command: %s %v", name, args)

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(input)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %v", err)
	}

	return nil
}

// RunCommandWithOutput はコマンドを実行し、出力を返します
func RunCommandWithOutput(logger *Logger, name string, args ...string) ([]byte, error) {
	logger.Info("Executing command: %s %v", name, args)

	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %v", err)
	}

	return output, nil
}
