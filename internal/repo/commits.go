package repo

import (
	"fmt"
	"strings"
	"time"

	gogitdiff "github.com/go-git/go-git/v5/plumbing/format/diff"

	"burlwood/internal/gitcli"
	"burlwood/internal/graph"
)

func (m *Manager) GetCommitGraph(repoPath string, limit int) ([]graph.GraphRow, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 2000
	}
	return getCommitGraphShell(repoPath, limit)
}


func getCommitGraphShell(repoPath string, limit int) ([]graph.GraphRow, error) {
	labelMap := buildLabelMapShell(repoPath)
	stashWIP, stashInternal := getStashGraph(repoPath)

	result, err := gitcli.Run(repoPath, "log", "--all", "--date-order",
		fmt.Sprintf("-n%d", limit),
		"--pretty=format:%H\t%P\t%an\t%ai\t%s")
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	var commits []graph.CommitNode
	for line := range strings.SplitSeq(result.Stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 5 {
			continue
		}

		hash := parts[0]
		if stashInternal[hash] {
			continue
		}

		var parents []string
		if parts[1] != "" {
			parents = strings.Fields(parts[1])
		}
		if stashWIP[hash] && len(parents) > 1 {
			parents = parents[:1]
		}

		t, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[3])
		dateStr := t.Format("2 Jan 2006")
		if t.IsZero() {
			dateStr = parts[3]
		}

		commits = append(commits, graph.CommitNode{
			Hash:         hash,
			Subject:      parts[4],
			Author:       parts[2],
			Date:         dateStr,
			Timestamp:    t.Unix(),
			ParentHashes: parents,
		})
	}

	return graph.BuildGraph(commits, labelMap), nil
}

func getStashGraph(repoPath string) (wip map[string]bool, internal map[string]bool) {
	wip = make(map[string]bool)
	internal = make(map[string]bool)
	result, err := gitcli.Run(repoPath, "stash", "list", "--format=%H\t%P")
	if err != nil {
		return
	}
	for line := range strings.SplitSeq(result.Stdout, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		wipHash := parts[0]
		wip[wipHash] = true
		parents := strings.Fields(parts[1])
		for i, p := range parents {
			if i > 0 {
				internal[p] = true
			}
		}
	}
	return
}

func buildLabelMapShell(repoPath string) map[string][]graph.Label {
	labelMap := make(map[string][]graph.Label)

	headResult, _ := gitcli.Run(repoPath, "symbolic-ref", "--short", "HEAD")
	headBranch := strings.TrimSpace(headResult.Stdout)

	// %(*objectname) is the peeled (commit) hash for annotated tags; empty otherwise
	result, err := gitcli.Run(repoPath, "for-each-ref",
		"--format=%(refname)\t%(objectname)\t%(*objectname)")
	if err != nil {
		return labelMap
	}

	for line := range strings.SplitSeq(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		refname := parts[0]
		hash := parts[1]
		if len(parts) == 3 && parts[2] != "" {
			hash = parts[2]
		}
		if hash == "" {
			continue
		}

		var label graph.Label
		switch {
		case strings.HasPrefix(refname, "refs/heads/"):
			shortName := strings.TrimPrefix(refname, "refs/heads/")
			if shortName == headBranch {
				label = graph.Label{Name: shortName, Type: "head"}
			} else {
				label = graph.Label{Name: shortName, Type: "branch"}
			}
		case strings.HasPrefix(refname, "refs/remotes/"):
			short := strings.TrimPrefix(refname, "refs/remotes/")
			// origin/HEAD only names the remote's default branch; it crowds out real refs.
			if strings.HasSuffix(short, "/HEAD") {
				continue
			}
			rparts := strings.SplitN(short, "/", 2)
			if len(rparts) == 2 {
				label = graph.Label{Name: rparts[1], Type: "remote", Remote: rparts[0]}
			} else {
				label = graph.Label{Name: short, Type: "remote"}
			}
		case strings.HasPrefix(refname, "refs/tags/"):
			label = graph.Label{Name: strings.TrimPrefix(refname, "refs/tags/"), Type: "tag"}
		default:
			continue
		}

		labelMap[hash] = append(labelMap[hash], label)
	}

	stashResult, _ := gitcli.Run(repoPath, "stash", "list", "--format=%gd\t%H")
	if stashResult.Stdout != "" {
		for line := range strings.SplitSeq(strings.TrimSpace(stashResult.Stdout), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) < 2 || parts[1] == "" {
				continue
			}
			labelMap[parts[1]] = append(labelMap[parts[1]], graph.Label{Name: parts[0], Type: "stash"})
		}
	}

	return labelMap
}


const diffContextLines = 3

type HunkLine struct {
	Type    string `json:"type"`    // "add", "del", "context"
	Content string `json:"content"`
	OldLine int    `json:"oldLine"` // 0 for added lines
	NewLine int    `json:"newLine"` // 0 for deleted lines
}

type Hunk struct {
	Header string     `json:"header"`
	Lines  []HunkLine `json:"lines"`
}

type FileDiff struct {
	OldPath  string `json:"oldPath"`
	NewPath  string `json:"newPath"`
	Hunks    []Hunk `json:"hunks"`
	IsBinary bool   `json:"isBinary"`
}

func (m *Manager) GetCommitDiff(repoPath, hash string) ([]FileDiff, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	return getCommitDiffShell(repoPath, hash)
}


func getCommitDiffShell(repoPath, hash string) ([]FileDiff, error) {
	prefs, _ := LoadPrefs()
	ctx := fmt.Sprintf("-U%d", prefs.DiffContextLines)
	result, err := gitcli.Run(repoPath, "diff-tree", "--no-commit-id", "-p", "--root", ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("git diff-tree: %w", err)
	}
	if result.Stdout != "" {
		return emptyIfNil(parseUnifiedDiff(result.Stdout)), nil
	}
	// Stash and merge commits have multiple parents and diff-tree prints nothing for them.
	// diff-tree ignores --first-parent, so "-m --first-parent" emitted one diff per parent and
	// every file showed up twice; compare against the first parent directly instead.
	result, err = gitcli.Run(repoPath, "diff", ctx, hash+"^1", hash)
	if err != nil || result.Stdout == "" {
		return []FileDiff{}, nil
	}
	return emptyIfNil(parseUnifiedDiff(result.Stdout)), nil
}

func buildHunks(fp gogitdiff.FilePatch) []Hunk {
	type flatLine struct {
		lineType string
		content  string
		oldLine  int
		newLine  int
	}

	var flat []flatLine
	oldLine, newLine := 1, 1

	for _, chunk := range fp.Chunks() {
		parts := strings.Split(chunk.Content(), "\n")
		if len(parts) > 0 && parts[len(parts)-1] == "" {
			parts = parts[:len(parts)-1]
		}
		switch chunk.Type() {
		case gogitdiff.Equal:
			for _, l := range parts {
				flat = append(flat, flatLine{"context", l, oldLine, newLine})
				oldLine++
				newLine++
			}
		case gogitdiff.Add:
			for _, l := range parts {
				flat = append(flat, flatLine{"add", l, 0, newLine})
				newLine++
			}
		case gogitdiff.Delete:
			for _, l := range parts {
				flat = append(flat, flatLine{"del", l, oldLine, 0})
				oldLine++
			}
		}
	}

	var changeIdx []int
	for i, l := range flat {
		if l.lineType != "context" {
			changeIdx = append(changeIdx, i)
		}
	}
	if len(changeIdx) == 0 {
		return nil
	}

	// Group change indices into windows, merging when gaps are ≤ 2*context.
	type win struct{ start, end int }
	var windows []win

	gs, ge := changeIdx[0], changeIdx[0]
	for _, idx := range changeIdx[1:] {
		if idx-ge <= 2*diffContextLines {
			ge = idx
		} else {
			windows = append(windows, win{max(0, gs-diffContextLines), min(len(flat)-1, ge+diffContextLines)})
			gs, ge = idx, idx
		}
	}
	windows = append(windows, win{max(0, gs-diffContextLines), min(len(flat)-1, ge+diffContextLines)})

	hunks := make([]Hunk, 0, len(windows))
	for _, w := range windows {
		slice := flat[w.start : w.end+1]

		oldStart, newStart, oldCount, newCount := 0, 0, 0, 0
		for _, l := range slice {
			if l.lineType != "add" {
				if oldStart == 0 {
					oldStart = l.oldLine
				}
				oldCount++
			}
			if l.lineType != "del" {
				if newStart == 0 {
					newStart = l.newLine
				}
				newCount++
			}
		}

		hunkLines := make([]HunkLine, len(slice))
		for i, l := range slice {
			hunkLines[i] = HunkLine{Type: l.lineType, Content: l.content, OldLine: l.oldLine, NewLine: l.newLine}
		}
		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)
		hunks = append(hunks, Hunk{Header: header, Lines: hunkLines})
	}
	return hunks
}
