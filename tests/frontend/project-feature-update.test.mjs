import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.featureUpdateReact = React;
const stubs = {
  "@plane/propel/toast":
    "export const setPromiseToast=(promise,options)=>globalThis.featureUpdateToast(promise,options);",
  "@plane/ui":
    'export const ToggleSwitch=({value,onChange,disabled})=>globalThis.featureUpdateReact.createElement("button",{disabled,"aria-pressed":value,onClick:()=>onChange(!value)},"Toggle feature");',
  "@/components/settings/boxed-control-item": "export const SettingsBoxedControlItem=({control})=>control;",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { ProjectSettingsFeatureControlItem } =
  await import("../../apps/web/core/components/settings/project/content/feature-control-item.tsx");

await test("failed feature updates report the error and release the pending control without an unhandled rejection", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, { userId: "alice", workspaceSlug: "acme", projectRoles: { p1: 20 } });
  const response = Promise.withResolvers();
  const started = Promise.withResolvers();
  const reported = Promise.withResolvers();
  const writes = [];
  t.mock.method(ProjectService.prototype, "updateProject", async (workspaceSlug, projectId, data) => {
    writes.push({ workspaceSlug, projectId, data });
    started.resolve();
    return response.promise;
  });
  globalThis.featureUpdateToast = (promise, options) =>
    void promise.then(
      () => reported.resolve({ kind: "success" }),
      (error) => reported.resolve({ kind: "error", error, message: options.error.message(error) })
    );
  t.after(() => {
    delete globalThis.featureUpdateToast;
  });
  await fixture.render(
    React.createElement(ProjectSettingsFeatureControlItem, {
      workspaceSlug: "acme",
      projectId: "p1",
      featureProperty: "cycle_view",
      title: "Cycles",
      value: true,
    })
  );
  await act(async () => {
    fixture.environment.container.querySelector("button").click();
    await started.promise;
  });
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector("button").disabled, true);
  const failure = new Error("Transport failed");
  await act(async () => {
    response.reject(failure);
    await reported.promise;
  });
  await settleQuery();
  assert.equal((await reported.promise).kind, "error");
  assert.equal((await reported.promise).error, failure);
  assert.equal(fixture.environment.container.querySelector("button").disabled, false);
  assert.equal(fixture.environment.container.querySelector("button").getAttribute("aria-pressed"), "true");
  assert.deepEqual(writes, [{ workspaceSlug: "acme", projectId: "p1", data: { cycle_view: false } }]);
});
