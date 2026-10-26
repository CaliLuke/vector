import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    tsconfigPaths: true,
    noExternal: ["prosemirror-codemark"],
    mainFields: ["module", "main"],
  },
  test: {
    maxWorkers: 1,
    fileParallelism: false,
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.{ts,tsx}"],
    experimental: { viteModuleRunner: true },
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary", "html"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.d.ts", "src/core/plugins/file/types.ts", "src/**/*.test.*"],
    },
  },
});
