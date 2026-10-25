# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

The primary user is the author: a developer who replaced GitKraken with Burlwood and uses it for daily git work across several repositories at once (work projects and hobby projects side by side in tabs). Trade-offs are decided by that workflow.

The repository is public and other developers are welcome to use it, but they are not a separate audience the product bends towards. A feature earns its place by being useful in real daily use, not by matching a competitor's checklist.

## Product Purpose

Burlwood is a desktop git client for reading history and shaping commits: browse the commit graph, inspect diffs, stage files or single hunks, commit, branch, stash, rebase, resolve conflicts, and sync with remotes. It exists so the author has a git client that is free, fast and fully under their own control. Success means it replaces GitKraken for every daily task without reaching for another client.

## Positioning

- **Free and open source:** GPL-3.0, no account, no subscription, no telemetry. Anyone may use, change and redistribute it; distributed versions stay GPL.
- **Small and fast:** one native binary built with Wails (Go backend, Vue frontend in the system webview), no Electron and no local web server. The commit graph scrolls smoothly through thousands of commits.

Native macOS feel, plain-language errors and undo are part of the product, but they are not the claims it competes on.

## Operating Context

- Runs as a macOS app; the interface is HTML/CSS rendered in the system webview (WKWebView), so web techniques apply but macOS conventions set expectations.
- Calls the system `git` for nearly every operation, so the user's own git config, hooks and SSH agent apply. The full git output is always one click away.
- Used alongside an editor and a terminal: the app offers Open in Editor and Open in Terminal, refreshes when the window regains focus, and auto-fetches in the background.
- Several repositories open at once as tabs, restored on the next launch.

## Capabilities and Constraints

- **Built:** commit graph with branch lanes, refs, tags and stashes; commit actions (checkout, branch, tag, revert, soft/mixed/hard reset with undo for hard reset); working tree staging per file or hunk, discard with undo; commit and amend with a warning before rewriting pushed commits; branch switch, rename, delete, push, rebase; stashes; merge and rebase conflict resolution (diff3/zdiff3 aware); fetch, pull with autostash, push with automatic upstream and force-with-lease on rejection; git errors rewritten as a sentence with a next step.
- **Not built yet:** merge into current branch, set upstream, HTTPS authentication for private repositories (SSH via the agent works).
- **Requires** git installed and on the user's PATH.
- **macOS is the target.** Linux (AppImage) and Windows (zip) builds ship because Wails makes them cheap; they are best-effort and design and testing aim at macOS.
- The macOS build is not signed with an Apple Developer ID; users clear the quarantine flag once.
- **Maturity:** early, pre-1.0 hobby project; the README asks users to keep backups.
- **Terminology:** the two main views are "Commits" and "Working Tree"; git's own terms are used as-is (stage, stash, rebase, HEAD, upstream).

## Brand Commitments

- **Name:** Burlwood, formerly gitgood. Written "Burlwood" in prose; the wordmark is lowercase "burlwood". New names avoid "git" as a syllable, per Git's trademark policy.
- **Icon:** wood grain flowing around a knot, with a branch forking at the knot. Source in `assets/brand/burlwood-icon.svg`.
- **Voice:** plain and direct. Messages name what happened and the next step; destructive actions name their consequence.

## Evidence on Hand

- `README.md`: features, install steps per platform, build instructions.
- `assets/brand/`: the app icon (SVG source and 1024px PNG).
- `assets/screenshots/`: commit graph and working tree screenshots, taken before the warm redesign and the rename; they need retaking.
- No user testimonials, download numbers, benchmarks or press exist. Do not invent them.

## Product Principles

1. **The author's daily workflow decides.** Build what real use needs; skip checklist features nobody here uses.
2. **Free, open, no strings.** No accounts, telemetry, upsells or network calls beyond the user's own git remotes.
3. **Small and fast is a feature.** Prefer the system git and existing dependencies over new heavy ones; large repos and big diffs must stay responsive.
4. **Git stays the source of truth.** No hidden state or private formats; what the app shows and does maps to plain git commands, with the raw output available.
5. **macOS first, others unbroken.** Design and test for macOS; don't knowingly break the Linux and Windows builds.

## Accessibility & Inclusion

Full keyboard use (arrow-key lists, Shift+F10 context menus, focus that returns where it came from) and screen-reader semantics are maintained. Text contrast stays at 4.5:1 or better, and motion respects Reduce Motion.
