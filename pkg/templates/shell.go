package templates

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Shell executes a shell command and returns its stdout as a string.
// It returns an error if the command fails or produces stderr output.
func Shell(command string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("shell command %q failed: %w\nstderr: %s", command, err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}
