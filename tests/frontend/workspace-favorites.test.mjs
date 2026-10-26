import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient } from "@tanstack/react-query";
import { favoriteQueryTestContext } from "./helpers/favorite-query-test-context.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

const {
  identityKeys,
  advanceIdentitySession,
  permissionKeys,
  addProjectFavoriteAfterAccessRefresh,
  captureWorkspaceFavoritesRefresh,
  favoriteProjectRolesRevision,
  favoriteKeys,
  favoriteMutationOptions,
  workspaceFavoritesOptions,
  FavoriteService,
  WorkspaceService,
  EUserWorkspaceRoles,
} = favoriteQueryTestContext;

const scope = {
  accountId: "actor",
  workspaceSlug: "acme",
  workspaceRole: EUserWorkspaceRoles.ADMIN,
  membershipRevision: "2026-10-24T00:00:00Z",
  projectRolesRevision: undefined,
};

function createClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
}

async function seed(client, workspaceRole = scope.workspaceRole, projectRoles = {}) {
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  await seedPermissionQueries(client, {
    userId: scope.accountId,
    workspaceSlug: scope.workspaceSlug,
    workspaceRole,
    projectRoles,
  });
  const membershipRevision = client.getQueryData(
    permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug)
  ).updated_at;
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    workspaceRole,
    membershipRevision
  );
  return {
    ...scope,
    workspaceRole,
    membershipRevision,
    projectRolesRevision: favoriteProjectRolesRevision(client.getQueryData(rolesKey)),
  };
}

function startFavoriteRead(t, queryClient, readScope) {
  const response = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(FavoriteService.prototype, "getFavorites", () => {
    started.resolve();
    return response.promise;
  });
  const request = queryClient.fetchQuery(workspaceFavoritesOptions(readScope));
  return { request, response, started };
}

await test("favorite all-list reads include account and active workspace membership scope", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  const readScope = await seed(queryClient, EUserWorkspaceRoles.MEMBER);
  const received = [];
  const rows = [{ id: "root" }, { id: "nested", parent: "folder" }];
  t.mock.method(FavoriteService.prototype, "getFavorites", async (...args) => {
    received.push(args);
    return rows;
  });

  const result = await queryClient.fetchQuery(workspaceFavoritesOptions(readScope));

  assert.deepEqual(result, rows);
  assert.equal(received.length, 1);
  assert.equal(received[0][0], "acme");
  assert.equal(received[0][1] instanceof AbortSignal, true);
  assert.deepEqual(favoriteKeys.all(readScope), [
    "workspace-favorites",
    "actor",
    "acme",
    EUserWorkspaceRoles.MEMBER,
    readScope.membershipRevision,
    readScope.projectRolesRevision,
    "all",
  ]);
});

await test("favorite reads reject stale project membership after a role refresh", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  const readScope = await seed(queryClient, scope.workspaceRole, { "project-1": 20 });
  const { request, response, started } = startFavoriteRead(t, queryClient, readScope);
  await started.promise;
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    readScope.workspaceRole,
    readScope.membershipRevision
  );
  queryClient.setQueryData(rolesKey, {});
  response.resolve([{ id: "private-project", project_id: "removed-project" }]);

  await assert.rejects(request, /scope changed/);
  assert.equal(queryClient.getQueryData(favoriteKeys.all(readScope)), undefined);
});

await test("favorite reads fail closed if workspace membership changes while the request is pending", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  const readScope = await seed(queryClient);
  const { request, response, started } = startFavoriteRead(t, queryClient, readScope);
  await started.promise;
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  queryClient.setQueryData(membershipKey, {
    is_active: true,
    role: EUserWorkspaceRoles.MEMBER,
    updated_at: "replacement-revision",
  });
  response.resolve([{ id: "stale" }]);

  await assert.rejects(request, /scope changed/);
  assert.equal(queryClient.getQueryData(favoriteKeys.all(readScope)), undefined);
});

await test("accepted favorite create caches the canonical row and failed writes keep the prior collection", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  const writeScope = await seed(queryClient);
  const key = favoriteKeys.all(writeScope);
  const existing = [{ id: "existing", entity_type: "project", entity_identifier: "p1" }];
  queryClient.setQueryData(key, existing);
  const canonical = { id: "canonical", entity_type: "project", entity_identifier: "p2" };
  const add = t.mock.method(FavoriteService.prototype, "addFavorite", async () => canonical);
  const update = t.mock.method(FavoriteService.prototype, "updateFavorite", async () => {
    throw new Error("write failed");
  });
  const observer = new MutationObserver(queryClient, favoriteMutationOptions(queryClient, writeScope));
  t.after(() => observer.reset());

  await observer.mutate({ type: "create", clientId: "optimistic", data: { entity_type: "project" } });
  assert.equal(add.mock.callCount(), 1);
  assert.deepEqual(queryClient.getQueryData(key), [...existing, canonical]);
  await assert.rejects(
    observer.mutate({ type: "update", favoriteId: "existing", data: { name: "changed" } }),
    /write failed/
  );
  assert.equal(update.mock.callCount(), 1);
  assert.deepEqual(queryClient.getQueryData(key), [...existing, canonical]);
});

await test("an accepted favorite write from a replaced identity cannot update its replacement query", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  const writeScope = await seed(queryClient);
  const key = favoriteKeys.all(writeScope);
  const previous = [{ id: "favorite", name: "Old" }];
  const replacement = [{ id: "favorite", name: "New account row" }];
  queryClient.setQueryData(key, previous);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(FavoriteService.prototype, "updateFavorite", () => {
    started.resolve();
    return response.promise;
  });
  const observer = new MutationObserver(queryClient, favoriteMutationOptions(queryClient, writeScope));
  t.after(() => observer.reset());

  const write = observer.mutate({ type: "update", favoriteId: "favorite", data: { name: "Saved" } });
  await started.promise;
  advanceIdentitySession(queryClient);
  queryClient.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  queryClient.setQueryData(key, replacement);
  response.resolve({ id: "favorite", name: "Saved" });
  await write;

  assert.deepEqual(queryClient.getQueryData(key), replacement);
});

await test("project creation favorites only after a refreshed canonical project role appears", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  await seed(queryClient, EUserWorkspaceRoles.MEMBER, {});
  const roles = t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({
    created: 20,
  }));
  const reads = t.mock.method(FavoriteService.prototype, "getFavorites", async () => []);
  const canonical = {
    id: "favorite-created",
    entity_type: "project",
    entity_identifier: "created",
    project_id: "created",
  };
  const add = t.mock.method(FavoriteService.prototype, "addFavorite", async () => canonical);

  const result = await addProjectFavoriteAfterAccessRefresh(queryClient, "acme", "created", "Created");

  assert.equal(roles.mock.callCount(), 1);
  assert.equal(reads.mock.callCount(), 1);
  assert.equal(add.mock.callCount(), 1);
  assert.equal(add.mock.calls[0].arguments[1].project_id, "created");
  assert.equal(result.id, canonical.id);
});

await test("project creation favorite follow-up fails closed when role refresh fails", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  await seed(queryClient, EUserWorkspaceRoles.MEMBER, {});
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => {
    throw new Error("roles unavailable");
  });
  const add = t.mock.method(FavoriteService.prototype, "addFavorite", async () => ({ id: "unexpected" }));

  await assert.rejects(
    addProjectFavoriteAfterAccessRefresh(queryClient, "acme", "created", "Created"),
    /roles unavailable/
  );
  assert.equal(add.mock.callCount(), 0);
});

await test("captured lifecycle refresh does not target a replacement identity", async (t) => {
  const queryClient = createClient();
  t.after(() => queryClient.clear());
  const readScope = await seed(queryClient);
  const key = favoriteKeys.all(readScope);
  queryClient.setQueryData(key, [{ id: "old-session" }]);
  let reads = 0;
  t.mock.method(FavoriteService.prototype, "getFavorites", async () => {
    reads += 1;
    return [{ id: "refetched" }];
  });
  const refresh = captureWorkspaceFavoritesRefresh(queryClient, "acme");
  advanceIdentitySession(queryClient);
  queryClient.setQueryData(identityKeys.currentUser(), { id: scope.accountId });

  await refresh();

  assert.equal(reads, 0);
  assert.deepEqual(queryClient.getQueryData(key), [{ id: "old-session" }]);
});
