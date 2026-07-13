import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI === undefined ? 0 : 1,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: "http://127.0.0.1:8080",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "../bin/local-totp serve",
    cwd: "..",
    env: {
      LOCAL_TOTP_DATA_DIR: "./web/test-results/e2e-data",
      LOCAL_TOTP_LISTEN_ADDR: "127.0.0.1:8080",
    },
    url: "http://127.0.0.1:8080/healthz",
    reuseExistingServer: false,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
