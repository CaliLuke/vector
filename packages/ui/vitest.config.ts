import { createOxcInstrumenter } from "oxc-coverage-instrument/vitest";
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    maxWorkers: 1,
    fileParallelism: false,
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "istanbul",
      instrumenter: (options) => createOxcInstrumenter({ ...options, compat: "istanbul" }),
      reporter: ["text", "json-summary", "html"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.d.ts", "src/**/types.ts", "src/**/*.test.*", "src/**/*.stories.*"],
    },
  },
});
