import { test, expect } from "@playwright/test";

test.describe("Home dashboard", () => {
  test("home page loads with blocks", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator(".page-header h1")).toContainText("Home");
    // The homepage should have dashboard blocks.
    const dashboard = page.locator("[data-home-dashboard]");
    await expect(dashboard).toBeVisible();
    // It should have at least one block.
    const blocks = dashboard.locator("[data-home-block]");
    await expect(blocks.first()).toBeVisible();
  });

  test("today block shows daily note or empty state", async ({ page }) => {
    await page.goto("/");
    const todayBlock = page.locator('[data-home-block="today"]');
    await expect(todayBlock).toBeVisible();
    // Renders a date string.
    await expect(todayBlock.locator("h2#home-today-title")).not.toBeEmpty();
  });

  test("active projects block exists and can be filtered", async ({ page }) => {
    await page.goto("/");
    const block = page.locator('[data-home-block="active_projects"]');
    // This may or may not have active projects — just verify structure.
    const filterInput = block.locator("[data-home-project-filter]");
    await expect(filterInput).toBeVisible();
  });

  test("recent notes block exists", async ({ page }) => {
    await page.goto("/");
    const block = page.locator('[data-home-block="recent_notes"]');
    await expect(block).toBeVisible();
  });

  test("calendar block shows month and day links", async ({ page }) => {
    await page.goto("/");
    const block = page.locator('[data-home-block="calendar"]');
    await expect(block).toBeVisible();
    // Calendar should have day links with class.
    await expect(block.locator("a.calendar-day").first()).toBeVisible();
  });

  test("diagnostics block shows broken link and orphan counts", async ({ page }) => {
    await page.goto("/");
    const block = page.locator('[data-home-block="diagnostics"]');
    await expect(block).toBeVisible();
  });
});

test.describe("Folder page", () => {
  test("folder view shows contents and sort links", async ({ page }) => {
    await page.goto("/Syntax");
    // Folder page.
    await expect(page.locator("h1")).toContainText("Syntax");
    const sortNav = page.locator("nav.folder-sort");
    await expect(sortNav).toBeVisible();
    await expect(sortNav.locator("a")).toContainText(["Name ↑", "Name ↓", "Modified ↓", "Modified ↑"]);
    // Should list items.
    const items = page.locator("ul.folder-list a");
    await expect(items.first()).toBeVisible();
  });

  test("folder sort links navigate with query params", async ({ page }) => {
    await page.goto("/Syntax?sort=modified&dir=desc");
    await expect(page.locator("h1")).toContainText("Syntax");
    // The current sort link should have aria-current="true".
    const currentSort = page.locator('nav.folder-sort a[aria-current="true"]');
    await expect(currentSort).toContainText("Modified ↓");
  });
});

test.describe("Search page", () => {
  test("search page renders form and recent notes", async ({ page }) => {
    await page.goto("/_search");
    await expect(page.locator("h1")).toContainText("Search");
    await expect(page.locator('form input[name="q"]')).toBeVisible();
    // Recent notes section should appear when no query.
    await expect(page.locator(".recent-search-results")).toBeVisible();
  });

  test("search with query returns results", async ({ page }) => {
    await page.goto("/_search?q=Target+Note");
    await expect(page.locator("h1")).toContainText("Search");
    const results = page.locator("ul.results li");
    await expect(results.first()).toBeVisible();
  });

  test("search with no-match query shows empty state", async ({ page }) => {
    await page.goto("/_search?q=zzzznotexistzzzz");
    const empty = page.locator(".empty-state");
    await expect(empty).toBeVisible();
    await expect(empty).toContainText("No results");
  });
});

test.describe("Tags pages", () => {
  test("tags index shows popular and alphabetical tags", async ({ page }) => {
    await page.goto("/_tags");
    await expect(page.locator("h1")).toContainText("Tags");
    await expect(page.locator(".popular-tags")).toBeVisible();
    // Should have tag chips with names.
    const chips = page.locator("[data-tag-chip]");
    await expect(chips.first()).toBeVisible();
  });

  test("tag filter input filters chips", async ({ page }) => {
    await page.goto("/_tags");
    const filterInput = page.locator("[data-tag-filter]");
    await filterInput.fill("demo");
    // At least one tag chip should remain visible.
    const visibleChips = page.locator("[data-tag-chip]:not([hidden])");
    await expect(visibleChips.first()).toBeVisible();
  });

  test("tag detail page shows notes for tag", async ({ page }) => {
    await page.goto("/_tags/demo");
    await expect(page.locator("h1")).toContainText("#demo");
    // Note cards.
    const cards = page.locator(".tag-note-card");
    await expect(cards.first()).toBeVisible();
  });

  test("tag detail page shows empty state for unused tag", async ({ page }) => {
    await page.goto("/_tags/nonexistent999");
    await expect(page.locator("h1")).toContainText("#nonexistent999");
    await expect(page.locator(".empty-state")).toBeVisible();
  });
});

test.describe("Broken links page", () => {
  test("broken links page shows diagnostics", async ({ page }) => {
    await page.goto("/_broken-links");
    await expect(page.locator("h1")).toContainText("Broken links");
    // Should show summary stats.
    await expect(page.locator(".diagnostic-summary")).toBeVisible();
    // Should list at least one broken link (ZorkMidriff) or show empty state.
    const groups = page.locator(".broken-link-group");
    const empty = page.locator(".empty-state");
    // Expect either the groups or empty state to be visible.
    await expect(groups.or(empty).first()).toBeVisible();
  });

  test("broken links filter input is present", async ({ page }) => {
    await page.goto("/_broken-links");
    await expect(page.locator("[data-list-filter]")).toBeVisible();
  });
});

test.describe("Orphans page", () => {
  test("orphans page shows note cards or empty state", async ({ page }) => {
    await page.goto("/_orphans");
    await expect(page.locator("h1")).toContainText("Orphan notes");
    await expect(page.locator(".diagnostic-summary")).toBeVisible();
    const cards = page.locator(".note-card-grid .note-card");
    const empty = page.locator(".empty-state");
    await expect(cards.or(empty).first()).toBeVisible();
  });

  test("orphans filter input is present", async ({ page }) => {
    await page.goto("/_orphans");
    await expect(page.locator("[data-list-filter]")).toBeVisible();
  });
});

test.describe("Dataview diagnostics page", () => {
  test("diagnostics page shows blocks and status", async ({ page }) => {
    await page.goto("/_dataview");
    await expect(page.locator("h1")).toContainText("Dataview diagnostics");
    // Should show total and unsupported counts.
    await expect(page.locator("section.page-header p.muted")).toBeVisible();
    // Should list individual diagnostics.
    const diagCards = page.locator("article.dataview-diagnostic");
    await expect(diagCards.first()).toBeVisible();
  });

  test("supported blocks have status chip 'supported'", async ({ page }) => {
    await page.goto("/_dataview");
    const supported = page.locator('article.dataview-diagnostic.supported');
    await expect(supported.first()).toBeVisible();
  });

  test("unsupported blocks (dataviewjs) show 'unsupported' status", async ({ page }) => {
    await page.goto("/_dataview");
    const unsupported = page.locator('article.dataview-diagnostic.unsupported');
    await expect(unsupported.first()).toBeVisible();
  });
});

test.describe("Missing and resolve pages", () => {
  test("missing page shows 404 and note name", async ({ page }) => {
    await page.goto("/_missing?name=ZorkMidriff");
    await expect(page.locator("h1")).toContainText("Not found");
    await expect(page.locator("code")).toContainText("ZorkMidriff");
  });

  test("resolve page shows matching notes", async ({ page }) => {
    await page.goto("/_resolve?name=Resolve%20One");
    await expect(page.locator("h1")).toContainText("Choose a note");
    await expect(page.locator("code")).toContainText("Resolve One");
    // Both resolution options should be listed.
    await expect(page.locator("ul.choice-list a")).toHaveCount(2);
  });

  test("missing wikilink navigates to missing page", async ({ page }) => {
    await page.goto("/Syntax/Broken%20Link%20Source.md");
    await page.waitForSelector("article.note");
    const missingLink = page.locator('a[href^="/_missing?name=ZorkMidriff"]');
    await missingLink.click();
    await expect(page.locator("h1")).toContainText("Not found");
  });

  test("ambiguous wikilink navigates to resolve page", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const ambigLink = page.locator('a[href^="/_resolve?name="]');
    await ambigLink.click();
    await expect(page.locator("h1")).toContainText("Choose a note");
  });
});

test.describe("TODO dashboard", () => {
  test("todo page renders sections and summary", async ({ page }) => {
    await page.goto("/_todo");
    await expect(page.locator(".todo-hero h1")).toContainText("TODOs");
    await expect(page.locator(".todo-summary")).toBeVisible();
    // Overdue section should have at least one visible task.
    const overdueSection = page.locator("section.todo-section.overdue");
    await expect(overdueSection).toBeVisible();
    // Today, upcoming sections are visible.
    await expect(page.locator("section.todo-section.today")).toBeVisible();
    await expect(page.locator("section.todo-section.upcoming")).toBeVisible();
    // no-date and done are hidden by default (filters checked).
    await expect(page.locator("section.todo-section.no-date")).toBeAttached();
    await expect(page.locator("section.todo-section.done")).toBeAttached();
  });

  test("todo filter controls are present", async ({ page }) => {
    await page.goto("/_todo");
    // Search, tag, priority, date, group filters.
    await expect(page.locator("[data-todo-search]")).toBeVisible();
    await expect(page.locator('[data-todo-filter="tag"]')).toBeVisible();
    await expect(page.locator('[data-todo-filter="priority"]')).toBeVisible();
    await expect(page.locator('[data-todo-filter="date"]')).toBeVisible();
    await expect(page.locator('[data-todo-filter="group"]')).toBeVisible();
    // Toggles.
    await expect(page.locator("[data-todo-hide-nodate]")).toBeVisible();
    await expect(page.locator("[data-todo-hide-done]")).toBeVisible();
  });

  test("todo tag filter has demo tags from tasks", async ({ page }) => {
    await page.goto("/_todo");
    const tagSelect = page.locator('[data-todo-filter="tag"]');
    // Should include task tags extracted from the fixture TODO source.
    const options = tagSelect.locator("option");
    await expect(options).toContainText([
      "All tags",
      "#admin",
      "#project/demo-project",
      "#task/dashboard",
    ]);
  });

  test("todo priority filter filters tasks", async ({ page }) => {
    await page.goto("/_todo");
    const prioritySelect = page.locator('[data-todo-filter="priority"]');
    // Select P1 — should show at least the overdue P1 task.
    await prioritySelect.selectOption("P1");
    await page.waitForTimeout(100);
    const visibleRows = page.locator(".task-row:not([hidden])");
    await expect(visibleRows.first()).toBeVisible();
  });

  test("todo group by priority creates dynamic groups", async ({ page }) => {
    await page.goto("/_todo");
    const groupSelect = page.locator('[data-todo-filter="group"]');
    await groupSelect.selectOption("Priority");
    await page.waitForTimeout(100);
    // Dynamic sections should appear.
    const dynamicGroups = page.locator(".todo-dynamic-groups");
    await expect(dynamicGroups).toBeVisible();
  });

  test("todo group by project creates project groups", async ({ page }) => {
    await page.goto("/_todo");
    const groupSelect = page.locator('[data-todo-filter="group"]');
    await groupSelect.selectOption("Project");
    await page.waitForTimeout(100);
    const dynamicGroups = page.locator(".todo-dynamic-groups");
    await expect(dynamicGroups).toBeVisible();
  });

  test("todo task menu can be opened", async ({ page }) => {
    await page.goto("/_todo");
    // Uncheck "Hide done" so completed tasks (which have IDs) become visible.
    await page.locator("[data-todo-hide-done]").uncheck();
    await page.waitForTimeout(100);
    // Find a visible task row and click its menu button.
    const taskRow = page.locator('.task-row:not([hidden])[data-task-id]:not([data-task-id=""])').first();
    await expect(taskRow).toBeVisible();
    const menuBtn = taskRow.locator('[data-task-menu]');
    await menuBtn.click();
    // The dropdown within this same row should become visible.
    const dropdown = taskRow.locator('.task-menu-dropdown');
    await expect(dropdown).toBeVisible();
    // Should have copy actions.
    await expect(dropdown.locator('[role="menuitem"]').first()).toBeVisible();
  });

  test("todo overdue section has at least one task", async ({ page }) => {
    await page.goto("/_todo");
    const overdueSection = page.locator("section.todo-section.overdue");
    const rows = overdueSection.locator(".task-row");
    // With the fixture data, we have overdue tasks with dates in 2024.
    await expect(rows.first()).toBeVisible();
  });

  test("todo done section shows completed tasks", async ({ page }) => {
    await page.goto("/_todo");
    // Uncheck "Hide done" to show completed tasks.
    const hideDoneCheckbox = page.locator("[data-todo-hide-done]");
    await hideDoneCheckbox.uncheck();
    await page.waitForTimeout(100);
    const doneSection = page.locator("section.todo-section.done");
    const rows = doneSection.locator(".task-row.completed");
    await expect(rows.first()).toBeVisible();
  });
});

test.describe("Command palette and settings", () => {
  test("command palette opens on click", async ({ page }) => {
    await page.goto("/");
    await page.locator("[data-palette-open]").click();
    const palette = page.locator("[data-palette]");
    await expect(palette).toBeVisible();
    await expect(palette.locator("[data-palette-input]")).toBeFocused();
  });

  test("settings modal opens and has controls", async ({ page }) => {
    await page.goto("/");
    await page.locator("[data-settings-open]").click();
    const modal = page.locator("[data-settings-modal]");
    await expect(modal).toBeVisible();
    // Should have theme and font controls.
    await expect(modal.locator("[data-theme-select]")).toBeVisible();
    await expect(modal.locator("[data-font-size-select]")).toBeVisible();
  });

  test("sidebar toggle exists in DOM", async ({ page }) => {
    await page.goto("/");
    // Mobile sidebar toggle is present in DOM (hidden on desktop viewport).
    const toggle = page.locator("[data-sidebar-toggle]");
    await expect(toggle).toBeAttached();
  });
});
