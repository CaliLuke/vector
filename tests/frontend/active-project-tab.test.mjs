import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { useActiveTab } = await import("../../apps/web/core/components/navigation/use-active-tab.ts");
const navigationItems = [
  { key: "work_items", href: "/project/issues" },
  { key: "epics", href: "/project/epics" },
  { key: "settings", href: "/project/settings" },
];
function Tabs(props) {
  const { activeItem, isActive } = useActiveTab({
    navigationItems,
    projectId: "project",
    pathname: "/browse/PRJ-1",
    ...props,
  });
  return React.createElement(
    "output",
    null,
    JSON.stringify({
      selected: activeItem?.key ?? null,
      matches: navigationItems.filter(isActive).map((item) => item.key),
    })
  );
}
await test("project navigation recognizes work items and epics only in their own project and matches complete route segments", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const render = async (props) => {
    await env.render(React.createElement(Tabs, props));
    return JSON.parse(env.container.textContent);
  };
  assert.deepEqual(await render({}), { selected: null, matches: [] });
  assert.deepEqual(await render({ workItemId: "item", workItem: { project_id: "project", is_epic: false } }), {
    selected: "work_items",
    matches: ["work_items"],
  });
  assert.deepEqual(await render({ workItemId: "item", workItem: { project_id: "project", is_epic: true } }), {
    selected: "epics",
    matches: ["epics"],
  });
  assert.deepEqual(await render({ workItemId: "item", workItem: { project_id: "other", is_epic: true } }), {
    selected: null,
    matches: [],
  });
  assert.deepEqual(await render({ pathname: "/project/settings/members" }), {
    selected: "settings",
    matches: ["settings"],
  });
  assert.deepEqual(await render({ pathname: "/project/issues-extra" }), { selected: null, matches: [] });
  assert.deepEqual(await render({ pathname: "/project/issues" }), { selected: "work_items", matches: ["work_items"] });
});
