import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
registerWebModuleHooks();
const fixtureModule = createModuleStub(`
  export const fixture = { enabled: true, supported: true, calls: [] };
  export const useWorkItemDiscoveryScope = (canLoad) => ({workspaceSlug:"acme", projectId:"p", canLoad, supported:fixture.supported});
  export const useWorkItemTypeCatalog = (_slug, _project, canLoad) => {
    fixture.calls.push(canLoad);
    return { identities: canLoad ? {enabled:fixture.enabled} : undefined,
      permissions:{isPending:false, workspaceRole:15,projectRole:15}};
  };
`);
registerHooks({
  resolve(specifier, context, next) {
    if (
      context.parentURL?.endsWith("/use-work-item-grouping-catalog.ts") &&
      ["./use-work-item-discovery-scope", "./use-work-item-type-catalog"].includes(specifier)
    )
      return fixtureModule;
    return next(specifier, context);
  },
});
const { fixture } = await import(fixtureModule.url);
const { useTypeGroupingOptions } = await import("../../apps/web/core/hooks/use-work-item-grouping-catalog.ts");
const original = Object.freeze(["state", "priority", null]);
function Probe({ layout, isEpic = false }) {
  const choices = useTypeGroupingOptions(layout, isEpic)(original);
  return React.createElement("output", null, JSON.stringify(choices));
}
await test("type grouping is available only for authorized supported list and board views", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const scenarios = [
    { layout: "list", enabled: true, supported: true, expected: true },
    { layout: "kanban", enabled: true, supported: true, expected: true },
    { layout: "list", enabled: false, supported: true, expected: false },
    { layout: "list", enabled: true, supported: false, expected: false },
    { layout: "list", enabled: true, supported: true, isEpic: true, expected: false },
    { layout: "spreadsheet", enabled: true, supported: true, expected: false },
  ];
  await Promise.all(
    scenarios.map((scenario, index) =>
      t.test(String(index), async () => {
        Object.assign(fixture, scenario);
        await env.render(React.createElement(Probe, scenario));
        const choices = JSON.parse(env.container.textContent);
        assert.equal(choices.includes("type_id"), scenario.expected, JSON.stringify(scenario));
        assert.deepEqual(
          choices.filter((choice) => choice !== "type_id"),
          original
        );
      })
    )
  );
  assert.equal(fixture.calls.at(-1), false, "table display must not request the type grouping catalog");
});
