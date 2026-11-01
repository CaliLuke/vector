import assert from "node:assert/strict";
import test from "node:test";
import { getFrontendViteConfig } from "../../scripts/vite-config.mts";

void test("frontend bundles receive only public VITE variables, preserving empty public values", () => {
  const env = {
    VITE_API_BASE_URL: "https://example.test",
    VITE_OPTIONAL: undefined,
    DATABASE_URL: "private-fixture",
    PLANE_DEV_HOST: "0.0.0.0",
  };
  const result = getFrontendViteConfig({ env, portVariable: "PLANE_WEB_PORT", defaultPort: 3000 });
  assert.deepEqual(JSON.parse(result.define["process.env"]), {
    VITE_API_BASE_URL: "https://example.test",
    VITE_OPTIONAL: "",
  });
  assert.equal(result.server.host, "0.0.0.0");
  assert.deepEqual(env, {
    VITE_API_BASE_URL: "https://example.test",
    VITE_OPTIONAL: undefined,
    DATABASE_URL: "private-fixture",
    PLANE_DEV_HOST: "0.0.0.0",
  });
});
void test("development upstreams honor per-app ports and fail rather than selecting another occupied port", () => {
  const web = getFrontendViteConfig({
    env: { PLANE_WEB_PORT: "3060", PLANE_ADMIN_PORT: "3061" },
    portVariable: "PLANE_WEB_PORT",
    defaultPort: 3000,
  });
  const admin = getFrontendViteConfig({
    env: { PLANE_WEB_PORT: "3060", PLANE_ADMIN_PORT: "3061" },
    portVariable: "PLANE_ADMIN_PORT",
    defaultPort: 3001,
  });
  assert.deepEqual(web.server, { host: "127.0.0.1", port: 3060, strictPort: true });
  assert.equal(admin.server.port, 3061);
  assert.equal(
    getFrontendViteConfig({ env: {}, portVariable: "PLANE_SPACE_PORT", defaultPort: 3002 }).server.port,
    3002
  );
  assert.equal(web.build.assetsInlineLimit, 0);
  assert.deepEqual(web.resolve, { tsconfigPaths: true, dedupe: ["react", "react-dom"] });
});
