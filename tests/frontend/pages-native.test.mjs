import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createIssueQueryClient } from "./helpers/issue-query-fixtures.mjs";
registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { QueryClient, MutationObserver } = webRequire("@tanstack/react-query");
const { EPageAccess } = webRequire("@plane/constants");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const {
  pageKeys,
  projectPagesOptions,
  projectPageOptions,
  updateProjectPageOptions,
  pageFilterList,
  updateProjectPageCache,
  removeProjectPageCache,
} = await import("../../apps/web/core/queries/pages.ts");
const { ProjectPageService } = await import("../../apps/web/core/services/page/project-page.service.ts");
const { pageRedirectLink } = await import("../../apps/web/helpers/page-permissions.ts");

const scope = {
  accountId: "alice",
  workspaceSlug: "acme",
  membershipRole: 15,
  membershipRevision: "membership-1",
  projectId: "project-1",
  projectMembershipRole: 15,
  projectRole: 15,
  rolesRevision: JSON.stringify([["project-1", 15]]),
  sessionGeneration: 0,
};

function createClient(t) {
  return createIssueQueryClient(t, QueryClient, scope).client;
}

const page = {
  id: "page-1",
  name: "Current name",
  owned_by: "alice",
  access: EPageAccess.PRIVATE,
  project_ids: [scope.projectId],
};

function replaceProjectRoleQuery(client) {
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { [scope.projectId]: scope.projectMembershipRole });
}

await test("page list read uses cancellation and retains successful empty results", async (t) => {
  const client = createClient(t);
  const fetchAll = t.mock.method(ProjectPageService.prototype, "fetchAll", async (_workspace, _project, signal) => {
    assert.equal(signal instanceof AbortSignal, true);
    return [];
  });

  assert.deepEqual(await client.fetchQuery(projectPagesOptions(client, scope)), []);
  assert.deepEqual(client.getQueryData(pageKeys.project(scope)), []);
  assert.equal(fetchAll.mock.callCount(), 1);
});

await test("page list read rejects a replaced project-role query before caching its response", async (t) => {
  const client = createClient(t);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(ProjectPageService.prototype, "fetchAll", () => {
    started.resolve();
    return response.promise;
  });
  const request = client.fetchQuery(projectPagesOptions(client, scope));
  await started.promise;
  replaceProjectRoleQuery(client);
  response.resolve([page]);

  await assert.rejects(request, /access changed/);
  assert.equal(client.getQueryData(pageKeys.project(scope)), undefined);
});

await test("page detail read validates canonical identity and requested-project membership", async (t) => {
  const client = createClient(t);
  const fetchById = t.mock.method(
    ProjectPageService.prototype,
    "fetchById",
    async (_workspace, _project, id, trackVisit, signal) => {
      assert.equal(id, page.id);
      assert.equal(trackVisit, false);
      assert.equal(signal instanceof AbortSignal, true);
      return { ...page, project_ids: ["other-project"] };
    }
  );

  await assert.rejects(
    client.fetchQuery(projectPageOptions(client, scope, page.id, false)),
    /outside the requested project/
  );
  assert.equal(client.getQueryData(pageKeys.detail(scope, page.id, false)), undefined);
  assert.equal(fetchById.mock.callCount(), 1);
});

await test("rejected optimistic page edit restores the visible query records", async (t) => {
  const client = createClient(t);
  const listKey = pageKeys.project(scope);
  const detailKey = pageKeys.detail(scope, page.id);
  client.setQueryData(listKey, [page]);
  client.setQueryData(detailKey, page);
  t.mock.method(ProjectPageService.prototype, "update", async () => {
    throw new Error("write rejected");
  });
  const mutation = new MutationObserver(client, updateProjectPageOptions(client, scope, page.id));
  t.after(() => mutation.reset());

  await assert.rejects(mutation.mutate({ name: "Draft title" }), /write rejected/);
  assert.equal(client.getQueryData(detailKey).name, page.name);
  assert.equal(client.getQueryData(listKey)[0].name, page.name);
});

await test("page list and links stay scoped to the active route project", async () => {
  const pages = [
    page,
    { ...page, id: "other-page", project_ids: ["other-project"] },
    { ...page, id: "shared-page", project_ids: [scope.projectId, "other-project"] },
  ];
  const filtered = pageFilterList(pages, scope.projectId, "private", {
    searchQuery: "",
    sortKey: "updated_at",
    sortBy: "desc",
  });

  assert.deepEqual(
    filtered.map(({ id }) => id).toSorted((left, right) => left.localeCompare(right)),
    ["page-1", "shared-page"]
  );
  assert.equal(
    pageRedirectLink(scope.workspaceSlug, scope.projectId, "shared-page"),
    "/acme/projects/project-1/pages/shared-page"
  );
});

await test("route-scoped page mutation commits and realtime cache updates/removes canonical records", async (t) => {
  const client = createClient(t);
  const listKey = pageKeys.project(scope);
  const detailKey = pageKeys.detail(scope, page.id);
  client.setQueryData(listKey, [page]);
  client.setQueryData(detailKey, page);
  const update = t.mock.method(ProjectPageService.prototype, "update", async (workspace, projectId, id, change) => {
    assert.equal(workspace, scope.workspaceSlug);
    assert.equal(projectId, scope.projectId);
    assert.equal(id, page.id);
    return { ...page, ...change };
  });
  const mutation = new MutationObserver(client, updateProjectPageOptions(client, scope, page.id));
  t.after(() => mutation.reset());

  await mutation.mutate({ name: "Saved title" });
  assert.equal(update.mock.callCount(), 1);
  assert.equal(client.getQueryData(detailKey).name, "Saved title");
  assert.equal(client.getQueryData(listKey)[0].name, "Saved title");

  updateProjectPageCache(client, scope, page.id, (current) => ({ ...current, is_locked: true }));
  assert.equal(client.getQueryData(detailKey).is_locked, true);
  removeProjectPageCache(client, scope, page.id);
  assert.deepEqual(client.getQueryData(listKey), []);
  assert.equal(client.getQueryData(detailKey), undefined);
});

await test("tracked and untracked page readers share canonical updates", async (t) => {
  const client = createClient(t);
  const tracked = projectPageOptions(client, scope, page.id, true);
  const untracked = projectPageOptions(client, scope, page.id, false);
  assert.deepEqual(tracked.queryKey, untracked.queryKey);
  client.setQueryData(untracked.queryKey, page);
  updateProjectPageCache(client, scope, page.id, (current) => ({ ...current, name: "Live title" }));
  assert.equal(client.getQueryData(tracked.queryKey).name, "Live title");
});

await test("an accepted page write preserves a newer collaborative change", async (t) => {
  const client = createClient(t);
  const detailKey = pageKeys.detail(scope, page.id);
  const listKey = pageKeys.project(scope);
  client.setQueryData(detailKey, page);
  client.setQueryData(listKey, [page]);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(ProjectPageService.prototype, "update", () => {
    started.resolve();
    return response.promise;
  });
  const mutation = new MutationObserver(client, updateProjectPageOptions(client, scope, page.id));
  t.after(() => mutation.reset());
  const request = mutation.mutate({ name: "Submitted title" });
  await started.promise;
  updateProjectPageCache(client, scope, page.id, (current) => ({ ...current, name: "Newer title", is_locked: true }));
  response.resolve({ ...page, name: "Submitted title", updated_at: "server-revision" });
  await request;
  for (const cached of [client.getQueryData(detailKey), client.getQueryData(listKey)[0]]) {
    assert.equal(cached.name, "Newer title");
    assert.equal(cached.is_locked, true);
    assert.equal(cached.updated_at, "server-revision");
  }
});

await test("page versions reject permission replacement before publishing their response", async (t) => {
  const { projectPageVersionsOptions } = await import("../../apps/web/core/queries/pages.ts");
  const { ProjectPageVersionService } =
    await import("../../apps/web/core/services/page/project-page-version.service.ts");
  const client = createClient(t);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(ProjectPageVersionService.prototype, "fetchAllVersions", (_workspace, _project, _page, signal) => {
    assert.equal(signal instanceof AbortSignal, true);
    started.resolve();
    return response.promise;
  });
  const options = projectPageVersionsOptions(client, scope, page.id);
  const request = client.fetchQuery(options);
  await started.promise;
  replaceProjectRoleQuery(client);
  response.resolve([{ id: "version-1", description_html: "private version" }]);
  await assert.rejects(request, /access changed/);
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("rejected page lock restores both visible records", async (t) => {
  const { projectPageActionOptions } = await import("../../apps/web/core/queries/pages.ts");
  const client = createClient(t);
  const unlocked = { ...page, is_locked: false };
  const detailKey = pageKeys.detail(scope, page.id);
  const listKey = pageKeys.project(scope);
  client.setQueryData(detailKey, unlocked);
  client.setQueryData(listKey, [unlocked]);
  t.mock.method(ProjectPageService.prototype, "lock", async () => {
    throw new Error("lock rejected");
  });
  const mutation = new MutationObserver(client, projectPageActionOptions(client, scope, page.id));
  t.after(() => mutation.reset());
  await assert.rejects(mutation.mutate("lock"), /lock rejected/);
  assert.equal(client.getQueryData(detailKey).is_locked, false);
  assert.equal(client.getQueryData(listKey)[0].is_locked, false);
});
