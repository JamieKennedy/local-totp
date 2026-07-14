import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

test("direct routes, internal links, metadata, and accessibility", async ({
  page,
  request,
}) => {
  await page.goto("docs/api/");
  await expect(page).toHaveTitle("Use the HTTP API | Local TOTP");
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
    "href",
    "https://jamiekennedy.github.io/local-totp/docs/api/",
  );
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute(
    "content",
    /social-card\.png$/,
  );

  const internalLinks = await page
    .locator('a[href^="/local-totp/"]')
    .evaluateAll((links) => [
      ...new Set(links.map((link) => (link as HTMLAnchorElement).href)),
    ]);
  for (const href of internalLinks) {
    const response = await request.get(href);
    expect(response.ok(), `${href} should resolve`).toBeTruthy();
  }

  const accessibility = await new AxeBuilder({ page }).analyze();
  expect(accessibility.violations).toEqual([]);
});

test("keyboard navigation exposes the skip link", async ({
  page,
  browserName,
}, testInfo) => {
  test.skip(
    browserName === "webkit" || testInfo.project.name === "mobile",
    "Windows WebKit and touch emulation do not expose desktop Tab focus consistently",
  );
  await page.goto("./");
  await page.keyboard.press("Tab");
  const skipLink = page.getByRole("link", { name: "Skip to content" });
  await expect(skipLink).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.locator("#main-content")).toBeFocused();
});

test("mobile navigation and responsive tables stay within the viewport", async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("docs/cli/");
  await page
    .locator('astro-island[component-export="MobileNavigation"][ssr]')
    .waitFor({ state: "detached" });
  await page.getByRole("button", { name: "Open navigation" }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await expect(page.getByRole("link", { name: "API" })).toBeVisible();

  await page.keyboard.press("Escape");
  const bodyWidth = await page
    .locator("body")
    .evaluate((body) => body.scrollWidth);
  expect(bodyWidth).toBeLessThanOrEqual(390);
  for (const table of await page.locator(".doc-table").all()) {
    await expect(table).toBeVisible();
    const box = await table.boundingBox();
    expect(box?.width ?? 391).toBeLessThanOrEqual(390);
  }
});

test("unknown routes use the noindex 404 page", async ({ page }) => {
  const response = await page.goto("missing-release-page/");
  expect(response?.status()).toBe(404);
  await expect(
    page.getByRole("heading", { name: "That page is outside the vault." }),
  ).toBeVisible();
  await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
    "content",
    "noindex,nofollow",
  );
});
