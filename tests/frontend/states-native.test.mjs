import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { createPermissionQueryClient } = await import("./helpers/permission-query-fixture.mjs");
const { webRequire } = await import("./helpers/react.mjs");
const { React, act } = await import("./helpers/react.mjs");
const { createQueryRouterFixture, settleQuery } = await import("./helpers/query-router.mjs");
const { MutationObserver } = webRequire("@tanstack/react-query");
const { EUserPermissions } = webRequire("@plane/constants");
const { EUserWorkspaceRoles } = webRequire("@plane/types");

const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const {
  currentStateRows,
  groupedStateRows,
  projectIntakeStateOptions,
  projectStateMutationOptions,
  projectStateOptions,
  projectStateProjection,
  stateOrderReadiness,
  stateKeys,
  stateRowsById,
  workspaceStateOptions,
  workspaceStateProjection,
} = await import("../../apps/web/core/queries/states.ts");
const { ProjectStateService } = await import("../../apps/web/core/services/project/project-state.service.ts");
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
const { useProjectStates } = await import("../../apps/web/core/hooks/use-project-states.ts");

const accountId = "account-a";
const workspaceSlug = "acme";
const projectId = "project-a";
const workspaceRole = EUserWorkspaceRoles.MEMBER;
const projectRole = EUserPermissions.ADMIN;
const membershipRevision = "membership-1";

function projectScope(overrides = {}) {
  return {
    accountId,
    workspaceSlug,
    projectId,
    workspaceRole,
    membershipRevision,
    projectRole,
    projectMembershipRole: projectRole,
    sessionGeneration: 0,
    ...overrides,
  };
}

function workspaceScope(overrides = {}) {
  return {
    accountId,
    workspaceSlug,
    workspaceRole,
    membershipRevision,
    projectRoles: [[projectId, projectRole]],
    sessionGeneration: 0,
    ...overrides,
  };
}

function createClient(t) {
  return createPermissionQueryClient(t, projectScope());
}

await test("project and intake reads forward cancellation and preserve canonical empty results", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  let projectSignal;
  let intakeSignal;
  t.mock.method(ProjectStateService.prototype, "getStates", async (_workspace, _project, signal) => {
    projectSignal = signal;
    return [];
  });
  t.mock.method(ProjectStateService.prototype, "getIntakeState", async (_workspace, _project, signal) => {
    intakeSignal = signal;
    return { id: "triage", project_id: projectId };
  });

  const states = await client.fetchQuery(projectStateOptions(client, scope));
  const intake = await client.fetchQuery(projectIntakeStateOptions(client, scope));

  assert.deepEqual(states, []);
  assert.equal(intake.id, "triage");
  assert.ok(projectSignal instanceof AbortSignal);
  assert.ok(intakeSignal instanceof AbortSignal);
});

await test("workspace query identity tracks the complete visible project-role snapshot", async (t) => {
  const { client, rolesKey } = await createClient(t);
  const firstScope = workspaceScope();
  const nextRoleMap = { [projectId]: projectRole, "project-b": EUserPermissions.MEMBER };
  const nextScope = workspaceScope({
    projectRoles: Object.entries(nextRoleMap).toSorted(([a], [b]) => a.localeCompare(b)),
  });
  t.mock.method(ProjectStateService.prototype, "getWorkspaceStates", async () => []);

  await client.fetchQuery(workspaceStateOptions(client, firstScope));
  client.setQueryData(rolesKey, nextRoleMap);
  await client.fetchQuery(workspaceStateOptions(client, nextScope));

  assert.notDeepEqual(stateKeys.workspace(firstScope), stateKeys.workspace(nextScope));
});

await test("workspace state projection expires when the visible project-role snapshot changes", async (t) => {
  const { client, rolesKey } = await createClient(t);
  const scope = workspaceScope();
  const states = [{ id: "state-a", name: "Started", project_id: projectId }];
  t.mock.method(ProjectStateService.prototype, "getWorkspaceStates", async () => states);
  await client.fetchQuery(workspaceStateOptions(client, scope));
  const query = client.getQueryState(stateKeys.workspace(scope));
  const projection = workspaceStateProjection(
    client,
    scope,
    { isSuccess: true, fetchStatus: "idle", dataUpdatedAt: query.dataUpdatedAt },
    stateRowsById(states)
  );

  assert.ok(projection?.isCurrent());
  client.setQueryData(rolesKey, { [projectId]: EUserPermissions.GUEST });
  assert.equal(projection?.isCurrent(), false);
});

await test("project query scope changes with role/session and failed reads cannot expose cached rows", async (t) => {
  const { client } = await createClient(t);
  const allowed = projectStateOptions(client, projectScope());
  const newSession = projectStateOptions(client, projectScope({ sessionGeneration: 1 }));
  const changedRole = projectStateOptions(client, projectScope({ projectRole: EUserPermissions.GUEST }));
  client.setQueryData(allowed.queryKey, [{ id: "private-state", name: "Stale", group: "started" }]);

  assert.notDeepEqual(allowed.queryKey, newSession.queryKey);
  assert.notDeepEqual(allowed.queryKey, changedRole.queryKey);
  assert.equal(currentStateRows({ isSuccess: false, data: client.getQueryData(allowed.queryKey) }), undefined);
  assert.deepEqual(currentStateRows({ isSuccess: true, data: [] }), []);
});

await test("project state response is discarded when project access changes during the read", async (t) => {
  const { client, rolesKey } = await createClient(t);
  const scope = projectScope();
  let finishRead;
  t.mock.method(ProjectStateService.prototype, "getStates", () => new Promise((resolve) => (finishRead = resolve)));

  const pendingRead = client.fetchQuery(projectStateOptions(client, scope));
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(typeof finishRead, "function");
  client.setQueryData(rolesKey, { [projectId]: EUserPermissions.GUEST });
  finishRead([{ id: "state-private", name: "Private", project_id: projectId }]);

  await assert.rejects(pendingRead, /scope changed or is unverified/);
  assert.equal(client.getQueryData(stateKeys.project(scope)), undefined);
});

await test("project state read rejects an authorization error retained with cached permission data", async (t) => {
  const { client, rolesKey } = await createClient(t);
  const scope = projectScope();
  const rolesQuery = client.getQueryCache().find({ queryKey: rolesKey, exact: true });
  rolesQuery.setState({ ...rolesQuery.state, status: "error", error: { status: 403 } });
  const read = t.mock.method(ProjectStateService.prototype, "getStates", async () => []);

  await assert.rejects(client.fetchQuery(projectStateOptions(client, scope)), /scope changed or is unverified/);
  assert.equal(read.mock.callCount(), 0);
});

await test("raw project membership changes invalidate a workspace-admin state read", async (t) => {
  const { client } = await createClient(t);
  const adminRolesKey = permissionKeys.workspaceProjectRoles(
    accountId,
    workspaceSlug,
    EUserWorkspaceRoles.ADMIN,
    membershipRevision
  );
  const scope = projectScope({
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    projectRole: EUserPermissions.ADMIN,
    projectMembershipRole: EUserPermissions.ADMIN,
  });
  client.setQueryData(permissionKeys.workspaceMembership(accountId, workspaceSlug), {
    is_active: true,
    role: EUserWorkspaceRoles.ADMIN,
    updated_at: membershipRevision,
  });
  client.setQueryData(adminRolesKey, { [projectId]: EUserPermissions.ADMIN });
  let finishRead;
  t.mock.method(ProjectStateService.prototype, "getStates", () => new Promise((resolve) => (finishRead = resolve)));

  const pendingRead = client.fetchQuery(projectStateOptions(client, scope));
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(typeof finishRead, "function");
  client.setQueryData(adminRolesKey, { [projectId]: EUserPermissions.MEMBER });
  finishRead([{ id: "state-private", project_id: projectId }]);

  await assert.rejects(pendingRead, /scope changed or is unverified/);
  assert.equal(client.getQueryData(stateKeys.project(scope)), undefined);
});

await test("state projection is invalidated when its canonical query data is replaced", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const key = stateKeys.project(scope);
  const states = [{ id: "state-a", name: "Started", project_id: projectId }];
  client.setQueryData(key, states, { updatedAt: 100 });
  const projection = projectStateProjection(
    client,
    scope,
    { isSuccess: true, fetchStatus: "idle", dataUpdatedAt: 100 },
    stateRowsById(states)
  );

  assert.ok(projection?.isCurrent());
  client.setQueryData(key, [{ ...states[0], name: "Review" }], { updatedAt: 101 });
  assert.equal(projection?.isCurrent(), false);
});

await test("state projection cannot adopt a replacement permission query with identical data", async (t) => {
  const { client, rolesKey } = await createClient(t);
  const scope = projectScope();
  const states = [{ id: "state-a", name: "Started", project_id: projectId }];
  const key = stateKeys.project(scope);
  client.setQueryData(key, states, { updatedAt: 100 });
  const projection = projectStateProjection(
    client,
    scope,
    { isSuccess: true, fetchStatus: "idle", dataUpdatedAt: 100 },
    stateRowsById(states)
  );

  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { [projectId]: projectRole });

  assert.equal(projection?.isCurrent(), false);
});

await test("reorder mutation preserves server rejection instead of silently succeeding", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  client.setQueryData(stateKeys.project(scope), [{ id: "state-a", name: "Started", project_id: projectId }]);
  const patch = t.mock.method(ProjectStateService.prototype, "patchState", async () => {
    throw new Error("state save rejected");
  });
  const mutation = new MutationObserver(client, projectStateMutationOptions(client, scope));

  await assert.rejects(
    mutation.mutate({ type: "reorder", stateId: "state-a", data: { sequence: 2 } }),
    /state save rejected/
  );
  assert.equal(patch.mock.callCount(), 1);
});

await test("accepted state writes invalidate the scoped query for a canonical refresh", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const key = stateKeys.project(scope);
  const original = [{ id: "state-a", name: "Started", project_id: projectId, sequence: 1 }];
  client.setQueryData(key, original);
  t.mock.method(ProjectStateService.prototype, "patchState", async (_workspace, _project, id, patch) => ({
    ...original[0],
    ...patch,
    id,
  }));
  const mutation = new MutationObserver(client, projectStateMutationOptions(client, scope));

  await mutation.mutate({ type: "patch", stateId: "state-a", data: { name: "Review" } });

  assert.deepEqual(client.getQueryData(key), original, "the mutation does not invent a local canonical record");
  assert.equal(
    client.getQueryState(key)?.isInvalidated,
    true,
    "the active reader will refresh server-owned state data"
  );
});

await test("accepted state writes cannot invalidate replacement intake or workspace query identities", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const state = { id: "state-a", name: "Started", project_id: projectId };
  client.setQueryData(stateKeys.project(scope), [state]);
  const intakeKey = stateKeys.intake(scope);
  const workspaceKey = stateKeys.workspace(workspaceScope());
  client.setQueryData(intakeKey, state);
  client.setQueryData(workspaceKey, [state]);
  let finishWrite;
  t.mock.method(ProjectStateService.prototype, "patchState", () => new Promise((resolve) => (finishWrite = resolve)));
  const mutation = new MutationObserver(client, projectStateMutationOptions(client, scope));

  const pending = mutation.mutate({ type: "patch", stateId: "state-a", data: { name: "Review" } });
  await new Promise((resolve) => setImmediate(resolve));
  for (const queryKey of [intakeKey, workspaceKey]) {
    client.removeQueries({ queryKey, exact: true });
    client.setQueryData(queryKey, queryKey === intakeKey ? state : [state]);
  }
  finishWrite(state);
  await pending;

  assert.equal(client.getQueryState(intakeKey)?.isInvalidated, false);
  assert.equal(client.getQueryState(workspaceKey)?.isInvalidated, false);
});

async function beginStatePatch(t) {
  const { client, rolesKey } = await createClient(t);
  const scope = projectScope();
  const key = stateKeys.project(scope);
  client.setQueryData(key, [{ id: "state-a", name: "Started", project_id: projectId }]);
  const response = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(ProjectStateService.prototype, "patchState", () => {
    started.resolve();
    return response.promise;
  });
  const reload = t.mock.method(ProjectStateService.prototype, "getStates", async () => []);
  const mutation = new MutationObserver(client, projectStateMutationOptions(client, scope));
  const pendingWrite = mutation.mutate({ type: "patch", stateId: "state-a", data: { name: "Review" } });
  await started.promise;
  return { client, rolesKey, key, pendingWrite, reload, finishWrite: response.resolve };
}

await test("state write completion after permission denial does not refetch retained data", async (t) => {
  const { client, rolesKey, key, pendingWrite, reload, finishWrite } = await beginStatePatch(t);
  const rolesQuery = client.getQueryCache().find({ queryKey: rolesKey, exact: true });
  rolesQuery.setState({ ...rolesQuery.state, status: "error", error: { status: 403 } });
  finishWrite({ id: "state-a", name: "Review", project_id: projectId });
  await pendingWrite;

  assert.equal(
    reload.mock.callCount(),
    0,
    "an accepted response after access denial cannot trigger a protected refetch"
  );
  assert.deepEqual(client.getQueryData(key), [{ id: "state-a", name: "Started", project_id: projectId }]);
});

await test("state write completion cannot adopt a replacement role query with identical data", async (t) => {
  const { client, rolesKey, key, pendingWrite, reload, finishWrite } = await beginStatePatch(t);
  const previousRoles = client.getQueryData(rolesKey);
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, previousRoles);
  finishWrite({ id: "state-a", name: "Review", project_id: projectId });
  await pendingWrite;

  assert.equal(reload.mock.callCount(), 0, "the write cannot capture permission lineage again after the response");
  assert.deepEqual(client.getQueryData(key), [{ id: "state-a", name: "Started", project_id: projectId }]);
});

await test("state writes reject a canonical response for another project", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  client.setQueryData(stateKeys.project(scope), []);
  t.mock.method(ProjectStateService.prototype, "createState", async () => ({ id: "foreign", project_id: "project-b" }));
  const mutation = new MutationObserver(client, projectStateMutationOptions(client, scope));

  await assert.rejects(mutation.mutate({ type: "create", data: { name: "Foreign" } }), /different project/);
});

await test("project state selectors preserve sorting and all workflow groups", () => {
  const grouped = groupedStateRows([
    { id: "state-2", group: "started", sequence: 2 },
    { id: "state-1", group: "started", sequence: 1 },
  ]);

  assert.deepEqual(
    grouped.started.map((state) => state.id),
    ["state-1", "state-2"]
  );
  assert.ok(grouped.backlog);
  assert.ok(grouped.unstarted);
  assert.ok(grouped.completed);
  assert.ok(grouped.cancelled);
});

await test("state-name ordering waits for reads, exposes retryable failures, and settles on denied scope", () => {
  assert.deepEqual(stateOrderReadiness(true, true, false, false, true, false), {
    isPending: true,
    isError: false,
  });
  assert.deepEqual(stateOrderReadiness(true, false, false, false, false, true), {
    isPending: false,
    isError: true,
  });
  assert.deepEqual(stateOrderReadiness(true, false, false, true, true, false), {
    isPending: false,
    isError: false,
  });
});

await test("current-user authorization failure hides cached state rows and retry restores them", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const scope = projectScope();
  fixture.client.setQueryData(identityKeys.currentUser(), { id: accountId });
  fixture.client.setQueryData(permissionKeys.workspaceMembership(accountId, workspaceSlug), {
    is_active: true,
    role: workspaceRole,
    updated_at: membershipRevision,
  });
  fixture.client.setQueryData(
    permissionKeys.workspaceProjectRoles(accountId, workspaceSlug, workspaceRole, membershipRevision),
    { [projectId]: projectRole }
  );
  fixture.client.setQueryData(stateKeys.project(scope), [
    { id: "cached-state", name: "Cached", project_id: projectId },
  ]);
  let shouldFail = true;
  t.mock.method(userService, "currentUser", async () => {
    if (shouldFail) throw Object.assign(new Error("unauthorized"), { status: 401 });
    return { id: accountId };
  });
  function Probe() {
    const state = useProjectStates(workspaceSlug, projectId);
    return React.createElement(
      "div",
      null,
      React.createElement("button", { onClick: () => void state.retryAccess() }, "Retry"),
      JSON.stringify({
        count: state.states?.length ?? null,
        cachedStateVisible: Boolean(state.getStateById("cached-state")),
        accessError: state.accessError,
        accessPending: state.accessPending,
      })
    );
  }
  await fixture.render(React.createElement(Probe));
  assert.match(fixture.environment.container.textContent, /"count":1/);

  await assert.rejects(fixture.client.fetchQuery({ ...currentUserOptions(), staleTime: 0 }), /unauthorized/);
  await settleQuery();
  assert.match(fixture.environment.container.textContent, /"count":null/);
  assert.match(fixture.environment.container.textContent, /"cachedStateVisible":false/);
  assert.match(fixture.environment.container.textContent, /"accessError":true/);
  assert.match(fixture.environment.container.textContent, /"accessPending":false/);

  shouldFail = false;
  await act(async () => {
    fixture.environment.container
      .querySelector("button")
      .dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
  });
  await settleQuery();
  assert.match(fixture.environment.container.textContent, /"count":1/);
  assert.match(fixture.environment.container.textContent, /"accessError":false/);
});
