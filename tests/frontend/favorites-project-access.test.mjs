import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { favoriteQueryTestContext } from "./helpers/favorite-query-test-context.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

const {
  identityKeys,
  permissionKeys,
  useFavorites,
  FavoriteService,
  WorkspaceService,
  EUserPermissions,
  EUserWorkspaceRoles,
} = favoriteQueryTestContext;

function FavoritesProbe() {
  const result = useFavorites("acme");
  return React.createElement(
    "output",
    { "data-favorites": JSON.stringify(result.favorites?.map((favorite) => favorite.id) ?? null) },
    result.isError ? "error" : result.isLoading ? "loading" : "ready"
  );
}

async function renderFavoritesForProjectRoles(t, favoriteRows) {
  const fixture = await createQueryRouterFixture(t);
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { p1: EUserPermissions.MEMBER },
  });
  const membershipRevision = fixture.client.getQueryData(
    permissionKeys.workspaceMembership("alice", "acme")
  ).updated_at;
  const rolesKey = permissionKeys.workspaceProjectRoles(
    "alice",
    "acme",
    EUserWorkspaceRoles.MEMBER,
    membershipRevision
  );
  t.mock.method(FavoriteService.prototype, "getFavorites", async () => favoriteRows);
  await fixture.render(React.createElement(FavoritesProbe));
  return { fixture, rolesKey };
}

await test("project favorites disappear when canonical project roles revoke membership", async (t) => {
  const favoriteRows = [
    { id: "project-favorite", project_id: "p1", entity_type: "cycle", entity_identifier: "cycle-1" },
    { id: "workspace-favorite", project_id: null, entity_type: "project", entity_identifier: "p2" },
  ];
  const { fixture, rolesKey } = await renderFavoritesForProjectRoles(t, favoriteRows);
  assert.equal(
    fixture.environment.container.querySelector("output")?.dataset.favorites,
    '["project-favorite","workspace-favorite"]'
  );

  fixture.client.setQueryData(rolesKey, {});
  await fixture.render(React.createElement(FavoritesProbe));

  assert.equal(fixture.environment.container.querySelector("output")?.dataset.favorites, '["workspace-favorite"]');

  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => {
    throw new Error("project roles unavailable");
  });
  await fixture.client.refetchQueries({ queryKey: rolesKey, exact: true });
  await fixture.render(React.createElement(FavoritesProbe));

  assert.equal(fixture.environment.container.querySelector("output")?.dataset.favorites, "null");
  assert.equal(fixture.environment.container.querySelector("output")?.textContent, "error");
});

await test("cached project favorites remain visible during a successful background role refresh", async (t) => {
  const { fixture, rolesKey } = await renderFavoritesForProjectRoles(t, [
    { id: "project-favorite", project_id: "p1", entity_type: "cycle", entity_identifier: "cycle-1" },
  ]);
  const response = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", () => response.promise);

  const refetch = fixture.client.refetchQueries({ queryKey: rolesKey, exact: true });
  await new Promise((resolve) => setImmediate(resolve));
  await fixture.render(React.createElement(FavoritesProbe));

  assert.equal(fixture.environment.container.querySelector("output")?.dataset.favorites, '["project-favorite"]');
  assert.equal(fixture.environment.container.querySelector("output")?.textContent, "ready");
  response.resolve({ p1: EUserPermissions.MEMBER });
  await refetch;
});
