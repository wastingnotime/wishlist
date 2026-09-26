import { defineConfig } from "@playwright/test";
import { existsSync } from "node:fs";

const chromium = process.env.WNT_WEB_E2E_CHROMIUM_PATH || "/usr/bin/chromium";

export default defineConfig({
  testDir: "./tests",
  timeout: 20_000,
  use: {
    baseURL: process.env.WNT_WEB_E2E_BASE_URL || "http://127.0.0.1:5173",
    browserName: process.env.WNT_WEB_E2E_BROWSER || "chromium",
    trace: process.env.WNT_WEB_E2E_TRACE || "retain-on-failure",
    launchOptions: existsSync(chromium) ? { executablePath: chromium } : {},
  },
  reporter: "line",
  webServer: [
    { command: "go run ./cmd/api", cwd: "../api", url: "http://127.0.0.1:8080/healthz", reuseExistingServer: !process.env.CI, timeout: 120_000 },
    { command: "npm run dev -- --host 127.0.0.1 --port 5173", url: "http://127.0.0.1:5173", reuseExistingServer: !process.env.CI, timeout: 120_000 },
  ],
});
