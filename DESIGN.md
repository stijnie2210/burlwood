---
name: gitgood
description: A calm macOS git client for reading history and shaping commits.
colors:
  accent-fill: "#2f6fdc"
  accent-fill-hover: "#2a66cc"
  accent-text: "#5aa5ff"
  surface-sunken: "#141416"
  surface-base: "#1b1b1d"
  surface-raised: "#222225"
  surface-chrome: "#28282b"
  surface-overlay: "#2e2e31"
  surface-field: "#19191b"
  surface-control: "#38383c"
  surface-control-hover: "#424246"
  surface-selected: "#22355a"
  border-subtle: "#303033"
  border: "#3c3c40"
  border-strong: "#4c4c51"
  text-primary: "#e8e8ea"
  text-secondary: "#c0c0c5"
  text-tertiary: "#a1a1a6"
  text-on-accent: "#ffffff"
  add: "#4fc977"
  add-fg: "#b6ecc6"
  del: "#ff7b72"
  del-fg: "#ffbdb7"
  modified: "#f0a43c"
  untracked: "#5ac8e0"
  danger-fill: "#c42f2a"
  danger-fill-hover: "#b02824"
  tag: "#e3b341"
  stash: "#be96ef"
typography:
  title:
    fontFamily: "-apple-system, BlinkMacSystemFont, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 600
  body:
    fontFamily: "-apple-system, BlinkMacSystemFont, system-ui, sans-serif"
    fontSize: "13px"
    fontWeight: 400
  row:
    fontFamily: "-apple-system, BlinkMacSystemFont, system-ui, sans-serif"
    fontSize: "12px"
    fontWeight: 400
  code:
    fontFamily: "ui-monospace, 'SF Mono', SFMono-Regular, Menlo, monospace"
    fontSize: "12px"
    lineHeight: "19px"
  label:
    fontFamily: "-apple-system, BlinkMacSystemFont, system-ui, sans-serif"
    fontSize: "11px"
    fontWeight: 600
rounded:
  chip: "3px"
  control: "4px"
  button: "5px"
  popover: "6px"
  modal: "10px"
spacing:
  row-y: "4px"
  row-x: "12px"
  section: "8px"
  panel: "24px"
components:
  button-primary:
    backgroundColor: "{colors.accent-fill}"
    textColor: "{colors.text-on-accent}"
    rounded: "{rounded.button}"
    padding: "8px"
  button-primary-hover:
    backgroundColor: "{colors.accent-fill-hover}"
  button-secondary:
    backgroundColor: "{colors.surface-control}"
    textColor: "{colors.text-secondary}"
    rounded: "{rounded.button}"
    padding: "8px"
  button-destructive:
    backgroundColor: "{colors.danger-fill}"
    textColor: "{colors.text-on-accent}"
    rounded: "{rounded.control}"
    padding: "4px 0"
  input:
    backgroundColor: "{colors.surface-field}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.control}"
    padding: "6px 8px"
  menu:
    backgroundColor: "{colors.surface-overlay}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.popover}"
    padding: "4px 0"
  menu-item-hover:
    backgroundColor: "{colors.accent-fill}"
    textColor: "{colors.text-on-accent}"
  list-row-selected:
    backgroundColor: "{colors.surface-selected}"
    textColor: "{colors.text-primary}"
  label-branch:
    textColor: "{colors.accent-text}"
    rounded: "{rounded.chip}"
    padding: "1px 5px"
---

# Design System: gitgood

## Overview

**Creative North Star: "The Lab Notebook"**

gitgood is where a developer reads what happened and decides what happens next. The notebook metaphor sets the priorities: the entries (commit history, diffs, conflicts) are the content, and everything around them is the binding. It should be quiet, neutral, and steady enough to read for hours. The chrome uses macOS's own vocabulary (system fonts, neutral greys, system-blue actions, blue-highlighted menus) so the app feels at home next to Xcode and Finder rather than like a themed web page.

Colour is reserved for meaning. A green line was added and a red line was removed. Orange is a modified file. Blue is the thing you can act on. If a colour does not carry one of those meanings, it should not be there. The UI is dense because git work is dense, but it is never cramped: text stays at or above 11px, every text level clears 4.5:1 contrast, and code gets its own monospace voice.

Destructive operations are written like careful notebook corrections: they name their consequence and can be undone where git allows (discard and hard reset show an Undo toast). Anything that can't be undone, such as force-pushing or force-deleting, gets its own explicit, red-filled step.

**Key Characteristics:**
- Neutral grey surfaces layered by tone, not by borders or shadows
- One accent (system blue) for actions, focus and selection
- Git semantics own the remaining colour: add, delete, modified, untracked, ours/theirs, tag, stash
- SF Pro for the interface, SF Mono for anything copied from or into git
- Four text sizes, no decorative type

## Colors

A neutral dark palette where hue only appears when it encodes git state or an action.

### Primary
- **System Blue** (accent-fill): solid fill for the single primary action in a context (Commit, Save & Stage, Continue Rebase, Push tag), the HEAD label, and the hovered item in every context menu. White text on it clears 4.7:1.
- **Link Blue** (accent-text): blue as text: active toolbar button, current branch, renamed-file status, focus borders, and the "theirs" side of a conflict.

### Secondary
- **Diff Green** (add) and **Diff Red** (del): added and removed lines, A/D status letters, ahead count, and the "ours" side of a conflict (green). Their pale variants (add-fg, del-fg) colour the code inside added and removed lines.
- **Signal Orange** (modified): modified files, the behind count, in-progress warnings such as the merge banner.
- **Untracked Cyan** (untracked): untracked files only.
- **Brass** (tag) and **Lilac** (stash): tag and stash labels, nothing else.

### Neutral
- **Ink Well** (surface-sunken): behind diffs and code, the deepest layer.
- **Notebook Page** (surface-base): the main canvas, commit graph background.
- **Ruled Line** (surface-raised): bars inside panels, such as hunk headers and file action bars.
- **Binding** (surface-chrome): sidebar, tab bar, toolbar, file panel. Lighter than the page, as macOS chrome is.
- **Loose Leaf** (surface-overlay): menus, popovers, modals, toasts.
- **Primary / Secondary / Tertiary text**: content, supporting text, and metadata (hashes, dates, line numbers). Tertiary is the floor: never go dimmer for readable text.

### Named Rules
**The Meaning-Only Rule.** Every hue in the interface must map to a git state or an action. A new colour needs a new meaning.

**The No-Shared-Red Rule.** Red means removal or destruction, never a neutral action. Unstage is blue, not red. Graph lanes never use the add or delete hues.

**The Token Rule.** Components use `var(--token)` from `frontend/src/style.css`. A hex value in a component is a bug.

## Typography

**Interface Font:** SF Pro via `-apple-system` (falls back to system-ui)
**Code Font:** SF Mono via `ui-monospace` (falls back to Menlo)

**Character:** The platform's own pairing. SF Pro reads as part of macOS; SF Mono marks anything that is literally git data (hashes, paths, diffs, branch names in inputs).

### Hierarchy
- **Title** (600, 15px): modal titles and the welcome line.
- **Body** (400, 13px): buttons, inputs, sidebar branches, messages.
- **Row** (400, 12px): list rows, menu items, commit subjects, and code lines (12px on 19px lines).
- **Label** (600, 11px, sentence case): section headings, badges, metadata, line numbers. No uppercase, no letter-spacing.

### Named Rules
**The Four Sizes Rule.** Text uses `--text-meta`, `--text-row`, `--text-body` or `--text-title`. The only exceptions are the 32px welcome wordmark and 16px glyph icons.

**The Real Mono Rule.** Never write bare `monospace`; WKWebView maps it to Courier. Use `var(--font-mono)`.

## Layout

A fixed desktop shell: repo tab bar, toolbar, then a resizable sidebar beside the main panel, which switches between the commit graph and the working tree. Panels resize by dragging 4px handles that turn blue on hover. Lists use tight rows (4px vertical, 10–12px horizontal padding) with no gaps between them. Modals and empty states use 24px padding. Base text is left-aligned; the welcome screen and centred empty-state messages set their centring explicitly.

## Elevation & Depth

Flat by default. Panels separate by tone (sunken → base → raised → chrome) and hairline borders (border-subtle), not by shadow. Only layers that float above the window carry a shadow, and there are exactly two.

### Shadow Vocabulary
- **Popover** (`box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5)`): context menus, dropdowns, repo picker, toasts.
- **Modal** (`box-shadow: 0 12px 40px rgba(0, 0, 0, 0.6)`): Preferences and the log viewer.

### Named Rules
**The Floaters-Only Rule.** A shadow means "this is above the window and will go away". Inline content, including image previews, gets a hairline border instead.

## Shapes

Small, consistent radii that echo macOS controls: 3px for ref labels and badges, 4px for inputs and small buttons, 5px for primary buttons and toolbar buttons, 6px for menus and popovers (8px for the larger pull-mode menu), 10px for modals. Fully rounded pills only for count badges, preset chips and toggle tracks; circles only for conflict order numbers and the commit-graph dots.

## Components

### Buttons
- **Shape:** gently rounded (5px; 4px for compact inline buttons)
- **Primary:** System Blue fill, white text, full width in the commit panel. Only one per context.
- **Secondary:** control grey fill, secondary text. The idle commit button uses this until the commit is ready.
- **Destructive:** red fill and white text, used only in a confirmation step that names the consequence (Force delete, Force-push tag, Delete from disk).
- **Toolbar:** transparent icon-over-label buttons; hover gets a faint white wash, the active state gets the control fill and blue text.

### Inputs / Fields
- **Style:** field background (slightly darker than the page), 1px border, 4px radius, row-size text.
- **Focus:** border turns Link Blue. No glow.

### Context Menus
- **Style:** overlay surface, 1px border, 6px radius, popover shadow, 12px row text.
- **Hover:** the whole row fills System Blue with white text, including destructive items, as native menus do.
- **Confirmation states:** the same menu switches mode in place to show a consequence sentence and a button row.

### Ref Labels (chips)
- **HEAD:** solid System Blue, white text.
- **Branch:** blue tint, Link Blue text, blue hairline.
- **Remote:** neutral control grey, secondary text.
- **Tag / Stash:** Brass / Lilac tint with matching text and hairline.

### Lists (branches, files, commits)
- **Rows:** 4px vertical padding, no dividers; hover is a faint white wash, selection is the muted blue surface-selected.
- **File status:** a single coloured letter (M orange, A green, D red, R blue, ? cyan) carries the state, so colour is never the only signal.

### Toasts
- **Style:** overlay surface with primary text; only the icon is coloured (✓ green, ✕ red, i blue). Undo toasts last 10 seconds and show a thin draining bar along the bottom edge.

### Diff Lines (signature component)
- 12px SF Mono on 19px lines over Ink Well. Added lines get a faint green wash with pale green code, removed lines a faint red wash with pale red code. Line numbers sit in 44px tertiary columns, and hunk headers sit on a raised bar.

## Do's and Don'ts

### Do:
- **Do** use the semantic tokens in `style.css` for every colour, size and shadow.
- **Do** give each context one blue primary action and make everything else neutral.
- **Do** name the consequence in any destructive confirmation ("leaves them reachable only through the reflog").
- **Do** offer Undo where git can restore the state, and an explicit red step where it can't.
- **Do** keep metadata at tertiary text or brighter; check 4.5:1 on the lightest surface it can appear on (overlay).

### Don't:
- **Don't** introduce navy or tinted backgrounds; surfaces stay neutral grey.
- **Don't** use green or red for anything other than add/delete (plus ours and destructive actions).
- **Don't** use uppercase letter-spaced micro-labels; labels are 11px semibold in sentence case.
- **Don't** put shadows on inline content; only popovers and modals float.
- **Don't** rely on inherited centring; alignment is left unless a rule centres explicitly.
