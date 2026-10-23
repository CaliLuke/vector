import assert from "node:assert/strict";
import test from "node:test";
import { createRouterFixture } from "./helpers/router.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const cases = [
  ["/sign-in", "sign-in", "/"],
  ["/signin", "signin", "/"],
  ["/login", "login", "/"],
  ["/register", "register", "/sign-up"],
  ["/accounts/sign-up", "accounts-signup", "/sign-up"],
  ["/acme/analytics", "analytics", "/acme/analytics/overview"],
  ["/acme/settings/api-tokens", "api-tokens", "/settings/profile/api-tokens"],
  ["/acme/projects/p1/inbox", "inbox", "/acme/projects/p1/intake"],
  [
    "/acme/projects/p1/settings/members?invite=1#people",
    "project-settings",
    "/acme/settings/projects/p1/members?invite=1#people",
  ],
  ["/profile/notifications?source=menu", "profile-settings", "/settings/profile/notifications?source=menu"],
  ["/acme/settings/account/?source=menu", "workspace-account-settings", "/settings/profile/general?source=menu"],
];

for (const [initial, file, expected] of cases) {
  void test(`legacy URL ${initial} redirects to its current destination`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const { Route } = await import(`../../apps/web/app/routes/redirects/core/${file}.tsx`);
    const { router } = createRouterFixture(initial, { modules: { [`routes/redirects/core/${file}.tsx`]: Route } });
    await router.load();
    assert.equal(router.state.location.href.replace(/\/$/, ""), expected.replace(/\/$/, ""));
    assert.equal(router.state.matches.at(-1).status, "success");
  });
}

await test("nested project navigation retains workspace and project params across history", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { router } = createRouterFixture("/acme/projects/p1/modules/m1?peekModule=m2#details");
  await router.load();
  assert.deepEqual(
    { ...router.state.matches.at(-1).params },
    { workspaceSlug: "acme", projectId: "p1", moduleId: "m1" }
  );
  assert.equal(router.state.location.search.peekModule, "m2");
  assert.equal(router.state.location.hash, "details");
  await router.navigate({
    to: "/$workspaceSlug/settings/projects/$projectId",
    params: { workspaceSlug: "acme", projectId: "p2" },
  });
  assert.equal(router.state.location.pathname, "/acme/settings/projects/p2");
  router.history.back();
  await router.load();
  assert.equal(router.state.location.pathname, "/acme/projects/p1/modules/m1");
  assert.equal(router.state.location.search.peekModule, "m2");
});
