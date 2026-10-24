package gitcli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Result struct {
	Stdout string
	Stderr string
}

type ExitError struct {
	Code   int
	Stderr string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("git exited %d: %s", e.Code, e.Stderr)
}

func Run(repoPath string, args ...string) (Result, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Env = mergedEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return result, &ExitError{Code: exitErr.ExitCode(), Stderr: stderr.String()}
		}
		return result, err
	}

	return result, nil
}

func RunWithInput(repoPath string, stdin []byte, args ...string) (Result, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Env = mergedEnv()
	cmd.Stdin = bytes.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return result, &ExitError{Code: exitErr.ExitCode(), Stderr: stderr.String()}
		}
		return result, err
	}

	return result, nil
}

// RepoRoot resolves any folder inside a working tree to its top level, so opening a
// subfolder opens the repository instead of failing.
func RepoRoot(path string) (string, error) {
	out, err := Run(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s isn't inside a git repository", path)
	}
	return strings.TrimSpace(out.Stdout), nil
}
