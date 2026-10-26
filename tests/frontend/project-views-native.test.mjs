import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { QueryClient, MutationObserver } = webRequire("@tanstack/react-query");
const { EUserPermissions, EUserWorkspaceRoles } = webRequire("@plane/types");
const { seedPermissionQueries } = await import("./helpers/permission-query-fixture.mjs");
const { ViewService } = await import("../../apps/web/core/services/view.service.ts");
const { advanceIdentitySession, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { projectViewKeys, projectViewMutationOptions, projectViewOptions, projectViewsOptions } =
  await import("../../apps/web/core/queries/project-views.ts");

async function createFixture(
  t,
  { workspaceRole = EUserWorkspaceRoles.MEMBER, projectRole = EUserPermissions.MEMBER } = {}
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false, gcTime: Infinity } },
  });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole,
    projectRoles: { p1: projectRole },
    revision: "membership-1",
  });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "p1",
    membershipRole: workspaceRole,
    membershipRevision: "membership-1",
    projectMembershipRole: projectRole,
    projectRole,
    sessionGeneration: 0,
  };
  return { client, scope };
}

await test("project view lists are canonical and scoped to verified project access", async (t) => {
  const { client, scope } = await createFixture(t);
  const canonical = [{ id: "view-a", project: "p1", name: "Shared" }];
  const read = t.mock.method(ViewService.prototype, "getViews", async (workspace, project) => {
    assert.equal(workspace, "acme");
    assert.equal(project, "p1");
    return canonical;
  });

  const views = await client.fetchQuery(projectViewsOptions(client, scope));
  assert.deepEqual(views, canonical);
  assert.equal(read.mock.callCount(), 1);
  assert.deepEqual(client.getQueryData(projectViewKeys.list(scope)), canonical);
});

await test("a permission replacement prevents an in-flight project view list from committing", async (t) => {
  const { client, scope } = await createFixture(t);
  const deferred = Promise.withResolvers();
  t.mock.method(ViewService.prototype, "getViews", () => deferred.promise);
  const request = client.fetchQuery(projectViewsOptions(client, scope));
  const rejected = assert.rejects(request, /access changed while loading/);
  client.setQueryData(
    permissionKeys.workspaceProjectRoles("alice", "acme", scope.membershipRole, scope.membershipRevision),
    {
      p1: EUserPermissions.GUEST,
    }
  );
  deferred.resolve([{ id: "stale", project: "p1" }]);
  await rejected;
  assert.equal(client.getQueryData(projectViewKeys.list(scope)), undefined);
});

await test("create reconciles the accepted server record and invalidates the active list", async (t) => {
  const { client, scope } = await createFixture(t);
  const listKey = projectViewKeys.list(scope);
  client.setQueryData(listKey, []);
  const canonical = {
    id: "created",
    project: "p1",
    name: "Saved",
    owned_by: "alice",
    display_filters: { layout: "kanban", group_by: "state" },
  };
  const create = t.mock.method(ViewService.prototype, "createView", async (_workspace, _project, data) => {
    assert.equal(data.display_filters.group_by, "state");
    return canonical;
  });
  const mutation = new MutationObserver(client, projectViewMutationOptions.create(client, scope, 0));
  const input = { name: "Saved", display_filters: { layout: "kanban", group_by: null } };
  const result = await mutation.mutate(input);
  assert.equal(result, canonical);
  assert.equal(create.mock.callCount(), 1);
  assert.equal(input.display_filters.group_by, null);
  assert.deepEqual(client.getQueryData(listKey), [canonical]);
  assert.equal(client.getQueryData(projectViewKeys.detail(scope, "created")), canonical);
});

await test("project guests can create views when they have active project membership", async (t) => {
  const { client, scope } = await createFixture(t, { projectRole: EUserPermissions.GUEST });
  const canonical = { id: "guest-created", project: "p1", owned_by: "alice" };
  const create = t.mock.method(ViewService.prototype, "createView", async () => canonical);
  const mutation = new MutationObserver(client, projectViewMutationOptions.create(client, scope, 0));

  assert.equal(await mutation.mutate({ name: "Guest view" }), canonical);
  assert.equal(create.mock.callCount(), 1);
});

await test("a failed detail read does not fall back to a cached list row", async (t) => {
  const { client, scope } = await createFixture(t);
  const cachedList = [{ id: "view-a", project: "p1", name: "Cached list copy" }];
  client.setQueryData(projectViewKeys.list(scope), cachedList);
  t.mock.method(ViewService.prototype, "getViewDetails", async () => {
    throw new Error("view is private");
  });

  await assert.rejects(client.fetchQuery(projectViewOptions(client, scope, "view-a")), /view is private/);
  assert.deepEqual(client.getQueryData(projectViewKeys.list(scope)), cachedList);
  assert.equal(client.getQueryData(projectViewKeys.detail(scope, "view-a")), undefined);
});

await test("an accepted owner update replaces the detail and list with the server record", async (t) => {
  const { client, scope } = await createFixture(t);
  const saved = { id: "view-a", project: "p1", owned_by: "alice", is_locked: false, name: "Before" };
  const canonical = { ...saved, name: "Canonical" };
  client.setQueryData(projectViewKeys.list(scope), [saved]);
  t.mock.method(ViewService.prototype, "getViewDetails", async () => saved);
  const update = t.mock.method(ViewService.prototype, "patchView", async () => canonical);
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));

  await mutation.mutate({ viewId: saved.id, data: { name: "Submitted" } });
  assert.equal(update.mock.callCount(), 1);
  assert.deepEqual(client.getQueryData(projectViewKeys.detail(scope, saved.id)), canonical);
  assert.deepEqual(client.getQueryData(projectViewKeys.list(scope)), [canonical]);
});

await test("a non-owner cannot update a view, even with a project admin role", async (t) => {
  const { client, scope } = await createFixture(t, { projectRole: EUserPermissions.ADMIN });
  const shared = { id: "view-a", project: "p1", owned_by: "bob", is_locked: false, name: "Shared" };
  t.mock.method(ViewService.prototype, "getViewDetails", async () => shared);
  const patch = t.mock.method(ViewService.prototype, "patchView", async () => shared);
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));

  await assert.rejects(mutation.mutate({ viewId: shared.id, data: { name: "Changed" } }), /Only the owner/);
  assert.equal(patch.mock.callCount(), 0);
});

async function beginViewUpdate(t) {
  const { client, scope } = await createFixture(t);
  const saved = { id: "view-a", project: "p1", owned_by: "alice", is_locked: false, name: "Before" };
  client.setQueryData(projectViewKeys.list(scope), [saved]);
  client.setQueryData(projectViewKeys.detail(scope, saved.id), saved);
  t.mock.method(ViewService.prototype, "getViewDetails", async () => saved);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(ViewService.prototype, "patchView", () => {
    started.resolve();
    return response.promise;
  });
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));
  const request = mutation.mutate({ viewId: saved.id, data: { name: "Submitted" } });
  await started.promise;
  return { client, scope, saved, response, request };
}

await test("an accepted update does not reconcile into a replacement identity scope", async (t) => {
  const { client, scope, saved, response, request } = await beginViewUpdate(t);
  advanceIdentitySession(client);
  response.resolve({ ...saved, name: "Accepted" });
  await request;

  assert.deepEqual(client.getQueryData(projectViewKeys.detail(scope, saved.id)), saved);
});

await test("an accepted update does not replace a recreated detail query", async (t) => {
  const { client, scope } = await createFixture(t);
  const saved = { id: "view-a", project: "p1", owned_by: "alice", is_locked: false, name: "Before" };
  const replacement = { ...saved, name: "New query owner" };
  const response = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(ViewService.prototype, "getViewDetails", async () => saved);
  t.mock.method(ViewService.prototype, "patchView", () => {
    started.resolve();
    return response.promise;
  });
  client.setQueryData(projectViewKeys.list(scope), [saved]);
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));
  const request = mutation.mutate({ viewId: saved.id, data: { name: "Submitted" } });
  await started.promise;
  const detailKey = projectViewKeys.detail(scope, saved.id);
  const oldQuery = client.getQueryCache().find({ queryKey: detailKey, exact: true });
  client.removeQueries({ queryKey: detailKey, exact: true });
  client.setQueryData(detailKey, replacement);
  const newQuery = client.getQueryCache().find({ queryKey: detailKey, exact: true });
  assert.notEqual(newQuery, oldQuery);

  response.resolve({ ...saved, name: "Accepted" });
  await request;
  assert.deepEqual(client.getQueryData(detailKey), replacement);
});

await test("an accepted update cancels a stale list response before reconciling", async (t) => {
  const { client, scope } = await createFixture(t);
  const saved = { id: "view-a", project: "p1", owned_by: "alice", is_locked: false, name: "Before" };
  const canonical = { ...saved, name: "Accepted" };
  const listStarted = Promise.withResolvers();
  const staleList = Promise.withResolvers();
  const patchStarted = Promise.withResolvers();
  const patchResponse = Promise.withResolvers();
  t.mock.method(ViewService.prototype, "getViews", () => {
    listStarted.resolve();
    return staleList.promise;
  });
  t.mock.method(ViewService.prototype, "getViewDetails", async () => saved);
  t.mock.method(ViewService.prototype, "patchView", () => {
    patchStarted.resolve();
    return patchResponse.promise;
  });
  client.setQueryData(projectViewKeys.list(scope), [saved]);
  const staleRequest = client.fetchQuery({ ...projectViewsOptions(client, scope), staleTime: 0 });
  await listStarted.promise;
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));
  const update = mutation.mutate({ viewId: saved.id, data: { name: "Submitted" } });
  await patchStarted.promise;
  patchResponse.resolve(canonical);
  await update;
  staleList.resolve([{ id: "stale", project: "p1", name: "Stale" }]);
  await staleRequest.catch(() => undefined);
  assert.deepEqual(client.getQueryData(projectViewKeys.list(scope)), [canonical]);
});

await test("a rejected owner update leaves the canonical view untouched", async (t) => {
  const { client, scope } = await createFixture(t);
  const saved = { id: "view-a", project: "p1", owned_by: "alice", is_locked: false, name: "Before" };
  client.setQueryData(projectViewKeys.list(scope), [saved]);
  client.setQueryData(projectViewKeys.detail(scope, saved.id), saved);
  t.mock.method(ViewService.prototype, "getViewDetails", async () => saved);
  t.mock.method(ViewService.prototype, "patchView", async () => {
    throw new Error("offline");
  });
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));

  await assert.rejects(mutation.mutate({ viewId: saved.id, data: { name: "Submitted" } }), /offline/);
  assert.deepEqual(client.getQueryData(projectViewKeys.detail(scope, saved.id)), saved);
  assert.deepEqual(client.getQueryData(projectViewKeys.list(scope)), [saved]);
});

await test("a role replacement during an accepted update prevents stale-scope reconciliation", async (t) => {
  const { client, scope, saved, response, request } = await beginViewUpdate(t);
  client.setQueryData(
    permissionKeys.workspaceProjectRoles("alice", "acme", scope.membershipRole, scope.membershipRevision),
    { p1: EUserPermissions.GUEST }
  );
  response.resolve({ ...saved, name: "Accepted" });
  await request;

  assert.deepEqual(client.getQueryData(projectViewKeys.detail(scope, saved.id)), saved);
  assert.deepEqual(client.getQueryData(projectViewKeys.list(scope)), [saved]);
});

await test("mismatched canonical update responses are rejected", async (t) => {
  const { client, scope } = await createFixture(t);
  const saved = { id: "view-a", project: "p1", owned_by: "alice", is_locked: false, name: "Before" };
  client.setQueryData(projectViewKeys.detail(scope, saved.id), saved);
  t.mock.method(ViewService.prototype, "getViewDetails", async () => saved);
  t.mock.method(ViewService.prototype, "patchView", async () => ({
    ...saved,
    project: "other-project",
    name: "Wrong",
  }));
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));

  await assert.rejects(mutation.mutate({ viewId: saved.id, data: { name: "Submitted" } }), /different project view/);
  assert.deepEqual(client.getQueryData(projectViewKeys.detail(scope, saved.id)), saved);
});

await test("same-scope saves serialize and the later accepted save remains canonical", async (t) => {
  const { client, scope } = await createFixture(t);
  const saved = { id: "view-a", project: "p1", owned_by: "alice", is_locked: false, name: "Before" };
  client.setQueryData(projectViewKeys.list(scope), [saved]);
  client.setQueryData(projectViewKeys.detail(scope, saved.id), saved);
  t.mock.method(ViewService.prototype, "getViewDetails", async () => saved);
  const firstResponse = Promise.withResolvers();
  const secondResponse = Promise.withResolvers();
  const firstStarted = Promise.withResolvers();
  const secondStarted = Promise.withResolvers();
  const patch = t.mock.method(ViewService.prototype, "patchView", (_workspace, _project, _id, data) => {
    if (data.name === "First") {
      firstStarted.resolve();
      return firstResponse.promise;
    }
    secondStarted.resolve();
    return secondResponse.promise;
  });
  const mutation = new MutationObserver(client, projectViewMutationOptions.update(client, scope, 0));
  const first = mutation.mutate({ viewId: saved.id, data: { name: "First" } });
  await firstStarted.promise;
  const second = mutation.mutate({ viewId: saved.id, data: { name: "Second" } });
  assert.equal(patch.mock.callCount(), 1);
  firstResponse.resolve({ ...saved, name: "First" });
  await first;
  await secondStarted.promise;
  secondResponse.resolve({ ...saved, name: "Second" });
  await second;

  assert.deepEqual(client.getQueryData(projectViewKeys.detail(scope, saved.id)), { ...saved, name: "Second" });
  assert.deepEqual(client.getQueryData(projectViewKeys.list(scope)), [{ ...saved, name: "Second" }]);
});
