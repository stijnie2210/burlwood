package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"burlwood/internal/gitcli"
	"burlwood/internal/graph"
	"burlwood/internal/repo"
)

type App struct {
	app     *application.App
	window  *application.WebviewWindow
	manager *repo.Manager
}

func NewApp() *App {
	return &App{
		manager: repo.NewManager(),
	}
}

func (a *App) attach(app *application.App, window *application.WebviewWindow) {
	a.app = app
	a.window = window
}

// Returns the repository root, which differs from path when a subfolder was picked.
func (a *App) OpenRepository(path string) (string, error) {
	root, err := gitcli.RepoRoot(path)
	if err != nil {
		return "", err
	}
	if err := a.manager.Open(root); err != nil {
		return "", err
	}
	_ = repo.SaveRecentRepo(root)
	return root, nil
}

func (a *App) CloseRepository(path string) {
	a.manager.Close(path)
}

func (a *App) ListOpenRepositories() []string {
	return a.manager.OpenPaths()
}

type RecentRepo struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func (a *App) ListRecentRepositories() ([]RecentRepo, error) {
	entries, err := repo.ListRecentRepos()
	if err != nil {
		return nil, err
	}
	result := make([]RecentRepo, len(entries))
	for i, e := range entries {
		result[i] = RecentRepo{Path: e.Path, Name: e.Name}
	}
	return result, nil
}

func (a *App) ListBranches(repoPath string) ([]repo.BranchInfo, error) {
	return a.manager.ListBranches(repoPath)
}

func (a *App) ListTags(repoPath string) ([]repo.TagInfo, error) {
	return a.manager.ListTags(repoPath)
}

func (a *App) GetAheadBehind(repoPath string) (repo.AheadBehind, error) {
	return a.manager.GetAheadBehind(repoPath)
}

func (a *App) PickDirectory() (string, error) {
	return a.app.Dialog.OpenFile().
		SetTitle("Open Repository").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		AttachToWindow(a.window).
		PromptForSingleSelection()
}

func (a *App) GetCommitGraph(repoPath string, limit int) ([]graph.GraphRow, error) {
	return a.manager.GetCommitGraph(repoPath, limit)
}

func (a *App) GetCommitDiff(repoPath, hash string) ([]repo.FileDiff, error) {
	return a.manager.GetCommitDiff(repoPath, hash)
}

func (a *App) GetStatus(repoPath string) ([]repo.FileStatus, error) {
	return a.manager.GetStatus(repoPath)
}

func (a *App) GetWorkingDiff(repoPath, path string, staged bool) ([]repo.FileDiff, error) {
	return a.manager.GetWorkingDiff(repoPath, path, staged)
}

func (a *App) StageFile(repoPath, path string) error {
	return a.manager.StageFile(repoPath, path)
}

func (a *App) UnstageFile(repoPath, path string) error {
	return a.manager.UnstageFile(repoPath, path)
}

func (a *App) StageFiles(repoPath string, paths []string) error {
	return a.manager.StageFiles(repoPath, paths)
}

func (a *App) UnstageFiles(repoPath string, paths []string) error {
	return a.manager.UnstageFiles(repoPath, paths)
}

func (a *App) StageHunk(repoPath, patch string) error {
	return a.manager.StageHunk(repoPath, patch)
}

func (a *App) UnstageHunk(repoPath, patch string) error {
	return a.manager.UnstageHunk(repoPath, patch)
}

func (a *App) DiscardFile(repoPath, path string) (gitcli.DiscardSnapshot, error) {
	return a.manager.DiscardFile(repoPath, path)
}

func (a *App) UndoDiscard(repoPath string, snap gitcli.DiscardSnapshot) error {
	return a.manager.UndoDiscard(repoPath, snap)
}

func (a *App) FetchAll(repoPath string) error {
	return a.manager.FetchAll(repoPath)
}

func (a *App) PullBranch(repoPath, mode string) error {
	return a.manager.PullBranch(repoPath, mode)
}

func (a *App) SyncAutostash(repoPath string) error {
	return a.manager.SyncAutostash(repoPath)
}

func (a *App) PushBranch(repoPath, branch string, force bool) (gitcli.PushResult, error) {
	return gitcli.PushBranch(repoPath, branch, force)
}

func (a *App) Stash(repoPath string) error {
	return a.manager.Stash(repoPath)
}

func (a *App) StashPop(repoPath string) error {
	return a.manager.StashPop(repoPath)
}

func (a *App) CreateBranch(repoPath, name string) error {
	return a.manager.CreateBranch(repoPath, name)
}

func (a *App) OpenTerminal(repoPath string) error {
	return gitcli.OpenTerminal(repoPath)
}

func (a *App) Commit(repoPath, message string, amend bool) error {
	return a.manager.Commit(repoPath, message, amend)
}

func (a *App) GetAmendInfo(repoPath string) (gitcli.AmendInfo, error) {
	return a.manager.GetAmendInfo(repoPath)
}

func (a *App) CheckoutRef(repoPath, ref string) error {
	return gitcli.CheckoutRef(repoPath, ref)
}

func (a *App) CreateBranchAt(repoPath, name, hash string) error {
	return gitcli.CreateBranchAt(repoPath, name, hash)
}

func (a *App) RevertCommit(repoPath, hash string, commitImmediately bool) error {
	return gitcli.RevertCommit(repoPath, hash, commitImmediately)
}

func (a *App) CreateTag(repoPath, name, hash string) error {
	return gitcli.CreateTag(repoPath, name, hash)
}

func (a *App) MoveTag(repoPath, name, hash string) error {
	return gitcli.MoveTag(repoPath, name, hash)
}

func (a *App) ResetBranch(repoPath, hash, mode string) error {
	return gitcli.ResetBranch(repoPath, hash, mode)
}

func (a *App) HardResetBranch(repoPath, hash string) (gitcli.ResetUndo, error) {
	return gitcli.HardResetWithUndo(repoPath, hash)
}

func (a *App) UndoHardReset(repoPath string, undo gitcli.ResetUndo) error {
	return gitcli.UndoHardReset(repoPath, undo)
}

func (a *App) IsInMerge(repoPath string) (bool, error) {
	return a.manager.IsInMerge(repoPath)
}

func (a *App) GetMergeMessage(repoPath string) (string, error) {
	return a.manager.GetMergeMessage(repoPath)
}

func (a *App) GetConflictContent(repoPath, path string) (repo.ConflictContent, error) {
	return a.manager.GetConflictContent(repoPath, path)
}

func (a *App) ResolveConflict(repoPath, path, content string) error {
	return a.manager.ResolveConflict(repoPath, path, content)
}

func (a *App) AbortMerge(repoPath string) error {
	return a.manager.AbortMerge(repoPath)
}

func (a *App) SwitchBranch(repoPath, target string, trackRemote bool) (repo.SwitchResult, error) {
	return a.manager.SwitchBranch(repoPath, target, trackRemote)
}

func (a *App) IsInRebase(repoPath string) (bool, error) {
	return a.manager.IsInRebase(repoPath)
}

func (a *App) GetRebaseState(repoPath string) (repo.RebaseState, error) {
	return a.manager.GetRebaseState(repoPath)
}

func (a *App) StartRebase(repoPath, onto string) error {
	return a.manager.StartRebase(repoPath, onto)
}

func (a *App) ContinueRebase(repoPath string) error {
	return a.manager.ContinueRebase(repoPath)
}

func (a *App) AbortRebase(repoPath string) error {
	return a.manager.AbortRebase(repoPath)
}

func (a *App) DeleteBranch(repoPath, name string, force bool) error {
	return gitcli.DeleteBranch(repoPath, name, force)
}

func (a *App) RenameBranch(repoPath, oldName, newName string) error {
	return gitcli.RenameBranch(repoPath, oldName, newName)
}

func (a *App) PushTag(repoPath, tagName string) error {
	return gitcli.PushTag(repoPath, tagName)
}

func (a *App) ForcePushTag(repoPath, tagName string) error {
	return gitcli.ForcePushTag(repoPath, tagName)
}

func (a *App) GetFileBase64(repoPath, filePath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, filePath))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (a *App) GetFileAtCommitBase64(repoPath, hash, filePath string) (string, error) {
	out, err := gitcli.Run(repoPath, "show", hash+":"+filePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(out.Stdout)), nil
}

func (a *App) OpenInDefaultApp(repoPath, filePath string) error {
	return gitcli.OpenInDefaultApp(repoPath, filePath)
}

func (a *App) ShowInFinder(repoPath, filePath string) error {
	return gitcli.ShowInFinder(repoPath, filePath)
}

func (a *App) OpenInEditor(repoPath, filePath string) error {
	prefs, _ := repo.LoadPrefs()
	return gitcli.OpenInEditor(repoPath, filePath, prefs.Editor)
}

func (a *App) GetPrefs() (repo.AppPrefs, error) {
	return repo.LoadPrefs()
}

func (a *App) SavePrefs(prefs repo.AppPrefs) error {
	return repo.SavePrefs(prefs)
}

func (a *App) StashFile(repoPath, filePath string) error {
	return gitcli.StashFile(repoPath, filePath)
}

func (a *App) ListStashes(repoPath string) ([]gitcli.StashEntry, error) {
	return a.manager.ListStashes(repoPath)
}

func (a *App) ApplyStash(repoPath, ref string) error {
	return a.manager.ApplyStash(repoPath, ref)
}

func (a *App) DropStash(repoPath, ref string) error {
	return a.manager.DropStash(repoPath, ref)
}

func (a *App) PopStash(repoPath, ref string) error {
	return a.manager.PopStash(repoPath, ref)
}

func (a *App) RenameStash(repoPath, ref, newMessage string) error {
	return a.manager.RenameStash(repoPath, ref, newMessage)
}

func (a *App) AppendToGitignore(repoPath, pattern string) error {
	return gitcli.AppendToGitignore(repoPath, pattern)
}

func (a *App) SavePatchFile(repoPath, filePath string) error {
	patch, err := gitcli.GetFilePatch(repoPath, filePath)
	if err != nil {
		return err
	}
	if patch == "" {
		return fmt.Errorf("no changes to patch")
	}
	savePath, err := a.app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:                "Save patch",
		Filename:             filepath.Base(filePath) + ".patch",
		CanCreateDirectories: true,
		Filters: []application.FileFilter{
			{DisplayName: "Patch files (*.patch)", Pattern: "*.patch"},
		},
		Window: a.window,
	}).PromptForSingleSelection()
	if err != nil || savePath == "" {
		return nil
	}
	return os.WriteFile(savePath, []byte(patch), 0644)
}

func (a *App) DeleteWorkingFile(repoPath, filePath string) error {
	return gitcli.DeleteWorkingFile(repoPath, filePath)
}

func (a *App) LoadSession() (repo.Session, error) {
	s, err := repo.LoadSession()
	if err != nil {
		return repo.Session{}, err
	}
	return *s, nil
}

func (a *App) RepositoryGone(path string) bool {
	return repo.RepoGone(path)
}

func (a *App) SaveSession(tabs []repo.SessionTab, activeIndex int) error {
	return repo.SaveSession(tabs, activeIndex)
}
