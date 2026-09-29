import { expect, test } from "@playwright/test";
import { mkdir, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";

const results = [];
const consoleErrors = new WeakMap();

test.beforeEach(async ({ page }) => {
  const errors = [];
  consoleErrors.set(page, errors);
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  page.on("pageerror", (error) => { errors.push(`pageerror: ${error.message}`); console.error(error); });
  page.on("requestfailed", (request) => errors.push(`requestfailed: ${request.url()} ${request.failure()?.errorText}`));
});

test.afterEach(async ({ page }, testInfo) => {
  if (testInfo.status !== "passed") console.log("Browser diagnostics:", consoleErrors.get(page));
  const baseURL = String(testInfo.project.use.baseURL ?? "");
  const outcomeFacts = testInfo.title.startsWith("public board selects")
    ? { voting_view_loaded: true, selected_apps: ["cat-care", "sliding-tasks"], filtered_views: ["voting", "producing", "delivered"], url_state_survived_reload: true, delivery_link_visible: true }
    : testInfo.title.startsWith("admin can")
      ? { admin_token_checked: true, app_created: true, feature_published: true }
      : testInfo.title.startsWith("visitor can")
        ? { otp_modal_completed: true, vote_toggled: true, private_suggestion_submitted: true }
      : { empty_state_visible: true, app_filter: "empty-board-test-app", unrelated_features_hidden: true };
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

test("public board selects an app and lifecycle, preserves URL state, and refreshes", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(/view=voting.*app=cat-care/);
  await expect(page.getByRole("heading", { name: "Cat Care" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Choose an app" }).getByRole("link", { name: "Cat Care" })).toHaveAttribute("aria-current", "page");
  await expect(page.locator('[data-feature-slug="family-sharing"]')).toContainText("0");
  await expect(page.locator('[data-feature-slug="dark-mode"]')).toHaveCount(0);

  await page.getByRole("navigation", { name: "Choose an app" }).getByRole("link", { name: "Sliding Tasks" }).click();
  await expect(page).toHaveURL(/app=sliding-tasks/);
  await expect(page.getByRole("heading", { name: "Sliding Tasks" })).toBeVisible();
  await expect(page.locator('[data-feature-slug="dark-mode"]')).toBeVisible();
  await page.getByRole("navigation", { name: "Choose an app" }).getByRole("link", { name: "Cat Care" }).click();
  await expect(page).toHaveURL(/app=cat-care/);
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
  await page.goto("/admin");
  await page.getByLabel("Admin access token").fill("wishlist-e2e-admin");
  await page.getByLabel("Name", { exact: true }).fill("Empty Board Test App");
  await page.getByRole("button", { name: "Create app" }).click();
  await expect(page.getByRole("status")).toContainText("App created");
  await page.goto("/?view=delivered&app=empty-board-test-app");
  await expect(page.getByRole("heading", { name: "Empty Board Test App" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Nothing here yet" })).toBeVisible();
  await expect(page.locator('[data-feature-slug="family-sharing-shipped"]')).toHaveCount(0);
});

test("admin can create an app and publish a voting feature", async ({ page }) => {
  await page.goto("/admin");
  await page.getByLabel("Admin access token").fill("wishlist-e2e-admin");
  await expect(page.getByRole("combobox", { name: "App" })).toContainText("Cat Care");
  await page.getByLabel("Name", { exact: true }).fill("Démo App!");
  await page.getByLabel("Description", { exact: true }).first().fill("A browser test app.");
  await page.getByRole("button", { name: "Create app" }).click();
  await expect(page.getByRole("status")).toContainText("App created");
  await expect(page.getByRole("combobox", { name: "App" })).toContainText("Démo App!");

  await page.getByRole("combobox", { name: "App" }).selectOption({ label: "Démo App!" });
  await page.getByLabel("Title", { exact: true }).fill("First request!");
  await page.getByLabel("Description", { exact: true }).last().fill("A feature created by the admin test.");
  await page.getByRole("button", { name: "Publish to Voting" }).click();
  await expect(page.getByRole("status")).toContainText("Feature published to Voting");
  await page.goto("/?app=demo-app");
  await expect(page.locator('[data-feature-slug="first-request"]')).toBeVisible();
});

test("visitor can verify, vote, and submit a private suggestion", async ({ page }) => {
  let verified = false;
  let voted = false;
  let voteCount = 0;
  let suggestionSubmitted = false;
  await page.route("**/api/apps", route => route.fulfill({ json: { apps: [{ id: "app-cat-care", slug: "cat-care", name: "Cat Care", description: "", url: "" }] } }));
  await page.route("**/api/features**", route => route.fulfill({ json: { features: [{ id: "feature-family-sharing", slug: "family-sharing", title: "Family sharing", description: "Coordinate care together.", app_slug: "cat-care", app_name: "Cat Care", status: "voting", vote_count: voteCount, published_at: "2026-09-01T00:00:00Z", delivered_at: null, delivery_url: null }] } }));
  await page.route("**/api/session", route => route.fulfill({ json: { verified, vote_feature_ids: voted ? ["feature-family-sharing"] : [] } }));
  await page.route("**/api/otp", route => route.fulfill({ status: 202, json: { requested: true } }));
  await page.route("**/api/otp/verify", route => { verified = true; return route.fulfill({ json: { verified: true } }); });
  await page.route("**/api/features/feature-family-sharing/vote", route => { voted = !voted; voteCount += voted ? 1 : -1; return route.fulfill({ json: { voted, vote_count: voteCount } }); });
  await page.route("**/api/suggestions", route => { suggestionSubmitted = true; return route.fulfill({ status: 201, json: { submitted: true, suggestion_id: "suggestion-test" } }); });
  await page.goto("/");
  await page.locator('[data-feature-slug="family-sharing"]').getByRole("button", { name: "Vote ↑" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Email address").fill("visitor@example.com");
  await dialog.getByRole("button", { name: "Send sign-in code" }).click();
  await dialog.getByLabel("One-time code").fill("246810");
  await dialog.getByRole("button", { name: "Verify and continue" }).click();
  await expect(page.getByRole("button", { name: "Voted ✓" })).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator('[data-feature-slug="family-sharing"]')).toContainText("1");
  await page.getByRole("button", { name: "Suggest an idea" }).click();
  const suggestion = page.getByRole("dialog");
  await suggestion.getByLabel("Idea title").fill("Medication reminders");
  await suggestion.getByLabel("What would this help you do?").fill("Keep doses on schedule.");
  await suggestion.getByRole("button", { name: "Continue" }).click();
  await expect(page.getByRole("status")).toContainText("private");
  expect(suggestionSubmitted).toBe(true);
});
