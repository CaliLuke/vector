import assert from "node:assert/strict";
import test from "node:test";
import {
  createPermissionUiFixture,
  React,
  registerPermissionUiStubs,
  settleQuery,
} from "./helpers/permission-ui-fixture.mjs";

registerPermissionUiStubs("/hooks/use-favorite-item-details.tsx", {
  "@/hooks/store/use-project": "export const useProject = () => ({getProjectById: () => undefined});",
  "@/hooks/store": "export const EPageStoreType = {PROJECT:'project'}; export const usePage = () => undefined;",
});
const { useFavoriteItemDetails } = await import("../../apps/web/core/hooks/use-favorite-item-details.tsx");
const { ViewService } = await import("../../apps/web/core/services/view.service.ts");

function Favorite({ favorite }) {
  const details = useFavoriteItemDetails("acme", favorite);
  return React.createElement("output", null, details.itemTitle);
}

const favorite = {
  id: "favorite",
  name: "Saved title",
  entity_data: { name: "Saved title" },
  entity_identifier: "entity",
  project_id: "p1",
};

async function setup(t) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues",
    workspaceRole: 15,
    projectRoles: { p1: 15 },
  });
  const calls = [];
  t.mock.method(ViewService.prototype, "getViewDetails", async (_workspace, project, id) => {
    calls.push({ project, id });
    return { id, project, name: "Canonical view", owned_by: "alice", is_locked: false };
  });
  return { fixture, calls };
}

await Promise.all(
  ["project", "page", "cycle", "module"].map((entity_type) =>
    test(`${entity_type} favorites do not request project view details`, async (t) => {
      const { fixture, calls } = await setup(t);
      await fixture.render(React.createElement(Favorite, { favorite: { ...favorite, entity_type } }));
      await settleQuery();
      assert.equal(fixture.environment.container.textContent, "Saved title");
      assert.deepEqual(calls, []);
    })
  )
);

await test("switching a favorite to a view fetches its canonical title", async (t) => {
  const { fixture, calls } = await setup(t);
  await fixture.render(React.createElement(Favorite, { favorite: { ...favorite, entity_type: "module" } }));
  assert.deepEqual(calls, []);
  await fixture.render(React.createElement(Favorite, { favorite: { ...favorite, entity_type: "view" } }));
  await settleQuery();
  assert.deepEqual(calls, [{ project: "p1", id: "entity" }]);
  assert.equal(fixture.environment.container.textContent, "Canonical view");
});
