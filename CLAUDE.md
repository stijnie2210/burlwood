# gitgood — Development Reference

GitKraken replacement. Single-binary desktop app: **Wails v2** (Go backend + Vue 3 frontend, no HTTP server, direct Go↔Vue bindings via auto-generated TypeScript).

## Code style

Write no comments by default. Only add one when the WHY is non-obvious: a hidden constraint, a subtle invariant, a workaround for a specific bug, or behaviour that would surprise a reader. Never describe what the code does — well-named identifiers do that. No multi-line docstrings or block comments.

Always use braces for if statements — never single-line `if (x) return` or `if (x) doThing()`.

## Commands

```bash
wails dev        # hot-reload dev build (regenerates TS bindings automatically)
wails build      # production .app bundle → build/bin/gitgood.app
go build ./...   # Go-only compile check
```

After adding Go methods to `App` or new structs crossing the boundary, `wails dev` / `wails build` regenerates `frontend/wailsjs/go/main/App.d.ts` and `frontend/wailsjs/go/models.ts`. Never edit those files by hand.

## Architecture

### Hybrid git rule
- **go-git** for reads: log, branches, refs, commit objects, tree diffs
- **system `git` (shell-out via `internal/gitcli/`)** for writes and complex ops: staging, hunk apply, rebase, non-FF merge, fetch/pull/push

### Layers
```
app.go              ← thin Wails-bound coordinator; all methods call manager
internal/repo/      ← Manager (open repos map), business logic methods
internal/gitcli/    ← pure exec.Command wrappers; no business logic
internal/graph/     ← lane assignment algorithm (no external deps)
frontend/src/stores/     ← Pinia stores (one per domain)
frontend/src/components/ ← Vue components, grouped by feature
```

### Store rules
- Every Pinia store ends with the `acceptHMRUpdate` block; without it `wails dev` keeps running the old store logic after an edit.
- Loads that depend on the repo check `isActive(repoPath)` (from `stores/repos.ts`) before starting and after every await, so late results (auto-fetch, slow git calls) never touch another repo's view.

### Wails binding rules
- Every Go struct that crosses the boundary needs `json:"camelCase"` tags on every field
- Methods on `*App` in `app.go` are auto-bound; keep `app.go` thin — real logic lives in `internal/repo/`
- TypeScript imports: methods from `wailsjs/go/main/App`, types from `wailsjs/go/models`

## File Map

```
app.go                              All bound methods
internal/gitcli/
  runner.go                         Run() / RunWithInput() — exec.Command wrapper
  staging.go                        StageFile, UnstageFile, StageFiles/UnstageFiles (one call, paths via stdin),
                                    ApplyPatch, DiscardFile
  remote.go                         FetchAll, Pull, PushTag/ForcePushTag
  push.go                           PushBranch(branch, force): upstream from config or -u on the default remote;
                                    non-fast-forward → Rejected result (not an error); force = --force-with-lease --force-if-includes
                                    (the bare lease is defeated by background auto-fetch)
  amend.go                          GetAmendInfo: full message (%B) + whether HEAD is already on its upstream
  fileops.go                        OpenInDefaultApp, ShowInFinder, OpenInEditor,
                                    StashFile, AppendToGitignore, GetFilePatch, DeleteWorkingFile
  undo.go                           SnapshotWorkingFile/RestoreWorkingFile (discard undo via hash-object),
                                    HardResetWithUndo/UndoHardReset (stash create + previous HEAD)
internal/repo/
  manager.go                        Manager{map[string]*git.Repository}; Open/Close/Get
                                    RecentEntry → ~/Library/Application Support/gitgood/repos.json
  branches.go                       ListBranches via go-git r.References()
  commits.go                        GetCommitGraph (go-git Log, All:true), GetCommitDiff
                                    buildHunks() — context-limited unified diff (3-line context)
  diff.go                           GetStatus (git status --porcelain), GetWorkingDiff
                                    parseUnifiedDiff() — parses git diff output → []FileDiff
  staging.go                        StageHunk/UnstageHunk (accept patch string from frontend)
                                    BuildHunkPatch, FetchAll, PullBranch
internal/graph/
  layout.go                         BuildGraph() — greedy lane assignment
                                    GraphRow{hash, column, color, edgesIn, edges, maxColumn}
frontend/src/
  App.vue                           Root: ToolBar (title-bar row, v-model:view-mode) + TabBar (2+ repos only)
                                    + RepoPicker + resizable Sidebar + main panel; viewMode ref ('commits'|'staging')
  composables/
    useContextMenu.ts               clampMenuPosition(el, x, y), handleMenuKeydown (↑/↓/Esc), menuEventFor (Shift+F10)
    useDialog.ts                    Esc, initial focus, Tab trap and focus return for modals
    useLineBudget.ts                renders the first 2000 diff lines until "Show more"; hunk actions keep the full hunk
    usePush.ts                      pushWithFeedback: names the destination; rejection → red "Force push (with lease)" toast
  utils/gitErrors.ts                describeGitError: raw git stderr → sentence + next step (toast.error uses it; raw text under Details)
  App.smoke.test.ts                 mounts the whole app on stubbed Wails bindings — catches setup-time crashes
  stores/
    repos.ts                        tabs[], activeRepo, openRepo, pickAndOpen, closeTab
    branches.ts                     local[], remote[] — loaded on activeRepo change
    commits.ts                      rows[], selectedHash, diff[] — GetCommitGraph(limit=2000)
    staging.ts                      files[], unstagedFiles, stagedFiles, diff[]
                                    buildHunkPatch() — constructs patch string for git apply
  components/
    layout/ToolBar.vue              Title-bar row (hidden-inset, --wails-draggable): repo button, Commits|Working Tree
                                    segmented control, git actions, Settings
    layout/TabBar.vue               Repo tabs (open/close/switch); only rendered with 2+ repos
    layout/RepoPicker.vue           Recent repos + Open…; shared by the toolbar repo button and the tab bar + (stores/repoPicker.ts)
    layout/Sidebar.vue              Branch list — width set by App.vue via :style
    graph/CommitGraph.vue           @tanstack/vue-virtual virtualizer; canvas-per-row
    graph/CommitRow.vue             Single row: canvas + shortHash + labels + subject
    graph/CommitDetail.vue          Commit diff view; collapsible files; hunk display
    graph/CommitContextMenu.vue     Right-click menu for commits; self-clamps via clampMenuPosition in onMounted
    graph/graphRenderer.ts          drawGraphCell() — CELL_W=14, bezier curves for diagonals
    staging/StagingView.vue         File list (unstaged/staged/conflicts) + merge banner + ConflictView/HunkSelector
                                    Arrow key navigation across all three file sections
                                    Right-click context menu (teleported); image preview for image file types
    staging/HunkSelector.vue        Per-hunk Stage/Unstage buttons; reuses same diff-line CSS
    staging/ConflictView.vue        Conflict resolution: parse markers → per-conflict Accept Ours/Theirs
```

## Key Implementation Details

### Commit graph layout (`internal/graph/layout.go`)
Greedy topological walk. Each commit finds/creates its lane column. When first parent is already in another lane (merge convergence), current lane is freed and a diagonal edge is drawn. Second pass: `rows[i].EdgesIn = rows[i-1].Edges` so canvas-per-row rendering works in virtual scroll context.

### Virtual scroll (`CommitGraph.vue`)
Uses `useVirtualizer(computed(() => ({ count, getScrollElement, estimateSize, overscan })))` — the entire options object must be wrapped in `computed()`, not just `count`. Uses `item.index` for `:key`, not `item.key`.

### Hunk staging flow
1. `GetWorkingDiff(repoPath, path, staged)` → `[]FileDiff`
2. User clicks "Stage Hunk" — frontend calls `buildHunkPatch(path, hunk)` in `stores/staging.ts`
3. `StageHunk(repoPath, patchString)` → `git apply --cached`
4. For unstage: `UnstageHunk` → `git apply --cached --reverse`
Frontend builds the patch (not backend) so the applied patch exactly matches what the user saw.

### Unified diff parsing (`internal/repo/diff.go`)
`parseUnifiedDiff()` handles: `diff --git` headers, `--- /dev/null` / `+++ /dev/null` (new/deleted files), `Binary files` lines, `\ No newline at end of file`, hunk headers `@@ -n,m +n,m @@`. Untracked files use `git diff --no-index /dev/null <path>` (exit code 1 = differences found, not an error).

### Resizable panels
`makeResizer(width: Ref<number>, min, max, direction)` in `App.vue` — returns a mousedown handler. Direction `'right'` = delta is +x; `'left'` = delta is −x (for right-edge panels). Each component that needs local resize (e.g. `StagingView`) defines its own inline version.

### Fonts
macOS system fonts via tokens in `style.css`: `var(--font-ui)` (SF Pro, set on `body`) and `var(--font-mono)` (SF Mono → Menlo) for hashes, paths and diffs. Never use bare `monospace` — WKWebView maps it to Courier. Base text is left-aligned; centre explicitly where intended.

Four text sizes, as tokens: `--text-meta` 11 (labels, hashes, line numbers), `--text-row` 12 (list rows, menus, code), `--text-body` 13 (buttons, inputs, messages), `--text-title` 15 (modal titles). Only exceptions: the welcome wordmark (32px) and glyph icons (16px).

### Color theme
All colours are semantic tokens in `style.css`; never hard-code hex in components. Neutral macOS-style greys, colour only where it means something:
- Surfaces: `--surface-sunken` (diffs) → `-base` (canvas) → `-raised` (bars in panels) → `-chrome` (sidebar, tab bar, toolbar) → `-overlay` (menus, modals, toasts); plus `-field`, `-control`, `-hover`, `-selected`
- Text: `--text-primary` / `-secondary` / `-tertiary` (all ≥4.5:1 on every surface), `--text-on-accent`
- Accent: derived from the macOS accent colour (`--accent-system` = `-apple-system-control-accent`, blue fallback): `--accent-fill` for primary buttons and menu hover, `--accent-text`, `--focus-ring`; `html.window-inactive` greys the selection
- Window is translucent (Mac options in main.go); the webview background is transparent, so every area except the sidebar (`--surface-sidebar`) must paint its own background. Controls use `cursor: default`, never `pointer`
- Git meaning: `--add` / `--del` (+ `-fg`, `-line-bg`), `--modified`, `--untracked`, `--ours` / `--theirs`, `--tag`, `--stash`; `--danger*` and `--warning*` for destructive and in-progress states
- Graph lanes (`internal/graph/layout.go`) are muted and avoid the add/del hues; syntax colours in `assets/prism-theme.css` follow Xcode's dark theme

## Phase Status

- **Phase 1** ✅ — Wails scaffold, repo manager, recent repos, tab bar, sidebar with branches
- **Phase 2** ✅ — Commit graph with lane layout, virtual scroll, commit diff viewer
- **Phase 3** ✅ — Working tree status/diff, stage/unstage files and hunks, fetch/pull, Staging view
- **Phase 4** ✅ — Conflict resolution: merge banner, conflicted file list, ConflictView (Accept Ours/Theirs per chunk), AbortMerge, merge message pre-fill
- **Phase 5** ✅ — Rebasing: StartRebase/ContinueRebase/AbortRebase, rebase banner in Working Tree, conflict resolution reuses ConflictView, sidebar branch context menu "Rebase onto X"
- **Phase 6A** ✅ — Branch context menu: Delete, Rename, Copy name, Push; file context menu in Working Tree (Stage/Discard/Ignore/Stash/Open/Finder/Editor/Copy path/Save patch/Delete)
- **Phase 6B** ⬜ — Branch context menu (medium): Create branch from here, Merge into current, Set upstream, Create tag here
- **Phase 6C** ⬜ — Working tree extras: arrow key navigation ✅, image preview ✅; HTTPS auth via go-keyring for push/pull on private repos

### Conflict resolution flow (`internal/repo/conflict.go`)
`IsInMerge()` checks for `MERGE_HEAD` / `CHERRY_PICK_HEAD`. `GetConflictContent()` calls `git show :1:/:2:/:3:` for base/ours/theirs plus reads the working file (with markers). `ResolveConflict()` writes the file and stages it. `AbortMerge()` runs `git merge --abort`.

Frontend parses markers with `utils/conflicts.ts` (`parseConflicts`, unit-tested) into context + conflict chunks; diff3/zdiff3 `|||||||` sections become a read-only base, never part of "ours". Save & Stage refuses output that still contains `<<<<<<<`/`|||||||`/`>>>>>>>`. Each conflict chunk has Accept Ours / Accept Theirs buttons. "Save & Stage" assembles accepted chunks and calls `ResolveConflict`. Merge message auto-fills commit summary when `isInMerge` becomes true.

### Rebase flow (`internal/gitcli/rebase.go`)
`StartRebase(onto)` → `git rebase <onto>`. `ContinueRebase()` → `git rebase --continue`. `AbortRebase()` → `git rebase --abort`. Working Tree view shows a rebase banner with step/total progress, reuses ConflictView for mid-rebase conflicts.

### Context menu pattern
All context menus are teleported to `<body>` to escape overflow clipping. Position clamping is shared via `composables/useContextMenu.ts`:
- `Sidebar.vue` / `StagingView.vue`: set position on open, then `await nextTick()` + `clampMenuPosition(getElementById(...), x, y)` to update
- `CommitContextMenu.vue`: self-contained component; clamps in `onMounted` using a `menuEl` ref
- Multi-mode menus use a `mode` ref (`'default' | 'ignore' | 'confirm-delete'` etc.) to render different states inside the same positioned div — avoids closing/reopening for confirmation flows
- Global `.ctx-menu` / `.ctx-menu-item` / `.ctx-menu-divider` styles live in an unscoped `<style>` block in `Sidebar.vue`

### Branch context menu (`Sidebar.vue` + `internal/gitcli/branches.go`)
**Phase 6A** items (all gitcli shell-outs):
- **Delete**: `git branch -d <name>`; on failure offer force-delete (`-D`) via confirmation
- **Rename**: `git branch -m <old> <new>`; name input via inline prompt
- **Copy name**: `navigator.clipboard.writeText()`
- **Push**: reuse existing `Push(repoPath, branch)` from `internal/gitcli/remote.go`

**Phase 6B** items:
- **Create branch from here**: `git checkout -b <name> <base>` — needs name input modal
- **Merge into current**: `git merge <branch>` — conflicts land in Working Tree view
- **Set upstream**: `git branch --set-upstream-to=origin/<name>`
- **Create tag**: `git tag <name> <hash>` — needs name input modal

### Working tree file context menu (`StagingView.vue` + `internal/gitcli/fileops.go`)
Right-click on any file item (conflicts, unstaged, staged). Items are contextual per section:
- **Unstaged**: Stage, Discard, Ignore (submenu: file/ext/folder → AppendToGitignore), Stash file, Open in default app, Show in Finder, Open in editor, Copy path, Save patch (runtime.SaveFileDialog), Delete (confirm-delete mode)
- **Staged**: Unstage, then common items
- **Conflict**: common items only
`OpenInEditor` checks `$VISUAL` then `$EDITOR` env vars, falls back to `open`.
`SavePatchFile` tries `git diff --`, then `git diff --cached --`, then `--no-index /dev/null` for untracked files.

### Image preview (`StagingView.vue` + `app.go`)
`GetFileBase64(repoPath, filePath)` reads the file and returns base64. Frontend detects image extensions (png, jpg, jpeg, gif, webp, svg, bmp, ico, tiff, avif) via `isImageFile()`, builds a `data:<mime>;base64,<b64>` src, and shows `<img>` instead of HunkSelector. Watches `[selectedPath, selectedMode]` to reload.

### Arrow key navigation (`StagingView.vue`)
`navigableFiles` computed flattens conflicts → unstaged → staged into a single list. Document-level `keydown` handler (added in `onMounted`, removed in `onUnmounted`) calls `navigateTo(±1)`, skipping events when `e.target` is an input or textarea. After selection, `scrollIntoView({ block: 'nearest' })` runs in `nextTick`.

## Pending Small Items

- HTTPS auth via go-keyring for push/pull on private repos (SSH agent already works)
