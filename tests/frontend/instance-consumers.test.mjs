import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";
import { seedInstanceInfo } from "./helpers/instance-info.mjs";

registerWebModuleHooks();
globalThis.instanceConsumersReact = React;
const stubs = {
  "@/hooks/oauth": `export const useOAuthConfig = () => ({isOAuthEnabled:false,oAuthOptions:[]});`,
  "@plane/ui": `const React = globalThis.instanceConsumersReact;
    export const OAuthOptions = () => null;
    export const Loader = Object.assign(({children}) => React.createElement("div", {role:"status"}, children), {Item:()=>null});`,
  "../terms-and-conditions": `export const TermsAndConditions = () => null;`,
  "./auth-header": `export const AuthHeader = () => null; export const AuthHeaderBase = ({header}) => header;`,
  "./auth-banner": `export const AuthBanner = ({handleBannerData}) => globalThis.instanceConsumersReact.createElement("button", {onClick:()=>handleBannerData(undefined)}, "Dismiss error");`,
  "./form-root": `const React = globalThis.instanceConsumersReact;
    export const AuthFormRoot = ({authMode,authStep,email,setAuthMode,setAuthStep,setEmail}) => React.createElement("div", null,
      React.createElement("output", null, [authMode,authStep,email].join("|")),
      React.createElement("button", {onClick:()=>{setAuthMode("SIGN_IN");setAuthStep("EMAIL");setEmail("edited@example.test");}}, "Change flow"));`,
  "@/components/global/product-updates/fallback": `export const ProductUpdatesFallback = ({variant}) => "Updates unavailable: " + variant;`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { AuthRoot } = await import("../../apps/web/core/components/account/auth-forms/auth-root.tsx");
const { ProductUpdatesChangelog } = await import("../../apps/web/core/components/global/product-updates/changelog.tsx");

await test("configuration updates retain authentication progress; a new callback URL starts the matching flow", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/sign-in/?email=first%40example.test&error_code=5035");
  seedInstanceInfo(fixture.client, { is_email_password_enabled: true });
  await fixture.render(React.createElement(AuthRoot, { authMode: "SIGN_IN" }));
  const { container } = fixture.environment;
  assert.equal(container.querySelector("output").textContent, "SIGN_UP|PASSWORD|first@example.test");
  await act(async () =>
    [...container.querySelectorAll("button")].find((button) => button.textContent === "Dismiss error").click()
  );
  await act(async () =>
    [...container.querySelectorAll("button")].find((button) => button.textContent === "Change flow").click()
  );
  await act(async () =>
    seedInstanceInfo(fixture.client, { is_email_password_enabled: true, is_magic_login_enabled: true })
  );
  await settleQuery();
  assert.equal(container.querySelector("output").textContent, "SIGN_IN|EMAIL|edited@example.test");
  assert.equal(
    [...container.querySelectorAll("button")].some((button) => button.textContent === "Dismiss error"),
    false
  );
  await act(async () =>
    fixture.router.navigate({ to: "/sign-in/", search: { email: "second@example.test", error_code: 5090 } })
  );
  await settleQuery();
  assert.equal(container.querySelector("output").textContent, "SIGN_IN|UNIQUE_CODE|second@example.test");
});

await test("authentication uses the query configuration when methods are disabled or enabled", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/sign-in/");
  seedInstanceInfo(fixture.client, { is_email_password_enabled: false, is_magic_login_enabled: false });
  await fixture.render(React.createElement(AuthRoot, { authMode: "SIGN_IN" }));
  assert.match(fixture.environment.container.textContent, /No authentication methods available/);
  await act(async () => seedInstanceInfo(fixture.client, { is_email_password_enabled: true }));
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector("output").textContent, "SIGN_IN|EMAIL|");
});

await test("changelog URL changes reset loading and recover after the previous frame times out", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  seedInstanceInfo(fixture.client, { is_self_managed: true });
  await fixture.render(React.createElement(ProductUpdatesChangelog));
  const { container } = fixture.environment;
  assert.match(container.textContent, /Updates unavailable: self-managed/);
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await act(async () => {
    seedInstanceInfo(fixture.client, { instance_changelog_url: "https://example.test/first", is_self_managed: true });
    t.mock.timers.tick(1);
  });
  assert.equal(container.querySelector("iframe").getAttribute("src"), "https://example.test/first");
  assert.ok(container.querySelector('[role="status"]'));
  await act(async () => t.mock.timers.tick(15000));
  assert.equal(container.querySelector("iframe"), null);
  assert.match(container.textContent, /Updates unavailable: self-managed/);
  await act(async () => {
    seedInstanceInfo(fixture.client, { instance_changelog_url: "https://example.test/second", is_self_managed: true });
    t.mock.timers.tick(1);
  });
  assert.equal(container.querySelector("iframe").getAttribute("src"), "https://example.test/second");
  assert.ok(container.querySelector('[role="status"]'));
  await act(async () => container.querySelector("iframe").dispatchEvent(new window.Event("load")));
  assert.equal(container.querySelector('[role="status"]'), null);
  await act(async () => t.mock.timers.tick(15000));
  assert.ok(container.querySelector("iframe"));
  await act(async () => {
    seedInstanceInfo(fixture.client, { is_self_managed: true });
    t.mock.timers.tick(1);
  });
  assert.equal(container.querySelector("iframe"), null);
  assert.match(container.textContent, /Updates unavailable: self-managed/);
});
