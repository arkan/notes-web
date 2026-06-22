import { test, expect } from "@playwright/test";

test.describe("Modern workbench shell", () => {
  test("desktop note renders left navigation, center content, and read-only context pane", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.goto("/Syntax/All%20Syntaxes.md");

    await expect(page.locator("[data-workbench-shell]")).toBeVisible();
    await expect(page.locator('aside[aria-label="Vault navigation"]')).toBeVisible();
    await expect(page.locator("main#main-content")).toBeVisible();
    await expect(page.locator("[data-context-pane]")).toBeVisible();
    await expect(page.locator("[data-context-pane]")).toContainText("Current page");
    await expect(page.locator("[data-context-pane]")).toContainText("Read-only context only");
    await expect(page.locator("article.note")).toBeVisible();
    await expect(page.locator("article.note [data-note-actions-toggle]")).toBeVisible();
    await expect(page.locator("[data-context-pane] [data-note-actions-toggle]")).toHaveCount(0);
  });

  test("app navigation exposes only implemented routes in order", async ({ page }) => {
    await page.goto("/");
    const appNav = page.locator('nav[aria-label="App navigation"]');
    const hrefs = await appNav.locator("a").evaluateAll((links) => links.map((link) => link.getAttribute("href")));
    expect(hrefs).toEqual(["/", "/_todo", "/_projects", "/_calendar", "/_search", "/_tags", "/_maintenance"]);
    await expect(appNav.locator('a[href="/"]')).toHaveAttribute("aria-current", "page");
    await expect(appNav.locator('a[href="/_todo"]')).toContainText("Tasks");
    await expect(appNav.locator('a[href="/_projects"]')).toContainText("Projects");
    await expect(appNav.locator('a[href="/_trash"]')).toHaveCount(0);
    await expect(appNav.locator('a[href="/_inbox"], a[href="/_settings"]')).toHaveCount(0);

    await page.goto("/_search");
    await expect(page.locator('nav[aria-label="App navigation"] a[href="/_search"]')).toHaveAttribute("aria-current", "page");
  });

  test("right context pane toggle hides, persists, and restores focus", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.goto("/Syntax/All%20Syntaxes.md");
    const pane = page.locator("[data-context-pane]");
    const primaryToggle = page.locator("[data-right-pane-toggle-primary]");

    await expect(pane).toBeVisible();
    await expect(primaryToggle).toHaveAttribute("aria-controls", "workbench-context-pane");
    await expect(primaryToggle).toHaveAttribute("aria-expanded", "true");

    await pane.getByLabel("Hide context pane").focus();
    await page.keyboard.press("Enter");
    await expect(pane).toBeHidden();
    await expect(pane).toHaveAttribute("aria-hidden", "true");
    await expect(pane).toHaveJSProperty("inert", true);
    await expect(primaryToggle).toHaveAttribute("aria-expanded", "false");
    await expect(primaryToggle).toBeFocused();

    await page.reload();
    await expect(page.locator("[data-context-pane]")).toBeHidden();
    await expect(page.locator("[data-right-pane-toggle-primary]")).toHaveAttribute("aria-expanded", "false");

    await page.locator("[data-right-pane-toggle-primary]").click();
    await expect(page.locator("[data-context-pane]")).toBeVisible();
    await expect(page.locator("[data-right-pane-toggle-primary]")).toHaveAttribute("aria-expanded", "true");
  });

  test("mobile note keeps content and primary controls accessible", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/Syntax/All%20Syntaxes.md");

    await expect(page.locator("main#main-content")).toBeVisible();
    await expect(page.locator("article.note")).toBeVisible();
    await expect(page.locator("[data-context-pane]")).toBeHidden();
    await expect(page.getByLabel("Open sidebar")).toBeVisible();
    await expect(page.getByLabel("Open command palette")).toBeVisible();
  });

  test("note page uses a calm reading stack and keeps actions out of context", async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto("/Syntax/All%20Syntaxes.md");

    const stack = page.locator(".note-page");
    await expect(stack).toBeVisible();
    await expect(page.locator(".note-page > article.note")).toBeVisible();
    await expect(page.locator(".note-page > .crumb")).toBeVisible();
    await expect(page.locator("article.note .note-actions [data-note-actions-toggle]")).toBeVisible();
    await expect(page.locator("[data-context-pane] [data-note-actions-toggle]")).toHaveCount(0);
    await expect(page.locator("[data-context-pane] [data-edit-open], [data-context-pane] [data-edit-trash]")).toHaveCount(0);

    const metrics = await page.evaluate(() => {
      const stackEl = document.querySelector<HTMLElement>(".note-page");
      const mainEl = document.querySelector<HTMLElement>("main#main-content");
      const contentEl = document.querySelector<HTMLElement>(".note-page .content");
      if (!stackEl || !mainEl || !contentEl) return null;
      const stackRect = stackEl.getBoundingClientRect();
      const mainRect = mainEl.getBoundingClientRect();
      const contentStyle = getComputedStyle(contentEl);
      return {
        stackWidth: stackRect.width,
        stackCenter: stackRect.left + stackRect.width / 2,
        mainCenter: mainRect.left + mainRect.width / 2,
        proseFont: Number.parseFloat(contentStyle.fontSize),
        proseLineHeight: Number.parseFloat(contentStyle.lineHeight),
      };
    });
    if (!metrics) throw new Error("note reading metrics were not available");
    expect(metrics.stackWidth).toBeLessThanOrEqual(850);
    expect(Math.abs(metrics.stackCenter - metrics.mainCenter)).toBeLessThan(12);
    expect(metrics.proseFont).toBeGreaterThanOrEqual(16);
    expect(metrics.proseLineHeight).toBeGreaterThanOrEqual(27);
  });

  test("mobile note reading stack avoids page overflow and keeps gear actions reachable", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/Syntax/All%20Syntaxes.md");

    await expect(page.locator(".note-page")).toBeVisible();
    const actions = page.locator("article.note .note-actions [data-note-actions-toggle]");
    await expect(actions).toBeVisible();
    const metrics = await page.evaluate(() => {
      const button = document.querySelector<HTMLElement>("article.note .note-actions [data-note-actions-toggle]");
      return {
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
        actionHeight: button?.getBoundingClientRect().height ?? 0,
      };
    });
    expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
    expect(metrics.actionHeight).toBeGreaterThanOrEqual(40);
  });

  test("mobile closed sidebar is hidden from keyboard and assistive tech", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/Syntax/All%20Syntaxes.md");

    const sidebar = page.locator('aside[aria-label="Vault navigation"]');
    await expect(sidebar).toHaveAttribute("aria-hidden", "true");
    await expect(sidebar).toHaveJSProperty("inert", true);

    const toggle = page.getByLabel("Open sidebar");
    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-expanded", "true");
    await expect(sidebar).not.toHaveAttribute("aria-hidden", "true");
    await expect(sidebar).toHaveJSProperty("inert", false);

    await page.keyboard.press("Escape");
    await expect(toggle).toHaveAttribute("aria-expanded", "false");
    await expect(sidebar).toHaveAttribute("aria-hidden", "true");
    await expect(sidebar).toHaveJSProperty("inert", true);
  });

  test("tablet-width drawer sidebar is hidden from keyboard and assistive tech", async ({ page }) => {
    await page.setViewportSize({ width: 1000, height: 760 });
    await page.goto("/Syntax/All%20Syntaxes.md");

    const sidebar = page.locator('aside[aria-label="Vault navigation"]');
    await expect(sidebar).toHaveAttribute("aria-hidden", "true");
    await expect(sidebar).toHaveJSProperty("inert", true);
  });

  test("future navigation routes stay hidden until implemented", async ({ page }) => {
    await page.goto("/");
    const appNav = page.locator('nav[aria-label="App navigation"]');
    await expect(appNav.locator('a[href="/_inbox"]')).toHaveCount(0);
    await expect(appNav.locator('a[href="/_projects"]')).toHaveCount(1);
    await expect(appNav.locator('a[href="/_calendar"]')).toHaveCount(1);
    await expect(appNav.locator('a[href="/_maintenance"]')).toHaveCount(1);
    await expect(appNav.locator('a[href="/_settings"]')).toHaveCount(0);
    await expect(appNav.locator('a[href="/_trash"]')).toHaveCount(0);
  });

  test("mobile command palette traps keyboard focus and has no preview pane", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");
    const opener = page.getByLabel("Open command palette");
    await opener.click();
    const palette = page.locator("[data-palette]");
    await expect(palette).toBeVisible();
    await expect(palette.locator("[data-palette-input]")).toBeFocused();
    await expect(palette.locator(".palette-preview")).toHaveCount(0);

    await palette.locator("[data-palette-input]").fill("Target");
    await expect(palette.locator("[data-palette-index]").first()).toBeVisible();
    await page.keyboard.press("Tab");
    const focusInsidePalette = await page.evaluate(() => Boolean(document.querySelector("[data-palette]")?.contains(document.activeElement)));
    expect(focusInsidePalette).toBe(true);
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("Escape");
    await expect(palette).toBeHidden();
    await expect(opener).toBeFocused();
  });
});

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

  test("home page reads as a daily cockpit without Quick capture", async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto("/");

    await expect(page.locator(".home-cockpit-header")).toContainText("Daily cockpit");
    await expect(page.locator(".home-header-summary")).toContainText("daily note");
    await expect(page.locator('[data-home-block="today"]')).toBeVisible();
    await expect(page.locator('.home-due-now-summary')).toContainText('today');
    await expect(page.locator('[data-home-block="todos"]')).toBeVisible();
    await expect(page.getByText("Quick capture")).toHaveCount(0);
    await expect(page.locator("[data-quick-capture], [data-capture]")).toHaveCount(0);

    const metrics = await page.evaluate(() => {
      const dashboard = document.querySelector<HTMLElement>("[data-home-dashboard]");
      const today = document.querySelector<HTMLElement>('[data-home-block="today"]');
      const todos = document.querySelector<HTMLElement>('[data-home-block="todos"]');
      const title = document.querySelector<HTMLElement>('[data-home-block="today"] h2');
      if (!dashboard || !today || !todos || !title) return null;
      const dashboardRect = dashboard.getBoundingClientRect();
      const todayRect = today.getBoundingClientRect();
      const todosRect = todos.getBoundingClientRect();
      const titleStyle = getComputedStyle(title);
      return {
        dashboardWidth: dashboardRect.width,
        todayTop: todayRect.top,
        todosTop: todosRect.top,
        todayRadius: Number.parseFloat(getComputedStyle(today).borderTopLeftRadius),
        heroTitleSize: Number.parseFloat(titleStyle.fontSize),
      };
    });
    if (!metrics) throw new Error("home cockpit metrics were not available");
    expect(metrics.dashboardWidth).toBeLessThanOrEqual(1042);
    expect(metrics.todayTop).toBeLessThan(metrics.todosTop);
    expect(metrics.todosTop).toBeLessThanOrEqual(900);
    expect(metrics.todayRadius).toBeGreaterThanOrEqual(16);
    expect(metrics.heroTitleSize).toBeGreaterThanOrEqual(32);
  });

  test("editing-disabled build hides Inbox route and Quick capture", async ({ page }) => {
    await page.goto("/");
    const appNav = page.locator('nav[aria-label="App navigation"]');
    await expect(appNav.locator('a[href="/_inbox"]')).toHaveCount(0);
    await expect(page.locator("[data-quick-capture]")).toHaveCount(0);

    const response = await page.goto("/_inbox");
    expect(response?.status()).toBe(404);
  });

  test("responsive home preserves configured block order and avoids overflow", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");

    const metrics = await page.evaluate(() => {
      const blocks = Array.from(document.querySelectorAll<HTMLElement>("[data-home-block]"));
      return {
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
        visualOrder: blocks
          .map((block) => ({ id: block.dataset.homeBlock || "", order: Number(block.dataset.homeOrder), top: block.getBoundingClientRect().top }))
          .sort((a, b) => a.top - b.top)
          .map((block) => block.id),
      };
    });
    expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
    expect(metrics.visualOrder.slice(0, 3)).toEqual(["today", "calendar", "todos"]);
    await expect(page.locator('[data-home-block="today"] .home-block-heading .btn')).toBeVisible();
    await expect(page.locator('.home-due-now-summary')).toBeInViewport();
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

test.describe("Projects page", () => {
  test("projects page renders overview and active project cards", async ({ page }) => {
    await page.goto("/_projects");

    await expect(page.locator(".projects-page")).toBeVisible();
    await expect(page.locator(".projects-header h1")).toContainText("Active projects");
    await expect(page.locator(".projects-overview")).toContainText("Active projects");
    await expect(page.locator("[data-project-filter]")).toBeVisible();
    await expect(page.locator("[data-project-card]").first()).toBeVisible();
    await expect(page.locator("[data-project-card]", { hasText: "Alpha" })).toBeVisible();
    await expect(page.locator("[data-project-card]", { hasText: "Gamma" })).toBeVisible();
    await expect(page.locator("[data-project-card]", { hasText: "Beta" })).toHaveCount(0);
    await expect(page.locator(".project-latest").first()).toBeVisible();
    await expect(page.locator('nav[aria-label="App navigation"] a[href="/_projects"]')).toHaveAttribute("aria-current", "page");
  });

  test("project filter narrows cards and shows filtered empty state", async ({ page }) => {
    await page.goto("/_projects");

    await page.locator("[data-project-filter]").fill("gamma");
    const visibleCards = page.locator("[data-project-card]:not([hidden])");
    await expect(visibleCards).toHaveCount(1);
    await expect(visibleCards.first()).toContainText("Gamma");
    await expect(page.locator("[data-project-filter-empty]")).toBeHidden();

    await page.locator("[data-project-filter]").fill("no active project with this name");
    await expect(page.locator("[data-project-card]:not([hidden])")).toHaveCount(0);
    await expect(page.locator("[data-project-filter-empty]")).toBeVisible();
  });

  test("mobile projects page avoids horizontal overflow", async ({ page }) => {
    for (const width of [390, 320]) {
      await page.setViewportSize({ width, height: 844 });
      await page.goto("/_projects");
      await expect(page.locator("[data-project-card]").first()).toBeVisible();
      const metrics = await page.evaluate(() => ({
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
      }));
      expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
    }
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
    await expect(page.locator('.search-page-form input[name="q"]')).toBeVisible();
    await expect(page.locator(".search-help")).toBeVisible();
    // Recent notes section should appear when no query.
    await expect(page.locator(".recent-search-results")).toBeVisible();
  });

  test("search with query returns results", async ({ page }) => {
    await page.goto("/_search?q=Target+Note");
    await expect(page.locator("h1")).toContainText("Search");
    const results = page.locator(".rich-results .search-result-card");
    await expect(results.first()).toBeVisible();
  });

  test("search with no-match query shows empty state", async ({ page }) => {
    await page.goto("/_search?q=zzzznotexistzzzz");
    const empty = page.locator(".empty-state");
    await expect(empty).toBeVisible();
    await expect(empty).toContainText("No results");
  });
});

test.describe("Maintenance page", () => {
  test("maintenance page shows grouped diagnostic links", async ({ page }) => {
    await page.goto("/_maintenance");
    const main = page.locator("#main-content");
    await expect(page.locator("h1")).toHaveText("Maintenance");
    await expect(main.locator(".maintenance-grid")).toBeVisible();
    await expect(main.getByRole("link", { name: "Broken links", exact: true })).toBeVisible();
    await expect(main.getByRole("link", { name: "Orphan notes", exact: true })).toBeVisible();
    await expect(main.getByRole("link", { name: "Dataview diagnostics", exact: true }).first()).toBeVisible();
  });

  test("maintenance page is mobile-readable", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/_maintenance");
    await expect(page.locator(".maintenance-card").first()).toBeVisible();
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);
    expect(overflow).toBe(false);
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

test.describe("Calendar page", () => {
  test("calendar page renders the premium daily-note layout", async ({ page }) => {
    await page.goto("/_calendar?date=2026-06-18");
    await expect(page.locator(".calendar-page")).toBeVisible();
    await expect(page.locator(".calendar-workbench")).toBeVisible();
    await expect(page.locator(".calendar-month-grid")).toBeVisible();
    await expect(page.locator(".calendar-selected-panel")).toBeVisible();
    await expect(page.locator(".calendar-selected-panel")).toContainText("Daily Note June 18");
    await expect(page.locator('.calendar-day.has-note[aria-selected="true"]')).toBeVisible();
  });

  test("calendar mobile layout avoids horizontal overflow", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/_calendar?date=2026-06-18");
    await expect(page.locator(".calendar-month-grid")).toBeVisible();
    await expect(page.locator(".calendar-selected-panel")).toBeVisible();
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);
    expect(overflow).toBe(false);
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

  test("command palette does not show Trash action when editing is disabled", async ({ page }) => {
    await page.goto("/");
    await page.locator("[data-palette-open]").click();
    await page.locator("[data-palette-input]").fill("Open Trash");
    await expect(page.locator("[data-palette-index]", { hasText: "Open Trash" })).toHaveCount(0);
  });

  test("settings modal opens and has controls", async ({ page }) => {
    await page.goto("/");
    await page.locator(".settings-button").click();
    const modal = page.locator("[data-settings-modal]");
    await expect(modal).toBeVisible();
    // Should have theme, font, density, reading focus controls.
    await expect(modal.locator("[data-theme-select]")).toBeVisible();
    await expect(modal.locator("[data-font-size-select]")).toBeVisible();
    await expect(modal.locator("[data-density-select]")).toBeVisible();
    await expect(modal.locator("[data-reading-focus-select]")).toBeVisible();
    await expect(modal.locator("[data-palette-recent-clear]")).toBeVisible();
  });

  test("density preference persists and applies", async ({ page }) => {
    await page.goto("/");
    await page.locator(".settings-button").click();
    const modal = page.locator("[data-settings-modal]");
    const densitySelect = modal.locator("[data-density-select]");
    // Default is Compact
    await expect(densitySelect).toHaveValue("compact");
    // Set to Comfortable
    await densitySelect.selectOption("comfortable");
    const htmlDensity = await page.evaluate(() => document.documentElement.dataset.density);
    expect(htmlDensity).toBe("comfortable");
    // Reload and verify persistence
    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("data-density", "comfortable");
    // Reset back to Compact
    await page.locator(".settings-button").click();
    await modal.locator("[data-density-select]").selectOption("compact");
    await page.reload();
    await expect(page.locator("html")).not.toHaveAttribute("data-density");
  });

  test("reading focus setting persists and applies", async ({ page }) => {
    await page.goto("/");
    await page.locator(".settings-button").click();
    const modal = page.locator("[data-settings-modal]");
    const rfSelect = modal.locator("[data-reading-focus-select]");
    // Default is Off
    await expect(rfSelect).toHaveValue("off");
    // Set to On
    await rfSelect.selectOption("on");
    await expect(page.locator("html")).toHaveAttribute("data-reading-focus", "true");
    await expect(page.locator("body")).toHaveClass(/reading-focus/);
    // Reload and verify persistence
    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("data-reading-focus", "true");
    await expect(page.locator("[data-palette-open]")).toBeVisible();
    await expect(page.locator("[data-sidebar-toggle]")).toBeHidden();
    // Turn off
    await page.locator(".settings-fab").click();
    await modal.locator("[data-reading-focus-select]").selectOption("off");
    await page.reload();
    await expect(page.locator("html")).not.toHaveAttribute("data-reading-focus");
  });

  test("palette recents appear after opening a URL-backed item", async ({ page }) => {
    await page.goto("/");
    // Clear any previous recents
    await page.evaluate(() => localStorage.removeItem("notes-web:palette-recent"));
    // Open palette
    await page.locator("[data-palette-open]").click();
    const palette = page.locator("[data-palette]");
    await expect(palette).toBeVisible();
    // Type to find a note-backed item
    await palette.locator("[data-palette-input]").fill("Target");
    await page.waitForTimeout(300);
    // Click the first result (should be a URL-backed note item)
    const firstResult = palette.locator('[data-palette-index]').filter({ hasText: 'Target' }).first();
    await expect(firstResult).toBeVisible();
    await firstResult.click();
    await expect(page).toHaveURL(/Target.*\.md$/);
    // Now navigate back home
    await page.goto("/");
    // Open palette with empty query - should show recents
    await page.locator("[data-palette-open]").click();
    await expect(palette).toBeVisible();
    // Recent header should be visible
    await expect(palette.locator(".palette-recent-header")).toBeVisible();
    // Should have at least one recent item
    await expect(palette.locator('[data-palette-index="0"]')).toBeVisible();
    // Clear recents from settings
    await page.keyboard.press("Escape");
    await page.locator(".settings-button").click();
    await page.locator("[data-palette-recent-clear]").click();
    // Re-open palette to verify recents are gone
    await page.keyboard.press("Escape");
    await page.locator("[data-palette-open]").click();
    await expect(palette.locator(".palette-recent-header")).toHaveCount(0);
  });

  test("palette recents ignore stale localStorage URLs", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => localStorage.setItem("notes-web:palette-recent", JSON.stringify([
      { title: "Stale hidden note", path: "_trash/stale.md", url: "/_trash/stale.md", kind: "note" },
      { title: "External", path: "https://example.com", url: "https://example.com", kind: "note" },
    ])));
    await page.reload();
    await page.locator("[data-palette-open]").click();
    const palette = page.locator("[data-palette]");
    await expect(palette.locator(".palette-recent-header")).toHaveCount(0);
    await expect(palette.locator("text=Stale hidden note")).toHaveCount(0);
  });

  test("mobile settings modal avoids horizontal overflow", async ({ page }) => {
    for (const width of [390, 320]) {
      await page.setViewportSize({ width, height: 844 });
      await page.goto("/");
      await page.locator(".settings-fab").click();
      const modal = page.locator("[data-settings-modal]");
      await expect(modal).toBeVisible();
      const metrics = await page.evaluate(() => ({
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
      }));
      expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
      await page.keyboard.press("Escape");
    }
  });

  test("mobile palette avoids horizontal overflow", async ({ page }) => {
    for (const width of [390, 320]) {
      await page.setViewportSize({ width, height: 844 });
      await page.goto("/");
      await page.locator("[data-palette-open]").click();
      await page.waitForTimeout(300);
      const palette = page.locator("[data-palette]");
      await expect(palette).toBeVisible();
      const metrics = await page.evaluate(() => ({
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
      }));
      expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
      await page.keyboard.press("Escape");
    }
  });

  test("dataview diagnostics avoid mobile horizontal page overflow", async ({ page }) => {
    for (const width of [390, 320]) {
      await page.setViewportSize({ width, height: 844 });
      await page.goto("/_dataview");
      await expect(page.locator(".dataview-diagnostic").first()).toBeVisible();
      const metrics = await page.evaluate(() => ({
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
      }));
      expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
    }
  });

  test("sidebar toggle exists in DOM", async ({ page }) => {
    await page.goto("/");
    // Mobile sidebar toggle is present in DOM (hidden on desktop viewport).
    const toggle = page.locator("[data-sidebar-toggle]");
    await expect(toggle).toBeAttached();
  });
});
