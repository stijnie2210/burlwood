package gitcli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func OpenInDefaultApp(repoPath, filePath string) error {
	return openCommand(filepath.Join(repoPath, filePath)).Start()
}

func ShowInFinder(repoPath, filePath string) error {
	return revealCommand(filepath.Join(repoPath, filePath)).Start()
}

func OpenTerminal(dir string) error {
	return terminalCommand(dir).Start()
}

func OpenInEditor(repoPath, filePath, preferredEditor string) error {
	full := filepath.Join(repoPath, filePath)
	editor := preferredEditor
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if editor == "" {
			editor = os.Getenv(env)
		}
	}
	if editor == "" {
		return openCommand(full).Start()
	}
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], full)...)
	noConsole(cmd)
	return cmd.Start()
}

func StashFile(repoPath, filePath string) error {
	_, err := Run(repoPath, "stash", "push", "--", filePath)
	return err
}

func AppendToGitignore(repoPath, pattern string) error {
	f, err := os.OpenFile(filepath.Join(repoPath, ".gitignore"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(pattern + "\n")
	return err
}

func GetFilePatch(repoPath, filePath string) (string, error) {
	res, err := Run(repoPath, "diff", "--", filePath)
	if err == nil && res.Stdout != "" {
		return res.Stdout, nil
	}
	res, err = Run(repoPath, "diff", "--cached", "--", filePath)
	if err == nil && res.Stdout != "" {
		return res.Stdout, nil
	}
	// --no-index exits 1 when differences are found, not an error
	full := filepath.Join(repoPath, filePath)
	res, err = Run(repoPath, "diff", "--no-index", "/dev/null", full)
	if res.Stdout != "" {
		return res.Stdout, nil
	}
	if err != nil {
		if ee, ok := err.(*ExitError); ok && ee.Code == 1 {
			return res.Stdout, nil
		}
		return "", err
	}
	return "", fmt.Errorf("no diff found for %s", filePath)
}

func DeleteWorkingFile(repoPath, filePath string) error {
	return os.Remove(filepath.Join(repoPath, filePath))
}
