---
name: Notes Web
description: A calm, keyboard-first web workbench for reading, retrieving, and maintaining a Markdown vault.
colors:
  parchment: "#f7f5ef"
  desk-paper: "#fffdfa"
  fresh-ink: "#27231d"
  faded-marginalia: "#786f63"
  pressed-fold: "#e2dccc"
  violet-thread: "#6b5cff"
  deep-violet-thread: "#3d32c2"
  washed-violet: "#efecff"
  workbench-ash-inline: "#eee9dc"
  workbench-ash-block: "#f1eee6"
  red-wax: "#b54747"
  amber-note: "#b76e00"
  moss-mark: "#2b8a3e"
  night-vault: "#11131a"
  night-paper: "#181b24"
  night-ink: "#ebe7df"
  sepia-parchment: "#f4ecd8"
  sepia-paper: "#fff8e8"
  sepia-ink: "#33291c"
typography:
  display:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "40px"
    fontWeight: 800
    lineHeight: 1.2
    letterSpacing: "-0.04em"
  headline:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "22px"
    fontWeight: 800
    lineHeight: 1.2
    letterSpacing: "-0.02em"
  title:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "18px"
    fontWeight: 800
    lineHeight: 1.3
  body:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "17px"
    fontWeight: 400
    lineHeight: 1.74
  label:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "12px"
    fontWeight: 800
    lineHeight: 1.2
    letterSpacing: "0.08em"
  code:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "0.95em"
    fontWeight: 400
    lineHeight: 1.35
rounded:
  sm: "8px"
  md: "12px"
  lg: "18px"
  xl: "22px"
  modal: "24px"
  pill: "999px"
spacing:
  space-1: "4px"
  space-2: "8px"
  space-3: "12px"
  space-4: "16px"
  space-5: "24px"
  space-6: "32px"
  sidebar: "22px"
  main-x: "56px"
  main-y: "40px"
components:
  button-default:
    backgroundColor: "{colors.desk-paper}"
    textColor: "{colors.fresh-ink}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
  button-primary:
    backgroundColor: "{colors.violet-thread}"
    textColor: "{colors.desk-paper}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
  chip:
    backgroundColor: "{colors.washed-violet}"
    textColor: "{colors.deep-violet-thread}"
    rounded: "{rounded.pill}"
    padding: "4px 10px"
  card:
    backgroundColor: "{colors.desk-paper}"
    textColor: "{colors.fresh-ink}"
    rounded: "{rounded.lg}"
    padding: "20px"
  search-input:
    backgroundColor: "{colors.desk-paper}"
    textColor: "{colors.fresh-ink}"
    rounded: "10px"
    padding: "10px 12px"
  palette-panel:
    backgroundColor: "{colors.desk-paper}"
    textColor: "{colors.fresh-ink}"
    rounded: "{rounded.xl}"
    padding: "0"
---

# Design System: Notes Web

## 1. Overview

**Creative North Star: "The Operative Notebook"**

Notes Web should feel like an operative notebook for a private vault: calm enough for long reading, precise enough for maintenance, and fast enough that retrieval feels almost physical. The interface is not neutral. It is personal-first, open source, keyboard-native, and intentionally opinionated.

The system uses warm paper tones, fresh ink contrast, and a single violet thread for active states, commands, focus, and selected navigation. Density is allowed because the product is a real tool, but every dense surface must remain quiet, legible, and purposeful.

The visual system rejects cold documentation wikis, heavy app chrome, generic SaaS dashboards, Obsidian mimicry, and bland neutrality. It should never look like a landing page trapped inside a knowledge tool.

**Key Characteristics:**
- Warm, paper-led product UI with strong reading comfort.
- Restrained accent use: violet marks action, focus, selection, and command flow.
- Tonal layering before shadow.
- Keyboard-first affordances with visible focus and predictable states.
- Personal craft made reusable through explicit tokens and rules.

## 2. Colors

The palette is a restrained paper-and-ink system with one violet accent that acts as a thread through navigation, commands, chips, and active states.

### Primary
- **Violet Thread** (`--accent`): The only primary accent. Use it for primary actions, active navigation, selected filters, command affordances, and focus emphasis.
- **Deep Violet Thread** (`--accent-ink`): Text and link accent on light surfaces. Use when the accent needs readable ink rather than filled color.
- **Washed Violet** (`--soft`): The quiet selected and hover field. Use for active tree branches, command palette rows, tags, and calm emphasis.

### Secondary
- **Red Wax** (`--danger`): Errors, overdue tasks, unresolved links, and destructive states. It should feel decisive, not theatrical.
- **Amber Note** (`--warning`): Warnings and due-soon attention. Use sparingly so it reads as signal, not decoration.
- **Moss Mark** (`--success`): Completion and resolved states. It should feel durable and quiet.

### Neutral
- **Parchment** (`--bg`): The ambient page background. It gives the app warmth without making the vault feel antique.
- **Desk Paper** (`--surface`): The main content and panel surface. It should stay close to the background so layers feel calm.
- **Fresh Ink** (`--ink`): Main text. It is warm black, never pure black.
- **Faded Marginalia** (`--muted`): Secondary metadata, paths, labels, timestamps, and helper text.
- **Pressed Fold** (`--line`): Borders and dividers. It should separate without cutting the page into boxes.
- **Workbench Ash** (`--code`, `--pre`): Inline and block code backgrounds. Technical content should feel matte and integrated.

### Named Rules

**The Violet Thread Rule.** Violet is for orientation and action only: current location, command flow, focus, selected filters, and primary actions. If violet appears as ornament, remove it.

**The Warm Neutral Rule.** Use tinted neutrals for every background and text layer. Pure black and pure white are forbidden in new UI work.

**The Semantic Restraint Rule.** Red Wax, Amber Note, and Moss Mark are state colors, not category colors. Do not decorate dashboards with them.

## 3. Typography

**Display Font:** System sans (`ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif`)  
**Body Font:** System sans (`ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif`)  
**Label/Mono Font:** System mono (`ui-monospace, SFMono-Regular, Menlo, monospace`)

**Character:** Native, fast, and unpretentious. Typography should feel like a tool that belongs on the user's system, with enough weight contrast to guide scanning and enough line-height to support long reading.

### Hierarchy
- **Display** (800, 40px, 1.2, -0.04em): Page titles, note titles, folder titles, and task dashboard heroes.
- **Headline** (800, 22px, 1.2, -0.02em): Brand mark and major section anchors.
- **Title** (800, 18px, 1.3): Search results, card headings, note cards, and dashboard modules.
- **Body** (400, 17px, 1.74): Note reading content. Keep prose readable and avoid squeezing long paragraphs into narrow widget columns.
- **Label** (800, 12px, 0.08em): Eyebrows, toolbar labels, calendar weekdays, compact section labels, and structured metadata.
- **Code** (400, 0.95em, 1.35): Inline code, fenced blocks, task IDs, and copyable technical fragments.

### Named Rules

**The Native Tool Rule.** Use the system sans for product UI. Do not introduce display fonts for labels, buttons, tables, task rows, or navigation.

**The Reading Breath Rule.** Note content earns extra line-height. Dense product modules can compress, but prose should stay comfortable.

## 4. Elevation

Depth comes from flat tonal layering: background, surface, borders, and raised overlays. Shadows are rare, short, and matte. They belong to temporary layers above the document: command palette, dialogs, menus, popovers, and hover feedback on actionable controls.

The existing system includes modal and card shadows, but the design direction is still flat by default. Blur is functional only. It may isolate a command mode, never create decorative glass.

### Shadow Vocabulary
- **Surface Hairline** (`0 1px 0 rgba(39, 35, 29, 0.08)`): A nearly flat separation for buttons and quiet cards.
- **Soft Card** (`0 2px 10px rgba(39, 35, 29, 0.08)`): Low elevation for cards and task groups.
- **Command Lift** (`0 24px 80px rgba(17, 19, 26, 0.34)`): Temporary elevation for command palette and settings dialogs.

### Named Rules

**The Flat Until Invoked Rule.** Resting surfaces are flat. Shadows appear when an element is actionable, hovered, opened, or temporarily above the vault.

**The No Glass Rule.** Backdrop blur may support a modal mode, but glassmorphism is forbidden as a decorative default.

## 5. Components

Every component should behave like a calm tool: obvious at keyboard speed, useful in one glance, simple in shape, and just distinctive enough to belong to Notes Web.

### Buttons
- **Shape:** Gently rounded rectangle (12px radius).
- **Default:** Desk Paper background, Pressed Fold border, Fresh Ink text, compact padding (8px 12px).
- **Primary:** Violet Thread fill with Desk Paper text. Reserve for the main action in a local context.
- **Ghost:** Transparent background with the same shape and focus vocabulary as default buttons.
- **Hover / Focus:** Move by transform only, never layout. Focus must be visible with a violet ring or equivalent high-contrast treatment.

### Chips
- **Style:** Washed Violet background, Deep Violet Thread text, pill radius, compact padding (4px 10px).
- **State:** Selected chips may use Violet Thread fill. Unselected chips should stay quiet.
- **Use:** Tags, filters, task metadata, folder sort controls, and compact status indicators.

### Cards / Containers
- **Corner Style:** Soft but not bubbly (18px radius).
- **Background:** Desk Paper for normal surfaces, never raw white in new work.
- **Shadow Strategy:** Flat by default, Soft Card only when a module needs separation from the paper.
- **Border:** Pressed Fold hairline. Avoid stacking nested cards.
- **Internal Padding:** 16px to 20px for cards, 14px for compact note and result rows, 18px for dense task sections.

### Inputs / Fields
- **Style:** Desk Paper background, Pressed Fold border, 10px to 12px radius, system sans text.
- **Focus:** Violet Thread border with a soft focus ring. Do not rely on color alone if the field has validation state.
- **Search:** Search fields can grow larger (18px to 20px) because retrieval is a primary task.

### Navigation
- **Sidebar:** Sticky desktop rail with warm paper surface, Pressed Fold divider, compact tree spacing, and a calm active branch using Washed Violet.
- **Tree:** Folders and notes should scan quickly. Active branch treatment uses background plus weight, not color alone.
- **Mobile:** Sidebar becomes a drawer overlay. Keep the same hierarchy and focus order.
- **Command Palette:** Treat as the fastest navigation surface. The panel is raised, centered, large enough for typing, and selected rows use Washed Violet.

### Task Rows
- **Style:** Dense grid with checkbox, priority badge, title, metadata, and menu access.
- **State:** Completed tasks reduce opacity and strike the title. Priority badges may use semantic fills, but only when the priority is meaningful.
- **Interaction:** Row menus stay quiet until hover or focus. Keyboard access must match mouse affordances.

### Code Blocks
- **Style:** Workbench Ash backgrounds with matte contrast, 12px radius, and a copy button in the top right.
- **Inline Code:** Small Workbench Ash chip with 5px radius.
- **Use:** Code should feel integrated with prose, not like a separate developer console.

### Callouts
- **Current State:** Legacy callouts use a colored side stripe.
- **Future Direction:** New callouts should use tinted panels, full borders, icons, or title weight instead of side-stripe accents.
- **Reason:** Side stripes create a generic documentation look and conflict with the calmer paper system.

## 6. Do's and Don'ts

### Do:
- **Do** keep Notes Web product-first: retrieval, reading, tasks, and diagnostics beat decorative composition.
- **Do** use Violet Thread only for orientation, action, focus, and selected states.
- **Do** preserve Calm Density: information-rich modules are allowed when spacing, hierarchy, and labels make them scannable.
- **Do** design keyboard-first states for buttons, tree items, command palette rows, task menus, and filters.
- **Do** make diagnostics feel like maintenance, not alarm: broken links and orphans should be clear without turning the page into an error wall.
- **Do** keep personal-first choices explicit so open source users can adapt them without erasing the point of view.

### Don't:
- **Don't** make Notes Web look like a cold documentation wiki that makes personal notes feel impersonal.
- **Don't** add heavy app chrome, decorative motion, unnecessary blur, or framework-like visual bulk.
- **Don't** use generic SaaS dashboard patterns: repeated metric cards, hero-stat layouts, gradient accents, or ornamental panels.
- **Don't** copy Obsidian's desktop app. Serve the web reading and retrieval context instead.
- **Don't** neutralize the product until it loses its personal-first point of view.
- **Don't** add colored side-stripe borders to new callouts, cards, list items, or alerts. Use full borders, tinted panels, icons, or title treatment.
- **Don't** use gradient text, glassmorphism as decoration, pure black, or pure white in new UI work.
