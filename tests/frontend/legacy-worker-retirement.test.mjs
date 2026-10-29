import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import test from "node:test";
const source = readFileSync(new URL("../../apps/web/public/sw.js", import.meta.url), "utf8");
function browser({ cacheDenied = false, claimDenied = false } = {}) {
  const handlers = new Map();
  const caches = new Map([
    ["start-url", "stale page"],
    ["unrelated", "keep this"],
  ]);
  const state = { registered: true, controller: "legacy", waiting: true };
  const scope = {
    addEventListener(name, handler) {
      handlers.set(name, handler);
    },
    async skipWaiting() {
      state.waiting = false;
    },
    clients: {
      async claim() {
        if (claimDenied) throw new Error("claim denied");
        state.controller = "retirement";
      },
    },
    registration: {
      async unregister() {
        const existed = state.registered;
        state.registered = false;
        return existed;
      },
    },
  };
  runInNewContext(source, {
    self: scope,
    caches: {
      async delete(name) {
        if (cacheDenied) throw new Error("cache unavailable");
        return caches.delete(name);
      },
    },
  });
  return {
    state,
    caches,
    async dispatch(name) {
      const completion = [];
      handlers.get(name)?.({
        waitUntil(promise) {
          completion.push(promise);
        },
      });
      await Promise.all(completion);
    },
    async request() {
      let response = Promise.resolve(new Response("fresh page"));
      handlers.get("fetch")?.({
        request: new Request("http://localhost/"),
        respondWith(value) {
          response = Promise.resolve(value);
        },
      });
      return (await response).text();
    },
  };
}
void test("retirement takes over open clients, removes only its cache and unregisters", async () => {
  const app = browser();
  await app.dispatch("install");
  assert.equal(app.state.waiting, false);
  await app.dispatch("activate");
  assert.equal(app.state.controller, "retirement");
  assert.equal(app.state.registered, false);
  assert.deepEqual([...app.caches], [["unrelated", "keep this"]]);
  assert.equal(await app.request(), "fresh page");
});
void test("repeated retirement is safe when the old cache and registration are already gone", async () => {
  const app = browser();
  await app.dispatch("activate");
  await app.dispatch("activate");
  assert.equal(app.state.registered, false);
  assert.deepEqual([...app.caches], [["unrelated", "keep this"]]);
  assert.equal(await app.request(), "fresh page");
});
void test("denied cache access does not retain registration or intercept requests", async () => {
  const app = browser({ cacheDenied: true });
  await app.dispatch("activate");
  assert.equal(app.state.registered, false);
  assert.equal(app.state.controller, "retirement");
  assert.equal(app.caches.get("unrelated"), "keep this");
  assert.equal(await app.request(), "fresh page");
});
void test("failed client claim still removes the old cache and unregisters", async () => {
  const app = browser({ claimDenied: true });
  await app.dispatch("activate");
  assert.equal(app.state.registered, false);
  assert.equal(app.caches.has("start-url"), false);
  assert.equal(app.caches.get("unrelated"), "keep this");
});
