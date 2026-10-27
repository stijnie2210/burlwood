# Burlwood — Development Reference

GitKraken replacement. Single-binary desktop app: **Wails v3** (Go backend + Vue 3 frontend, no HTTP server, direct Go↔Vue bindings via auto-generated TypeScript).

## Code style

Write no comments by default. Only add one when the WHY is non-obvious: a hidden constraint, a subtle invariant, a workaround for a specific bug, or behaviour that would surprise a reader. Never describe what the code does — well-named identifiers do that. No multi-line docstrings or block comments.

Always use braces for if statements — never single-line `if (x) return` or `if (x) doThing()`.

## Commands

```bash
wails3 dev                     # hot-reload dev build (regenerates TS bindings automatically)
wails3 task darwin:package     # production .app bundle → bin/Burlwood.app (wails3 build: just the binary)
wails3 generate bindings -clean=true -ts -i   # regenerate frontend/bindings only
go build ./...                 # Go-only compile check
go test ./internal/... && go vet ./internal/...

# in frontend/
npm test             # vitest
npm run lint         # eslint
npm run build        # vue-tsc type-check + vite build
npx prettier --check src
```

CI (`.github/workflows/ci.yml`) runs all of the above on PRs to `main`; lint and format are non-blocking there.

After adding Go methods to `App` or new structs crossing the boundary, `wails3 dev` / `wails3 build` regenerates `frontend/bindings/` (commit it: CI's frontend checks use it). Never edit those files by hand.

Builds run through Taskfiles: `Taskfile.yml` and `build/{darwin,windows,linux}/Taskfile.yml`, trimmed from the `wails3 init` template. `build/config.yml` holds the product info that `wails3 task common:update:build-assets` renders into `build/darwin/Info*.plist` and `build/windows/`. macOS builds always add `-tags private_mac_apis`: without it the webview stays opaque and the sidebar loses its vibrancy. Linux builds use GTK 4 / WebKitGTK 6.0 (Wails' default; the `gtk3` tag goes away in v3.1).

## Architecture

### Git access
- **System `git`** for nearly everything, reads included: log, refs, diffs, status, staging, rebase, fetch/pull/push. Plain wrappers live in `internal/gitcli/`; parsing and multi-step logic in `internal/repo/`
- **go-git** only for `PlainOpen` in the `Manager` and the `FilePatch` types `buildHunks()` consumes
- `gitcli.Run` passes the user's login-shell `PATH` (`env.go`): macOS launches GUI apps with a bare `PATH`, so repo hooks wouldn't find Homebrew/nvm tools (skipped on Windows)
- Start git only through `gitcli.Run`/`RunWithInput`: they hide the console window on Windows

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
- Exported methods on `*App` in `app.go` are bound (it's the one registered service); keep `app.go` thin — real logic lives in `internal/repo/`. `main.go` builds the window and menu and hands them to `App.attach` for dialogs
- TypeScript imports: methods from `bindings/burlwood/app`, types as namespaces: `import type * as repo from '.../bindings/burlwood/internal/repo'`
- Go slices arrive as `T[] | null` (a nil slice encodes as `null`): default them with `?? []` where results come in
- Events: `Events.On(name, (ev) => …ev.data)` from `@wailsio/runtime`; Go emits with `app.Event.Emit`

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
  fileops.go                        OpenInDefaultApp, ShowInFinder, OpenTerminal, OpenInEditor,
                                    StashFile, AppendToGitignore, GetFilePatch, DeleteWorkingFile
  open_darwin.go / _windows.go /    per-OS commands behind fileops: open, reveal, terminal
    _other.go                       (open / explorer.exe, wt or PowerShell / xdg-open, $TERMINAL)
  hidewindow_windows.go             git runs with CREATE_NO_WINDOW; without it every call opens a console window
  undo.go                           SnapshotWorkingFile/RestoreWorkingFile (discard undo via hash-object),
                                    HardResetWithUndo/UndoHardReset (stash create + previous HEAD)
internal/repo/
  manager.go                        Manager{map[string]*git.Repository}; Open/Close/Get
                                    RecentEntry → ~/Library/Application Support/burlwood/repos.json; ListRecentRepos drops repos that are gone
  prefs.go / session.go /           AppPrefs (editor, auto-fetch, graph limit, diff context, date format, pull mode),
    windowprefs.go                  open tabs + active tab, window frame — JSON files next to repos.json
  configdir.go                      configPath(): the settings folder; moves the pre-rename `gitgood` folder once
  branches.go                       ListBranches (for-each-ref), GetAheadBehind
  tags.go                           ListTags: newest first, annotated tags peeled to their commit
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
internal/desktop/
  appimage.go                       Run as an AppImage on Linux ($APPIMAGE): writes ~/.local/share/applications/burlwood.desktop
                                    + icon, so GNOME on Wayland shows the icon; skipped when another entry already launches it.
                                    AppID is the GTK application id, which GTK 4 uses as the Wayland app id: keep
                                    StartupWMClass in both .desktop files equal to it, or the dock shows a generic icon
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
    useDiffLayout.ts                shared Unified|Split layout and Wrap toggle for every diff view (localStorage)
  utils/
    gitErrors.ts                    describeGitError: raw git stderr → sentence + next step (toast.error uses it; raw text under Details)
    conflicts.ts                    parseConflicts (see Conflict resolution flow)
    highlight.ts                    Prism syntax highlighting for diff lines
    imageUtils.ts                   isImageFile, imageDataUrl
    date.ts                         formatDate (relative | absolute, from prefs)
    diffRows.ts                     changedRanges (word-level change within a paired line), pairChanges, splitRows
    initials.ts                     author initials for the avatar circles
    folders.ts                      parentDir, orderByFolder (commit diffs are stored in folder order)
    platform.ts                     mac | windows | linux (sets :root.platform-*), shortcut() → ⌘F or Ctrl+F for UI hints
  App.smoke.test.ts                 mounts the whole app on stubbed Wails bindings — catches setup-time crashes
  assets/labels.css                 ref/tag/stash label pills, rendered by graph/RefLabel.vue
  assets/fonts/                     Bricolage Grotesque subset for the welcome wordmark only (SIL OFL)
  stores/
    repos.ts                        tabs[], activeRepo, openRepo, pickAndOpen, closeTab; restoreSession (App shows a loading
                                    screen while `restoring`; drops only tabs whose repo is gone, see RepoGone in session.go)
    branches.ts                     local[], remote[], tags[], aheadBehind, lastFetchedAt — loaded on activeRepo change
    commits.ts                      rows[], selectedHash, diff[], searchQuery — GetCommitGraph(prefs.commitGraphLimit), loads more on demand
    staging.ts                      files[], unstagedFiles, stagedFiles, diff[]
                                    buildHunkPatch() — constructs patch string for git apply
    stash.ts                        stash list for the sidebar
    prefs.ts                        AppPrefs + Preferences modal open state
    toast.ts                        toast queue (ToastStack)
    logModal.ts                     title/content for LogViewerModal (full git output)
  components/
    layout/ToolBar.vue              Title-bar row (hidden-inset, --wails-draggable): repo button, current-branch chip,
                                    Commits|Working Tree segmented control, icon-only git actions (tooltips +
                                    aria-labels), Find commit field (⌘F via the 'burlwood:find' event), Settings
    layout/TabBar.vue               Repo tabs (open/close/switch); only rendered with 2+ repos
    layout/RepoPicker.vue           Recent repos + Open…; shared by the toolbar repo button and the tab bar + (stores/repoPicker.ts)
    layout/Sidebar.vue              Branches, remotes grouped per remote, tags, stashes (collapsible sections); arrow keys move
                                    between options with the same data-list — width set by App.vue via :style
    layout/PreferencesModal.vue     Edits AppPrefs (stores/prefs.ts)
    layout/LogViewerModal.vue       Full git output with Copy (stores/logModal.ts)
    layout/ToastStack.vue           Renders stores/toast.ts
    graph/CommitGraph.vue           @tanstack/vue-virtual virtualizer; canvas-per-row; resizable graph column
                                    (width in localStorage `burlwood:graph-width`); column headers, filter from
                                    commits.searchQuery, status bar (auto-fetch, last fetch, commit count)
    graph/CommitRow.vue             Single row: canvas + shortHash + RefLabels + subject + author initials
    graph/RefLabel.vue              One ref pill (HEAD check, branch, remote cloud, tag, stash)
    graph/CommitDetail.vue          Commit metadata (copy hash, parent links) + CommitFileList / FileDiffView
    graph/CommitFileList.vue        Changed files grouped by folder with A/M/D/R status and +/− totals
    graph/FileDiffView.vue          Diff of the selected file; resizable file list
    graph/CommitContextMenu.vue     Right-click menu for commits; self-clamps via clampMenuPosition in onMounted
    graph/graphRenderer.ts          drawGraphCell() — CELL_W=14, bezier curves for diagonals; the HEAD commit is
                                    drawn as the icon's knot (--brand-sapwood ring, --brand-heartwood core)
    staging/StagingView.vue         File list (unstaged/staged/conflicts) + merge banner + ConflictView/HunkSelector
                                    Arrow key navigation across all three file sections
                                    Right-click context menu (teleported); image preview for image file types
    staging/HunkSelector.vue        Per-hunk Stage/Unstage buttons around diff/DiffLines
    diff/DiffLines.vue              Shared diff renderer: unified or split, wrap, changed-word highlight
    diff/DiffLayoutControls.vue     Unified|Split + Wrap controls (composables/useDiffLayout.ts)
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

Four text sizes, as tokens: `--text-meta` 11 (labels, hashes, line numbers), `--text-row` 12 (list rows, menus, code), `--text-body` 13 (buttons, inputs, messages), `--text-title` 15 (modal titles). Only exceptions: the welcome wordmark (44px, Bricolage Grotesque subset in `assets/fonts/`, used nowhere else) and glyph icons (16px).

### Color theme
All colours are semantic tokens in `style.css`; never hard-code hex in components. Neutral macOS-style greys, colour only where it means something:
- Surfaces: `--surface-sunken` (diffs) → `-base` (canvas) → `-raised` (bars in panels) → `-chrome` (sidebar, tab bar, toolbar) → `-overlay` (menus, modals, toasts); plus `-field`, `-control`, `-hover`, `-selected`
- Text: `--text-primary` / `-secondary` / `-tertiary` (all ≥4.5:1 on every surface), `--text-on-accent`
- Accent: derived from the macOS accent colour (`--accent-system` = `-apple-system-control-accent`, blue fallback): `--accent-fill` for primary buttons and menu hover, `--accent-text`, `--focus-ring`; `html.window-inactive` greys the selection
- Window is translucent (Mac options in main.go); the webview background is transparent, so every area except the sidebar (`--surface-sidebar`) must paint its own background. Other platforms have no vibrancy: `:root:not(.platform-mac)` makes the sidebar opaque and drops the traffic-light inset (`utils/platform.ts` sets the class). Controls use `cursor: default`, never `pointer`
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
- A commit selected outside the loaded rows (a parent past the commit limit, an old tag, or one hidden by Find) shows
  only its files: CommitDetail has no row for its hash, subject and parent links
- Changed-word highlight (`utils/diffRows.ts`) works on UTF-16 units and ASCII word characters: an emoji can split,
  and a mark inside a string literal breaks the syntax colours after it
- Copy hash gives no feedback when the clipboard write fails; select-by-hash and copy-hash each exist twice
  (CommitGraph/CommitDetail, Sidebar selectTag/selectStash)
- ListTags peels nested annotated tags one level only, and a tag that shares a branch's name shows as `tags/<name>`
