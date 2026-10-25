package repo

import (
	"fmt"
	"strconv"
	"strings"

	"burlwood/internal/gitcli"
)

type FileStatus struct {
	Path     string `json:"path"`
	OldPath  string `json:"oldPath"` // non-empty for renames
	Staged   string `json:"staged"`   // X code: ' ', 'M', 'A', 'D', 'R', '?'
	Unstaged string `json:"unstaged"` // Y code: ' ', 'M', 'D', '?'
}

// Single source of truth for which porcelain status codes mean "unresolved conflict".
func IsConflictedStatus(staged, unstaged string) bool {
	return staged == "U" || unstaged == "U" ||
		(staged == "A" && unstaged == "A") ||
		(staged == "D" && unstaged == "D")
}

func (m *Manager) GetStatus(repoPath string) ([]FileStatus, error) {
	result, err := gitcli.Run(repoPath, "status", "--porcelain", "-u")
	if err != nil {
		return nil, err
	}
	files := parseGitStatus(result.Stdout)
	if files == nil {
		files = []FileStatus{}
	}
	return files, nil
}

func parseGitStatus(output string) []FileStatus {
	var files []FileStatus
	for line := range strings.SplitSeq(strings.TrimRight(output, "\n"), "\n") {
		if len(line) < 3 {
			continue
		}
		staged := string(line[0])
		unstaged := string(line[1])
		rest := line[3:]

		path := rest
		oldPath := ""

		// Renames show as "old -> new"
		if before, after, ok := strings.Cut(rest, " -> "); ok {
			oldPath = before
			path = after
		}

		files = append(files, FileStatus{
			Path:     path,
			OldPath:  oldPath,
			Staged:   staged,
			Unstaged: unstaged,
		})
	}
	return files
}

func (m *Manager) GetWorkingDiff(repoPath, path string, staged bool) ([]FileDiff, error) {
	prefs, _ := LoadPrefs()
	ctx := fmt.Sprintf("-U%d", prefs.DiffContextLines)

	if !staged {
		// Check for untracked file: diff against /dev/null
		st, _ := gitcli.Run(repoPath, "status", "--porcelain", "--", path)
		if len(st.Stdout) >= 2 && st.Stdout[0] == '?' {
			result, err := gitcli.Run(repoPath, "-c", "core.quotePath=false", "diff", "--no-index", ctx, "/dev/null", path)
			if err != nil {
				// exit code 1 = differences found — not an error for --no-index
				if e, ok := err.(*gitcli.ExitError); ok && e.Code == 1 {
					err = nil
				}
			}
			if err != nil {
				return nil, err
			}
			return emptyIfNil(parseUnifiedDiff(result.Stdout)), nil
		}
	}

	var args []string
	if staged {
		args = []string{"-c", "core.quotePath=false", "diff", "--cached", ctx, "--", path}
	} else {
		args = []string{"-c", "core.quotePath=false", "diff", ctx, "--", path}
	}
	result, err := gitcli.Run(repoPath, args...)
	if err != nil {
		if e, ok := err.(*gitcli.ExitError); ok && e.Code == 1 {
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}
	return emptyIfNil(parseUnifiedDiff(result.Stdout)), nil
}

func emptyIfNil(diffs []FileDiff) []FileDiff {
	if diffs == nil {
		return []FileDiff{}
	}
	return diffs
}

func parseUnifiedDiff(output string) []FileDiff {
	var diffs []FileDiff
	var cur *FileDiff
	var curHunk *Hunk
	var curHeader string
	oldLineNo, newLineNo := 0, 0

	finish := func() {
		if curHunk != nil {
			cur.Hunks = append(cur.Hunks, *curHunk)
			curHunk = nil
		}
		if cur.OldPath == "" && cur.NewPath == "" {
			cur.OldPath = headerPath(curHeader)
			cur.NewPath = cur.OldPath
		}
		diffs = append(diffs, *cur)
	}

	for line := range strings.SplitSeq(output, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if cur != nil {
				finish()
			}
			cur = &FileDiff{Hunks: []Hunk{}}
			curHeader = line

		case cur == nil:
		case strings.HasPrefix(line, "--- "):
			p := line[4:]
			if p != "/dev/null" {
				cur.OldPath = strings.TrimPrefix(p, "a/")
			}

		case strings.HasPrefix(line, "+++ "):
			p := line[4:]
			if p != "/dev/null" {
				cur.NewPath = strings.TrimPrefix(p, "b/")
			}

		case strings.HasPrefix(line, "Binary files"):
			cur.IsBinary = true
			if cur.OldPath == "" && cur.NewPath == "" {
				cur.OldPath, cur.NewPath = binaryPaths(line)
			}

		case strings.HasPrefix(line, "@@ "):
			if curHunk != nil {
				cur.Hunks = append(cur.Hunks, *curHunk)
			}
			oldLineNo, newLineNo = parseHunkHeader(line)
			curHunk = &Hunk{Header: line, Lines: []HunkLine{}}

		case curHunk != nil && len(line) > 0:
			if line[0] == '\\' {
				continue // "\ No newline at end of file"
			}
			content := ""
			if len(line) > 1 {
				content = line[1:]
			}
			switch line[0] {
			case ' ':
				curHunk.Lines = append(curHunk.Lines, HunkLine{Type: "context", Content: content, OldLine: oldLineNo, NewLine: newLineNo})
				oldLineNo++
				newLineNo++
			case '+':
				curHunk.Lines = append(curHunk.Lines, HunkLine{Type: "add", Content: content, OldLine: 0, NewLine: newLineNo})
				newLineNo++
			case '-':
				curHunk.Lines = append(curHunk.Lines, HunkLine{Type: "del", Content: content, OldLine: oldLineNo, NewLine: 0})
				oldLineNo++
			}
		}
	}

	if cur != nil {
		finish()
	}
	return diffs
}

// A mode-only change has no ---/+++ or Binary line, so the path comes from "diff --git a/P b/P".
// The header is ambiguous when the path contains " b/", but both halves name the same path here.
func headerPath(header string) string {
	rest := strings.TrimPrefix(header, "diff --git ")
	n := (len(rest) - len("a/ b/")) / 2
	if n <= 0 || len(rest) != 2*n+len("a/ b/") || !strings.HasPrefix(rest, "a/") {
		return ""
	}
	p := rest[2 : 2+n]
	if rest[2+n:] != " b/"+p {
		return ""
	}
	return p
}

// Binary diffs have no ---/+++ lines; the paths only appear in "Binary files a/x and b/x differ".
func binaryPaths(line string) (oldPath, newPath string) {
	body := strings.TrimSuffix(strings.TrimPrefix(line, "Binary files "), " differ")
	switch {
	case strings.HasPrefix(body, "/dev/null and "):
		return "", strings.TrimPrefix(strings.TrimPrefix(body, "/dev/null and "), "b/")
	case strings.HasSuffix(body, " and /dev/null"):
		return strings.TrimPrefix(strings.TrimSuffix(body, " and /dev/null"), "a/"), ""
	}
	if before, after, ok := strings.Cut(body, " and b/"); ok {
		return strings.TrimPrefix(before, "a/"), after
	}
	return "", ""
}

func parseHunkHeader(header string) (oldStart, newStart int) {
	s := strings.TrimPrefix(header, "@@ ")
	if before, _, ok := strings.Cut(s, " @@"); ok {
		s = before
	}
	for part := range strings.FieldsSeq(s) {
		if strings.HasPrefix(part, "-") {
			n, _, _ := strings.Cut(part[1:], ",")
			oldStart, _ = strconv.Atoi(n)
		} else if strings.HasPrefix(part, "+") {
			n, _, _ := strings.Cut(part[1:], ",")
			newStart, _ = strconv.Atoi(n)
		}
	}
	return
}
