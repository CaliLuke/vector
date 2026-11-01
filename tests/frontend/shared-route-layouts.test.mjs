import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebTestModules, reactModuleUrl } from "./helpers/web-test-modules.mjs";
const react = `import React from ${JSON.stringify(reactModuleUrl)};`;
registerWebTestModules({
  "@/components/core/app-header":
    react + `export const AppHeader=({header,mobileHeader})=>React.createElement('header',null,header,mobileHeader);`,
  "@/components/core/content-wrapper":
    react + `export const ContentWrapper=({children})=>React.createElement('main',null,children);`,
  "@/components/projects/mobile-header":
    react + `export const ProjectsListMobileHeader=()=>React.createElement('nav',{'aria-label':'Projects on mobile'});`,
  "@/components/project/header":
    react + `export const ProjectsBaseHeader=()=>React.createElement('h1',null,'Projects');`,
  "@/layouts/default-layout": `export default ({children})=>children;`,
  "@/lib/wrappers/authentication-wrapper":
    react +
    `export const AuthenticationWrapper=({children,pageType})=>React.createElement('section',{'data-auth-policy':pageType},children);`,
  "@/components/auth-screens/header":
    react + `export const AuthHeader=({type})=>React.createElement('header',{'data-auth-mode':type});`,
  "@/components/account/auth-forms/set-password":
    react + `export const SetPasswordForm=()=>React.createElement('form',{'aria-label':'Password setup'});`,
  "@/components/account/auth-forms/reset-password":
    react + `export const ResetPasswordForm=()=>React.createElement('form',{'aria-label':'Password recovery'});`,
});
const list = (await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(list)/layout.tsx")).Route
  .options.component;
const archives = (
  await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/archives/layout.tsx")
).Route.options.component;
for (const { path, Component } of [
  { path: "/acme/projects/", Component: list },
  { path: "/acme/projects/archives/", Component: archives },
]) {
  void test(`${path} retains the shared project heading, mobile navigation and content outlet`, async (t) => {
    assert.equal(list, archives, "both routes use the same layout owner");
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    await createRouterFixture(path).render(env, React.createElement(Component));
    assert.equal(env.container.querySelector("h1").textContent, "Projects");
    assert.ok(env.container.querySelector('nav[aria-label="Projects on mobile"]'));
    assert.ok(env.container.querySelector("main"));
  });
}
const { EPageTypes } = await import("../../apps/web/helpers/authentication.helper.tsx");
const { EAuthModes } = await import("../../packages/constants/dist/index.mjs");
const reset = (await import("../../apps/web/app/(all)/accounts/reset-password/page.tsx")).Route.options.component;
const set = (await import("../../apps/web/app/(all)/accounts/set-password/page.tsx")).Route.options.component;
for (const { path, Component, policy } of [
  { path: "/accounts/reset-password/", Component: reset, policy: EPageTypes.NON_AUTHENTICATED },
  { path: "/accounts/set-password/", Component: set, policy: EPageTypes.SET_PASSWORD },
]) {
  void test(`${path} retains its own authentication policy inside the shared password screen`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    await createRouterFixture(path).render(env, React.createElement(Component));
    assert.equal(env.container.querySelector("[data-auth-policy]").dataset.authPolicy, policy);
    assert.equal(env.container.querySelector("[data-auth-mode]").dataset.authMode, EAuthModes.SIGN_IN);
    assert.equal(env.container.querySelectorAll("form").length, 1);
    assert.equal(
      env.container.querySelector("form").getAttribute("aria-label"),
      policy === EPageTypes.SET_PASSWORD ? "Password setup" : "Password recovery"
    );
  });
}
const { createAppMetadata } = await import("../../packages/utils/src/metadata.ts");
void test("application metadata retains each entry point's identity and search keywords", () => {
  const metadata = createAppMetadata("Vector Publish & boards", "Public project boards", "work item tracking");
  const tags = new Map(metadata.map((tag) => [tag.property ?? tag.name ?? "title", tag.content ?? tag.title]));
  assert.equal(tags.get("title"), "Vector Publish & boards");
  assert.equal(tags.get("og:title"), tags.get("title"));
  assert.equal(tags.get("description"), "Public project boards");
  assert.equal(tags.get("og:description"), tags.get("description"));
  assert.equal(tags.get("keywords"), "work item tracking");
  assert.equal(tags.get("og:url"), "https://github.com/CaliLuke/vector");
  assert.equal(tags.get("twitter:site"), "Vector");
  assert.equal(tags.size, 7, "each metadata field is emitted once");
});

const { AuthHeader: AdminAuthHeader } = await import("../../apps/admin/app/(all)/(home)/auth-header.tsx");
const { AuthHeader: PublishAuthHeader } = await import("../../apps/space/components/views/header.tsx");
const { MemoryRouter } = await import("react-router");
for (const { label, Header } of [
  { label: "Administration", Header: AdminAuthHeader },
  { label: "Publish", Header: PublishAuthHeader },
]) {
  void test(`${label} branding keeps the native home link and lockup dimensions`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    await env.render(React.createElement(MemoryRouter, null, React.createElement(Header)));
    const link = env.container.querySelector("a");
    assert.equal(link.getAttribute("href"), "/");
    assert.equal(env.container.querySelectorAll("a").length, 1);
    const lockup = link.querySelector("svg");
    assert.ok(lockup);
    assert.equal(lockup.getAttribute("height"), "20");
    assert.equal(lockup.getAttribute("width"), "95");
  });
}
