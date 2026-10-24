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
go test ./internal/... && go vet ./internal/...

# in frontend/
npm test             # vitest
npm run lint         # eslint
npm run build        # vue-tsc type-check + vite build
npx prettier --check src
```

CI (`.github/workflows/ci.yml`) runs all of the above on PRs to `main`; lint and format are non-blocking there.

After adding Go methods to `App` or new structs crossing the boundary, `wails dev` / `wails build` regenerates `frontend/wailsjs/go/main/App.d.ts` and `frontend/wailsjs/go/models.ts`. Never edit those files by hand.

## Architecture

### Git access
- **System `git`** for nearly everything, reads included: log, refs, diffs, status, staging, rebase, fetch/pull/push. Plain wrappers live in `internal/gitcli/`; parsing and multi-step logic in `internal/repo/`
- **go-git** only for `PlainOpen` in the `Manager` and the `FilePatch` types `buildHunks()` consumes
- `gitcli.Run` passes the user's login-shell `PATH` (`env.go`): macOS launches GUI apps with a bare `PATH`, so repo hooks wouldn't find Homebrew/nvm tools

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
  env.go                            login-shell PATH for subprocesses (cached once)
  commit.go                         Commit (incl. amend)
  commits.go                        CheckoutRef, CreateBranchAt, RevertCommit, CreateTag, MoveTag, ResetBranch(mode)
  branches.go                       DeleteBranch, RenameBranch
  stash.go                          ListStashes, Apply/Pop/Drop/RenameStash, Stash/StashPop, StashPushNamed, CreateBranch
  rebase.go                         StartRebase, ContinueRebase, AbortRebase
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
  prefs.go / session.go /           AppPrefs (editor, auto-fetch, graph limit, diff context, date format, pull mode),
    windowprefs.go                  open tabs + active tab, window frame — JSON files next to repos.json
  branches.go                       ListBranches (for-each-ref), GetAheadBehind
  commits.go                        GetCommitGraph (git log --all --date-order + stashes + ref labels), GetCommitDiff
                                    buildHunks() — context lines from prefs.DiffContextLines
  diff.go                           GetStatus (git status --porcelain), GetWorkingDiff
                                    parseUnifiedDiff() — parses git diff output → []FileDiff
  checkout.go                       SwitchBranch: plain checkout; if local changes block it, stash → checkout → pop,
                                    returning the conflicted files when the pop conflicts
  pull.go                           PullBranch with autostash; a conflicted pull leaves the stash pending and
                                    SyncAutostash pops/drops it once the merge/rebase concludes (phase re-derived on restart)
  rebase.go                         IsInRebase, GetRebaseState (step/total), Start/Continue/AbortRebase
  conflict.go                       IsInMerge, GetMergeMessage, GetConflictContent, ResolveConflict, AbortMerge
  stash.go                          stash list/apply/pop/drop/rename
  staging.go                        StageHunk/UnstageHunk (accept patch string from frontend), UndoDiscard,
                                    FetchAll, Commit, GetAmendInfo; BuildHunkPatch (test reference for the TS version)
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
    useDetailWidth.ts               shared, localStorage-backed width of the detail / file-list panel
  utils/
    gitErrors.ts                    describeGitError: raw git stderr → sentence + next step (toast.error uses it; raw text under Details)
    conflicts.ts                    parseConflicts (see Conflict resolution flow)
    highlight.ts                    Prism syntax highlighting for diff lines
    imageUtils.ts                   isImageFile, imageDataUrl
    date.ts                         formatDate (relative | absolute, from prefs)
  App.smoke.test.ts                 mounts the whole app on stubbed Wails bindings — catches setup-time crashes
  assets/labels.css                 ref/tag/stash label pills (CommitRow, CommitDetail)
  stores/
    repos.ts                        tabs[], activeRepo, openRepo, pickAndOpen, closeTab
    branches.ts                     local[], remote[] — loaded on activeRepo change
    commits.ts                      rows[], selectedHash, diff[] — GetCommitGraph(prefs.commitGraphLimit), loads more on demand
    staging.ts                      files[], unstagedFiles, stagedFiles, diff[]
                                    buildHunkPatch() — constructs patch string for git apply
    stash.ts                        stash list for the sidebar
    prefs.ts                        AppPrefs + Preferences modal open state
    toast.ts                        toast queue (ToastStack)
    logModal.ts                     title/content for LogViewerModal (full git output)
  components/
    layout/ToolBar.vue              Title-bar row (hidden-inset, --wails-draggable): repo button, Commits|Working Tree
                                    segmented control, git actions, Settings
    layout/TabBar.vue               Repo tabs (open/close/switch); only rendered with 2+ repos
    layout/RepoPicker.vue           Recent repos + Open…; shared by the toolbar repo button and the tab bar + (stores/repoPicker.ts)
    layout/Sidebar.vue              Local/remote branches + stashes (collapsible sections) — width set by App.vue via :style
    layout/PreferencesModal.vue     Edits AppPrefs (stores/prefs.ts)
    layout/LogViewerModal.vue       Full git output with Copy (stores/logModal.ts)
    layout/ToastStack.vue           Renders stores/toast.ts
    graph/CommitGraph.vue           @tanstack/vue-virtual virtualizer; canvas-per-row; resizable graph column
                                    (width in localStorage `gitgood:graph-width`)
    graph/CommitRow.vue             Single row: canvas + shortHash + labels + subject
    graph/CommitDetail.vue          Commit metadata + CommitFileList / FileDiffView
    graph/CommitFileList.vue        Changed files with A/M/D/R status
    graph/FileDiffView.vue          Diff of the selected file; resizable file list
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

### Conflict resolution flow (`internal/repo/conflict.go`)
`IsInMerge()` checks for `MERGE_HEAD` / `CHERRY_PICK_HEAD`. `GetConflictContent()` calls `git show :1:/:2:/:3:` for base/ours/theirs plus reads the working file (with markers). `ResolveConflict()` writes the file and stages it. `AbortMerge()` runs `git merge --abort`.

Frontend parses markers with `utils/conflicts.ts` (`parseConflicts`, unit-tested) into context + conflict chunks; diff3/zdiff3 `|||||||` sections become a read-only base, never part of "ours". Save & Stage refuses output that still contains `<<<<<<<`/`|||||||`/`>>>>>>>`. Each conflict chunk has Accept Ours / Accept Theirs buttons. "Save & Stage" assembles accepted chunks and calls `ResolveConflict`. Merge message auto-fills commit summary when `isInMerge` becomes true.

### Rebase flow (`internal/gitcli/rebase.go` + `internal/repo/rebase.go`)
`StartRebase(onto)` → `git rebase <onto>`. `ContinueRebase()` → `git rebase --continue`. `AbortRebase()` → `git rebase --abort`. `GetRebaseState()` reads step/total from the rebase dir for the Working Tree rebase banner; mid-rebase conflicts reuse ConflictView.

### Context menu pattern
All context menus are teleported to `<body>` to escape overflow clipping. Position clamping is shared via `composables/useContextMenu.ts`:
- `Sidebar.vue` / `StagingView.vue`: set position on open, then `await nextTick()` + `clampMenuPosition(getElementById(...), x, y)` to update
- `CommitContextMenu.vue`: self-contained component; clamps in `onMounted` using a `menuEl` ref
- `RepoPicker.vue`: clamps the same way when it opens
- Multi-mode menus use a `mode` ref (`'default' | 'ignore' | 'confirm-delete'` etc.) to render different states inside the same positioned div — avoids closing/reopening for confirmation flows
- Global `.ctx-menu` / `.ctx-menu-item` / `.ctx-menu-divider` styles live in an unscoped `<style>` block in `Sidebar.vue`

### Branch context menu (`Sidebar.vue` + `internal/gitcli/branches.go`)
- **Delete**: `git branch -d <name>`; on failure offer force-delete (`-D`) via confirmation
- **Rename**: `git branch -m <old> <new>`; name input via inline prompt
- **Copy name**: `navigator.clipboard.writeText()`
- **Push**: `pushWithFeedback` (`composables/usePush.ts`) → `PushBranch` in `internal/gitcli/push.go`
- **Rebase onto X**: `StartRebase`
- Not built yet (see Open Work): **Merge into current** (`git merge <branch>`, conflicts land in Working Tree) and **Set upstream** (`git branch --set-upstream-to=origin/<name>`)

The stash section of the sidebar has its own menu: Apply, Pop, Rename, Drop.

### Commit context menu (`CommitContextMenu.vue` + `internal/gitcli/commits.go`)
Checkout, Create branch here (`checkout -b <name> <hash>`), Create tag (then optionally push it), Move tag, Push/Force-push tag, Revert (confirm mode), Reset branch soft/mixed, hard reset via `HardResetBranch` (undoable with `UndoHardReset`), Copy hash. Name inputs and confirmations are `mode` states (`'branch'`, `'tag'`, `'move-tag'`, `'revert-confirm'`, `'tag-created'`).

### Working tree file context menu (`StagingView.vue` + `internal/gitcli/fileops.go`)
Right-click on any file item (conflicts, unstaged, staged). Items are contextual per section:
- **Unstaged**: Stage, Discard, Ignore (submenu: file/ext/folder → AppendToGitignore), Stash file, Open in default app, Show in Finder, Open in editor, Copy path, Save patch (runtime.SaveFileDialog), Delete (confirm-delete mode)
- **Staged**: Unstage, then common items
- **Conflict**: common items only
`OpenInEditor` uses the editor from Preferences, then `$VISUAL`, then `$EDITOR`, and falls back to `open`.
`SavePatchFile` tries `git diff --`, then `git diff --cached --`, then `--no-index /dev/null` for untracked files.

### Image preview (`StagingView.vue` + `app.go`)
`GetFileBase64(repoPath, filePath)` reads the file and returns base64 (`GetFileAtCommitBase64` for commit diffs). Frontend detects image extensions (png, jpg, jpeg, gif, webp, svg, bmp, ico, tiff, avif) via `isImageFile()` in `utils/imageUtils.ts`, builds a `data:<mime>;base64,<b64>` src, and shows `<img>` instead of HunkSelector. Watches `[selectedPath, selectedMode]` to reload.

### Arrow key navigation (`StagingView.vue`)
`navigableFiles` computed flattens conflicts → unstaged → staged into a single list. Document-level `keydown` handler (added in `onMounted`, removed in `onUnmounted`) calls `navigateTo(±1)`, skipping events when `e.target` is an input or textarea. After selection, `scrollIntoView({ block: 'nearest' })` runs in `nextTick`.

## Open Work

- Branch context menu: Merge into current, Set upstream
- HTTPS auth via go-keyring for push/pull on private repos (SSH agent already works)
