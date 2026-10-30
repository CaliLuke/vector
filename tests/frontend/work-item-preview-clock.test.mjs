import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

// Initialize the real Query polling scheduler in a browser environment.
const bootstrapEnvironment = await createReactDomTestEnvironment();
registerWebModuleHooks();
const fixture = { calls: 0 };
globalThis.previewClockFixture = fixture;
const stubs = {
  "@/hooks/use-current-user": `export const useCurrentUserId=()=>"reader";`,
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>({projectRole:20});`,
  "@/queries/work-item-access": `export const captureVisibleItemRead=()=>()=>{};`,
  "@/queries/work-item-schema": `export const itemSchemaOptions=()=>({queryKey:["schema"],queryFn:async()=>globalThis.previewClockFixture.schema,initialData:globalThis.previewClockFixture.schema});export const itemTypeChoicesOptions=()=>({queryKey:["types"],queryFn:async()=>({enabled:true,types:[]})});`,
  "@/services/work-item-schema.service": `export const workItemSchemaService={preview:async()=>{globalThis.previewClockFixture.calls++;return {enabled:true,revision:1,type_id:"task",computed:{clock:{kind:"Date",status:"value",value:new Date().toISOString()}}};}};`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { QueryClient, QueryClientProvider, focusManager, notifyManager } = await import("@tanstack/react-query");
const { useForm } = webRequire("@tanstack/react-form");
const { useInlinePropertyPreview } =
  await import("../../apps/web/core/components/work-item-types/use-inline-property-preview.ts");
const { useItemTypeFields } = await import("../../apps/web/core/components/issues/issue-modal/use-item-type-fields.ts");
notifyManager.setScheduler(queueMicrotask);
const midnight = Date.parse("2026-10-27T00:00:00Z");
const item = { id: "item", name: "Item", project_id: "project", type_id: "task", custom_properties: {} };
const scope = { workspaceSlug: "workspace", projectId: "project", issueId: "item" };
function InlinePreview() {
  const preview = useInlinePropertyPreview(scope, fixture.schema, item, item.custom_properties);
  return React.createElement("output", null, preview.computed?.clock?.value);
}
function FormPreview() {
  const form = useForm({ defaultValues: item });
  const fields = useItemTypeFields(form, "workspace", item, "edit");
  return React.createElement("output", null, fields.computed?.clock?.value);
}
async function mount(t, component, formulas = true) {
  const env = await createReactDomTestEnvironment();
  t.mock.timers.enable({ apis: ["Date", "setInterval"], now: midnight - 1000 });
  fixture.calls = 0;
  fixture.schema = {
    enabled: true,
    revision: 1,
    type_id: "task",
    eligible_type_ids: ["task"],
    properties: formulas ? [{ id: "clock", kind: "Formula", configuration: { active: true } }] : [],
  };
  const client = new QueryClient({
    defaultOptions: { queries: { gcTime: Infinity, staleTime: 30_000, retry: false } },
  });
  focusManager.setFocused(true);
  const render = () => env.render(React.createElement(QueryClientProvider, { client }, React.createElement(component)));
  t.after(async () => {
    await env.unmount(() => client.clear());
    focusManager.setFocused(undefined);
  });
  await render();
  return { render, env };
}

async function testPreviewSurface(name, component) {
  await test(`${name} refreshes at UTC minute boundaries and stops when unmounted`, async (t) => {
    const { env } = await mount(t, component);
    assert.equal(env.container.textContent, "2026-10-26T23:59:59.000Z");
    await act(async () => t.mock.timers.tick(1000));
    assert.equal(env.container.textContent, "2026-10-27T00:00:00.000Z");
    await act(async () => t.mock.timers.tick(60_000));
    assert.equal(env.container.textContent, "2026-10-27T00:01:00.000Z");
    await env.render(null);
    const calls = fixture.calls;
    await act(async () => t.mock.timers.tick(60_000));
    assert.equal(fixture.calls, calls);
  });
  await test(`${name} refreshes fresh cached formulas on focus and remount`, async (t) => {
    const { render, env } = await mount(t, component);
    focusManager.setFocused(false);
    await act(async () => t.mock.timers.tick(1000));
    await act(async () => focusManager.setFocused(true));
    assert.equal(env.container.textContent, "2026-10-27T00:00:00.000Z");
    await env.render(null);
    await act(async () => t.mock.timers.tick(1000));
    await render();
    assert.equal(env.container.textContent, "2026-10-27T00:00:01.000Z");
  });
  await test(`${name} does not request previews without formulas`, async (t) => {
    await mount(t, component, false);
    await act(async () => t.mock.timers.tick(120_000));
    await act(async () => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
    });
    assert.equal(fixture.calls, 0);
  });
}

await testPreviewSurface("inline editor", InlinePreview);
await testPreviewSurface("creation/editing form", FormPreview);
await bootstrapEnvironment.unmount();
