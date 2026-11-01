import assert from "node:assert/strict";
import test from "node:test";
import { React, act, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";

globalThis.analyticsLoadingReact = React;
globalThis.analyticsLoadingState = { cycle: undefined, isError: false, retry: async () => {} };
const loader = `export {Loader} from ${JSON.stringify(new URL("../../packages/ui/src/loader.tsx", import.meta.url).href)};`;
registerPermissionUiStubs("/cycles/analytics-sidebar/root.tsx", {
  "@plane/ui": loader,
  "../active-cycle/use-cycles-details": "export default ()=>globalThis.analyticsLoadingState;",
  "./sidebar-header": `export const CycleSidebarHeader=({cycleDetails,handleClose})=>globalThis.analyticsLoadingReact.createElement('button',{onClick:handleClose},cycleDetails.name);`,
  "./sidebar-details": `export const CycleSidebarDetails=({workspaceSlug,projectId,cycleDetails})=>globalThis.analyticsLoadingReact.createElement('output',null,[workspaceSlug,projectId,cycleDetails.id].join('/'));`,
  "./issue-progress": "export const CycleAnalyticsProgress=()=>null;",
});
registerPermissionUiStubs("/common/analytics-sidebar-loader.tsx", { "@plane/ui": loader });

await test("cycle sidebar loading replaces the previous scope and resolves to the current details", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { CycleDetailsSidebar } = await import("../../apps/web/core/components/cycles/analytics-sidebar/root.tsx");
  globalThis.analyticsLoadingState = { cycle: undefined, isError: false, retry: async () => {} };
  let closed = 0;
  const render = (projectId, cycleId) =>
    env.render(
      React.createElement(CycleDetailsSidebar, {
        workspaceSlug: "acme",
        projectId,
        cycleId,
        handleClose: () => {
          closed++;
        },
      })
    );
  await render("p1", "first");
  assert.ok(env.container.querySelector('[aria-busy="true"]'));
  assert.match(env.container.textContent, /Loading/);
  globalThis.analyticsLoadingState.cycle = { id: "first", name: "First cycle" };
  await render("p1", "first");
  assert.equal(Boolean(env.container.querySelector('[aria-busy="true"]')), false);
  assert.equal(env.container.querySelector("output").textContent, "acme/p1/first");
  globalThis.analyticsLoadingState.cycle = undefined;
  await render("p2", "second");
  assert.ok(env.container.querySelector('[aria-busy="true"]'));
  assert.doesNotMatch(env.container.textContent, /First cycle/);
  globalThis.analyticsLoadingState.cycle = { id: "second", name: "Second cycle" };
  await render("p2", "second");
  assert.equal(env.container.querySelector("output").textContent, "acme/p2/second");
  await act(async () => env.container.querySelector("button").click());
  assert.equal(closed, 1);
});
