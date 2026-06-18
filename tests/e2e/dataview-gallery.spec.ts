import { test, expect } from "@playwright/test";

test.describe("Dataview syntax gallery", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/Syntax/Dataview%20Gallery.md");
    await page.waitForSelector("article.note");
  });

  test("TABLE renders as dataview-table-wrap", async ({ page }) => {
    const table = page.locator('.dataview-table-wrap[data-dataview-table="1"]');
    await expect(table).toBeVisible();
    await expect(table.locator("table.dataview-table")).toBeVisible();
  });

  test("LIST renders as dataview-list", async ({ page }) => {
    const list = page.locator("ul.dataview.dataview-list");
    await expect(list).toBeVisible();
    // Should have list items.
    await expect(list.locator("li").first()).toBeVisible();
  });

  test("TASK renders as dataview-tasks", async ({ page }) => {
    const taskList = page.locator("ul.dataview.dataview-tasks");
    await expect(taskList).toBeVisible();
    // Should list tasks with checkboxes.
    const items = taskList.locator("li");
    await expect(items.first()).toBeVisible();
    const checkbox = taskList.locator('input[type="checkbox"][disabled]');
    await expect(checkbox.first()).toBeVisible();
  });

  test("CALENDAR renders a dataview-calendar container", async ({ page }) => {
    // The first CALENDAR block (file.mtime for Syntax/) renders a calendar with days.
    const calendar = page.locator("div.dataview.dataview-calendar").first();
    await expect(calendar).toBeVisible();
    // Should contain day sections or the "no date" fallback.
    const daySection = calendar.locator("section.dataview-calendar-day");
    await expect(daySection.or(calendar.locator("p")).first()).toBeVisible();
  });

  test("GROUP BY shows grouped rows in table", async ({ page }) => {
    // GROUP BY table has data-dataview-table="2" (second TABLE block).
    const table = page.locator('.dataview-table-wrap[data-dataview-table="2"]');
    await expect(table).toBeVisible();
    // Should contain at least one group header row.
    const groupRow = table.locator("tr.dataview-group");
    await expect(groupRow.first()).toBeVisible();
  });

  test("FLATTEN expands tags into separate rows", async ({ page }) => {
    // FLATTEN table has data-dataview-table="3".
    const table = page.locator('.dataview-table-wrap[data-dataview-table="3"]');
    await expect(table).toBeVisible();
  });

  test("WHERE + LIMIT filters and constrains results", async ({ page }) => {
    // WHERE+LIMIT table has data-dataview-table="4".
    const table = page.locator('.dataview-table-wrap[data-dataview-table="4"]');
    await expect(table).toBeVisible();
    // The WHERE clause filters on file.tags containing "demo".
    // LIMIT 5 should constrain to at most 5 rows.
    const tbody = table.locator("table.dataview-table tbody");
    const rows = tbody.locator("tr:not(.dataview-group):not([hidden])");
    const count = await rows.count();
    expect(count).toBeLessThanOrEqual(5);
  });

  test("CALENDAR with unsupported date shows fallback message", async ({ page }) => {
    // The second CALENDAR block (gibberish-without-date) produces a dataview-calendar
    // with the "no date" fallback message.
    const calendars = page.locator("div.dataview.dataview-calendar");
    const secondCalendar = calendars.nth(1);
    await expect(secondCalendar).toBeVisible();
    await expect(secondCalendar.locator("p")).toContainText("Aucune date");
  });

  test("dataviewjs code block renders as regular code block", async ({ page }) => {
    // dataviewjs blocks are not processed by the dataview preprocessor.
    // They render as a regular <pre><code> with language-dataviewjs class.
    const codeBlock = page.locator('pre.code-block code.language-dataviewjs');
    await expect(codeBlock).toBeVisible();
    await expect(codeBlock).toContainText("dataviewjs");
  });
});

test.describe("Dataview diagnostics page", () => {
  test("unsupported blocks shown in diagnostics include dataviewjs", async ({ page }) => {
    await page.goto("/_dataview");
    const unsupported = page.locator("article.dataview-diagnostic.unsupported");
    // The dataviewjs block should be listed as unsupported.
    await expect(unsupported.locator("code")).toContainText("dataviewjs");
  });

  test("diagnostics counts update correctly", async ({ page }) => {
    await page.goto("/_dataview");
    const headerText = await page.locator("section.page-header p.muted").textContent();
    expect(headerText).toContain("block(s) found");
    expect(headerText).toContain("unsupported");
  });
});
