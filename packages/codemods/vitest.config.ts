import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    maxWorkers: 1,
    fileParallelism: false,
    environment: "node",
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary", "html"],
      include: ["*.ts"],
      exclude: ["tests/**", "vitest.config.ts", "**/*.d.ts"],
    },
  },
});
