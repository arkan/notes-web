import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { cp, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import net from "node:net";
import { tmpdir } from "node:os";
import path from "node:path";
import { expect, test, type Page } from "@playwright/test";

const repoRoot = process.cwd();

async function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close(() => reject(new Error("unable to allocate port")));
        return;
      }
      const port = address.port;
      server.close(() => resolve(port));
    });
  });
}

async function delay(ms: number): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForServer(url: string, child: ChildProcessWithoutNullStreams, logs: () => string): Promise<void> {
  for (let i = 0; i < 100; i++) {
    if (child.exitCode !== null) {
      throw new Error(`notes-web exited before readiness:\n${logs()}`);
    }
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // Server not ready yet.
    }
    await delay(100);
  }
  throw new Error(`notes-web did not become ready:\n${logs()}`);
}

async function openHeaderActions(page: Page) {
  const trigger = page.getByRole("button", { name: "Actions" });
  await expect(trigger).toBeVisible();
  await trigger.click();
  const menu = page.locator("[data-note-actions-menu]");
  await expect(menu).toBeVisible();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  return menu;
}

async function clickHeaderAction(page: Page, name: string): Promise<void> {
  const menu = await openHeaderActions(page);
  await menu.getByRole("menuitem", { name, exact: true }).click();
}

async function createInboxCapture(page: Page, baseURL: string, title: string, body = "Captured body."): Promise<string> {
  await page.goto(`${baseURL}/`);
  const input = page.locator("[data-quick-capture-input]");
  await expect(input).toBeVisible();
  await input.fill(`${title}\n${body}`);
  await input.press("Control+Enter");
  const message = page.locator("[data-quick-capture-message]");
  await expect(message).toContainText("Saved to Inbox");
  const href = await message.locator("a").getAttribute("href");
  if (!href) throw new Error("capture link was not rendered");
  return href;
}

test.describe.serial("Edit mode CRUD", () => {
  let baseURL = "";
  let serverProcess: ChildProcessWithoutNullStreams | undefined;
  let serverLogs = "";
  let tempRoot = "";
  let vaultDir = "";

  test.beforeAll(async () => {
    tempRoot = await mkdtemp(path.join(tmpdir(), "notes-web-edit-mode-"));
    vaultDir = path.join(tempRoot, "vault");
    await cp(path.join(repoRoot, "testdata/e2e-vault"), vaultDir, { recursive: true });
    await writeFile(path.join(vaultDir, ".notes-web.yaml"), "editing:\n  enabled: true\n  trash_path: _trash\n  template_name: _template.md\n  hide_templates: true\n  slug: kebab_lowercase\ntodo:\n  todo_file: Tasks/Inbox.md\n", "utf8");
    await writeFile(path.join(vaultDir, "Tasks", "Inbox.md"), "# Inbox tasks\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "_template.md"), "# {{title}}\n\nPath: {{path}}\nDate: {{date}}\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "Edit Mode Fixture.md"), "# Edit Mode Fixture\n\nOriginal text.\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "Conflict Fixture.md"), "# Conflict Fixture\n\nOriginal conflict text.\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "Rename Target.md"), "# Rename Target\n\nRename body.\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "Rename Source.md"), "# Rename Source\n\nSee [[Rename Target]] and [the target](Rename Target.md#details).\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "Missing Link Source.md"), "# Missing Link Source\n\nSee [[Browser Missing|the missing note]].\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "Trash Me.md"), "# Trash Me\n\nMove me to trash.\n", "utf8");
    await writeFile(path.join(vaultDir, "Syntax", "Restore Collision.md"), "# Restore Collision\n\nOriginal trashed content.\n", "utf8");

    const port = await freePort();
    baseURL = `http://127.0.0.1:${port}`;
    serverProcess = spawn("go", ["run", "./cmd/notes-web", "--vault", vaultDir, "--host", "127.0.0.1", "--port", String(port)], {
      cwd: repoRoot,
      env: process.env,
    });
    serverProcess.stdout.on("data", (chunk: Buffer) => { serverLogs += chunk.toString(); });
    serverProcess.stderr.on("data", (chunk: Buffer) => { serverLogs += chunk.toString(); });
    await waitForServer(baseURL, serverProcess, () => serverLogs);
  });

  test.afterAll(async () => {
    if (serverProcess && serverProcess.exitCode === null) {
      serverProcess.kill();
      await new Promise((resolve) => serverProcess?.once("exit", resolve));
    }
    if (tempRoot) await rm(tempRoot, { recursive: true, force: true });
  });

  test("opens and closes the compact header actions menu", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    const trigger = page.getByRole("button", { name: "Actions" });
    const menu = await openHeaderActions(page);
    await expect(menu.getByRole("menuitem", { name: "Edit" })).toBeVisible();
    await expect(menu.getByRole("menuitem", { name: "Copy path" })).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(menu).toBeHidden();
    await expect(trigger).toBeFocused();

    await trigger.click();
    await expect(menu).toBeVisible();
    await page.keyboard.press("Tab");
    await expect(menu).toBeHidden();

    await trigger.click();
    await expect(menu).toBeVisible();
    await page.locator("article.note > header h1").click();
    await expect(menu).toBeHidden();
  });

  test("edits, manually previews stale content, and saves back to read mode", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    await expect(page.locator("article.note")).toBeVisible();

    await clickHeaderAction(page, "Edit");
    const textarea = page.locator("textarea[data-edit-textarea]");
    await expect(textarea).toBeVisible();
    await expect(textarea).toHaveValue(/Original text/);

    const updated = "# Edit Mode Fixture\n\nUpdated **browser** text.\n\nSee [[Target Note]].\n";
    await textarea.fill(updated);
    await expect(page.locator("[data-edit-status]")).toContainText("Unsaved changes");

    await page.getByRole("button", { name: /^Preview/ }).click();
    await expect(page.locator(".edit-preview-panel")).toBeVisible();
    await expect(page.locator(".edit-preview-panel strong")).toContainText("browser");
    await expect(page.locator('.edit-preview-panel a[href^="/Syntax/Target%20Note.md"]')).toBeVisible();

    await page.getByRole("tab", { name: /^Source/ }).click();
    await textarea.fill(`${updated}\nStale after preview.\n`);
    await expect(page.getByText("Preview stale").first()).toBeVisible();

    await page.getByRole("button", { name: /^Save/ }).click();
    await page.waitForFunction(() => !document.querySelector(".edit-workbench"));
    await expect(page.locator("article.note")).toBeVisible();
    await expect(page.locator("article.note > .content")).toContainText("Updated browser text.");
    await expect(page.locator("article.note > .content")).toContainText("Stale after preview.");

    const disk = await readFile(path.join(vaultDir, "Syntax", "Edit Mode Fixture.md"), "utf8");
    expect(disk).toContain("Updated **browser** text.");
    expect(disk).toContain("Stale after preview.");
  });

  test("confirms dirty cancel and internal navigation", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    await clickHeaderAction(page, "Edit");
    const textarea = page.locator("textarea[data-edit-textarea]");
    await textarea.fill("# Edit Mode Fixture\n\nDraft to discard.\n");

    page.once("dialog", async (dialog) => {
      expect(dialog.message()).toContain("Discard unsaved changes");
      await dialog.dismiss();
    });
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.locator(".edit-workbench")).toBeVisible();

    page.once("dialog", async (dialog) => {
      expect(dialog.message()).toContain("Leave this note");
      await dialog.accept();
    });
    await page.locator('nav.crumb a[href="/Syntax"]').click();
    await expect(page).toHaveURL(`${baseURL}/Syntax`);
    await expect(page.locator(".folder-view h1")).toContainText("Syntax");
  });

  test("enters edit mode via E keyboard shortcut", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    await expect(page.locator("article.note")).toBeVisible();

    await page.keyboard.press("e");
    const textarea = page.locator("textarea[data-edit-textarea]");
    await expect(textarea).toBeVisible();
    // Textarea should have content loaded from the file.
    await expect(textarea).not.toHaveValue("");
  });

  test("Ctrl+Enter triggers preview and Ctrl+S saves", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    await clickHeaderAction(page, "Edit");
    const textarea = page.locator("textarea[data-edit-textarea]");
    await expect(textarea).toBeVisible();

    await textarea.fill("# Shortcut Test\n\n**Preview** content.\n");

    // Ctrl+Enter triggers preview.
    await page.keyboard.press("Control+Enter");
    await expect(page.locator(".edit-preview-panel:not([hidden])")).toBeVisible();
    await expect(page.locator(".edit-preview-panel strong")).toContainText("Preview");

    // Ctrl+S saves and returns to read mode.
    await page.keyboard.press("Control+s");
    await page.waitForFunction(() => !document.querySelector(".edit-workbench"));
    await expect(page.locator("article.note > .content")).toContainText("Shortcut Test");
  });

  test("Esc triggers dirty cancel confirmation", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    await clickHeaderAction(page, "Edit");
    const textarea = page.locator("textarea[data-edit-textarea]");
    await textarea.fill("# Esc Test\n\nDraft to confirm discard.\n");

    page.once("dialog", async (dialog) => {
      expect(dialog.message()).toContain("Discard unsaved changes");
      await dialog.dismiss();
    });
    await page.keyboard.press("Escape");
    await expect(page.locator(".edit-workbench")).toBeVisible();
  });

  test("mobile viewport applies full-screen edit workbench CSS", async ({ page }) => {
    // Set a mobile viewport.
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    await clickHeaderAction(page, "Edit");
    await expect(page.locator("textarea[data-edit-textarea]")).toBeVisible();

    // body should have edit-mode-active class.
    const hasClass = await page.evaluate(() => document.body.classList.contains("edit-mode-active"));
    expect(hasClass).toBe(true);

    // The note surface should have position: fixed (mobile full-screen).
    const position = await page.evaluate(() => {
      const surface = document.querySelector(".note.is-editing");
      if (!surface) return "";
      return getComputedStyle(surface).position;
    });
    expect(position).toBe("fixed");
  });

  test("keeps the draft on save conflict and can reload disk content", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Conflict%20Fixture.md`);
    await clickHeaderAction(page, "Edit");
    const textarea = page.locator("textarea[data-edit-textarea]");
    await expect(textarea).toHaveValue(/Original conflict text/);

    await writeFile(path.join(vaultDir, "Syntax", "Conflict Fixture.md"), "# Conflict Fixture\n\nChanged on disk.\n", "utf8");
    await textarea.fill("# Conflict Fixture\n\nBrowser draft kept.\n");
    await page.getByRole("button", { name: /^Save/ }).click();

    await expect(page.locator("[data-edit-message] strong")).toContainText("Save conflict");
    await expect(textarea).toHaveValue(/Browser draft kept/);
    await expect(page.getByRole("button", { name: "Copy draft" })).toBeVisible();

    page.once("dialog", async (dialog) => {
      expect(dialog.message()).toContain("Reload disk");
      await dialog.accept();
    });
    await page.getByRole("button", { name: "Reload disk" }).click();
    await expect(textarea).toHaveValue(/Changed on disk/);
    await expect(textarea).not.toHaveValue(/Browser draft kept/);
  });

  test("creates a templated note from a folder context", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax`);
    await clickHeaderAction(page, "New");

    const title = page.locator('input[name="edit-create-title"]');
    const targetPath = page.locator('input[name="edit-create-path"]');
    await title.fill("Éléphant Café");
    await expect(targetPath).toHaveValue("Syntax/elephant-cafe.md");

    await page.getByRole("button", { name: "Create note" }).click();
    await expect(page).toHaveURL(`${baseURL}/Syntax/elephant-cafe.md`);
    await expect(page.locator("article.note > .content")).toContainText("Éléphant Café");
    await expect(page.locator("article.note > .content")).toContainText("Path: Syntax/elephant-cafe.md");

    const disk = await readFile(path.join(vaultDir, "Syntax", "elephant-cafe.md"), "utf8");
    expect(disk).toContain("# Éléphant Café");
    expect(disk).toContain("Path: Syntax/elephant-cafe.md");
  });

  test("creates a preserved-name folder from the New dialog", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax`);
    await clickHeaderAction(page, "New");
    await page.getByLabel("Create folder").check();
    await page.locator('input[name="edit-create-title"]').fill("Folder With Space");
    await expect(page.locator('input[name="edit-create-path"]')).toHaveValue("Syntax/Folder With Space/");

    await page.getByRole("button", { name: "Create folder" }).click();
    await expect(page).toHaveURL(`${baseURL}/Syntax/Folder%20With%20Space`);
    await expect(page.locator(".folder-view h1")).toContainText("Folder With Space");
  });

  test("renames an empty folder without slugging or .md", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Folder%20With%20Space`);
    await clickHeaderAction(page, "Rename");
    const renameModal = page.locator(".edit-rename-modal");
    await renameModal.locator('input[name="edit-rename-title"]').fill("Renamed Folder");
    await expect(renameModal.locator('input[name="edit-rename-path"]')).toHaveValue("Syntax/Renamed Folder");
    await renameModal.getByRole("button", { name: "Preview impact" }).click();
    await expect(renameModal.getByRole("button", { name: "Rename" })).toBeEnabled();
    await renameModal.getByRole("button", { name: "Rename" }).click();
    await expect(page).toHaveURL(`${baseURL}/Syntax/Renamed%20Folder`);
    await expect(page.locator(".folder-view h1")).toContainText("Renamed Folder");
  });

  test("renames a note after impact preview and rewrites exact links", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Rename%20Target.md`);
    await clickHeaderAction(page, "Rename");
    const renameModal = page.locator(".edit-rename-modal");
    await renameModal.locator('input[name="edit-rename-title"]').fill("Renamed Target");
    await expect(renameModal.locator('input[name="edit-rename-path"]')).toHaveValue("Syntax/renamed-target.md");

    await renameModal.getByRole("button", { name: "Preview impact" }).click();
    await expect(renameModal.locator(".edit-impact")).toContainText("Syntax/Rename Source.md");
    await expect(renameModal.getByRole("button", { name: "Rename" })).toBeEnabled();
    await renameModal.getByRole("button", { name: "Rename" }).click();

    await expect(page).toHaveURL(`${baseURL}/Syntax/renamed-target.md`);
    await expect(page.locator("article.note > header h1")).toContainText("Rename Target");
    const source = await readFile(path.join(vaultDir, "Syntax", "Rename Source.md"), "utf8");
    expect(source).toContain("[[renamed-target]]");
    expect(source).toContain("[the target](renamed-target.md#details)");
    await expect(async () => readFile(path.join(vaultDir, "Syntax", "Rename Target.md"), "utf8")).rejects.toThrow();
  });

  test("creates a note from a missing wikilink and repairs the source", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Missing%20Link%20Source.md`);
    await page.getByRole("link", { name: "the missing note" }).click();
    await expect(page.locator(".missing-note-state")).toContainText("Referenced from Syntax/Missing Link Source.md");

    await page.getByRole("button", { name: "Create this note" }).click();
    await page.getByRole("button", { name: "Preview impact" }).click();
    await expect(page.locator("[data-edit-created-path]")).toContainText("Syntax/browser-missing.md");
    await expect(page.locator(".edit-impact")).toContainText("Syntax/Missing Link Source.md");
    await page.getByRole("button", { name: "Create note" }).click();

    await expect(page).toHaveURL(`${baseURL}/Syntax/browser-missing.md`);
    await expect(page.locator("article.note > .content")).toContainText("Browser Missing");
    const source = await readFile(path.join(vaultDir, "Syntax", "Missing Link Source.md"), "utf8");
    expect(source).toContain("[[browser-missing|the missing note]]");
  });

  test("moves a note to Trash and restores it to the original path", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Trash%20Me.md`);
    page.once("dialog", async (dialog) => {
      expect(dialog.message()).toContain("Move this note to Trash");
      await dialog.accept();
    });
    await clickHeaderAction(page, "Move to Trash");
    await expect(page).toHaveURL(`${baseURL}/_trash`);
    const card = page.locator("[data-trash-entry]").filter({ hasText: "Syntax/Trash Me.md" });
    await expect(card).toBeVisible();

    await card.getByRole("button", { name: "Restore", exact: true }).click();
    await expect(card).toBeHidden();
    await page.goto(`${baseURL}/Syntax/Trash%20Me.md`);
    await expect(page.locator("article.note > header h1")).toHaveText("Trash Me");
  });

  test("offers Restore as when original path collides", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Restore%20Collision.md`);
    page.once("dialog", async (dialog) => {
      await dialog.accept();
    });
    await clickHeaderAction(page, "Move to Trash");
    await expect(page).toHaveURL(`${baseURL}/_trash`);
    await writeFile(path.join(vaultDir, "Syntax", "Restore Collision.md"), "# Replacement\n\nExisting replacement.\n", "utf8");

    const card = page.locator("[data-trash-entry]").filter({ hasText: "Syntax/Restore Collision.md" });
    await expect(card).toBeVisible();
    page.once("dialog", async (dialog) => {
      expect(dialog.type()).toBe("prompt");
      await dialog.accept("Syntax/restored-as.md");
    });
    await card.getByRole("button", { name: "Restore", exact: true }).click();
    await expect(card).toBeHidden();
    await page.goto(`${baseURL}/Syntax/restored-as.md`);
    await expect(page.locator("article.note > header h1")).toHaveText("Restore Collision");
    await expect(page.locator("article.note .content")).toContainText("Original trashed content");
  });

  test("opens Trash from the command palette action", async ({ page }) => {
    await page.goto(`${baseURL}/Syntax/Edit%20Mode%20Fixture.md`);
    const appNav = page.locator('nav[aria-label="App navigation"]');
    const hrefs = await appNav.locator("a").evaluateAll((links) => links.map((link) => link.getAttribute("href")));
    expect(hrefs).toEqual(["/", "/_inbox", "/_todo", "/_projects", "/_calendar", "/_search", "/_tags", "/_maintenance", "/_trash"]);
    await page.getByLabel("Open command palette").click();
    await page.locator("[data-palette-input]").fill("Open Trash");
    const action = page.locator("[data-palette-index]").filter({ hasText: "Open Trash" }).first();
    await expect(action).toBeVisible();
    await action.click();
    await expect(page).toHaveURL(`${baseURL}/_trash`);
  });

  test("captures from Home and lists the Inbox item", async ({ page }) => {
    const title = `UI Capture ${Date.now()}`;
    const href = await createInboxCapture(page, baseURL, title, "Remember this from the browser test.");

    await page.goto(`${baseURL}/_inbox`);
    await expect(page.locator('nav[aria-label="App navigation"] a[href="/_inbox"]')).toHaveAttribute("aria-current", "page");
    const card = page.locator("[data-inbox-entry]").filter({ hasText: title });
    await expect(card).toBeVisible();
    await expect(card.locator("h2")).toContainText(title);
    await expect(card.getByRole("link", { name: /open/i }).first()).toHaveAttribute("href", href);
  });

  test("archives an Inbox capture from the list", async ({ page }) => {
    const title = `Archive Capture ${Date.now()}`;
    await createInboxCapture(page, baseURL, title);
    await page.goto(`${baseURL}/_inbox`);
    const card = page.locator("[data-inbox-entry]").filter({ hasText: title });
    await expect(card).toBeVisible();

    await card.getByRole("button", { name: "Archive" }).click();
    await expect(card).toBeHidden();
    await expect(page.locator("[data-inbox-message]")).toContainText("Archived");
  });

  test("moves an Inbox capture with missing-folder confirmation", async ({ page }) => {
    const title = `Move Capture ${Date.now()}`;
    const targetPath = `Projects/Captured/${title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "")}.md`;
    await createInboxCapture(page, baseURL, title);
    await page.goto(`${baseURL}/_inbox`);
    const card = page.locator("[data-inbox-entry]").filter({ hasText: title });
    await expect(card).toBeVisible();

    await card.getByRole("button", { name: "Move…" }).click();
    const modal = page.locator(".edit-modal-panel");
    await expect(modal).toBeVisible();
    await modal.locator('input[name="inbox-move-target"]').fill(targetPath);
    await modal.getByRole("button", { name: "Move capture" }).click();
    await expect(modal.locator("[data-edit-modal-message]")).toContainText("Confirmation needed");
    await modal.getByRole("button", { name: "Continue" }).click();

    await expect(page).toHaveURL(`${baseURL}/${targetPath}`);
    await expect(page.locator("article.note .content")).toContainText(title);
  });

  test("converts an Inbox capture to a task and archives it", async ({ page }) => {
    const title = `Convert Capture ${Date.now()}`;
    await createInboxCapture(page, baseURL, title);
    await page.goto(`${baseURL}/_inbox`);
    const card = page.locator("[data-inbox-entry]").filter({ hasText: title });
    await expect(card).toBeVisible();

    await card.getByRole("button", { name: "Convert to task" }).click();
    await expect(card).toBeHidden();
    await expect(page.locator("[data-inbox-message]")).toContainText("Task added");

    const tasks = await readFile(path.join(vaultDir, "Tasks", "Inbox.md"), "utf8");
    expect(tasks).toContain(`- [ ] ${title} 📥 [[Inbox/Archive/`);
  });

  test("convert warning keeps Inbox capture actions usable", async ({ page }) => {
    const title = `Warning Capture ${Date.now()}`;
    await createInboxCapture(page, baseURL, title);
    await page.goto(`${baseURL}/_inbox`);
    const card = page.locator("[data-inbox-entry]").filter({ hasText: title });
    await expect(card).toBeVisible();

    await page.route("**/_api/edit/inbox/convert-task", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ code: "task_written_archive_failed", message: "Task was written, but archive failed.", task_file: "Tasks/Inbox.md" }),
      });
    });
    await card.getByRole("button", { name: "Convert to task" }).click();
    await expect(page.locator("[data-inbox-message]")).toContainText("Task written with warning");
    await expect(card.getByRole("button", { name: "Archive" })).toBeEnabled();
    await expect(card.getByRole("button", { name: "Move…" })).toBeEnabled();
    await expect(card.getByRole("button", { name: "Convert to task" })).toBeEnabled();
  });

  test("mobile Inbox keeps capture actions usable without overflow", async ({ page }) => {
    const title = `Mobile Capture ${Date.now()}`;
    await createInboxCapture(page, baseURL, title);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${baseURL}/_inbox`);

    const metrics = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
    }));
    expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
    const card = page.locator("[data-inbox-entry]").filter({ hasText: title });
    await expect(card.getByRole("link", { name: "Open" })).toBeVisible();
    await expect(card.getByRole("button", { name: "Archive" })).toBeVisible();
    await expect(card.getByRole("button", { name: "Move…" })).toBeVisible();
    await expect(card.getByRole("button", { name: "Convert to task" })).toBeVisible();
  });

  test("narrow desktop Inbox collapses cards before tri-pane gets cramped", async ({ page }) => {
    const title = `Narrow Capture ${Date.now()}`;
    await createInboxCapture(page, baseURL, title);
    await page.setViewportSize({ width: 1121, height: 820 });
    await page.goto(`${baseURL}/_inbox`);
    const card = page.locator("[data-inbox-entry]").filter({ hasText: title });
    await expect(card).toBeVisible();
    const metrics = await card.evaluate((el) => {
      const cardStyle = getComputedStyle(el as HTMLElement);
      const actions = (el as HTMLElement).querySelector<HTMLElement>(".inbox-actions");
      return {
        columns: cardStyle.gridTemplateColumns.split(" ").length,
        actionsLeft: actions?.getBoundingClientRect().left ?? 0,
        cardLeft: (el as HTMLElement).getBoundingClientRect().left,
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
      };
    });
    expect(metrics.columns).toBe(1);
    expect(Math.abs(metrics.actionsLeft - metrics.cardLeft)).toBeLessThanOrEqual(24);
    expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 2);
  });
});
