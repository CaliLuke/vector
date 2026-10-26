import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { favoriteQueryTestContext } from "./helpers/favorite-query-test-context.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

const { identityKeys, useFavorites, FavoriteService, EUserWorkspaceRoles } = favoriteQueryTestContext;

let actions;

function FavoritesProbe() {
  const result = useFavorites("acme");
  const { addFavorite, deleteFavorite, reOrderFavorite, updateFavorite } = result;
  React.useEffect(() => {
    actions = {
      create: () => addFavorite({ entity_type: "project", entity_identifier: "new", project_id: null, name: "New" }),
      rename: () => updateFavorite("folder", { name: "Renamed" }),
      renameChild: () => updateFavorite("child", { name: "Saved" }),
      deleteFolder: () => deleteFavorite("folder"),
      reorder: (favoriteId, destinationId, edge) => reOrderFavorite(favoriteId, destinationId, edge),
    };
  }, [addFavorite, deleteFavorite, reOrderFavorite, updateFavorite]);
  return React.createElement(
    "output",
    { "data-favorites": JSON.stringify(result.favorites?.map((favorite) => [favorite.id, favorite.name]) ?? null) },
    result.isError ? "error" : result.isLoading ? "loading" : "ready"
  );
}

async function favoritesFixture(t, initialRows) {
  const fixture = await createQueryRouterFixture(t);
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: {},
  });
  let rows = initialRows;
  let failReads = false;
  const getFavorites = t.mock.method(FavoriteService.prototype, "getFavorites", async () => {
    if (failReads) throw new Error("refresh failed");
    return rows;
  });
  await fixture.render(React.createElement(FavoritesProbe));
  return {
    ...fixture,
    get rows() {
      return rows;
    },
    setRows(nextRows) {
      rows = nextRows;
    },
    failReads() {
      failReads = true;
    },
    getFavorites,
  };
}

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

async function runPendingAction(action) {
  let request;
  await act(async () => {
    request = action();
    await Promise.resolve();
  });
  request.catch(() => undefined);
  return { request };
}

function folderAndChildFavorites() {
  return [
    {
      id: "folder",
      entity_type: "folder",
      name: "Folder",
      is_folder: true,
      parent: null,
      sequence: 1,
      project_id: null,
    },
    {
      id: "child",
      entity_type: "project",
      name: "Child",
      is_folder: false,
      parent: "folder",
      sequence: 2,
      project_id: null,
    },
  ];
}

async function settleFailure(fixture, deferredRequest) {
  await act(async () => {
    deferredRequest.reject(new Error("write failed"));
    await deferredRequest.promise.catch(() => undefined);
  });
  await fixture.render(React.createElement(FavoritesProbe));
}

await test("failed create stops projecting its optimistic favorite", async (t) => {
  const fixture = await favoritesFixture(t, []);
  const response = deferred();
  const create = t.mock.method(FavoriteService.prototype, "addFavorite", () => response.promise);
  const { request } = await runPendingAction(actions.create);
  await fixture.render(React.createElement(FavoritesProbe));

  assert.equal(create.mock.callCount(), 1);
  assert.match(fixture.environment.container.querySelector("output")?.dataset.favorites ?? "", /New/);

  await settleFailure(fixture, response);
  await request.catch(() => undefined);
  assert.equal(fixture.environment.container.querySelector("output")?.dataset.favorites, "[]");
});

await test("failed folder rename and delete remove their pending overlays", async (t) => {
  const fixture = await favoritesFixture(t, folderAndChildFavorites());
  const renameResponse = deferred();
  const update = t.mock.method(FavoriteService.prototype, "updateFavorite", () => renameResponse.promise);
  const { request: rename } = await runPendingAction(actions.rename);
  await fixture.render(React.createElement(FavoritesProbe));

  assert.equal(update.mock.callCount(), 1);
  assert.match(fixture.environment.container.querySelector("output")?.dataset.favorites ?? "", /Renamed/);

  await settleFailure(fixture, renameResponse);
  await rename.catch(() => undefined);
  assert.match(fixture.environment.container.querySelector("output")?.dataset.favorites ?? "", /Folder/);

  const deleteResponse = deferred();
  const remove = t.mock.method(FavoriteService.prototype, "deleteFavorite", () => deleteResponse.promise);
  const { request: deletion } = await runPendingAction(actions.deleteFolder);
  await fixture.render(React.createElement(FavoritesProbe));

  assert.equal(remove.mock.callCount(), 1);
  assert.equal(fixture.environment.container.querySelector("output")?.dataset.favorites, "[]");

  await settleFailure(fixture, deleteResponse);
  await deletion.catch(() => undefined);
  assert.match(fixture.environment.container.querySelector("output")?.dataset.favorites ?? "", /child/);
  assert.match(fixture.environment.container.querySelector("output")?.dataset.favorites ?? "", /folder/);
});

await test("settled create uses one canonical row and does not keep its overlay", async (t) => {
  const fixture = await favoritesFixture(t, []);
  const canonical = {
    id: "canonical",
    entity_type: "project",
    entity_identifier: "new",
    entity_data: { name: "New" },
    name: "New",
    is_folder: false,
    sequence: 1,
    sort_order: 1,
    parent: null,
    children: [],
    project_id: null,
    workspace_id: "workspace",
  };
  t.mock.method(FavoriteService.prototype, "addFavorite", async () => {
    fixture.setRows([canonical]);
    return canonical;
  });

  await runPendingAction(actions.create).then(({ request }) => request);
  await fixture.render(React.createElement(FavoritesProbe));

  assert.equal(
    fixture.environment.container.querySelector("output")?.dataset.favorites,
    JSON.stringify([["canonical", "New"]])
  );
});

await test("accepted folder deletion removes its cascade subtree even when refresh fails", async (t) => {
  const fixture = await favoritesFixture(t, [
    {
      id: "folder",
      entity_type: "folder",
      name: "Folder",
      is_folder: true,
      parent: null,
      sequence: 1,
      project_id: null,
    },
    {
      id: "child",
      entity_type: "folder",
      name: "Child folder",
      is_folder: true,
      parent: "folder",
      sequence: 2,
      project_id: null,
    },
    {
      id: "nested",
      entity_type: "project",
      name: "Nested",
      is_folder: false,
      parent: "child",
      sequence: 3,
      project_id: null,
    },
  ]);
  const remove = t.mock.method(FavoriteService.prototype, "deleteFavorite", async () => {
    fixture.setRows([]);
    fixture.failReads();
  });

  await runPendingAction(actions.deleteFolder).then(({ request }) => request);

  assert.equal(remove.mock.callCount(), 1);
  assert.equal(fixture.getFavorites.mock.callCount(), 2);
  const favoriteQuery = fixture.client
    .getQueryCache()
    .findAll({ queryKey: ["workspace-favorites", "alice", "acme"] })[0];
  assert.deepEqual(favoriteQuery?.state.data, []);
});

await test("favorite reordering uses sibling midpoints and preserves zero-valued sequence", async (t) => {
  const nestedRows = [
    { id: "top", entity_type: "folder", is_folder: true, parent: null, sequence: 30_000, project_id: null },
    { id: "middle", entity_type: "folder", is_folder: true, parent: null, sequence: 20_000, project_id: null },
    { id: "bottom", entity_type: "folder", is_folder: true, parent: null, sequence: 10_000, project_id: null },
    { id: "zero", entity_type: "folder", is_folder: true, parent: null, sequence: 0, project_id: null },
    { id: "parent", entity_type: "folder", is_folder: true, parent: null, sequence: 1, project_id: null },
    { id: "nested", entity_type: "folder", is_folder: true, parent: "parent", sequence: 25_000, project_id: null },
  ];
  const fixture = await favoritesFixture(t, nestedRows);
  const favoriteQuery = fixture.client
    .getQueryCache()
    .findAll({ queryKey: ["workspace-favorites", "alice", "acme"] })[0];
  const updates = [];
  t.mock.method(FavoriteService.prototype, "updateFavorite", async (_workspaceSlug, id, data) => {
    updates.push({ id, ...data });
    return { ...nestedRows.find((favorite) => favorite.id === id), ...data };
  });

  const cases = [
    { moving: "top", destination: "middle", edge: "reorder-below", expected: 15_000 },
    { moving: "bottom", destination: "middle", edge: "reorder-above", expected: 25_000 },
    { moving: "bottom", destination: "zero", edge: "reorder-below", expected: -10_000 },
  ];
  await assertReorderCase(fixture, favoriteQuery, updates, nestedRows, cases[0]);
  await assertReorderCase(fixture, favoriteQuery, updates, nestedRows, cases[1]);
  await assertReorderCase(fixture, favoriteQuery, updates, nestedRows, cases[2]);
});

async function assertReorderCase(fixture, favoriteQuery, updates, nestedRows, scenario) {
  await act(async () => {
    await actions.reorder(scenario.moving, scenario.destination, scenario.edge);
  });
  assert.equal(updates.at(-1)?.id, scenario.moving);
  assert.equal(updates.at(-1)?.sequence, scenario.expected);
  fixture.client.setQueryData(favoriteQuery.queryKey, nestedRows);
  await fixture.render(React.createElement(FavoritesProbe));
}
