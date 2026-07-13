import { expect, test } from "@playwright/test";

test("sets up the vault and opens the responsive workbench", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "Create your local vault" })).toBeVisible();
  await page.locator('input[name="password"]').fill("synthetic-password-123");
  await page.locator('input[name="confirmation"]').fill("synthetic-password-123");
  await page.getByRole("button", { name: "Create encrypted vault" }).click();

  await expect(page.getByRole("heading", { name: "Save your recovery key" })).toBeVisible();
  await page.getByRole("checkbox").check();
  await page.getByRole("button", { name: "Open workbench" }).click();

  await expect(page.getByRole("heading", { name: "Local TOTP" })).toBeVisible();
  await expect(page.getByRole("button", { name: /Add credential/ })).toBeVisible();

  await page.setViewportSize({ width: 412, height: 915 });
  await expect(page.getByRole("button", { name: /Add credential/ })).toBeVisible();
});
