import { test, expect } from "@playwright/test";

test.describe("Markdown rendering", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
  });

  test("frontmatter panel is rendered", async ({ page }) => {
    const fm = page.locator("details.frontmatter");
    await expect(fm).toBeVisible();
    await expect(fm.locator("summary")).toContainText("Frontmatter");
    await expect(fm.locator("dt")).toContainText(["created", "tags", "title"]);
  });

  test("table of contents is rendered with heading links", async ({ page }) => {
    const toc = page.locator('details[data-panel-state="toc"]');
    await expect(toc).toBeVisible();
    // The TOC should contain links to major headings.
    await expect(toc.locator("a[href='#headings']")).toBeVisible();
    await expect(toc.locator("a[href='#gfm-table']")).toBeVisible();
  });

  test("heading has an auto-generated anchor id", async ({ page }) => {
    const heading = page.getByRole("heading", { name: "Headings", exact: true });
    await expect(heading).toBeVisible();
    // Goldmark auto-heading-id generates a slug.
    await expect(heading).toHaveId("headings");
  });

  test("GFM table is wrapped in markdown-table-wrap", async ({ page }) => {
    const wrap = page.locator(".markdown-table-wrap");
    await expect(wrap).toBeVisible();
    await expect(wrap.locator("table")).toBeVisible();
    await expect(wrap.locator("th")).toContainText(["Feature", "Status", "Priority"]);
  });

  test("task checkboxes render with disabled input and class", async ({ page }) => {
    // Completed checkbox should have checked and disabled.
    const completedCheckbox = page.locator('li.task-list-item input[type="checkbox"][checked][disabled]');
    await expect(completedCheckbox.first()).toBeVisible();
    // Uncompleted checkbox should be disabled but not checked.
    const unchecked = page.locator('li.task-list-item input[type="checkbox"]:not([checked])[disabled]');
    await expect(unchecked.first()).toBeVisible();
  });

  test("task metadata spans are rendered (due, done, priority)", async ({ page }) => {
    await expect(page.locator("span.task-meta.due-date").first()).toBeVisible();
    await expect(page.locator("span.task-meta.done-date").first()).toBeVisible();
    await expect(page.locator("span.task-meta.priority-meta").first()).toBeVisible();
  });

  test("task ID button is rendered with data-copy attribute", async ({ page }) => {
    const tidBtn = page.locator('button.task-id[data-copy="DEMO-001"]');
    await expect(tidBtn).toBeVisible();
    await expect(tidBtn).toContainText("tid:DEMO-001");
  });

  test("code block has copy button", async ({ page }) => {
    const pre = page.locator("pre.code-block");
    await expect(pre).toBeVisible();
    await expect(pre.locator('button.code-copy[data-copy-code]')).toBeVisible();
  });

  test("callouts render with correct structure", async ({ page }) => {
    const noteCallout = page.locator('div.callout[data-callout="note"]');
    await expect(noteCallout).toBeVisible();
    await expect(noteCallout.locator(".callout-title-text")).toContainText("Custom title");

    const warningCallout = page.locator('div.callout[data-callout="warning"]');
    await expect(warningCallout).toBeVisible();
    await expect(warningCallout.locator(".callout-title-text")).toContainText("Warning");

    const collapsedCallout = page.locator("div.callout.is-collapsed");
    await expect(collapsedCallout).toBeVisible();
  });

  test("mermaid pre element is rendered", async ({ page }) => {
    const mermaid = page.locator("pre.mermaid");
    await expect(mermaid).toBeVisible();
  });

  test("footnote section is rendered", async ({ page }) => {
    // Goldmark renders footnotes as a div with role="doc-endnotes".
    const fnSection = page.locator("div.footnotes[role='doc-endnotes']");
    await expect(fnSection).toBeVisible();
    await expect(fnSection.locator("li")).toContainText("This is the footnote content.");
  });

  test("raw HTML block is preserved", async ({ page }) => {
    const raw = page.locator('div.raw-html-marker[data-testid="raw-html"]');
    await expect(raw).toBeVisible();
    await expect(raw).toContainText("Raw HTML block");
  });

  test("local image renders with correct src", async ({ page }) => {
    const img = page.locator('.content img[src="/Assets/tiny.svg"]');
    await expect(img).toBeVisible();
    await expect(img).toHaveAttribute("alt", "Tiny SVG");
  });

  test("missing and non-previewable media use compact placeholders", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    const missing = page.locator('.content .media-placeholder[href="/Assets/missing-preview.png"]');
    const pdf = page.locator('.content .media-placeholder[href="/Assets/reference.pdf"]');
    await expect(missing).toBeVisible();
    await expect(missing).toContainText("missing-preview.png");
    await expect(missing).toContainText("Image unavailable");
    await expect(pdf).toBeVisible();
    await expect(pdf).toContainText("reference.pdf");
    await expect(pdf).toContainText("Media not previewable");
    await expect(pdf.locator(".media-placeholder-icon")).toHaveText("PDF");
    await expect(page.locator('.content img[src="/Assets/missing-preview.png"]')).toHaveCount(0);
    await expect(page.locator('.content img[src="/Assets/reference.pdf"]')).toHaveCount(0);

    const metrics = await missing.evaluate((el) => {
      const box = el.getBoundingClientRect();
      const style = getComputedStyle(el);
      const icon = el.querySelector<HTMLElement>(".media-placeholder-icon");
      const iconBox = icon?.getBoundingClientRect();
      return {
        width: box.width,
        height: box.height,
        shadow: style.boxShadow,
        borderWidth: style.borderTopWidth,
        iconWidth: iconBox?.width ?? 0,
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
      };
    });
    expect(metrics.height).toBeLessThan(90);
    expect(metrics.iconWidth).toBeLessThanOrEqual(36);
    expect(metrics.shadow).toBe("none");
    expect(metrics.borderWidth).toBe("1px");
    expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
  });

  test("media placeholders stay flat on mobile", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/Syntax/All%20Syntaxes.md");
    const missing = page.locator('.content .media-placeholder[href="/Assets/missing-preview.png"]');
    await expect(missing).toBeVisible();
    const metrics = await missing.evaluate((el) => ({
      width: el.getBoundingClientRect().width,
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
    }));
    expect(metrics.width).toBeLessThanOrEqual(metrics.clientWidth);
    expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
  });

  test("external link is rendered", async ({ page }) => {
    const extLink = page.locator('a[href="https://example.com"]');
    await expect(extLink).toBeVisible();
    await expect(extLink).toContainText("Notes Web");
  });
});

test.describe("Wikilinks and navigation panels", () => {
  test("unique wikilink renders as anchor", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const link = page.locator('.content a[href^="/Syntax/Target%20Note.md"]').first();
    await expect(link).toBeVisible();
    await expect(link).toContainText("Target Note");
  });

  test("wikilink with alias uses alias text", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const aliasLink = page.locator('.content a').filter({ hasText: "Aliased Link" });
    await expect(aliasLink).toBeVisible();
  });

  test("wikilink with heading anchor navigates to section", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const headingLink = page.locator('.content a[href$="#section-a"]');
    await expect(headingLink).toBeVisible();
  });

  test("missing wikilink links to missing page", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const missingLink = page.locator('.content a[href^="/_missing?name="]');
    await expect(missingLink).toBeVisible();
    await expect(missingLink).toContainText("NonExistentTarget");
  });

  test("ambiguous wikilink links to resolve page", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const ambigLink = page.locator('.content a[href^="/_resolve?name="]');
    await expect(ambigLink).toBeVisible();
    await expect(ambigLink).toContainText("Resolve One");
  });

  test("forward links panel lists unique/missing/ambiguous", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const panel = page.locator("details.forward-links");
    await expect(panel).toBeVisible();
    await expect(panel.locator(".panel-title")).toContainText("Forward links");
    // Should contain "unique", "missing", "ambiguous" kind labels.
    await expect(panel.locator("small")).toContainText(["unique", "missing", "ambiguous"]);
  });

  test("backlinks panel shows incoming links", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const panel = page.locator("details.backlinks");
    await expect(panel).toBeVisible();
    // Backlink Source links to All Syntaxes.
    await expect(panel.locator("a")).toContainText("Syntax/Backlink Source.md");
  });

  test("note without backlinks shows empty state", async ({ page }) => {
    await page.goto("/Syntax/Broken%20Link%20Source.md");
    await page.waitForSelector("article.note");
    const panel = page.locator("details.backlinks");
    await expect(panel.locator(".empty-state")).toBeVisible();
    await expect(panel.locator(".empty-state")).toContainText("No backlinks.");
  });
});

test.describe("Copy buttons", () => {
  test("code copy button exists and has aria-label", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    await expect(page.locator('button.code-copy[data-copy-code]')).toHaveCount(1);
  });

  test("task ID copy button has data-copy attribute", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const tidBtn = page.locator('button.task-id[data-copy="DEMO-001"]');
    await expect(tidBtn).toBeVisible();
    await expect(tidBtn).toHaveAttribute("title", "Copy task ID");
  });

  test("copy path button is present on note page", async ({ page }) => {
    await page.addInitScript(() => {
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: { writeText: () => Promise.resolve() },
      });
    });
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");
    const trigger = page.getByRole("button", { name: "Actions" });
    await trigger.click();
    const menu = page.locator("[data-note-actions-menu]");
    const copyPath = menu.getByRole("menuitem", { name: "Copy path" });
    await expect(copyPath).toBeVisible();
    await copyPath.click();
    await expect(menu).toBeHidden();
    await expect(trigger).toBeFocused();
  });
});
