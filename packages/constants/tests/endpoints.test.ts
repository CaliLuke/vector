/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { afterEach, describe, expect, it, vi } from "vitest";

async function loadGodModeUrl(baseUrl: string, basePath: string) {
  vi.stubEnv("VITE_ADMIN_BASE_URL", baseUrl);
  vi.stubEnv("VITE_ADMIN_BASE_PATH", basePath);
  vi.resetModules();

  const { GOD_MODE_URL } = await import("../src/endpoints");

  return GOD_MODE_URL;
}

describe("GOD_MODE_URL", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it("adds the trailing slash required by the admin app", async () => {
    expect(await loadGodModeUrl("http://localhost:3001", "/god-mode")).toBe("http://localhost:3001/god-mode/");
  });

  it("does not duplicate an existing trailing slash", async () => {
    expect(await loadGodModeUrl("http://localhost:3001", "/god-mode/")).toBe("http://localhost:3001/god-mode/");
  });

  it("collapses duplicate trailing slashes", async () => {
    expect(await loadGodModeUrl("http://localhost:3001", "/god-mode///")).toBe("http://localhost:3001/god-mode/");
  });

  it("preserves the empty fallback", async () => {
    expect(await loadGodModeUrl("", "")).toBe("");
  });
});
