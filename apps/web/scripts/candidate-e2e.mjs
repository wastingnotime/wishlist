import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { chromium } from "@playwright/test";

const phase = process.argv[2];
if (!(["publish", "persistence"].includes(phase))) throw new Error("Expected publish or persistence phase");
if ((process.env.WNT_WEB_E2E_BROWSER ?? "chromium") !== "chromium") throw new Error("Candidate validation currently supports Chromium");

const baseURL = (process.env.WNT_WEB_E2E_BASE_URL ?? "http://127.0.0.1:18083/wishlist").replace(/\/$/, "");
const artifactDir = resolve(process.env.WNT_WEB_E2E_ARTIFACT_DIR ?? "sandboxes/integration/artifacts");
const output = resolve(process.env.WNT_WEB_E2E_OUTPUT ?? `${artifactDir}/browser-e2e.json`);
await mkdir(artifactDir, { recursive: true });
await mkdir(dirname(output), { recursive: true });

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
    outcome_facts: { app_slug: "candidate-ci-app", feature_slug: "candidate-feature", visible_after_reload: status === "passed", durable_after_restart: phase === "persistence" && status === "passed" },
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
      semantic_slices: ["admin app and feature publication", "public board selection and reload", "durable feature storage"],
      scenario: { browser_e2e: prior },
    };
    await writeFile(resolve(artifactDir, "validation-result.json"), `${JSON.stringify(validation, null, 2)}\n`);
    await writeFile(resolve(artifactDir, "evidence.md"), `# Wishlist candidate validation\n\nStatus: ${validation.status}\n\nThe built API and web images were exercised through Chromium at ${baseURL}. The browser created an app, published a feature, reloaded the public board, and checked the feature after an API restart. See browser-e2e.json and validation-result.json for details.\n`);
  }
  await browser.close();
}
if (status === "failed") throw new Error(errorMessage);
