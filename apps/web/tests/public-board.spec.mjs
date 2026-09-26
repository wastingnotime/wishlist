import { expect, test } from "@playwright/test";
import { mkdir, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";

const results = [];
const consoleErrors = new WeakMap();

test.beforeEach(async ({ page }) => {
  const errors = [];
  consoleErrors.set(page, errors);
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
});

test.afterEach(async ({ page }, testInfo) => {
  const baseURL = String(testInfo.project.use.baseURL ?? "");
  const outcomeFacts = testInfo.title.startsWith("public board filters")
    ? { voting_view_loaded: true, app_filter: "cat-care", filtered_views: ["voting", "producing", "delivered"], url_state_survived_reload: true, delivery_link_visible: true }
    : { empty_state_visible: true, app_filter: "cat-care-missing", unrelated_features_hidden: true };
  results.push({
    browser: testInfo.project.use.browserName ?? "chromium",
    flow: testInfo.title,
    base_url: baseURL,
    final_page_url: page.url(),
    status: testInfo.status,
    outcome_facts: { ...outcomeFacts, test_passed: testInfo.status === "passed" },
    console_errors: consoleErrors.get(page) ?? [],
  });
  if (testInfo.status !== "passed" && process.env.WNT_WEB_E2E_ARTIFACT_DIR) {
    const screenshot = resolve(process.env.WNT_WEB_E2E_ARTIFACT_DIR, "browser-failure.png");
    await mkdir(dirname(screenshot), { recursive: true });
    await page.screenshot({ path: screenshot, fullPage: true });
    results.at(-1).screenshot_path = screenshot;
  }
});

test.afterAll(async () => {
  if (!process.env.WNT_WEB_E2E_OUTPUT) return;
  const output = resolve(process.env.WNT_WEB_E2E_OUTPUT);
  await mkdir(dirname(output), { recursive: true });
  await writeFile(output, `${JSON.stringify({ browser: "chromium", results }, null, 2)}\n`);
});

test("public board filters app and lifecycle, preserves URL state, and refreshes", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Community requests" })).toBeVisible();
  await expect(page.locator('[data-feature-slug="family-sharing"]')).toContainText("184");
  await expect(page.locator('[data-feature-slug="dark-mode"]')).toBeVisible();

  await page.getByLabel("Filter by app").selectOption("cat-care");
  await expect(page.locator('[data-feature-slug="dark-mode"]')).toHaveCount(0);
  await page.getByRole("link", { name: "Producing" }).click();
  await expect(page).toHaveURL(/view=producing.*app=cat-care|app=cat-care.*view=producing/);
  await expect(page.locator('[data-feature-slug="shared-calendar"]')).toBeVisible();
  await expect(page.locator('[data-feature-slug="batch-edit"]')).toHaveCount(0);

  await page.reload();
  await expect(page.locator('[data-feature-slug="shared-calendar"]')).toBeVisible();
  await page.getByRole("link", { name: "Delivered" }).click();
  await expect(page.getByRole("link", { name: "View release" })).toHaveAttribute("href", /releases\/cat-care/);
  await expect(page.locator('[data-feature-slug="quick-add-shipped"]')).toHaveCount(0);
  await page.getByRole("button", { name: "Refresh wishlist" }).click();
  await expect(page.locator('[data-feature-slug="family-sharing-shipped"]')).toBeVisible();
});

test("valid app without features shows an empty state", async ({ page }) => {
  await page.goto("/?view=delivered&app=cat-care-missing");
  await expect(page.getByRole("heading", { name: "Nothing here yet" })).toBeVisible();
  await expect(page.locator('[data-feature-slug="family-sharing-shipped"]')).toHaveCount(0);
});
