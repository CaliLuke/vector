import { projectIssueScope } from "./helpers/issue-query-fixtures.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { QueryClient } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedIssueQueryScope } from "./helpers/issue-query-fixtures.mjs";

const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
registerWebModuleHooks();
const { IssueService } = await fromRoot("apps/web/core/services/issue/index.ts");
const { ModuleService } = await fromRoot("apps/web/core/services/module.service.ts");
const { IssueArchiveService } = await fromRoot("apps/web/core/services/issue/issue_archive.service.ts");
const { UserService } = await fromRoot("apps/web/core/services/user.service.ts");
const { identityKeys } = await fromRoot("apps/web/core/queries/identity.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { issueListGroupOptions, issueListKeys, issueListOptions } =
  await import("../../apps/web/core/queries/issue-lists.ts");
const { currentIssueListMutationOwner, issueListMutation, patchIssueMutation } =
  await import("../../apps/web/core/queries/issue-list-mutations.ts");

const scope = projectIssueScope();
const issue = (id, projectId = "p1") => ({ id, project_id: projectId, state_id: "state-1" });
const response = (results, totalCount = 2) => ({
  grouped_by: "state",
  next_cursor: "20:1:0",
  prev_cursor: "",
  next_page_results: true,
  prev_page_results: false,
  total_count: totalCount,
  count: 1,
  total_pages: 2,
  extra_stats: null,
  total_results: totalCount,
  results,
});
function read(overrides = {}) {
  const input = {
    workspaceSlug: "acme",
    projectId: "p1",
    richFilters: {},
    displayFilters: { layout: "list", group_by: "state", sub_group_by: null },
  };
  return {
    scope,
    pagination: { canGroup: true, perPageCount: 20, groupedBy: "state" },
    request: { family: "project", input, isReadCurrent: () => true },
    ...overrides,
  };
}
function queryClient() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedIssueQueryScope(client, scope);
  return client;
}
function projectRolesKey() {
  return permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.workspaceRole,
    scope.membershipRevision
  );
}

async function pendingListRead(t, client) {
  const started = Promise.withResolvers(),
    finish = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    started.resolve();
    return finish.promise;
  });
  const pending = client.fetchQuery(issueListOptions(read()));
  await started.promise;
  return { finish, pending };
}
async function pendingGroupPage(t, client, listRead) {
  const started = Promise.withResolvers(),
    finish = Promise.withResolvers();
  let calls = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    if (++calls === 1) return response({ "state-1": { results: [issue("first")], total_results: 2 } }, 2);
    started.resolve();
    return finish.promise;
  });
  const fullRead = await client.fetchQuery(issueListOptions(listRead));
  const options = issueListGroupOptions(listRead, fullRead, { groupId: "state-1" });
  client.getQueryCache().build(client, options);
  const page = options.queryFn({ client, signal: new AbortController().signal, pageParam: "1:1:0" });
  await started.promise;
  return { finish, page };
}

await test("project list Query sends cancellation and keeps only the authorized project response", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  let signal;
  t.mock.method(IssueService.prototype, "getIssues", async (_workspace, _project, _params, options) => {
    signal = options.signal;
    return response({ "state-1": { results: [issue("i1")], total_results: 2 } });
  });

  const result = await client.fetchQuery(issueListOptions(read()));
  assert.equal(result.response.results["state-1"].results[0].id, "i1");
  assert.equal(signal instanceof AbortSignal, true);
});

await test("a replaced permission Query cannot authorize an in-flight list response", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const { finish, pending } = await pendingListRead(t, client);
  const rolesKey = projectRolesKey();
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { p1: 20 });
  finish.resolve(response({ "state-1": { results: [issue("i1")], total_results: 1 } }, 1));
  await assert.rejects(pending, /access or list preferences changed during the read/);
});

await test("group pages have independent Query identities and seed from the accepted full read", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  t.mock.method(IssueService.prototype, "getIssues", async () =>
    response(
      {
        "state-1": { results: [issue("i1")], total_results: 2 },
        "state-2": { results: [issue("i2")], total_results: 1 },
      },
      3
    )
  );
  const readInput = read();
  const fullRead = await client.fetchQuery(issueListOptions(readInput));
  const stateOne = issueListGroupOptions(readInput, fullRead, { groupId: "state-1" });
  const stateTwo = issueListGroupOptions(readInput, fullRead, { groupId: "state-2" });
  const nextGeneration = issueListGroupOptions(
    readInput,
    { ...fullRead, generation: "g-next" },
    { groupId: "state-1" }
  );

  assert.notDeepEqual(stateOne.queryKey, stateTwo.queryKey);
  assert.notDeepEqual(stateOne.queryKey, nextGeneration.queryKey);
  assert.equal(stateOne.initialData.pages[0].results["state-1"].results[0].id, "i1");
});

await test("a saved PATCH returns its canonical row and refreshes only the captured list owner", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const listQueryKey = issueListKeys.list(listRead);
  const listQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "g1", response: response([issue("i1")], 1) });
  let requestCount = 0;
  const saved = { ...issue("i1"), type_id: "bug" };
  t.mock.method(IssueService.prototype, "patchIssue", async () => {
    requestCount += 1;
    return saved;
  });
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  const result = await patchIssueMutation(client, owner, "i1", { type_id: "bug" });

  assert.equal(requestCount, 1);
  assert.equal(result, saved);
  assert.equal(client.getQueryCache().find({ queryKey: listQueryKey, exact: true }), listQuery);
  assert.equal(listQuery.state.isInvalidated, true);
});

await test("a replaced permission Query cannot receive an accepted PATCH list refresh", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const rolesKey = projectRolesKey();
  const roleQuery = client.getQueryCache().find({ queryKey: rolesKey, exact: true });
  const listRead = read();
  const listQueryKey = issueListKeys.list(listRead);
  const listQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "g1", response: response([issue("i1")], 1) });
  const started = Promise.withResolvers();
  const finish = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "patchIssue", async () => {
    started.resolve();
    return finish.promise;
  });
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);
  const pending = patchIssueMutation(client, owner, "i1", { name: "Updated" });
  await started.promise;
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { p1: scope.projectMembershipRole });
  finish.resolve({ ...issue("i1"), name: "Updated" });

  assert.equal((await pending).name, "Updated");
  assert.notEqual(client.getQueryCache().find({ queryKey: rolesKey, exact: true }), roleQuery);
  assert.equal(listQuery.state.isInvalidated, false);
});

await test("bulk module assignment uses the canonical service and refreshes its captured list", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const listQueryKey = issueListKeys.list(listRead);
  const listQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "g1", response: response([issue("i1")], 1) });
  let call;
  t.mock.method(ModuleService.prototype, "addIssuesToModule", async (...args) => {
    call = args;
  });
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  await issueListMutation(client, owner, { kind: "module-add", moduleId: "m1", issueIds: ["i1", "i2"] });

  assert.deepEqual(call, ["acme", "p1", "m1", { issues: ["i1", "i2"] }]);
  assert.equal(listQuery.state.isInvalidated, true);
});

await test("archive and restore writes invalidate the captured native issue list", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listQueryKey = issueListKeys.list(read());
  const listQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "g1", response: response([issue("i1")], 1) });
  const archives = [];
  t.mock.method(IssueArchiveService.prototype, "archiveIssue", async (...args) => {
    archives.push(["archive", ...args]);
    return { archived_at: "2026-10-26T00:00:00Z" };
  });
  t.mock.method(IssueArchiveService.prototype, "restoreIssue", async (...args) => {
    archives.push(["restore", ...args]);
  });
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  await issueListMutation(client, owner, { kind: "archive", issueId: "i1" });
  assert.equal(listQuery.state.isInvalidated, true);
  client.setQueryData(listQueryKey, { generation: "g2", response: response([issue("i1")], 1) });
  await issueListMutation(client, owner, { kind: "restore", issueId: "i1" });

  assert.deepEqual(archives, [
    ["archive", "acme", "p1", "i1"],
    ["restore", "acme", "p1", "i1"],
  ]);
  assert.equal(listQuery.state.isInvalidated, true);
});

await test("a list read cannot use cached identity after its identity Query enters an error state", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const userQuery = client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  userQuery.setState({ ...userQuery.state, status: "error", error: new Error("identity refresh failed") });
  const getIssues = t.mock.method(IssueService.prototype, "getIssues", async () => response([issue("i1")], 1));

  await assert.rejects(client.fetchQuery(issueListOptions(read())), /Work item access or list preferences changed/);
  assert.equal(getIssues.mock.callCount(), 0);
});

await test("replacing the identity Query discards the in-flight native list response", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const { finish, pending } = await pendingListRead(t, client);
  client.removeQueries({ queryKey: identityKeys.currentUser(), exact: true });
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  finish.resolve(response([issue("late")], 1));

  await assert.rejects(pending, /Work item access or list preferences changed during the read/);
  assert.equal(client.getQueryData(issueListKeys.list(read())), undefined);
});

await test("a native group page is discarded when its captured project permission changes", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const { finish, page } = await pendingGroupPage(t, client, listRead);
  const rolesKey = projectRolesKey();
  client.setQueryData(rolesKey, {});
  finish.resolve(response({ "state-1": { results: [issue("late")], total_results: 1 } }, 1));

  await assert.rejects(page, /access or list changed during pagination/);
});

await test("native group pagination does not start after project permission verification becomes stale", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  let calls = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    calls += 1;
    return response({ "state-1": { results: [issue("first")], total_results: 2 } }, 2);
  });
  const fullRead = await client.fetchQuery(issueListOptions(listRead));
  const options = issueListGroupOptions(listRead, fullRead, { groupId: "state-1" });
  client.getQueryCache().build(client, options);
  const rolesKey = projectRolesKey();
  client.setQueryData(rolesKey, {});

  await assert.rejects(
    options.queryFn({ client, signal: new AbortController().signal, pageParam: "1:1:0" }),
    /list changed before pagination/
  );
  assert.equal(calls, 1);
});

await test("native group pagination rejects a response after its immutable request input becomes stale", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  let current = true;
  const listRead = read({ request: { ...read().request, isReadCurrent: () => current } });
  const { finish, page } = await pendingGroupPage(t, client, listRead);
  current = false;
  finish.resolve(response({ "state-1": { results: [issue("late")], total_results: 1 } }, 1));

  await assert.rejects(page, /access or list changed during pagination/);
});

await test("an older full response cannot replace the current immutable request Query", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  let oldCurrent = true;
  const oldRead = read({ request: { ...read().request, isReadCurrent: () => oldCurrent } });
  const currentRead = read({
    request: { ...read().request, input: { ...read().request.input, richFilters: { priority: "urgent" } } },
  });
  const oldStarted = Promise.withResolvers();
  const oldFinish = Promise.withResolvers();
  const currentStarted = Promise.withResolvers();
  const currentFinish = Promise.withResolvers();
  let calls = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    calls += 1;
    if (calls === 1) {
      oldStarted.resolve();
      return oldFinish.promise;
    }
    currentStarted.resolve();
    return currentFinish.promise;
  });
  const oldPending = client.fetchQuery(issueListOptions(oldRead));
  await oldStarted.promise;
  oldCurrent = false;
  const currentPending = client.fetchQuery(issueListOptions(currentRead));
  await currentStarted.promise;
  currentFinish.resolve(response([issue("current")], 1));
  await currentPending;
  oldFinish.resolve(response([issue("obsolete")], 1));

  await assert.rejects(oldPending, /access or list preferences changed during the read/);
  assert.deepEqual(
    client.getQueryData(issueListKeys.list(currentRead)).response.results.map((row) => row.id),
    ["current"]
  );
  assert.equal(client.getQueryData(issueListKeys.list(oldRead)), undefined);
});

await test("a failed native PATCH leaves its captured Query response unchanged for retry", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const listQueryKey = issueListKeys.list(listRead);
  const listQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  const original = { generation: "g1", response: response([issue("i1")], 1) };
  client.setQueryData(listQueryKey, original);
  t.mock.method(IssueService.prototype, "patchIssue", async () => {
    throw new Error("write failed");
  });
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  await assert.rejects(
    issueListMutation(client, owner, { kind: "patch", issueId: "i1", data: { priority: "high" } }),
    /write failed/
  );

  assert.equal(client.getQueryData(listQueryKey), original);
  assert.equal(listQuery.state.isInvalidated, false);
});

await test("an accepted native create invalidates its exact captured list owner", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const listQueryKey = issueListKeys.list(listRead);
  const listQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "g1", response: response([], 0) });
  const created = { ...issue("i2"), name: "Created" };
  let args;
  t.mock.method(IssueService.prototype, "createIssue", async (...input) => {
    args = input;
    return created;
  });
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  assert.equal(await issueListMutation(client, owner, { kind: "create", data: { name: "Created" } }), created);

  assert.deepEqual(args, ["acme", "p1", { name: "Created" }]);
  assert.equal(client.getQueryCache().find({ queryKey: listQueryKey, exact: true }), listQuery);
  assert.equal(listQuery.state.isInvalidated, true);
});

await test("an accepted create cancels a captured in-flight list read before its stale receipt can commit", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const started = Promise.withResolvers();
  const finishRead = Promise.withResolvers();
  let readSignal;
  t.mock.method(IssueService.prototype, "getIssues", async (_workspace, _project, _params, options) => {
    readSignal = options.signal;
    started.resolve();
    return finishRead.promise;
  });
  const pendingRead = client.fetchQuery(issueListOptions(listRead)).catch(() => undefined);
  await started.promise;
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);
  t.mock.method(IssueService.prototype, "createIssue", async () => ({ ...issue("created"), name: "Created" }));

  await issueListMutation(client, owner, { kind: "create", data: { name: "Created" } });
  assert.equal(readSignal.aborted, true);
  finishRead.resolve(response([issue("late")], 1));
  await pendingRead;
  assert.equal(client.getQueryData(issueListKeys.list(listRead)), undefined);
});

await test("a label-catalogue revision requires its own guarded full-list Query", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const firstRead = read({ catalogueRevision: "labels-1" });
  const secondRead = read({ catalogueRevision: "labels-2" });
  let calls = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    calls += 1;
    return response([issue(`i${calls}`)], 1);
  });

  const first = await client.fetchQuery(issueListOptions(firstRead));
  const second = await client.fetchQuery(issueListOptions(secondRead));

  assert.notDeepEqual(issueListKeys.list(firstRead), issueListKeys.list(secondRead));
  assert.equal(calls, 2);
  assert.equal(first.response.results[0].id, "i1");
  assert.equal(second.response.results[0].id, "i2");
});

await test("an accepted project create invalidates its captured workspace all-issues Query", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const workspaceScope = {
    accountId: scope.accountId,
    sessionGeneration: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    workspaceRole: scope.workspaceRole,
    membershipRevision: scope.membershipRevision,
    projectRoles: [[scope.projectId, scope.projectMembershipRole]],
  };
  const input = {
    kind: "workspace",
    workspaceSlug: scope.workspaceSlug,
    viewId: "all-issues",
    accountId: scope.accountId,
    richFilters: {},
    displayFilters: { layout: "list", group_by: "state" },
    isScopeCurrent: () => true,
    isReadScopeCurrent: () => true,
  };
  const workspaceRead = {
    scope: workspaceScope,
    pagination: { canGroup: true, perPageCount: 20, groupedBy: "state" },
    request: { family: "workspace", input, isReadCurrent: () => true },
  };
  const listQueryKey = issueListKeys.list(workspaceRead);
  const listQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "g1", response: response([], 0) });
  t.mock.method(IssueService.prototype, "createIssue", async () => ({ ...issue("created"), name: "Created" }));
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  await issueListMutation(client, owner, { kind: "create", data: { name: "Created" } });

  assert.equal(client.getQueryCache().find({ queryKey: listQueryKey, exact: true }), listQuery);
  assert.equal(listQuery.state.isInvalidated, true);
});

async function testParentPagination(kind) {
  await test(`${kind} list Queries keep their parent filter through native group pagination`, async (t) => {
    const client = queryClient();
    t.after(() => client.clear());
    const parentId = `${kind}-1`;
    const input = {
      kind,
      workspaceSlug: scope.workspaceSlug,
      projectId: scope.projectId,
      viewId: parentId,
      accountId: scope.accountId,
      richFilters: {},
      displayFilters: { layout: "list", group_by: "state", order_by: "-created_at", sub_issue: true },
      isScopeCurrent: () => true,
      isReadScopeCurrent: () => true,
    };
    const listRead = {
      scope,
      pagination: { canGroup: true, perPageCount: 1, groupedBy: "state" },
      request: { family: kind, input, isReadCurrent: () => true },
    };
    const requests = [];
    t.mock.method(IssueService.prototype, "getIssues", async (_workspace, _project, params) => {
      requests.push(params);
      return response({ "state-1": { results: [issue(`i${requests.length}`)], total_results: 2 } }, 2);
    });
    const fullRead = await client.fetchQuery(issueListOptions(listRead));
    const options = issueListGroupOptions(listRead, fullRead, { groupId: "state-1" });
    client.getQueryCache().build(client, options);
    await options.queryFn({ client, signal: new AbortController().signal, pageParam: "1:1:0" });

    assert.equal(requests[0][kind], parentId);
    assert.equal(requests[1][kind], parentId);
    assert.equal(requests[1].cursor, "1:1:0");
  });
}
await testParentPagination("cycle");
await testParentPagination("module");

await test("archive and profile lists use scoped native Query transport with family-specific access owners", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  let archiveParams;
  let profileParams;
  t.mock.method(IssueArchiveService.prototype, "getArchivedIssues", async (_workspace, _project, params) => {
    archiveParams = params;
    return response([issue("archived")], 1);
  });
  t.mock.method(UserService.prototype, "getUserProfileIssues", async (_workspace, _user, params) => {
    profileParams = params;
    return response([issue("profile")], 1);
  });
  const archivedInput = {
    workspaceSlug: "acme",
    projectId: "p1",
    accountId: "alice",
    richFilters: {},
    displayFilters: { layout: "list", order_by: "-created_at" },
    isCurrent: () => true,
    isReadScopeCurrent: () => true,
  };
  const archivedRead = {
    scope,
    pagination: { canGroup: false, perPageCount: 20 },
    request: { family: "archived", input: archivedInput, isReadCurrent: () => true },
  };
  const workspaceScope = {
    accountId: scope.accountId,
    sessionGeneration: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    workspaceRole: scope.workspaceRole,
    membershipRevision: scope.membershipRevision,
    projectRoles: [["p1", 20]],
  };
  const profileInput = {
    workspaceSlug: "acme",
    userId: "alice",
    accountId: "alice",
    view: "assigned",
    richFilters: {},
    displayFilters: { layout: "list" },
    isCurrent: () => true,
    isReadScopeCurrent: () => true,
  };
  const profileRead = {
    scope: workspaceScope,
    pagination: { canGroup: false, perPageCount: 20 },
    request: { family: "profile", input: profileInput, isReadCurrent: () => true },
  };

  await client.fetchQuery(issueListOptions(archivedRead));
  await client.fetchQuery(issueListOptions(profileRead));

  assert.equal(archiveParams.per_page, "20");
  assert.equal(profileParams.assignees, "alice");
});

await test("an estimate catalogue retry leaves the captured issue-list Query and transport untouched", async (t) => {
  const { estimateKeys } = await import("../../apps/web/core/queries/estimates.ts");
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const estimateScope = {
    kind: "project",
    accountId: scope.accountId,
    sessionGeneration: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    membershipRole: scope.workspaceRole,
    membershipRevision: scope.membershipRevision,
    projectId: scope.projectId,
    projectMembershipRole: scope.projectMembershipRole,
    projectRole: scope.projectRole,
  };
  let issueCalls = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    issueCalls += 1;
    return response([issue("i1")], 1);
  });
  const first = await client.fetchQuery(issueListOptions(listRead));
  client.setQueryData(estimateKeys.list(estimateScope), []);
  client.setQueryData(estimateKeys.list(estimateScope), [{ id: "estimate", points: [{ id: "small", key: 1 }] }]);

  assert.equal(client.getQueryData(issueListKeys.list(listRead)), first);
  assert.equal(issueCalls, 1);
});

await test("a pending accepted write cannot invalidate a replacement Query owner", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const listRead = read();
  const listQueryKey = issueListKeys.list(listRead);
  const oldQuery = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "old", response: response([issue("i1")], 1) });
  const started = Promise.withResolvers();
  const finish = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "patchIssue", async () => {
    started.resolve();
    return finish.promise;
  });
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);
  const pending = issueListMutation(client, owner, { kind: "patch", issueId: "i1", data: { priority: "high" } });
  await started.promise;
  client.removeQueries({ queryKey: listQueryKey, exact: true });
  const replacement = client.getQueryCache().build(client, { queryKey: listQueryKey });
  client.setQueryData(listQueryKey, { generation: "new", response: response([issue("replacement")], 1) });
  finish.resolve({ ...issue("i1"), priority: "high" });

  await pending;

  assert.notEqual(client.getQueryCache().find({ queryKey: listQueryKey, exact: true }), oldQuery);
  assert.equal(client.getQueryCache().find({ queryKey: listQueryKey, exact: true }), replacement);
  assert.equal(replacement.state.isInvalidated, false);
  assert.deepEqual(
    client.getQueryData(listQueryKey).response.results.map((row) => row.id),
    ["replacement"]
  );
});

await test("a native write refuses missing or stale access before calling the service", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const patch = t.mock.method(IssueService.prototype, "patchIssue", async () => issue("i1"));
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => false);

  await assert.rejects(
    issueListMutation(client, owner, { kind: "patch", issueId: "i1", data: { priority: "high" } }),
    /Work item access changed/
  );
  assert.equal(patch.mock.callCount(), 0);
});

await test("HTTP 204 PATCH is accepted and invalidates the list without inventing a server row", async (t) => {
  const client = queryClient();
  t.after(() => client.clear());
  const key = issueListKeys.list(read());
  const before = { generation: "g1", response: response([issue("i1")], 1) };
  client.setQueryData(key, before);
  const transport = t.mock.method(IssueService.prototype, "patch", async () => ({ status: 204, data: "" }));
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  assert.equal(await patchIssueMutation(client, owner, "i1", { name: "Saved title" }), undefined);
  assert.equal(
    await issueListMutation(client, owner, { kind: "patch", issueId: "i1", data: { priority: "high" } }),
    undefined
  );
  assert.equal(transport.mock.callCount(), 2);
  assert.equal(client.getQueryState(key).isInvalidated, true);
  assert.deepEqual(client.getQueryData(key), before);
});

await test("an empty HTTP 200 PATCH is rejected rather than mistaken for the no-content contract", async (t) => {
  t.mock.method(IssueService.prototype, "patch", async () => ({ status: 200, data: "" }));
  await assert.rejects(new IssueService().patchIssue("acme", "p1", "i1", { name: "Title" }), /response was empty/);
});
