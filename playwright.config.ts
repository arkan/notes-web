import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:18081",
    trace: "on-first-retry",
  },
  webServer: {
    command:
      "go run ./cmd/notes-web --vault ./testdata/e2e-vault --host 127.0.0.1 --port 18081",
    port: 18081,
    reuseExistingServer: !process.env.CI,
    cwd: ".",
  },
});
