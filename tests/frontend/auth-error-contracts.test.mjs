import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const web = await import("../../apps/web/helpers/authentication.helper.tsx");
const space = await import("../../apps/space/helpers/authentication.helper.tsx");
const { authErrorHandler: shared, createAuthErrorHandler } = await import("../../packages/utils/dist/auth.js");
const { MemoryRouter } = await import("react-router");
const codes = Object.values(web.EAuthenticationErrorCodes);
const webOnly = new Set(["5016", "5018", "5021", "5138", "5190", "5900"]);
const email = "name+tag&next=//example.test@example.test";

for (const { app, helper } of [
  { app: "web", helper: web },
  { app: "space", helper: space },
])
  for (const suppliedEmail of [undefined, email]) {
    void test(`${app} renders supported authentication failures with safe ${suppliedEmail ? "email" : "default"} links`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      const errors = codes.map((code) => helper.authErrorHandler(code, suppliedEmail)).filter(Boolean);
      assert.equal(errors.length, app === "web" ? 52 : 46);
      const content = React.createElement(
        "div",
        null,
        errors.map((error) =>
          React.createElement(
            "article",
            { key: error.code, "data-code": error.code },
            React.createElement("h2", null, error.title),
            error.message
          )
        )
      );
      if (app === "web") await createRouterFixture("/").render(env, content);
      else await env.render(React.createElement(MemoryRouter, null, content));
      const existing = env.container.querySelector('[data-code="5030"] a');
      const missing = env.container.querySelector('[data-code="5060"] a');
      for (const anchor of [existing, missing]) {
        assert.ok(anchor);
        const url = new URL(anchor.getAttribute("href"), "http://localhost");
        assert.equal(url.origin, "http://localhost");
        assert.equal(url.searchParams.get("email"), suppliedEmail ?? null);
        assert.equal(url.searchParams.has("next"), false);
      }
      assert.equal(new URL(existing.href).pathname, app === "web" ? "/" : "/sign-in");
      assert.equal(new URL(missing.href).pathname, "/");
      assert.match(env.container.querySelector('[data-code="5180"]').textContent, /Admin user already exists/);
      assert.equal(
        new URL(env.container.querySelector('[data-code="5180"] a').href).pathname.replace(/\/$/, ""),
        "/admin"
      );
      const invalidToken = env.container.querySelector('[data-code="5125"]').textContent;
      if (app === "web") assert.doesNotMatch(invalidToken, /Please try again/);
      else assert.match(invalidToken, /Please try again/);
      assert.match(env.container.querySelector('[data-code="5017"]').textContent, /personal account/);
    });
  }
void test("authentication error resolution preserves each consumer's supported protocol codes", () => {
  for (const code of codes) {
    assert.equal(web.authErrorHandler(code)?.code, code);
    assert.equal(space.authErrorHandler(code)?.code, webOnly.has(code) ? undefined : code);
    assert.equal(shared(code)?.code, webOnly.has(code) ? undefined : code);
  }
  for (const invalid of [undefined, null, 5000, {}, "unknown", "toString", "__proto__"]) {
    assert.equal(web.authErrorHandler(invalid), undefined);
    assert.equal(space.authErrorHandler(invalid), undefined);
    assert.equal(shared(invalid), undefined);
  }
  assert.equal(web.authErrorHandler("5900").title, "Error");
  const fallback = createAuthErrorHandler({ 5000: { title: "", message: () => "" } });
  assert.equal(fallback("5000").message, "Something went wrong. Please try again.");
});
