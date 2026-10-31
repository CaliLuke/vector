import { defineConfig } from "vitest/config";
import path from "path";

export default defineConfig({
  test: {
    maxWorkers: 1,
    fileParallelism: false,
    environment: "node",
    env: {
      API_BASE_URL: "http://api.test",
      LIVE_SERVER_SECRET_KEY: "live-test-secret",
    },
    globals: true,
    include: ["tests/**/*.test.ts", "tests/**/*.spec.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary", "html"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.d.ts", "src/**/types.ts", "src/**/*.test.*"],
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
});
