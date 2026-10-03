import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { chromium } from "@playwright/test";

const phase = process.argv[2];
if (!(["publish", "persistence"].includes(phase))) throw new Error("Expected publish or persistence phase");
if ((process.env.WNT_WEB_E2E_BROWSER ?? "chromium") !== "chromium") throw new Error("Candidate validation currently supports Chromium");

const baseURL = (process.env.WNT_WEB_E2E_BASE_URL ?? "http://127.0.0.1:18083/wishlist").replace(/\/$/, "");
const basePath = new URL(baseURL).pathname.replace(/\/$/, "");
const artifactDir = resolve(process.env.WNT_WEB_E2E_ARTIFACT_DIR ?? "sandboxes/integration/artifacts");
const output = resolve(process.env.WNT_WEB_E2E_OUTPUT ?? `${artifactDir}/browser-e2e.json`);
await mkdir(artifactDir, { recursive: true });
await mkdir(dirname(output), { recursive: true });
if (phase === "publish") {
  const plan = {
    source_revision: process.env.GIT_SHA ?? "local",
    base_url: baseURL,
    semantic_slices: ["admin app and feature publication", "public board selection and reload", "production base-path navigation", "durable feature storage"],
    phases: ["publish", "persistence after API restart"],
  };
  await writeFile(resolve(artifactDir, "validation-plan.json"), `${JSON.stringify(plan, null, 2)}\n`);
}

const chromiumPath = process.env.WNT_WEB_E2E_CHROMIUM_PATH ?? "/usr/bin/chromium";
const browser = await chromium.launch(existsSync(chromiumPath) ? { executablePath: chromiumPath } : {});
const context = await browser.newContext();
const page = await context.newPage();
const consoleErrors = [];
page.on("console", message => { if (message.type() === "error") consoleErrors.push(message.text()); });
page.on("pageerror", error => consoleErrors.push(`pageerror: ${error.message}`));
page.on("requestfailed", request => consoleErrors.push(`requestfailed: ${request.url()} ${request.failure()?.errorText}`));
if (process.env.WNT_WEB_E2E_TRACE !== "off") await context.tracing.start({ screenshots: true, snapshots: true });

let status = "passed";
let errorMessage = "";
let screenshotPath;
let tracePath;
let basePathNavigationValidated = false;
try {
  if (phase === "publish") {
    await page.goto(`${baseURL}/admin`);
    await page.getByLabel("Admin access token").fill("wishlist-candidate-admin");
    await page.getByLabel("Name", { exact: true }).fill("Candidate CI App");
    await page.getByRole("button", { name: "Create app" }).click();
    await page.getByRole("status").filter({ hasText: "App created." }).waitFor();
    await page.getByRole("combobox", { name: "App" }).selectOption({ label: "Candidate CI App" });
    await page.getByLabel("Title", { exact: true }).fill("Candidate feature");
    await page.getByRole("button", { name: "Publish to Voting" }).click();
    await page.getByRole("status").filter({ hasText: "Feature published to Voting." }).waitFor();
  }

  await page.goto(`${baseURL}/?view=voting&app=candidate-ci-app`);
  const internalHrefs = await page.locator('a[href^="/"]').evaluateAll(anchors => anchors.map(anchor => anchor.getAttribute("href")));
  const escapedHrefs = internalHrefs.filter(href => {
    const pathname = new URL(href, baseURL).pathname;
    return pathname !== basePath && !pathname.startsWith(`${basePath}/`);
  });
  assert.deepEqual(escapedHrefs, [], `Internal links escaped base path ${basePath}: ${escapedHrefs.join(", ")}`);
  await page.getByRole("link", { name: "Your data" }).click();
  await page.waitForURL(url => new URL(url).pathname === `${basePath}/privacy`);
  assert.equal(new URL(page.url()).pathname, `${basePath}/privacy`);
  await page.getByRole("heading", { name: "Verify your email" }).waitFor();
  assert.deepEqual(consoleErrors, ["Failed to load resource: the server responded with a status of 401 (Unauthorized)"]);
  consoleErrors.length = 0;
  await page.getByRole("link", { name: "Back to board" }).click();
  await page.waitForURL(url => new URL(url).pathname.replace(/\/$/, "") === basePath);
  assert.equal(new URL(page.url()).pathname.replace(/\/$/, ""), basePath);
  await page.getByRole("link", { name: "Admin", exact: true }).click();
  await page.waitForURL(url => new URL(url).pathname === `${basePath}/admin`);
  assert.equal(new URL(page.url()).pathname, `${basePath}/admin`);
  await page.getByRole("link", { name: "Public Wishlist" }).click();
  await page.waitForURL(url => new URL(url).pathname.replace(/\/$/, "") === basePath);
  assert.equal(new URL(page.url()).pathname.replace(/\/$/, ""), basePath);
  basePathNavigationValidated = true;
  await page.goto(`${baseURL}/?view=voting&app=candidate-ci-app`);
  await page.getByRole("heading", { name: "Candidate CI App" }).waitFor();
  await page.locator('[data-feature-slug="candidate-feature"]').waitFor();
  assert.equal(await page.getByRole("navigation", { name: "Choose an app" }).getByRole("link", { name: "Candidate CI App" }).getAttribute("aria-current"), "page");
  await page.reload();
  await page.locator('[data-feature-slug="candidate-feature"]').waitFor();
  assert.deepEqual(consoleErrors, []);
} catch (error) {
  status = "failed";
  errorMessage = String(error);
  screenshotPath = resolve(artifactDir, `${phase}-failure.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true }).catch(() => {});
} finally {
  if (process.env.WNT_WEB_E2E_TRACE !== "off") {
    if (status === "failed" || process.env.WNT_WEB_E2E_TRACE === "on") {
      tracePath = resolve(artifactDir, `${phase}-trace.zip`);
      await context.tracing.stop({ path: tracePath });
    } else await context.tracing.stop();
  }
  const prior = phase === "persistence" ? JSON.parse(await readFile(output, "utf8")) : { browser: "chromium", base_url: baseURL, results: [] };
  prior.results.push({
    flow: phase === "publish" ? "admin creates an app and publishes a feature through built images" : "published feature survives API restart",
    phase,
    status,
    final_page_url: page.url(),
    outcome_facts: { app_slug: "candidate-ci-app", feature_slug: "candidate-feature", visible_after_reload: status === "passed", base_path_navigation: basePathNavigationValidated, durable_after_restart: phase === "persistence" && status === "passed" },
    console_errors: consoleErrors,
    ...(errorMessage ? { error: errorMessage } : {}),
    ...(screenshotPath ? { screenshot_path: screenshotPath } : {}),
    ...(tracePath ? { trace_path: tracePath } : {}),
  });
  await writeFile(output, `${JSON.stringify(prior, null, 2)}\n`);
  if (phase === "persistence" || status === "failed") {
    const validation = {
      status: prior.results.every(result => result.status === "passed") && phase === "persistence" ? "passed" : "failed",
      source_revision: process.env.GIT_SHA ?? "local",
      semantic_slices: ["admin app and feature publication", "public board selection and reload", "production base-path navigation", "durable feature storage"],
      scenario: { browser_e2e: prior },
    };
    await writeFile(resolve(artifactDir, "validation-result.json"), `${JSON.stringify(validation, null, 2)}\n`);
    await writeFile(resolve(artifactDir, "result.json"), `${JSON.stringify({ status: validation.status, source_revision: validation.source_revision, browser_e2e: output }, null, 2)}\n`);
    await writeFile(resolve(artifactDir, "evidence.md"), `# Wishlist candidate validation\n\nStatus: ${validation.status}\n\nThe built API and web images were exercised through Chromium at ${baseURL}. The browser created an app, published a feature, reloaded the public board, and checked the feature after an API restart. See browser-e2e.json and validation-result.json for details.\n`);
  }
  await browser.close();
}
if (status === "failed") throw new Error(errorMessage);
