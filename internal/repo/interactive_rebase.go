package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"burlwood/internal/gitcli"
)

type RebaseTodoCommit struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Pushed    bool   `json:"pushed"`
}

type RebaseTodo struct {
	Commits  []RebaseTodoCommit `json:"commits"`
	Upstream string             `json:"upstream"`
}

type RebaseStep struct {
	Hash    string `json:"hash"`
	Action  string `json:"action"` // "pick" | "reword" | "squash" | "fixup" | "drop"
	Message string `json:"message"`
}

type InteractiveRebaseOutcome string

const (
	InteractiveRebaseDone    InteractiveRebaseOutcome = "done"
	InteractiveRebaseStopped InteractiveRebaseOutcome = "stopped"
)

type InteractiveRebaseResult struct {
	Outcome      InteractiveRebaseOutcome `json:"outcome"`
	PreviousHead string                   `json:"previousHead"`
	NewHead      string                   `json:"newHead"`
}

var fullHash = regexp.MustCompile(`^[0-9a-f]{40,64}$`)

// The commits from fromHash up to HEAD, oldest first: the order git replays them in.
func (m *Manager) GetRebaseTodo(repoPath, fromHash string) (RebaseTodo, error) {
	if _, err := m.Get(repoPath); err != nil {
		return RebaseTodo{}, err
	}
	if !fullHash.MatchString(fromHash) {
		return RebaseTodo{}, fmt.Errorf("invalid commit %q", fromHash)
	}
	if _, err := gitcli.Run(repoPath, "merge-base", "--is-ancestor", fromHash, "HEAD"); err != nil {
		return RebaseTodo{}, errors.New("this commit isn't part of the current branch")
	}
	rng := rebaseRange(repoPath, fromHash)
	merges, err := gitcli.Run(repoPath, "rev-list", "--min-parents=2", "--max-count=1", rng)
	if err != nil {
		return RebaseTodo{}, err
	}
	if strings.TrimSpace(merges.Stdout) != "" {
		return RebaseTodo{}, errors.New("the commits from here to HEAD include a merge commit, which an interactive rebase would flatten")
	}

	out, err := gitcli.Run(repoPath, "log", "--reverse", "--format=%H%x1f%h%x1f%an%x1f%B%x1e", rng)
	if err != nil {
		return RebaseTodo{}, err
	}
	todo := RebaseTodo{Commits: []RebaseTodoCommit{}, Upstream: upstreamOf(repoPath, "HEAD")}
	pushed := pushedCommits(repoPath, fromHash, todo.Upstream)
	for _, record := range strings.Split(out.Stdout, "\x1e") {
		parts := strings.SplitN(strings.TrimLeft(record, "\n"), "\x1f", 4)
		if len(parts) != 4 {
			continue
		}
		message := strings.TrimSpace(parts[3])
		subject, _, _ := strings.Cut(message, "\n")
		todo.Commits = append(todo.Commits, RebaseTodoCommit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author:    parts[2],
			Subject:   subject,
			Message:   message,
			Pushed:    pushed[parts[0]],
		})
	}
	return todo, nil
}

// A root commit has no parent to name as the base, so the range is everything up to HEAD.
func rebaseRange(repoPath, fromHash string) string {
	if revParse(repoPath, fromHash+"^") == "" {
		return "HEAD"
	}
	return fromHash + "^..HEAD"
}

// Commits of the range that HEAD's upstream already has: rewriting them means a force push.
func pushedCommits(repoPath, fromHash, upstream string) map[string]bool {
	pushed := map[string]bool{}
	if upstream == "" {
		return pushed
	}
	base, err := gitcli.Run(repoPath, "merge-base", "HEAD", upstream)
	if err != nil {
		return pushed
	}
	shared := strings.TrimSpace(base.Stdout)
	rng := shared
	if revParse(repoPath, fromHash+"^") != "" {
		rng = fromHash + "^.." + shared
	}
	out, err := gitcli.Run(repoPath, "rev-list", rng)
	if err != nil {
		return pushed
	}
	for _, h := range strings.Fields(out.Stdout) {
		pushed[h] = true
	}
	return pushed
}

// git reads the todo list through GIT_SEQUENCE_EDITOR, which copies the prepared one over
// git's own. Reword is a pick followed by an exec that amends the message from a file, so
// several rewords can each carry their own text without an editor.
func (m *Manager) StartInteractiveRebase(repoPath, fromHash string, steps []RebaseStep) (InteractiveRebaseResult, error) {
	todo, err := m.GetRebaseTodo(repoPath, fromHash)
	if err != nil {
		return InteractiveRebaseResult{}, err
	}
	if err := validateRebaseSteps(todo, steps); err != nil {
		return InteractiveRebaseResult{}, err
	}
	gitDir, err := getGitDir(repoPath)
	if err != nil {
		return InteractiveRebaseResult{}, err
	}
	workDir := filepath.Join(gitDir, "burlwood-rebase")
	if err := os.RemoveAll(workDir); err != nil {
		return InteractiveRebaseResult{}, err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return InteractiveRebaseResult{}, err
	}

	var lines []string
	for i, step := range steps {
		switch step.Action {
		case "reword":
			msgPath := filepath.Join(workDir, fmt.Sprintf("%d.msg", i))
			if err := os.WriteFile(msgPath, []byte(strings.TrimSpace(step.Message)+"\n"), 0o644); err != nil {
				return InteractiveRebaseResult{}, err
			}
			lines = append(lines,
				"pick "+step.Hash,
				"exec git commit --amend --quiet --no-verify -F "+shellQuote(msgPath))
		default:
			lines = append(lines, step.Action+" "+step.Hash)
		}
	}
	todoPath := filepath.Join(workDir, "todo")
	if err := os.WriteFile(todoPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return InteractiveRebaseResult{}, err
	}

	args := []string{"rebase", "--interactive", "--autostash"}
	if revParse(repoPath, fromHash+"^") == "" {
		args = append(args, "--root")
	} else {
		args = append(args, fromHash+"^")
	}
	result := InteractiveRebaseResult{PreviousHead: revParse(repoPath, "HEAD")}
	env := []string{"GIT_SEQUENCE_EDITOR=cp " + shellQuote(todoPath), "GIT_EDITOR=true"}
	if _, err := gitcli.RunWithEnv(repoPath, env, args...); err != nil {
		if inRebase, _ := m.IsInRebase(repoPath); inRebase {
			result.Outcome = InteractiveRebaseStopped
			return result, nil
		}
		return InteractiveRebaseResult{}, err
	}
	result.Outcome = InteractiveRebaseDone
	result.NewHead = revParse(repoPath, "HEAD")
	return result, nil
}

func validateRebaseSteps(todo RebaseTodo, steps []RebaseStep) error {
	if len(steps) != len(todo.Commits) {
		return errors.New("the branch has changed since the list was loaded; open it again")
	}
	inRange := map[string]bool{}
	for _, c := range todo.Commits {
		inRange[c.Hash] = true
	}
	kept := 0
	for _, step := range steps {
		if !inRange[step.Hash] {
			return errors.New("the branch has changed since the list was loaded; open it again")
		}
		delete(inRange, step.Hash)
		switch step.Action {
		case "pick":
			kept++
		case "reword":
			if strings.TrimSpace(step.Message) == "" {
				return errors.New("a reworded commit needs a message")
			}
			kept++
		case "squash", "fixup":
			if kept == 0 {
				return errors.New("the first commit can't be squashed: there is nothing before it to squash into")
			}
		case "drop":
		default:
			return fmt.Errorf("unknown action %q", step.Action)
		}
	}
	if kept == 0 {
		return errors.New("every commit would be dropped")
	}
	return nil
}

// git runs the editor and exec lines through sh on every platform (its own on Windows), so
// paths use forward slashes and POSIX single quotes.
func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(filepath.ToSlash(path), "'", `'\''`) + "'"
}

// --keep moves the branch back while carrying uncommitted changes along, and refuses rather
// than overwrite them.
func (m *Manager) UndoInteractiveRebase(repoPath string, r InteractiveRebaseResult) error {
	if _, err := m.Get(repoPath); err != nil {
		return err
	}
	if !fullHash.MatchString(r.PreviousHead) || revParse(repoPath, "HEAD") != r.NewHead {
		return errors.New("the branch has moved since the rebase, so it can't be undone from here")
	}
	_, err := gitcli.Run(repoPath, "reset", "--keep", r.PreviousHead)
	return err
}
