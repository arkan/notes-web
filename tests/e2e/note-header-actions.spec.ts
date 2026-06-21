import { test, expect, type Page } from "@playwright/test";

async function openHeaderActions(page: Page) {
  await page.getByRole("button", { name: "Actions" }).click();
  const menu = page.locator("[data-note-actions-menu]");
  await expect(menu).toBeVisible();
  return menu;
}

test.describe("Note header actions", () => {
  test("shows Open URL for safe source_url and opens in the same tab", async ({ page }) => {
    await page.goto("/Syntax/External%20Source.md");
    await page.waitForSelector("article.note");

    const menu = await openHeaderActions(page);
    const openURL = menu.getByRole("menuitem", { name: "Open URL" });
    await expect(openURL).toBeVisible();
    await expect(openURL).toHaveAttribute("href", "https://example.com/article");
    await expect(openURL).not.toHaveAttribute("target");
  });

  test("does not show Open URL without source_url", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");

    const menu = await openHeaderActions(page);
    await expect(menu.getByRole("menuitem", { name: "Open URL" })).toHaveCount(0);
  });

  test("does not show Open URL for unsafe source_url", async ({ page }) => {
    await page.goto("/Syntax/Unsafe%20Source.md");
    await page.waitForSelector("article.note");

    const menu = await openHeaderActions(page);
    await expect(menu.getByRole("menuitem", { name: "Open URL" })).toHaveCount(0);
  });

  test("shows Copy reading prompt and copies the prompt for reading_list notes", async ({ page }) => {
    await page.addInitScript(() => {
      let copiedText = "";
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: {
          readText: () => Promise.resolve(copiedText),
          writeText: (text: string) => {
            copiedText = text;
            return Promise.resolve();
          },
        },
      });
    });
    await page.goto("/Syntax/Reading%20List.md");
    await page.waitForSelector("article.note");

    const menu = await openHeaderActions(page);
    const promptButton = menu.getByRole("menuitem", { name: "Copy reading prompt" });
    await expect(promptButton).toBeVisible();
    await expect(promptButton).toHaveAttribute("data-copy", /Reading list item: Syntax\/Reading List\.md/);
    await expect(promptButton).toHaveAttribute("data-copy", /Mark as read/);
    await expect(promptButton).toHaveAttribute("data-copy", /Mark as unread/);
    await expect(promptButton).toHaveAttribute("data-copy", /Archive/);

    await promptButton.click();
    await expect(page.locator("article.note header button.note-actions-item.copied")).toContainText("copied");

    const copiedText = await page.evaluate(() => navigator.clipboard.readText());
    expect(copiedText).toContain("Reading list item: Syntax/Reading List.md");
    expect(copiedText).toContain("Mark as read");
    expect(copiedText).toContain("Mark as unread");
    expect(copiedText).toContain("Archive");
  });

  test("does not show Copy reading prompt for regular notes", async ({ page }) => {
    await page.goto("/Syntax/All%20Syntaxes.md");
    await page.waitForSelector("article.note");

    const menu = await openHeaderActions(page);
    await expect(menu.getByRole("menuitem", { name: "Copy reading prompt" })).toHaveCount(0);
  });
});
