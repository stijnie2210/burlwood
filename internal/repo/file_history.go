package repo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"burlwood/internal/gitcli"
)

type FileHistoryEntry struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Subject   string `json:"subject"`
	Author    string `json:"author"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status"` // "A" | "M" | "D" | "R" | "C"
	Path      string `json:"path"`
	OldPath   string `json:"oldPath"`
}

type BlameChunk struct {
	Hash        string   `json:"hash"`
	ShortHash   string   `json:"shortHash"`
	Author      string   `json:"author"`
	Timestamp   int64    `json:"timestamp"`
	Summary     string   `json:"summary"`
	StartLine   int      `json:"startLine"`
	Lines       []string `json:"lines"`
	Uncommitted bool     `json:"uncommitted"`
}

const fileHistoryLimit = 1000

// --follow keeps going across renames, so each entry carries the name the file had in that commit.
func (m *Manager) GetFileHistory(repoPath, path string) ([]FileHistoryEntry, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	out, err := gitcli.Run(repoPath, "-c", "core.quotePath=false", "log", "--follow", "-M",
		"--format=%x1e%H%x1f%h%x1f%an%x1f%at%x1f%s", "--name-status",
		"-n", strconv.Itoa(fileHistoryLimit), "--", path)
	if err != nil {
		return nil, err
	}
	entries := []FileHistoryEntry{}
	for _, record := range strings.Split(out.Stdout, "\x1e") {
		header, rest, _ := strings.Cut(strings.TrimLeft(record, "\n"), "\n")
		fields := strings.SplitN(header, "\x1f", 5)
		if len(fields) != 5 {
			continue
		}
		ts, _ := strconv.ParseInt(fields[3], 10, 64)
		entry := FileHistoryEntry{
			Hash: fields[0], ShortHash: fields[1], Author: fields[2], Timestamp: ts, Subject: fields[4],
			Status: "M", Path: path,
		}
		for _, line := range strings.Split(rest, "\n") {
			cols := strings.Split(line, "\t")
			if len(cols) < 2 || cols[0] == "" {
				continue
			}
			entry.Status = cols[0][:1]
			entry.Path = cols[len(cols)-1]
			if len(cols) == 3 {
				entry.OldPath = cols[1]
			}
			break
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (m *Manager) GetFileDiffAtCommit(repoPath, hash, path, oldPath string) ([]FileDiff, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	if !fullHash.MatchString(hash) {
		return nil, fmt.Errorf("invalid commit %q", hash)
	}
	prefs, _ := LoadPrefs()
	ctx := fmt.Sprintf("-U%d", prefs.DiffContextLines)
	paths := []string{"--", path}
	if oldPath != "" {
		paths = append(paths, oldPath)
	}
	args := append([]string{"-c", "core.quotePath=false", "diff-tree", "--no-commit-id", "-p", "--root", "-M", ctx, hash}, paths...)
	result, err := gitcli.Run(repoPath, args...)
	if err != nil {
		return nil, err
	}
	if result.Stdout == "" && revParse(repoPath, hash+"^2") != "" {
		args = append([]string{"-c", "core.quotePath=false", "diff", "-M", ctx, hash + "^1", hash}, paths...)
		if result, err = gitcli.Run(repoPath, args...); err != nil {
			return nil, err
		}
	}
	return emptyIfNil(parseUnifiedDiff(result.Stdout)), nil
}

// rev is a full commit hash, or empty for the working tree (uncommitted lines included).
func (m *Manager) GetBlame(repoPath, path, rev string) ([]BlameChunk, error) {
	if _, err := m.Get(repoPath); err != nil {
		return nil, err
	}
	args := []string{"-c", "core.quotePath=false", "blame", "--porcelain"}
	if rev != "" {
		if !fullHash.MatchString(rev) {
			return nil, fmt.Errorf("invalid commit %q", rev)
		}
		args = append(args, rev)
	}
	out, err := gitcli.Run(repoPath, append(args, "--", path)...)
	if err != nil {
		return nil, err
	}
	if strings.ContainsRune(out.Stdout, 0) {
		return nil, errors.New("this looks like a binary file, so there are no lines to blame")
	}
	return parseBlame(out.Stdout), nil
}

// Porcelain blame: a "<hash> <orig> <final> [<count>]" header, the commit's details the first
// time that commit appears, then the line itself after a tab. Consecutive lines from one commit
// become one chunk.
func parseBlame(out string) []BlameChunk {
	type meta struct {
		author  string
		ts      int64
		summary string
	}
	commits := map[string]*meta{}
	chunks := []BlameChunk{}
	var hash string
	var lineNo int
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if content, ok := strings.CutPrefix(line, "\t"); ok {
			info := commits[hash]
			if n := len(chunks); n > 0 && chunks[n-1].Hash == hash && chunks[n-1].StartLine+len(chunks[n-1].Lines) == lineNo {
				chunks[n-1].Lines = append(chunks[n-1].Lines, content)
				continue
			}
			chunk := BlameChunk{Hash: hash, StartLine: lineNo, Lines: []string{content}}
			chunk.Uncommitted = strings.Trim(hash, "0") == ""
			if len(hash) >= 7 {
				chunk.ShortHash = hash[:7]
			}
			if info != nil {
				chunk.Author, chunk.Timestamp, chunk.Summary = info.author, info.ts, info.summary
			}
			chunks = append(chunks, chunk)
			continue
		}
		key, value, _ := strings.Cut(line, " ")
		switch {
		case fullHash.MatchString(key):
			hash = key
			if fields := strings.Fields(value); len(fields) >= 2 {
				lineNo, _ = strconv.Atoi(fields[1])
			}
			if commits[hash] == nil {
				commits[hash] = &meta{}
			}
		case key == "author":
			commits[hash].author = value
		case key == "author-time":
			commits[hash].ts, _ = strconv.ParseInt(value, 10, 64)
		case key == "summary":
			commits[hash].summary = value
		}
	}
	return chunks
}
