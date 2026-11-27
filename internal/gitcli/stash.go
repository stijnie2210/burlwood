package gitcli

import (
	"fmt"
	"strings"
	"time"
)

type StashEntry struct {
	Index   int    `json:"index"`
	Ref     string `json:"ref"`
	Message string `json:"message"`
	Branch  string `json:"branch"`
	Hash    string `json:"hash"`
	Date    string `json:"date"`
}

func ListStashes(repoPath string) ([]StashEntry, error) {
	result, err := Run(repoPath, "stash", "list", "--format=%gd\t%H\t%gs\t%ci")
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(result.Stdout)
	if trimmed == "" {
		return nil, nil
	}

	var entries []StashEntry
	for _, line := range strings.Split(trimmed, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		ref := parts[0]
		hash := parts[1]
		message := parts[2]
		dateStr := parts[3]

		var idx int
		fmt.Sscanf(ref, "stash@{%d}", &idx)

		branch := ""
		if rest, ok := strings.CutPrefix(message, "WIP on "); ok {
			if b, _, found := strings.Cut(rest, ":"); found {
				branch = b
			}
		} else if rest, ok := strings.CutPrefix(message, "On "); ok {
			if b, _, found := strings.Cut(rest, ":"); found {
				branch = b
			}
		}

		t, _ := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		date := dateStr
		if !t.IsZero() {
			date = t.Format("2 Jan 2006")
		}

		entries = append(entries, StashEntry{
			Index:   idx,
			Ref:     ref,
			Hash:    hash,
			Message: message,
			Branch:  branch,
			Date:    date,
		})
	}
	return entries, nil
}

func ApplyStash(repoPath, ref string) error {
	_, err := Run(repoPath, "stash", "apply", ref)
	return err
}

func DropStash(repoPath, ref string) error {
	_, err := Run(repoPath, "stash", "drop", ref)
	return err
}

func PopStash(repoPath, ref string) error {
	_, err := Run(repoPath, "stash", "pop", ref)
	return err
}

func RenameStash(repoPath, ref, newMessage string) error {
	// stash store always inserts at stash@{0}; rebuild to preserve original order.
	listResult, err := Run(repoPath, "stash", "list", "--format=%gd\t%H\t%gs")
	if err != nil {
		return fmt.Errorf("could not list stashes: %w", err)
	}

	type entry struct{ hash, msg string }
	var entries []entry
	found := false
	for _, line := range strings.Split(strings.TrimSpace(listResult.Stdout), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		msg := parts[2]
		if parts[0] == ref {
			msg = newMessage
			found = true
		}
		entries = append(entries, entry{parts[1], msg})
	}
	if !found {
		return fmt.Errorf("stash %s not found", ref)
	}

	for range entries {
		if _, err = Run(repoPath, "stash", "drop", "stash@{0}"); err != nil {
			return err
		}
	}

	for i := len(entries) - 1; i >= 0; i-- {
		if _, err = Run(repoPath, "stash", "store", "-m", entries[i].msg, entries[i].hash); err != nil {
			return err
		}
	}
	return nil
}

func Stash(repoPath string, includeUntracked bool) error {
	args := []string{"stash", "push"}
	if includeUntracked {
		args = append(args, "--include-untracked")
	}
	_, err := Run(repoPath, args...)
	return err
}

func StashPop(repoPath string) error {
	_, err := Run(repoPath, "stash", "pop")
	return err
}

func StashPushNamed(repoPath, msg string) error {
	_, err := Run(repoPath, "stash", "push", "-m", msg)
	return err
}

func CreateBranch(repoPath, name string) error {
	_, err := Run(repoPath, "checkout", "-b", name)
	return err
}
