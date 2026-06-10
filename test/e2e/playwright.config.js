const { defineConfig } = require("@playwright/test");

module.exports = defineConfig({
  testDir: "./tests",
  outputDir: process.env.PLAYWRIGHT_OUTPUT_DIR || "test-results",
  timeout: 30000,
  reporter: "list",
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || "http://localhost:8080",
    trace: "retain-on-failure",
  },
});
