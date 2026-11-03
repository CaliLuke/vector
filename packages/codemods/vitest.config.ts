import { createOxcInstrumenter } from "oxc-coverage-instrument/vitest";
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    maxWorkers: 1,
    fileParallelism: false,
    environment: "node",
    coverage: {
      provider: "istanbul",
      instrumenter: (options) =>
        createOxcInstrumenter({ ...options, compat: "istanbul" }),
      reporter: ["text", "json-summary", "html"],
      include: ["*.ts"],
      exclude: ["tests/**", "vitest.config.ts", "**/*.d.ts"],
    },
  },
});
