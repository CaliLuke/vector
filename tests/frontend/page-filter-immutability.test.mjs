import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.pageFilterReact = React;
registerWebTestModules({
  "@/components/core/filters/favorites-created-date-filters":
    "export const FavoritesCreatedDateFilters=({children})=>children('');",
  "@/components/core/filters/workspace-member-filter": `export const WorkspaceMemberFilter=({handleUpdate})=>globalThis.pageFilterReact.createElement('button',{onClick:()=>handleUpdate('bob')},'Bob');`,
});
const { PageFiltersSelection } = await import("../../apps/web/core/components/pages/list/filters/root.tsx");

await test("changing a page creator filter preserves the prior selection snapshot", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const selected = Object.freeze(["alice"]);
  const filters = Object.freeze({ filters: Object.freeze({ created_by: selected }) });
  const updates = [];
  await env.render(
    React.createElement(PageFiltersSelection, {
      filters,
      workspaceSlug: "acme",
      memberIds: ["alice", "bob"],
      handleFiltersUpdate: (key, value) => updates.push([key, value]),
    })
  );
  await act(async () => env.container.querySelector("button").click());
  assert.deepEqual(updates, [["filters", { created_by: ["alice", "bob"] }]]);
  assert.deepEqual(selected, ["alice"]);
});
