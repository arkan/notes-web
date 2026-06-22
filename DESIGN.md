---
name: Notes Web
description: A calm, keyboard-first web workbench for reading, retrieving, and maintaining a Markdown vault.
colors:
  graphite-canvas: "#0f1117"
  graphite-shell: "#151821"
  graphite-panel: "#1b1f2a"
  graphite-surface: "#f4f6fb"
  graphite-card: "#fbfcff"
  graphite-ink: "#111827"
  graphite-muted: "#667085"
  graphite-line: "#d9dee8"
  graphite-line-dark: "#2a3040"
  electric-blue: "#3b82f6"
  electric-blue-ink: "#1d4ed8"
  electric-blue-soft: "#dbeafe"
  danger: "#dc2626"
  warning: "#d97706"
  success: "#16a34a"
  code-inline: "#eef2f7"
  code-block: "#111827"
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
    backgroundColor: "{colors.graphite-card}"
    textColor: "{colors.graphite-ink}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
  button-primary:
    backgroundColor: "{colors.electric-blue}"
    textColor: "{colors.graphite-card}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
  chip:
    backgroundColor: "{colors.electric-blue-soft}"
    textColor: "{colors.electric-blue-ink}"
    rounded: "{rounded.pill}"
    padding: "4px 10px"
  card:
    backgroundColor: "{colors.graphite-card}"
    textColor: "{colors.graphite-ink}"
    rounded: "{rounded.lg}"
    padding: "20px"
  search-input:
    backgroundColor: "{colors.graphite-card}"
    textColor: "{colors.graphite-ink}"
    rounded: "10px"
    padding: "10px 12px"
  palette-panel:
    backgroundColor: "{colors.graphite-card}"
    textColor: "{colors.graphite-ink}"
    rounded: "{rounded.xl}"
    padding: "0"
---

# Design System: Notes Web

## 1. Overview

**Creative North Star: "Modern Workbench"**

Notes Web should feel like a precise local workbench for a trusted Markdown vault: calm enough for long reading, structured enough for daily review, and fast enough that keyboard retrieval feels immediate. The interface is a real application shell, not a decorative notebook skin.

The Modern Workbench system uses neutral graphite structure, fine borders, compact premium density, and a single electric-blue thread for active states, commands, focus, and selected navigation. Density is allowed because the product is a real tool, but every dense surface must remain quiet, legible, and purposeful.

The visual system rejects cold documentation wikis, generic SaaS dashboard noise, glassy decoration, gradient-heavy marketing UI, Obsidian mimicry, and bland neutrality. It should feel like a premium local workbench inspired by Linear-level precision while serving Notes Web's retrieval, reading, task review, and vault maintenance goals.

**Key Characteristics:**
- Graphite-led product UI with strong reading comfort.
- Restrained accent use: electric blue marks action, focus, selection, and command flow.
- Flat premium layering before shadow.
- Keyboard-first affordances with visible focus and predictable states.
- Personal craft made reusable through explicit tokens and rules.

## 2. Colors

The palette is a restrained graphite system with one electric-blue accent that acts as a thread through navigation, commands, chips, focus, and active states.

### Primary
- **Electric Blue** (`--accent`): The only primary accent. Use it for primary actions, active navigation, selected filters, command affordances, and focus emphasis.
- **Electric Blue Ink** (`--accent-ink`): Text and link accent on light surfaces. Use when the accent needs readable ink rather than filled color.
- **Electric Blue Soft** (`--soft`): The quiet selected and hover field. Use for active branches, command palette rows, tags, and calm emphasis.

### Secondary
- **Danger** (`--danger`): Errors, overdue tasks, unresolved links, and destructive states. It should feel decisive, not theatrical.
- **Warning** (`--warning`): Warnings and due-soon attention. Use sparingly so it reads as signal, not decoration.
- **Success** (`--success`): Completion and resolved states. It should feel durable and quiet.

### Neutral
- **Graphite Canvas** (`--bg`): The ambient app background. It creates a focused workbench frame without pure black.
- **Graphite Surface** (`--surface`): The main content and card surface. In light mode it stays crisp; in dark mode it stays matte.
- **Graphite Ink** (`--ink`): Main text. It is high contrast, never pure black or pure white.
- **Graphite Muted** (`--muted`): Secondary metadata, paths, labels, timestamps, and helper text.
- **Graphite Line** (`--line`): Borders and dividers. It should separate with fine precision, not box the page aggressively.
- **Code Inline / Code Block** (`--code`, `--pre`): Matte technical surfaces integrated with prose.

### Named Rules

**The Blue Thread Rule.** Electric blue is for orientation and action only: current location, command flow, focus, selected filters, and primary actions. If blue appears as ornament, remove it.

**The Graphite Neutral Rule.** Use graphite neutrals for every background and text layer. Pure black and pure white are forbidden in new UI work.

**The Semantic Restraint Rule.** Danger, Warning, and Success are state colors, not category colors. Do not decorate dashboards with them.

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

Depth comes from flat premium layering: canvas, shell, surfaces, hairline borders, and raised overlays. Shadows are rare, short, and matte. They belong to temporary layers above the document: command palette, dialogs, menus, popovers, and hover feedback on actionable controls.

The existing system includes modal and card shadows, but the design direction is still flat by default. Blur is functional only. It may isolate a command mode, never create decorative glass.

### Shadow Vocabulary
- **Surface Hairline** (`0 1px 0 rgba(17, 24, 39, 0.08)`): A nearly flat separation for buttons and quiet cards.
- **Soft Card** (`0 2px 10px rgba(17, 24, 39, 0.08)`): Low elevation for cards and task groups.
- **Command Lift** (`0 24px 80px rgba(17, 19, 26, 0.34)`): Temporary elevation for command palette and settings dialogs.

### Named Rules

**The Flat Until Invoked Rule.** Resting surfaces are flat. Shadows appear when an element is actionable, hovered, opened, or temporarily above the vault.

**The No Glass Rule.** Backdrop blur may support a modal mode, but glassmorphism is forbidden as a decorative default.

## 5. Components

Every component should behave like a calm tool: obvious at keyboard speed, useful in one glance, simple in shape, and just distinctive enough to belong to Notes Web.

### Buttons
- **Shape:** Gently rounded rectangle (12px radius).
- **Default:** Graphite Card background, Graphite Line border, Graphite Ink text, compact padding (8px 12px).
- **Primary:** Electric Blue fill with light text. Reserve for the main action in a local context.
- **Ghost:** Transparent background with the same shape and focus vocabulary as default buttons.
- **Hover / Focus:** Move by transform only, never layout. Focus must be visible with an electric-blue ring or equivalent high-contrast treatment.

### Chips
- **Style:** Electric Blue Soft background, Electric Blue Ink text, pill radius, compact padding (4px 10px).
- **State:** Selected chips may use Electric Blue fill. Unselected chips should stay quiet.
- **Use:** Tags, filters, task metadata, folder sort controls, and compact status indicators.

### Cards / Containers
- **Corner Style:** Soft but not bubbly (18px radius).
- **Background:** Graphite Card or Graphite Surface for normal surfaces, never raw white in new work.
- **Shadow Strategy:** Flat by default, Soft Card only when a module needs separation from the paper.
- **Border:** Graphite Line hairline. Avoid stacking nested cards.
- **Internal Padding:** 16px to 20px for cards, 14px for compact note and result rows, 18px for dense task sections.

### Inputs / Fields
- **Style:** Graphite Card background, Graphite Line border, 10px to 12px radius, system sans text.
- **Focus:** Electric Blue border with a soft focus ring. Do not rely on color alone if the field has validation state.
- **Search:** Search fields can grow larger (18px to 20px) because retrieval is a primary task.

### Navigation
- **Sidebar:** Persistent desktop rail with graphite shell structure, fine divider, compact tree spacing, and a calm active branch using Electric Blue Soft.
- **Tree:** Folders and notes should scan quickly. Active branch treatment uses background plus weight, not color alone.
- **Mobile:** Sidebar becomes a drawer overlay. Keep the same hierarchy and focus order.
- **Command Palette:** Treat as the fastest navigation surface. The panel is raised, centered, large enough for typing, and selected rows use Electric Blue Soft.

### Workbench Shell
- **Desktop:** Tri-pane by default: left app navigation/favorites/tree, center reading or app surface, right contextual pane. The center pane remains the source of truth; the right pane supports context and secondary actions.
- **Right Pane:** Collapsible, persistent by local preference, and hidden from tab order when collapsed. Modules should be small, labeled, and contextual.
- **Mobile:** Equal-quality app model, not a squeezed desktop. Sidebar is a drawer, page actions use compact gear affordances, and Settings remains reachable even in reading focus.
- **Settings:** A modal for browser-local UI preferences: theme, font size, density, reading focus, and palette recents.
- **Local Preferences:** Browser-local only. Do not require server persistence for density, pane state, reading focus, or palette recents.

### Task Rows
- **Style:** Dense grid with checkbox, priority badge, title, metadata, and menu access.
- **State:** Completed tasks reduce opacity and strike the title. Priority badges may use semantic fills, but only when the priority is meaningful.
- **Interaction:** Row menus stay quiet until hover or focus. Keyboard access must match mouse affordances.

### App Surfaces
- **Home:** Daily cockpit in a calm single-column center flow: Quick capture when editing is enabled, daily note hero, due-now signal, configured blocks lower in the flow.
- **Inbox:** Processing surface for root `Inbox/` captures. Cards are action-oriented but quiet; disabled action reasons must be explicit.
- **Projects:** Read-only project overview cards sourced from indexed project notes; filtering is local display help, not a new server model.
- **Calendar:** Daily-notes calendar only. Days are clear, touch-friendly, and path-policy safe.
- **Maintenance:** Grouped diagnostics with counts and links, not an alarm wall or a dump of paths.
- **Trash:** Restore / Restore as only. No purge affordance in v1.

### Command Palette Recents
- **Storage:** Browser-local history only.
- **Trust Model:** Render/navigate only items still present in the current server palette payload. Treat `localStorage` as an untrusted cache.
- **Mobile:** Limit recents visually so search remains primary.

### Density Modes
- **Compact:** Default premium density for workbench scanning.
- **Comfortable:** Must preserve or increase spacing on key cards and reading surfaces. It must never shrink a surface that is already more spacious than the base token.

### Code Blocks
- **Style:** Matte graphite/code backgrounds with 12px radius and a copy button in the top right.
- **Inline Code:** Small graphite code chip with 5px radius.
- **Use:** Code should feel integrated with prose, not like a separate developer console.

### Callouts
- **Direction:** Use tinted panels, full borders, icons, or title weight instead of side-stripe accents.
- **Reason:** Side stripes create a generic documentation look and conflict with the calmer workbench system.

## 6. Do's and Don'ts

### Do:
- **Do** keep Notes Web product-first: retrieval, reading, tasks, and diagnostics beat decorative composition.
- **Do** use Electric Blue only for orientation, action, focus, and selected states.
- **Do** preserve Calm Density: information-rich modules are allowed when spacing, hierarchy, and labels make them scannable.
- **Do** design keyboard-first states for buttons, tree items, command palette rows, task menus, and filters.
- **Do** make diagnostics feel like maintenance, not alarm: broken links and orphans should be clear without turning the page into an error wall.
- **Do** keep local-first product choices explicit so open source users can adapt them without erasing the point of view.

### Don't:
- **Don't** make Notes Web look like a cold documentation wiki that makes personal notes feel impersonal.
- **Don't** add decorative motion, unnecessary blur, or framework-like visual bulk.
- **Don't** use generic SaaS dashboard patterns: repeated metric cards, hero-stat layouts, gradient accents, or ornamental panels.
- **Don't** copy Obsidian's desktop app. Serve the web reading and retrieval context instead.
- **Don't** neutralize the product until it loses its precise local-workbench point of view.
- **Don't** add colored side-stripe borders to new callouts, cards, list items, or alerts. Use full borders, tinted panels, icons, or title treatment.
- **Don't** use gradient text, glassmorphism as decoration, pure black, or pure white in new UI work.
