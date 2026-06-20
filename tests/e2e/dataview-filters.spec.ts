import { test, expect, type Page, type Locator } from "@playwright/test";

/** Navigate to the project index page and wait for the first Dataview table. */
async function gotoProjectIndex(page: Page) {
  await page.goto("/Projects/Index.md");
  await page.waitForSelector(
    '.dataview-table-wrap[data-dataview-action="renderDataviewTable"][data-dataview-table="1"]',
  );
}

/** Return the first Dataview table wrapper (table=1 with filters). */
function tableWrap(page: Page): Locator {
  return page.locator(
    '.dataview-table-wrap[data-dataview-action="renderDataviewTable"][data-dataview-table="1"]',
  );
}

/** Return visible data rows (non-group, non-noresult) within the first table. */
function dataRows(page: Page): Locator {
  return tableWrap(page).locator(
    ".dataview-table tbody tr:not(.dataview-group):not([hidden]):not(:has(.dataview-no-rows))",
  );
}

/** Wait for an AJAX Dataview update and let the DOM replacement settle. */
async function waitForAjax(page: Page): Promise<void> {
  await page.waitForFunction(
    () => !document.querySelector(".dataview-table-wrap[data-dataview-loading]"),
  );
  // Allow the DOM replacement microtask (replaceWith + initDataviewWrapper) to complete.
  await page.waitForTimeout(50);
}

test.describe("Dataview table filters", () => {
  test("filtered tables with more than ten rows are not implicitly capped", async ({
    page,
  }) => {
    await page.goto("/DataviewCap/Filter%20Cap.md");
    await page.waitForSelector(
      '.dataview-table-wrap[data-dataview-action="renderDataviewTable"][data-dataview-table="1"]',
    );

    const rows = dataRows(page);
    await expect(page.locator(".dataview-cap-note")).toHaveCount(0);
    await expect(rows).toHaveCount(12);
    await expect(rows.nth(0)).toContainText("Active01");
    await expect(rows.nth(11)).toContainText("Active12");

    const statusSelect = tableWrap(page).locator(
      'select[data-dataview-filter="status"]',
    );
    await statusSelect.selectOption("done");
    await waitForAjax(page);
    await expect(page.locator(".dataview-cap-note")).toHaveCount(0);
    await expect(rows).toHaveCount(1);
    await expect(rows.nth(0)).toContainText("Done01");

    await statusSelect.selectOption("active");
    await waitForAjax(page);
    await expect(page.locator(".dataview-cap-note")).toHaveCount(0);
    await expect(rows).toHaveCount(12);
  });

  test("single status filter defaults to active and is clearable", async ({
    page,
  }) => {
    await gotoProjectIndex(page);

    const statusSelect = tableWrap(page).locator(
      'select[data-dataview-filter="status"]',
    );

    // Default is "active".
    await expect(statusSelect).toHaveValue("active");

    // Only active projects (Alpha, Gamma) should be visible.
    const rows = dataRows(page);
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toContainText("Alpha");
    await expect(rows.nth(1)).toContainText("Gamma");

    // Clear to "All" — show all 5 project notes (Index.md is also in Projects/).
    await statusSelect.selectOption("");
    await waitForAjax(page);
    await expect(rows).toHaveCount(5);

    // Select "done" — only Beta.
    await statusSelect.selectOption("done");
    await waitForAjax(page);
    await expect(rows).toHaveCount(1);
    await expect(rows.nth(0)).toContainText("Beta");
  });

  test("URL unchanged after filter change (no history update)", async ({
    page,
  }) => {
    await gotoProjectIndex(page);

    const originalUrl = page.url();
    expect(originalUrl).toContain("/Projects/Index.md");

    // Change a filter.
    const statusSelect = tableWrap(page).locator(
      'select[data-dataview-filter="status"]',
    );
    await statusSelect.selectOption("done");
    await waitForAjax(page);

    // URL must remain identical.
    expect(page.url()).toBe(originalUrl);
  });

  test("multi tags filter shows #tag values", async ({ page }) => {
    await gotoProjectIndex(page);

    // Open the multi filter menu.
    const multiBtn = tableWrap(page).locator(
      '.dataview-multi-btn[data-dataview-filter="tags"]',
    );
    await multiBtn.click();

    const menu = tableWrap(page).locator(".dataview-multi-menu");
    await expect(menu).toBeVisible();

    // Options should be displayed with # prefix.
    await expect(menu.locator("label")).toContainText([
      "#active",
      "#dashboard",
    ]);
  });

  test("multi tags filter filters by checkboxes", async ({ page }) => {
    await gotoProjectIndex(page);

    // Open the multi filter menu.
    const multiBtn = tableWrap(page).locator(
      '.dataview-multi-btn[data-dataview-filter="tags"]',
    );
    await multiBtn.click();

    const menu = tableWrap(page).locator(".dataview-multi-menu");

    // Toggle #dashboard — should show only Gamma.
    await menu.locator('input[type="checkbox"][value="#dashboard"]').check();
    await waitForAjax(page);

    const rows = dataRows(page);
    await expect(rows).toHaveCount(1);
    await expect(rows.nth(0)).toContainText("Gamma");
  });

  test("multi filter sends repeated params", async ({ page }) => {
    await gotoProjectIndex(page);

    // First: toggle #active.
    const btn1 = tableWrap(page).locator(
      '.dataview-multi-btn[data-dataview-filter="tags"]',
    );
    await btn1.click();
    await tableWrap(page)
      .locator(".dataview-multi-menu")
      .locator('input[type="checkbox"][value="#active"]')
      .check();
    await waitForAjax(page);

    // Second: toggle #dashboard in the fresh wrapper.
    const btn2 = tableWrap(page).locator(
      '.dataview-multi-btn[data-dataview-filter="tags"]',
    );
    await btn2.click();
    // Observe the outgoing request — it should carry both filter.tags params.
    const [request] = await Promise.all([
      page.waitForRequest(
        (req) =>
          req.url().includes("filter.tags=%23active") &&
          req.url().includes("filter.tags=%23dashboard"),
      ),
      tableWrap(page)
        .locator(".dataview-multi-menu")
        .locator('input[type="checkbox"][value="#dashboard"]')
        .check(),
    ]);
    expect(request).toBeTruthy();
    // Wait for the AJAX response to complete and DOM to settle before next test.
    await waitForAjax(page);
  });

  test("multi keyboard navigation: arrow keys and escape", async ({
    page,
  }) => {
    await gotoProjectIndex(page);

    const multiBtn = tableWrap(page).locator(
      '.dataview-multi-btn[data-dataview-filter="tags"]',
    );
    const menu = tableWrap(page).locator(".dataview-multi-menu");

    // Open the multi menu by clicking the button.
    await multiBtn.click();
    await expect(menu).toBeVisible();

    // After opening, focus the button again, then press ArrowDown.
    // The button keydown handler forwards ArrowDown into the menu.
    await multiBtn.focus();
    await page.keyboard.press("ArrowDown");
    const inputs = tableWrap(page).locator(
      ".dataview-multi-menu input[type='checkbox']",
    );
    await expect(inputs.nth(0)).toBeFocused();

    // ArrowDown again to second checkbox.
    await page.keyboard.press("ArrowDown");
    await expect(inputs.nth(1)).toBeFocused();

    // Escape closes the menu and focus returns to the button.
    await page.keyboard.press("Escape");
    await expect(menu).not.toBeVisible();
    await expect(multiBtn).toBeFocused();

    // Enter on the button opens the menu.
    await page.keyboard.press("Enter");
    await expect(menu).toBeVisible();

    // Home/End move focus to first/last menu item.
    const inputsAfterEnter = tableWrap(page).locator(
      ".dataview-multi-menu input[type='checkbox']",
    );
    await page.keyboard.press("End");
    await expect(inputsAfterEnter.last()).toBeFocused();
    await page.keyboard.press("Home");
    await expect(inputsAfterEnter.first()).toBeFocused();

    // Tab closes the menu naturally.
    await page.keyboard.press("Tab");
    await expect(menu).not.toBeVisible();

    // Space on the button also opens it, and clicking outside closes it.
    await multiBtn.focus();
    await page.keyboard.press("Space");
    await expect(menu).toBeVisible();
    await page.locator("#project-index").click();
    await expect(menu).not.toBeVisible();
  });

  test("shows no matching rows when default is absent from options", async ({
    page,
  }) => {
    await page.goto("/NoResultsTest.md");
    await page.waitForSelector(
      '.dataview-table-wrap[data-dataview-action="renderDataviewTable"]',
    );

    const noRows = page.locator(".dataview-table .dataview-no-rows");
    await expect(noRows).toBeVisible();
    await expect(noRows).toContainText("No matching rows");
  });

  test("text q and dropdown combine with AND", async ({ page }) => {
    // Navigate fresh to avoid cross-test state issues.
    await page.goto("/Projects/Index.md");
    await page.waitForSelector(
      '.dataview-table-wrap[data-dataview-action="renderDataviewTable"][data-dataview-table="1"]',
    );

    const textFilter = page
      .locator(
        '.dataview-table-wrap[data-dataview-action="renderDataviewTable"][data-dataview-table="1"]',
      )
      .locator('input.dataview-filter[data-dataview-filter]')
      .first();

    // Type text — the initial status default "active" is still in the DOM,
    // so the debounced AJAX request carries q=Alpha + filter.status=active.
    await textFilter.fill("Alpha");
    // Wait for debounce (200ms) + AJAX response + DOM replacement.
    await page.waitForTimeout(600);

    // Only Alpha matches both "Alpha" text and "active" status.
    const rows = dataRows(page);
    await expect(rows).toHaveCount(1);
    await expect(rows.nth(0)).toContainText("Alpha");
  });

  test("q debounce respects latest response", async ({ page }) => {
    await gotoProjectIndex(page);

    const textFilter = tableWrap(page).locator(
      'input.dataview-filter[data-dataview-filter]',
    );

    // Count requests triggered by text filter changes.
    let requestCount = 0;
    page.on("request", (req) => {
      if (req.url().includes("action=renderDataviewTable")) {
        requestCount++;
      }
    });

    // Type quickly — only the final debounced value should fire.
    await textFilter.fill("A");
    await textFilter.fill("Al");
    await textFilter.fill("Alp");
    await textFilter.fill("Alph");
    await textFilter.fill("Alpha");
    await page.waitForTimeout(500); // longer than 200ms debounce

    // Only one request should have been made for the final value.
    expect(requestCount).toBe(1);
  });

  test("pagination resets to page 1 and preserves Rows value", async ({
    page,
  }) => {
    await gotoProjectIndex(page);

    // Set a non-default page size.
    const pageSizeSelect = tableWrap(page).locator(
      "[data-dataview-page-size]",
    );
    await pageSizeSelect.selectOption("25");
    await expect(pageSizeSelect).toHaveValue("25");

    // Apply a filter (triggers AJAX which resets page to 1).
    const statusSelect = tableWrap(page).locator(
      'select[data-dataview-filter="status"]',
    );
    await statusSelect.selectOption("active");
    await waitForAjax(page);

    // Rows value must be preserved after AJAX response.
    await expect(pageSizeSelect).toHaveValue("25");

    // Page 1 is shown (either as "Page 1 / …" when rows > page size,
    // or as "N rows" when all rows fit on one page).
    const pager = tableWrap(page).locator("[data-dataview-pager]");
    await expect(pager).not.toBeEmpty();
  });

  test("server-side sort works with active filters", async ({ page }) => {
    await gotoProjectIndex(page);

    // Apply a filter to show only active projects.
    const statusSelect = tableWrap(page).locator(
      'select[data-dataview-filter="status"]',
    );
    await statusSelect.selectOption("active");
    await waitForAjax(page);

    // Find the "Name" column header in the first table.
    const nameHeader = tableWrap(page).locator(
      'th[data-dataview-sort-field="file.name"]',
    );

    // First click: descending (Gamma before Alpha).
    await nameHeader.click();
    await waitForAjax(page);
    await expect(nameHeader).toHaveAttribute("aria-sort", "descending");
    let rows = dataRows(page);
    await expect(rows.nth(0)).toContainText("Gamma");
    await expect(rows.nth(1)).toContainText("Alpha");

    // Second click: ascending (Alpha before Gamma).
    await nameHeader.click();
    await waitForAjax(page);
    await expect(nameHeader).toHaveAttribute("aria-sort", "ascending");
    rows = dataRows(page);
    await expect(rows.nth(0)).toContainText("Alpha");
    await expect(rows.nth(1)).toContainText("Gamma");
  });

  test("complex header does not trigger sort fetch", async ({ page }) => {
    await gotoProjectIndex(page);

    const complexHeader = tableWrap(page).locator("th", { hasText: "Updated" });
    await expect(complexHeader).not.toHaveAttribute("data-dataview-sort", "");
    await expect(complexHeader).not.toHaveAttribute(
      "data-dataview-sort-field",
      /.+/,
    );

    let requestCount = 0;
    page.on("request", (req) => {
      if (req.url().includes("action=renderDataviewTable")) requestCount++;
    });
    await complexHeader.click();
    await page.waitForTimeout(250);
    expect(requestCount).toBe(0);
  });

  test("Rows value preserved after AJAX response", async ({ page }) => {
    await gotoProjectIndex(page);

    // Change Rows to a non-default value.
    const pageSizeSelect = tableWrap(page).locator(
      "[data-dataview-page-size]",
    );
    await pageSizeSelect.selectOption("25");
    await expect(pageSizeSelect).toHaveValue("25");

    // Trigger an AJAX request by changing a filter.
    const statusSelect = tableWrap(page).locator(
      'select[data-dataview-filter="status"]',
    );
    await statusSelect.selectOption("done");
    await waitForAjax(page);

    // Rows must still be 25 after AJAX replacement.
    await expect(pageSizeSelect).toHaveValue("25");
  });

  test("direct action URL returns partial HTML", async ({ page }) => {
    const response = await page.request.get(
      "/Projects/Index.md?action=renderDataviewTable&table=1&filter.status=done",
    );
    expect(response.ok()).toBeTruthy();
    expect(response.headers()["content-type"]).toContain("text/html");

    const text = await response.text();

    expect(text).toContain('class="dataview dataview-table-wrap"');
    expect(text).toContain('data-dataview-action="renderDataviewTable"');
    expect(text).toContain("Beta");
    expect(text).not.toContain("Alpha");
    expect(text).not.toContain("<html");
    expect(text).not.toContain("</html>");
  });

  test("direct action URL with unknown table returns error fragment", async ({
    page,
  }) => {
    const response = await page.request.get(
      "/Projects/Index.md?action=renderDataviewTable&table=999",
    );
    expect(response.ok()).toBeFalsy();
    expect(response.status()).toBe(400);

    const text = await response.text();
    expect(text).toContain("dataview-error");
  });

  test("direct action URL with non-GET returns 405", async ({ page }) => {
    const response = await page.request.post(
      "/Projects/Index.md?action=renderDataviewTable&table=1",
    );
    expect(response.status()).toBe(405);
    expect(await response.text()).toContain("dataview-error");
  });

  test("direct action URL for table=2 works", async ({ page }) => {
    const response = await page.request.get(
      "/Projects/Index.md?action=renderDataviewTable&table=2",
    );
    expect(response.ok()).toBeTruthy();

    const text = await response.text();
    expect(text).toContain('data-dataview-table="2"');
    expect(text).toContain("dataview-table-wrap");
  });
});
