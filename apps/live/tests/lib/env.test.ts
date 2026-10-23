import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("dotenv", () => ({ config: vi.fn() }));

describe("Live environment parsing", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
    vi.resetModules();
  });

  it("retains defaults and numeric transforms when parsing valid configuration", async () => {
    vi.stubEnv("API_BASE_URL", "https://api.example.test");
    vi.stubEnv("LIVE_SERVER_SECRET_KEY", "test-secret");
    vi.stubEnv("PORT", undefined);
    vi.stubEnv("COMPRESSION_LEVEL", undefined);
    vi.stubEnv("COMPRESSION_THRESHOLD", undefined);
    vi.stubEnv("REDIS_PORT", undefined);

    const { env } = await import("@/env");

    expect(env).toMatchObject({
      PORT: "3000",
      LIVE_BASE_PATH: "/live",
      COMPRESSION_LEVEL: 6,
      COMPRESSION_THRESHOLD: 5000,
      REDIS_PORT: 6379,
    });
  });

  it("logs structured Zod issues before exiting for invalid configuration", async () => {
    vi.stubEnv("API_BASE_URL", "not a URL");
    vi.stubEnv("LIVE_SERVER_SECRET_KEY", "test-secret");
    const logError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    vi.spyOn(process, "exit").mockImplementation((): never => {
      throw new Error("process.exit called");
    });

    await expect(import("@/env")).rejects.toThrow("process.exit called");

    expect(logError).toHaveBeenCalledWith("❌ Invalid environment variables:", expect.stringContaining("API_BASE_URL"));
  });
});
