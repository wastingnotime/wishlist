import { defineConfig } from "@playwright/test";
import { existsSync } from "node:fs";

const chromium = process.env.WNT_WEB_E2E_CHROMIUM_PATH || "/usr/bin/chromium";
const apiPort = process.env.WNT_WEB_E2E_API_PORT || "8080";
const webPort = process.env.WNT_WEB_E2E_WEB_PORT || "5173";

export default defineConfig({
  testDir: "./tests",
  timeout: 20_000,
  use: {
    baseURL: process.env.WNT_WEB_E2E_BASE_URL || `http://127.0.0.1:${webPort}`,
    browserName: process.env.WNT_WEB_E2E_BROWSER || "chromium",
    trace: process.env.WNT_WEB_E2E_TRACE || "retain-on-failure",
    launchOptions: existsSync(chromium) ? { executablePath: chromium } : {},
  },
  reporter: "line",
  webServer: [
    { command: "WISHLIST_ADMIN_TOKEN=wishlist-e2e-admin go run ./cmd/api", cwd: "../api", url: `http://127.0.0.1:${apiPort}/healthz`, reuseExistingServer: false, timeout: 120_000 },
    { command: `npm run dev -- --host 127.0.0.1 --port ${webPort}`, url: `http://127.0.0.1:${webPort}`, reuseExistingServer: false, timeout: 120_000 },
  ],
});
