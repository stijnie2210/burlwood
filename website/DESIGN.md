---
name: Burlwood website
description: The Flitch, a single-page veneer sample book for a free Mac git client.
colors:
  heartwood: "#2a1a10"
  heartwood-raised: "#3a2618"
  sapwood: "#f3ddb8"
  sapwood-dim: "#d9bf98"
  amber-crown: "#cf8d50"
  amber: "#cc8a4e"
  amber-low: "#b97a44"
  burl: "#8a5230"
  burl-figure-1: "#a86a3c"
  burl-figure-2: "#9d6237"
  burl-figure-3: "#935a33"
  grain-line: "#5a3420"
typography:
  display:
    fontFamily: "'Bricolage Grotesque Variable', 'Bricolage Grotesque', -apple-system, system-ui, sans-serif"
    fontSize: "clamp(60px, 10.4vw, 150px)"
    fontWeight: 750
    lineHeight: 0.92
    letterSpacing: "-0.04em"
  headline:
    fontFamily: "'Bricolage Grotesque Variable', 'Bricolage Grotesque', -apple-system, system-ui, sans-serif"
    fontSize: "clamp(42px, 5.4vw, 72px)"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "-0.035em"
  title:
    fontFamily: "'Bricolage Grotesque Variable', 'Bricolage Grotesque', -apple-system, system-ui, sans-serif"
    fontSize: "clamp(24px, 2.4vw, 32px)"
    fontWeight: 680
    lineHeight: 1.2
    letterSpacing: "-0.015em"
  title-panel:
    fontFamily: "'Bricolage Grotesque Variable', 'Bricolage Grotesque', -apple-system, system-ui, sans-serif"
    fontSize: "30px"
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "-0.02em"
  body:
    fontFamily: "'Bricolage Grotesque Variable', 'Bricolage Grotesque', -apple-system, system-ui, sans-serif"
    fontSize: "18px"
    fontWeight: 400
    lineHeight: 1.55
  body-dense:
    fontFamily: "'Bricolage Grotesque Variable', 'Bricolage Grotesque', -apple-system, system-ui, sans-serif"
    fontSize: "17px"
    fontWeight: 400
    lineHeight: 1.45
  label:
    fontFamily: "ui-monospace, 'SF Mono', SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "13px"
    fontWeight: 600
    lineHeight: 1.4
  code:
    fontFamily: "ui-monospace, 'SF Mono', SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "13.5px"
    fontWeight: 400
    lineHeight: 1.6
rounded:
  none: "0px"
  sm: "4px"
  md: "6px"
spacing:
  gutter: "clamp(16px, 4vw, 56px)"
  leaf-block: "clamp(80px, 10vw, 136px)"
  frame: "8px"
  measure: "34em"
  sheet: "1248px"
components:
  button-solid:
    backgroundColor: "{colors.heartwood}"
    textColor: "{colors.sapwood}"
    rounded: "{rounded.md}"
    padding: "0 22px"
    height: "52px"
  button-solid-hover:
    backgroundColor: "{colors.heartwood-raised}"
  button-outline:
    backgroundColor: "transparent"
    textColor: "{colors.heartwood}"
    rounded: "{rounded.md}"
    padding: "0 22px"
    height: "52px"
  stamp:
    backgroundColor: "{colors.sapwood}"
    textColor: "{colors.heartwood}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "3px 9px"
  inlay:
    backgroundColor: "{colors.heartwood}"
    rounded: "{rounded.none}"
    padding: "{spacing.frame}"
  frame:
    backgroundColor: "{colors.heartwood-raised}"
    rounded: "{rounded.none}"
    padding: "{spacing.frame}"
  cutting-list:
    backgroundColor: "{colors.sapwood}"
    textColor: "{colors.heartwood}"
    typography: "{typography.body-dense}"
    rounded: "{rounded.none}"
    padding: "15px 24px"
  platform-panel:
    backgroundColor: "{colors.sapwood}"
    textColor: "{colors.heartwood}"
    typography: "{typography.body-dense}"
    rounded: "{rounded.none}"
    padding: "clamp(22px, 2.4vw, 32px)"
  command-block:
    backgroundColor: "{colors.heartwood}"
    textColor: "{colors.sapwood}"
    typography: "{typography.code}"
    rounded: "{rounded.sm}"
    padding: "14px 16px"
  copy-button:
    backgroundColor: "{colors.heartwood-raised}"
    textColor: "{colors.sapwood}"
    rounded: "{rounded.sm}"
    padding: "0 10px"
    height: "32px"
---

# Design System: Burlwood website

This file covers the marketing website in `website/` only. The desktop app is a separate Operate surface; its system lives in the repo-root `DESIGN.md`. The two share the brand assets: the icon (grain flowing around a knot), the sapwood and heartwood colours of that knot, and the lowercase "burlwood" wordmark in Bricolage Grotesque. The app keeps those assets as trim on neutral greys. The website builds its whole world out of them.

## Overview

**Creative North Star: "The Flitch"**

A flitch is a log sawn into consecutive leaves that are kept in order and numbered, so their figure can be bookmatched. The page is one flitch. Amber veneer leaves, drawn with the icon's contour grain, open and close it (the hero, and the install leaf). Heartwood leaves sit between them and carry the product: real app screenshots set in as inlays, two short headed passages, and a cutting list. Each leaf has its number stamped in a corner, the way samples in a veneer book are marked.

The grain is the one decorative system, and it is drawn rather than photographed. Contour lines flow around elliptical knots. The whole field is mirrored across the centre gutter, wave as well as knots, so the two halves read as bookmatched leaves. In the hero a stepped burl figure sits behind the split wordmark "burl ● wood", with the HEAD knot from the app icon between the halves. On a fine pointer the grain bends around the cursor, and the bend is mirrored on the opposite leaf. Under Reduce Motion, and on touch, the grain stays still.

Density is low and the type is large: a 150px wordmark, 72px leaf headlines, 18px body. Depth comes from material and borders, never from shadow. Nothing floats. Screenshots are set into their leaf with a solid border.

**Key Characteristics:**
- Two materials alternate. Amber veneer leaves have heartwood text, and heartwood leaves have sapwood text.
- Contour grain is bookmatched across the centre and sanded down to 12–18% wherever copy sits on it.
- Every leaf carries a corner stamp with its number: sapwood fill, heartwood rule, system mono.
- Screenshots are inlays: a flush 8px solid border, no radius, no shadow.
- Bricolage Grotesque is used for everything except git data, which is set in system mono.

## Colors

The palette is one piece of wood, from the dark core through the light rim, with no hue from outside it.

### Primary
- **Heartwood** (heartwood): the dark core. It is the ground of the content leaves and the footer, the text colour on every amber and sapwood surface, the fill of the primary button, the rule around stamps, panels and the cutting list, and the inlay border on veneer. It also sets `theme-color` and the knot's core.
- **Sapwood** (sapwood): the light rim. It is the text colour on heartwood, the wordmark colour, the ring of the HEAD knot, and the paper of stamps, the cutting list and the platform panels.

### Secondary
- **Amber Crown** (amber-crown): the top of the hero veneer and the nav strip's ground, which fades out over 28% of its height. It appears only where the hero gradient starts.
- **Amber** (amber) and **Amber Low** (amber-low): the veneer gradient. The install leaf runs from amber to amber-low. The hero reaches amber-low at 640px and continues to burl.
- **Burl** (burl): the deepest veneer tone. It ends the hero gradient, is the darkest step of the burl figure, and colours the scrollbar thumb.

### Tertiary
- **Burl Figure** (burl-figure-1 → burl-figure-2 → burl-figure-3 → burl): four nested wobbling ellipses behind the wordmark, darkening towards the knot. The steps are sized so the sapwood wordmark sits on the darker three (3.74:1, 4.22:1 and 4.76:1). The outermost step (3.31:1) only carries the tips of the letters. At 60px and up, all of these pass the large-text threshold.
- **Grain Line** (grain-line): contour strokes at 2.4px, 50% opacity, non-scaling.

### Neutral
- **Heartwood Raised** (heartwood-raised): the frame of screenshots set on heartwood leaves, the hover state of the solid button, and the fill of the copy button.
- **Sapwood Dim** (sapwood-dim): secondary text on heartwood, used for leaf copy, intros, the footer and shell comments (9.46:1).
- Hairlines are derived from these colours, not separate tokens. A leaf divider is sapwood at 12%. Table rows are heartwood at 22%, the "Not yet" divider is heartwood at 45% dashed, and kbd and copy-button borders are sapwood at 35%.

### Named Rules
**The Burl Carries Sapwood Rule.** The burl and the burl figure are a ground for sapwood display type only. Heartwood text never sits on burl (2.66:1). It belongs on amber-crown, amber, amber-low (4.73:1 or better) and sapwood (12.64:1). In the hero, all heartwood copy ends above the 640px gradient stop.

**The One Log Rule.** Every colour on the page is a tone of the same wood. Git meaning appears only inside the screenshots.

## Typography

**Display Font:** Bricolage Grotesque Variable (self-hosted through `@fontsource-variable`, with -apple-system and system-ui as fallbacks)
**Body Font:** Bricolage Grotesque Variable, with optical sizing on
**Label/Mono Font:** the system mono (ui-monospace, SF Mono, Menlo)

**Character:** One expressive grotesque runs from the 150px wordmark down to 15px footer text. Tight tracking at display sizes gives it the stamped, carved weight. The system mono marks anything that is git or machine data: versions, leaf numbers, commands and keys.

### Hierarchy
- **Display** (750, clamp(60px, 10.4vw, 150px), 0.92, -0.04em): the split wordmark only.
- **Headline** (700, clamp(42px, 5.4vw, 72px), 1, -0.035em): one per leaf, a short imperative sentence with a full stop ("Read the history.", "Take a leaf.").
- **Title** (680, clamp(24px, 2.4vw, 32px), 1.2): the hero line under the wordmark. **Title Panel** (700, 30px, 1.1) is for platform names.
- **Body** (400, 18px, 1.55): leaf copy, capped at the 34em measure. The hero sub uses clamp(17px, 1.4vw, 19px) at a 46em cap.
- **Body Dense** (17px, 1.45): the cutting list and the platform panels. The cutting-list part names are 19px 700, and its column heads are 14px 600.
- **UI weights**: buttons are 18px 650, nav links 16px 650, the brand wordmark 22px 700 at -0.03em (20px under 720px).
- **Label** (mono 600, 13px): stamps, plus the hero meta line in mono 13px. **Code** (mono, 13.5px, 1.6) is used in command blocks.

### Named Rules
**The Mono Means Machine Rule.** Mono is reserved for what git or the shell would print: version tags, leaf numbers, commands, key glyphs. Prose is never set in mono.

## Layout

The page is one centred column of full-bleed leaves. Content inside a leaf sits on a sheet of at most 1248px, inset by the fluid gutter. Leaves have the leaf-block padding top and bottom. Consecutive heartwood leaves are split by a sapwood hairline at 12%.

- **Hero:** centred. The nav strip (brand on the left, three links on the right, 22px block padding) is followed by mirrored stamps at both top corners (82px from the top), then the wordmark grid (1fr auto 1fr, so the knot holds the exact centre), then the line, the sub, the CTA pair (12px gap, wraps), the mono meta line, and the commit-graph inlay rising into the bottom third.
- **Content leaves:** a 5fr/7fr grid (gap clamp(32px, 4.5vw, 64px)). Copy is on the left and the native-size framed screenshot is pushed to the right edge. A leaf can also put the headline and copy side by side at the top, with a wide frame below aligned to the sheet's left edge.
- **Cutting list and install:** the headline, then an intro (18px above, 44px below), then a full-width table or panel row.
- Screenshots are shown at their native pixel size (`fit-content`, max 100%), never scaled up to fill.

**Responsive:** at 959px and below, leaf grids and the platform row become one column, and the panel dividers turn horizontal. At 719px and below, the right hero stamp is hidden (the mirror needs both corners), stamps move up to 72px, and the hero mask narrows to a taller 300×280 ellipse. Inlays swap to narrower crops of the same app (the commit detail, a narrow hunk), and the cutting list restacks into blocks with the part name above its description.

## Elevation & Depth

There are no shadows anywhere. Depth comes from material and layering. Veneer leaves are lit amber, and content leaves are dark heartwood. Inlays sit in the leaf like a contrasting wood, behind a solid border. The only z-axis is the grain field, set at z-index -1 behind the hero and install copy, with masks that sand it down where copy sits.

### Named Rules
**The Sanded Pitch Rule.** Grain never runs at full strength under copy. In the hero, a radial mask (640×205px, 300×280px on mobile) holds the lines at 12% around the text. On the install leaf, a horizontal mask holds the middle 56% at 18% and leaves both outer edges full.

**The Flush Inlay Rule.** Screenshots get a solid border in the leaf's darker wood (heartwood on veneer, heartwood-raised on heartwood) of exactly the frame width. They get no shadow, no radius, no tilt and no floating.

## Shapes

The form language is square cut. Inlays, frames, stamps, the cutting list and the platform panels all have 0 radius and hard 1–2px rules or the 8px frame. Only the things you press are eased. Buttons use 6px, and command blocks, copy buttons and kbd keys use 4px. The organic shapes come only from the grain: knots are ellipses, and the burl figure is four ellipses perturbed by low-frequency sine (2.5% horizontal, 6% vertical) so they read as cut figure and not as geometry.

## Components

### Buttons
- **Shape:** gently eased corners (6px), a 2px heartwood rule, 52px tall, 0 22px padding, a 10px gap to a 20px stroke icon.
- **Solid (primary):** a heartwood fill with sapwood text. It is used for every download action, with a download icon.
- **Outline:** transparent with heartwood text. It is the secondary route ("Linux · Windows").
- **Hover / Focus:** solid moves to heartwood-raised, and outline fills with heartwood at 12%. Background and colour ease over 0.2s on cubic-bezier(0.16, 1, 0.3, 1). The focus ring is a 2px heartwood outline at a 3px offset. Elsewhere on the page the focus ring is currentColor.
- Buttons live only on amber or sapwood grounds, never on heartwood.

### Stamps (leaf numbers)
- **Style:** a sapwood box, a 2px heartwood rule, mono 13px 600, 3px 9px padding, no radius, no wrapping.
- **Content:** `LEAF NN`, with the release version added on leaf 01 (live from the GitHub releases API, falling back to the bundled version). Nothing else.
- **Placement:** pinned to a corner, never in the reading flow. The hero carries a mirrored pair at both top corners, and the right copy is aria-hidden. Content leaves carry one aria-hidden stamp at the top right.

### Inlays and Frames
- **Inlay** (in the hero, on veneer): a heartwood frame-width border and fill, at min(sheet, 100% minus the gutters).
- **Frame** (on heartwood leaves): heartwood-raised. Its border and fill are set at native size, or wide (at most 893px plus the frame).
- Every screenshot is a real build of the app. Each has a `<picture>` source with a narrower crop under 720px.

### Cutting List (signature)
- A real `<table>` on sapwood paper with heartwood text. The header row has a 2px heartwood rule. Rows are divided by heartwood at 22%, with 15px 24px cells. The 200px part column is in 19px bold.
- The final "Not yet" row is set off by a 2px dashed rule at 45%, and it names honestly what is missing.
- Under 720px the head is hidden and each row restacks into a block.

### Platform Panels
- Three equal panels share one sapwood plate inside a 2px heartwood rule, divided by 2px heartwood rules (horizontal when stacked).
- Each panel is a column with a 14px gap: a 30px name, short instructions, command blocks, then a solid download button pinned to the bottom (`margin-top: auto`), so the buttons align across panels.

### Command Block
- Heartwood `pre` with sapwood mono 13.5/1.6, 4px radius, and 14px 16px padding. Shell comments are in sapwood-dim.
- A single command gets a copy button pinned top-right: heartwood-raised, a 1px sapwood rule at 35% that turns full sapwood on hover, 32px tall, a 16px icon and a 13px 600 label. The label changes to "Copied", or to "Select and copy" if the clipboard write fails, then resets after 2s.

### Navigation
- The nav strip sits on amber-crown, with brand text at 22px 700 and links at 16px 650 in heartwood. Links have no underline and gain one on hover. Link underlines are 1.5px thick at a 0.2em offset.
- The footer is on heartwood: the sapwood brand, sapwood-dim text at 15px, and a sapwood "Source on GitHub" link with an 18px icon.

### Grain Field (signature)
- A procedural SVG (`lib/grain.ts`). Horizontal rows run every 60px (64px on install), each a sine wave of 5px amplitude mirrored about the centre. Each row is displaced away from every knot by a Gaussian falloff. Knots are bookmatched, and the burl figure keeps its own rings out of the knot-ring layer. Every other knot gets two ellipse rings, at 0.6× and 1.05×.
- **Pointer:** only on a fine hover pointer, and only without Reduce Motion. The cursor becomes a 120×78 knot, mirrored onto the opposite leaf. Position eases at 0.16 per frame and strength at 0.08. The animation loop stops once it settles, and on pointerleave the knot relaxes back to zero.

## Do's and Don'ts

### Do:
- **Do** alternate the two materials: amber veneer with heartwood text, and heartwood leaves with sapwood text.
- **Do** mirror every grain field across the centre, knots and wave alike, including the pointer knot.
- **Do** mask the grain to 12–18% under any copy.
- **Do** set screenshots as flush inlays with an 8px solid border in the darker wood, at native size.
- **Do** number each leaf with a corner stamp (`LEAF NN`), in order.
- **Do** keep sapwood display type on the darker burl-figure steps (3.3:1 or better at 60px and up), and heartwood copy on amber or sapwood only.
- **Do** keep the grain static under Reduce Motion and on touch.

### Don't:
- **Don't** add shadows, glows or floating cards. Depth here is material.
- **Don't** put heartwood text on burl (2.66:1).
- **Don't** introduce hues from outside the wood. Git colours stay inside the screenshots.
- **Don't** use a stamp as a topic label or eyebrow above a headline. It carries the leaf number and nothing else.
- **Don't** round inlays, stamps, tables or panels. Only pressable things get 4–6px.
- **Don't** set prose in mono or git data in Bricolage.
