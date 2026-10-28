import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { createPermissionQueryClient } = await import("./helpers/permission-query-fixture.mjs");
const { webRequire } = await import("./helpers/react.mjs");
const { MutationObserver, QueryObserver } = webRequire("@tanstack/react-query");
const { EUserPermissions } = webRequire("@plane/constants");
const { EUserWorkspaceRoles } = webRequire("@plane/types");

const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
const { ProjectArchiveService } = await import("../../apps/web/core/services/project/project-archive.service.ts");
const { projectMutationOptions } = await import("../../apps/web/core/queries/projects.ts");
const { advanceIdentitySession, currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
const {
  labelKeys,
  projectLabelMutationOptions,
  projectLabelOptions,
  projectLabelPositionPatch,
  projectLabelTree,
  workspaceLabelOptions,
} = await import("../../apps/web/core/queries/labels.ts");

const accountId = "alice";
const workspaceSlug = "acme";
const projectId = "project-a";
const membershipRevision = "membership-1";
const workspaceRole = EUserWorkspaceRoles.MEMBER;
const projectMembershipRole = EUserPermissions.ADMIN;

function projectScope(overrides = {}) {
  return {
    accountId,
    workspaceSlug,
    projectId,
    workspaceRole,
    membershipRevision,
    projectRole: EUserPermissions.ADMIN,
    projectMembershipRole,
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
    projectRoles: [[projectId, projectMembershipRole]],
    sessionGeneration: 0,
    ...overrides,
  };
}

function createClient(t) {
  return createPermissionQueryClient(t, projectScope());
}

await test("project label reads forward cancellation and replace stale rows with canonical empty data", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const query = { ...projectLabelOptions(scope), staleTime: 0 };
  client.setQueryData(query.queryKey, [{ id: "deleted-label", name: "Old" }]);
  let requestSignal;
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async (_slug, _projectId, signal) => {
    requestSignal = signal;
    return [];
  });

  const labels = await client.fetchQuery(query);

  assert.deepEqual(labels, []);
  assert.deepEqual(client.getQueryData(query.queryKey), []);
  assert.ok(requestSignal instanceof AbortSignal);
});

await test("label reads and new writes require verified identity while accepted writes reconcile by lineage", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const options = projectLabelOptions(scope);
  const cached = [{ id: "label-a", name: "Before", project_id: projectId, sort_order: 1 }];
  client.setQueryData(options.queryKey, cached);
  const currentUser = t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(client.fetchQuery({ ...currentUserOptions(), staleTime: 0 }), /identity refresh failed/);

  const read = t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => []);
  await assert.rejects(client.fetchQuery({ ...options, staleTime: 0 }), /scope changed or is unverified/);
  assert.equal(read.mock.callCount(), 0);

  const create = t.mock.method(IssueLabelService.prototype, "createIssueLabel", async (_slug, _project, data) => ({
    ...data,
    id: "label-b",
    project_id: projectId,
    sort_order: 2,
  }));
  const rejectedWrite = new MutationObserver(client, projectLabelMutationOptions(client, scope));
  await assert.rejects(
    rejectedWrite.mutate({ type: "create", data: { name: "Blocked" } }),
    /scope changed or is unverified/
  );
  assert.equal(create.mock.callCount(), 0);

  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  create.mock.mockImplementation(async (_slug, _project, data) => {
    started.resolve();
    return pending.promise.then((label) => ({ ...data, ...label }));
  });
  currentUser.mock.mockImplementation(async () => ({ id: accountId }));
  await client.fetchQuery({ ...currentUserOptions(), staleTime: 0 });
  const accepted = new MutationObserver(client, projectLabelMutationOptions(client, scope));
  const save = accepted.mutate({ type: "create", data: { name: "Accepted" } });
  await started.promise;
  currentUser.mock.mockImplementation(async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(client.fetchQuery({ ...currentUserOptions(), staleTime: 0 }), /identity refresh failed/);
  pending.resolve({ id: "label-b", project_id: projectId, name: "Accepted", sort_order: 2 });
  await save;
  assert.deepEqual(client.getQueryData(options.queryKey), [
    ...cached,
    { id: "label-b", project_id: projectId, name: "Accepted", sort_order: 2 },
  ]);
});

await test("workspace label cache identity includes visible project roles", async (t) => {
  const { client, rolesKey } = await createClient(t);
  const first = workspaceScope();
  const secondRoles = { [projectId]: projectMembershipRole, "project-b": EUserPermissions.MEMBER };
  const second = workspaceScope({
    projectRoles: Object.entries(secondRoles).toSorted(([left], [right]) => left.localeCompare(right)),
  });
  const firstOptions = workspaceLabelOptions(first);
  const secondOptions = workspaceLabelOptions(second);
  t.mock.method(IssueLabelService.prototype, "getWorkspaceIssueLabels", async () => []);

  await client.fetchQuery(firstOptions);
  client.setQueryData(rolesKey, secondRoles);
  await client.fetchQuery(secondOptions);

  assert.notDeepEqual(labelKeys.workspace(first), labelKeys.workspace(second));
  assert.equal(client.getQueryData(firstOptions.queryKey).length, 0);
  assert.equal(client.getQueryData(secondOptions.queryKey).length, 0);
});

await test("a failed project-role read cannot expose labels from the previous permission key", async (t) => {
  const { client } = await createClient(t);
  const allowed = projectLabelOptions(projectScope());
  const unavailable = projectLabelOptions(projectScope({ projectRole: undefined, projectMembershipRole: undefined }));
  client.setQueryData(allowed.queryKey, [{ id: "private-label", name: "Private" }]);

  assert.notDeepEqual(unavailable.queryKey, allowed.queryKey);
  assert.equal(client.getQueryData(unavailable.queryKey), undefined);
});

await test("a project label response arriving after membership replacement is discarded", async (t) => {
  const { client, membershipKey } = await createClient(t);
  const requestStarted = Promise.withResolvers();
  const resolveResponse = Promise.withResolvers();
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", () => {
    requestStarted.resolve();
    return resolveResponse.promise;
  });
  const options = projectLabelOptions(projectScope());
  const pending = client.fetchQuery(options);
  await requestStarted.promise;
  client.setQueryData(membershipKey, {
    is_active: true,
    role: workspaceRole,
    updated_at: "membership-2",
  });
  resolveResponse.resolve([{ id: "stale-label", name: "Stale" }]);

  await assert.rejects(pending, /scope changed/);
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("captured label reads and queued writes reject a replaced same-account session", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  let reads = 0;
  let writes = 0;
  const firstWriteStarted = Promise.withResolvers();
  const acceptFirstWrite = Promise.withResolvers();
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => {
    reads += 1;
    return [];
  });
  t.mock.method(IssueLabelService.prototype, "patchIssueLabel", async (_slug, _projectId, labelId, data) => {
    writes += 1;
    if (writes === 1) {
      firstWriteStarted.resolve();
      await acceptFirstWrite.promise;
    }
    return { id: labelId, ...data };
  });
  const options = projectLabelOptions(scope);
  const queryKey = labelKeys.project(scope);
  client.setQueryData(queryKey, [{ id: "label-a", name: "Original" }]);
  const observer = new MutationObserver(client, projectLabelMutationOptions(client, scope));
  const firstWrite = observer.mutate({ type: "update", labelId: "label-a", data: { name: "Changed" } });
  await firstWriteStarted.promise;
  const queuedWrite = observer.mutate({ type: "update", labelId: "label-a", data: { name: "Queued" } });
  advanceIdentitySession(client);
  acceptFirstWrite.resolve();
  await firstWrite;
  await assert.rejects(queuedWrite, /scope changed/);
  await assert.rejects(client.fetchQuery({ ...options, staleTime: 0 }), /scope changed/);

  assert.equal(reads, 0);
  assert.equal(writes, 1);
  assert.deepEqual(client.getQueryData(queryKey), [{ id: "label-a", name: "Original" }]);
});

await test("failed label writes leave canonical data intact and expose mutation failure", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const queryKey = labelKeys.project(scope);
  const labels = [{ id: "label-a", name: "A", sort_order: 1 }];
  client.setQueryData(queryKey, labels);
  t.mock.method(IssueLabelService.prototype, "patchIssueLabel", async () => {
    throw new Error("write rejected");
  });
  const observer = new MutationObserver(client, projectLabelMutationOptions(client, scope));

  await assert.rejects(
    observer.mutate({ type: "update", labelId: "label-a", data: { name: "Saved" } }),
    /write rejected/
  );

  assert.deepEqual(client.getQueryData(queryKey), labels);
  assert.equal(observer.getCurrentResult().isError, true);
});

await test("workspace members can create labels but cannot update or delete them", async (t) => {
  const { client, rolesKey } = await createClient(t);
  const memberScope = projectScope({
    projectRole: EUserPermissions.MEMBER,
    projectMembershipRole: EUserPermissions.MEMBER,
  });
  client.setQueryData(rolesKey, { [projectId]: EUserPermissions.MEMBER });
  const queryKey = labelKeys.project(memberScope);
  client.setQueryData(queryKey, []);
  const created = { id: "label-a", name: "A", project_id: projectId, workspace_id: "workspace" };
  let deniedWriteCalls = 0;
  t.mock.method(IssueLabelService.prototype, "createIssueLabel", async () => created);
  t.mock.method(IssueLabelService.prototype, "patchIssueLabel", async () => {
    deniedWriteCalls += 1;
    return created;
  });
  t.mock.method(IssueLabelService.prototype, "deleteIssueLabel", async () => {
    deniedWriteCalls += 1;
  });

  const createObserver = new MutationObserver(client, projectLabelMutationOptions(client, memberScope));
  await createObserver.mutate({ type: "create", data: { name: "A" } });
  const updateObserver = new MutationObserver(client, projectLabelMutationOptions(client, memberScope));
  const deleteObserver = new MutationObserver(client, projectLabelMutationOptions(client, memberScope));

  await assert.rejects(
    updateObserver.mutate({ type: "update", labelId: created.id, data: { name: "Changed" } }),
    /Project administrator access/
  );
  await assert.rejects(deleteObserver.mutate({ type: "delete", labelId: created.id }), /Project administrator access/);

  assert.equal(deniedWriteCalls, 0);
  assert.deepEqual(client.getQueryData(queryKey), [created]);
});

await test("rejected label reorder remains visible to its caller and leaves ordering unchanged", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const queryKey = labelKeys.project(scope);
  const labels = [
    { id: "first", name: "First", parent: null, sort_order: 1 },
    { id: "second", name: "Second", parent: null, sort_order: 2 },
  ];
  client.setQueryData(queryKey, labels);
  t.mock.method(IssueLabelService.prototype, "patchIssueLabel", async () => {
    throw new Error("reorder rejected");
  });
  const observer = new MutationObserver(client, projectLabelMutationOptions(client, scope));

  await assert.rejects(
    observer.mutate({
      type: "reorder",
      position: {
        draggingLabelId: "first",
        droppedParentId: null,
        droppedLabelId: "second",
        dropAtEndOfList: false,
      },
    }),
    /reorder rejected/
  );

  assert.deepEqual(client.getQueryData(queryKey), labels);
});

await test("an accepted old write does not cancel a replacement project query", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const queryKey = labelKeys.project(scope);
  client.setQueryData(queryKey, [{ id: "label-a", name: "Original" }]);
  const writeStarted = Promise.withResolvers();
  const acceptWrite = Promise.withResolvers();
  const replacementReadStarted = Promise.withResolvers();
  const resolveReplacementRead = Promise.withResolvers();
  let replacementSignal;
  t.mock.method(IssueLabelService.prototype, "patchIssueLabel", async () => {
    writeStarted.resolve();
    await acceptWrite.promise;
    return { id: "label-a", name: "Saved" };
  });
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async (_slug, _projectId, signal) => {
    replacementSignal = signal;
    replacementReadStarted.resolve();
    return resolveReplacementRead.promise;
  });
  const observer = new MutationObserver(client, projectLabelMutationOptions(client, scope));
  const write = observer.mutate({ type: "update", labelId: "label-a", data: { name: "Saved" } });
  await writeStarted.promise;

  client.removeQueries({ queryKey, exact: true });
  const replacementRead = client.fetchQuery({ ...projectLabelOptions(scope), staleTime: 0 });
  await replacementReadStarted.promise;
  acceptWrite.resolve();
  await write;

  assert.equal(replacementSignal.aborted, false);
  resolveReplacementRead.resolve([{ id: "label-a", name: "Replacement" }]);
  await replacementRead;
  assert.deepEqual(client.getQueryData(queryKey), [{ id: "label-a", name: "Replacement" }]);
});

await test("create during the initial read refetches labels and stays accepted when that refresh fails", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const query = projectLabelOptions(scope);
  const initialReadStarted = Promise.withResolvers();
  const keepInitialReadPending = Promise.withResolvers();
  let reads = 0;
  let initialSignal;
  const created = { id: "label-created", name: "Created", project_id: projectId, workspace_id: "workspace" };
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async (_slug, _projectId, signal) => {
    reads += 1;
    if (reads === 1) {
      initialSignal = signal;
      initialReadStarted.resolve();
      return keepInitialReadPending.promise;
    }
    throw new Error("refresh failed");
  });
  t.mock.method(IssueLabelService.prototype, "createIssueLabel", async () => created);
  const queryObserver = new QueryObserver(client, query);
  const unsubscribe = queryObserver.subscribe(() => {});
  t.after(unsubscribe);
  await initialReadStarted.promise;
  const mutation = new MutationObserver(client, projectLabelMutationOptions(client, scope));

  const result = await mutation.mutate({ type: "create", data: { name: "Created" } });

  assert.equal(result.label, created);
  assert.equal(reads, 2);
  assert.equal(initialSignal.aborted, true);
  assert.equal(queryObserver.getCurrentResult().isError, true);
});

await test("project label writes serialize on their project scope", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const queryKey = labelKeys.project(scope);
  client.setQueryData(queryKey, [
    { id: "label-a", name: "A", sort_order: 1 },
    { id: "label-b", name: "B", sort_order: 2 },
  ]);
  const firstStarted = Promise.withResolvers();
  const acceptFirst = Promise.withResolvers();
  let calls = 0;
  t.mock.method(IssueLabelService.prototype, "patchIssueLabel", async (_slug, _project, labelId, data) => {
    calls += 1;
    if (calls === 1) {
      firstStarted.resolve();
      await acceptFirst.promise;
    }
    return { id: labelId, project_id: projectId, workspace_id: "workspace", ...data };
  });
  const observer = new MutationObserver(client, projectLabelMutationOptions(client, scope));
  const first = observer.mutate({ type: "update", labelId: "label-a", data: { name: "A1" } });
  await firstStarted.promise;
  const second = observer.mutate({ type: "update", labelId: "label-b", data: { name: "B1" } });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(calls, 1);

  acceptFirst.resolve();
  await first;
  await second;

  assert.equal(calls, 2);
  assert.deepEqual(
    client.getQueryData(queryKey).map(({ id, name }) => [id, name]),
    [
      ["label-a", "A1"],
      ["label-b", "B1"],
    ]
  );
});

await test("archiving and restoring a project refresh the canonical workspace label catalogue", async (t) => {
  const { client } = await createClient(t);
  const scope = workspaceScope();
  const query = workspaceLabelOptions(scope);
  let archived = false;
  const label = { id: "label-a", project_id: projectId, name: "A" };
  t.mock.method(IssueLabelService.prototype, "getWorkspaceIssueLabels", async () => (archived ? [] : [label]));
  const observer = new QueryObserver(client, query);
  const unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  await observer.refetch();

  const archivedProjectScope = {
    ...scope,
    projectId,
    projectRole: EUserPermissions.ADMIN,
    projectMembershipRole,
  };
  t.mock.method(ProjectArchiveService.prototype, "archiveProject", async () => ({
    archived_at: "2026-01-01T00:00:00Z",
  }));
  t.mock.method(ProjectArchiveService.prototype, "restoreProject", async () => undefined);
  const projectMutation = new MutationObserver(client, projectMutationOptions(client, archivedProjectScope));

  archived = true;
  await projectMutation.mutate({ type: "archive", projectId });
  assert.deepEqual(client.getQueryData(query.queryKey), []);

  archived = false;
  await projectMutation.mutate({ type: "restore", projectId });
  assert.deepEqual(client.getQueryData(query.queryKey), [label]);
});

await test("deleting a group removes its descendants from the flat project catalogue", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const queryKey = labelKeys.project(scope);
  client.setQueryData(queryKey, [
    { id: "group", name: "Group", parent: null, sort_order: 1 },
    { id: "child", name: "Child", parent: "group", sort_order: 2 },
    { id: "grandchild", name: "Grandchild", parent: "child", sort_order: 3 },
    { id: "other", name: "Other", parent: null, sort_order: 4 },
  ]);
  t.mock.method(IssueLabelService.prototype, "deleteIssueLabel", async () => undefined);
  const observer = new MutationObserver(client, projectLabelMutationOptions(client, scope));

  await observer.mutate({ type: "delete", labelId: "group" });

  assert.deepEqual(
    client.getQueryData(queryKey).map(({ id }) => id),
    ["other"]
  );
});

await test("accepted label reorder is visible from the sorted tree before a refetch", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const queryKey = labelKeys.project(scope);
  client.setQueryData(queryKey, [
    { id: "first", name: "First", parent: null, sort_order: 1 },
    { id: "second", name: "Second", parent: null, sort_order: 2 },
  ]);
  t.mock.method(IssueLabelService.prototype, "patchIssueLabel", async () => ({
    id: "first",
    name: "First",
    parent: null,
    project_id: projectId,
    workspace_id: "workspace",
    sort_order: 3,
  }));
  const observer = new MutationObserver(client, projectLabelMutationOptions(client, scope));

  await observer.mutate({ type: "update", labelId: "first", data: { sort_order: 3 } });

  assert.deepEqual(
    projectLabelTree(client.getQueryData(queryKey)).map(({ id }) => id),
    ["second", "first"]
  );
});

await test("label tree projection is pure and position fallback preserves drop-at-end behavior", () => {
  const labels = [
    { id: "parent", name: "Group", parent: null, sort_order: 1 },
    { id: "child", name: "Child", parent: "parent", sort_order: 2 },
    { id: "other", name: "Other", parent: null, sort_order: 3 },
  ];

  const tree = projectLabelTree(labels);
  const endPatch = projectLabelPositionPatch(labels, {
    draggingLabelId: "other",
    droppedParentId: "parent",
    droppedLabelId: "missing",
    dropAtEndOfList: true,
  });

  assert.equal(tree[0].children[0].id, "child");
  assert.equal(Object.hasOwn(labels[0], "children"), false);
  assert.deepEqual(endPatch, { parent: "parent", sort_order: 10002 });
  assert.equal(
    projectLabelPositionPatch(labels, {
      draggingLabelId: "child",
      droppedParentId: "parent",
      droppedLabelId: undefined,
      dropAtEndOfList: true,
    }),
    undefined
  );
});

await test("label positions handle zero and negative sort order values", () => {
  const labels = [
    { id: "negative", name: "Negative", parent: null, sort_order: -10 },
    { id: "zero", name: "Zero", parent: null, sort_order: 0 },
    { id: "dragged", name: "Dragged", parent: null, sort_order: 5 },
  ];

  assert.deepEqual(
    projectLabelPositionPatch(labels, {
      draggingLabelId: "dragged",
      droppedParentId: null,
      droppedLabelId: "zero",
      dropAtEndOfList: false,
    }),
    { parent: null, sort_order: -5 }
  );
  const beforeNegative = projectLabelPositionPatch(labels, {
    draggingLabelId: "zero",
    droppedParentId: null,
    droppedLabelId: "negative",
    dropAtEndOfList: false,
  });
  assert.ok(beforeNegative);
  const negativeOrder = labels
    .map((label) => (label.id === "zero" ? Object.assign({}, label, beforeNegative) : label))
    .toSorted((left, right) => left.sort_order - right.sort_order)
    .map((label) => label.id);
  assert.ok(negativeOrder.indexOf("zero") < negativeOrder.indexOf("negative"));

  const beforeZero = projectLabelPositionPatch(
    [
      { id: "zero", name: "Zero", parent: null, sort_order: 0 },
      { id: "dragged", name: "Dragged", parent: null, sort_order: 5 },
    ],
    {
      draggingLabelId: "dragged",
      droppedParentId: null,
      droppedLabelId: "zero",
      dropAtEndOfList: false,
    }
  );
  assert.equal(beforeZero?.sort_order, -10000);
  assert.equal(
    projectLabelPositionPatch(
      [
        { id: "group", name: "Group", parent: null, sort_order: 1 },
        { id: "child", name: "Child", parent: "group", sort_order: 2 },
        { id: "grandchild", name: "Nested", parent: "child", sort_order: 3 },
      ],
      {
        draggingLabelId: "group",
        droppedParentId: "grandchild",
        droppedLabelId: undefined,
        dropAtEndOfList: false,
      }
    ),
    undefined,
    "a label cannot be moved under a descendant"
  );
  assert.equal(
    projectLabelPositionPatch(
      [
        { id: "first", name: "First", parent: "second", sort_order: 1 },
        { id: "second", name: "Second", parent: "first", sort_order: 2 },
        { id: "moving", name: "Moving", parent: null, sort_order: 3 },
      ],
      {
        draggingLabelId: "moving",
        droppedParentId: "first",
        droppedLabelId: undefined,
        dropAtEndOfList: false,
      }
    ),
    undefined,
    "a malformed existing parent cycle cannot be extended"
  );
});

await test("label reorder finds sibling groups nested below the root", () => {
  const labels = [
    { id: "root", name: "Root", parent: null, sort_order: 1 },
    { id: "nested", name: "Nested", parent: "root", sort_order: 2 },
    { id: "child", name: "Child", parent: "nested", sort_order: 3 },
    { id: "moving", name: "Moving", parent: null, sort_order: 4 },
  ];

  assert.deepEqual(
    projectLabelPositionPatch(labels, {
      draggingLabelId: "moving",
      droppedParentId: "nested",
      droppedLabelId: undefined,
      dropAtEndOfList: true,
    }),
    { parent: "nested", sort_order: 10003 }
  );
});

await test("accepted group deletion refreshes descendants changed by another writer", async (t) => {
  const { client } = await createClient(t);
  const scope = projectScope();
  const query = projectLabelOptions(scope);
  const child = { id: "child", name: "Child", parent: "group", sort_order: 2 };
  client.setQueryData(query.queryKey, [{ id: "group", name: "Group", parent: null, sort_order: 1 }, child]);
  const readStarted = Promise.withResolvers();
  const canonical = Promise.withResolvers();
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", () => {
    readStarted.resolve();
    return canonical.promise;
  });
  t.mock.method(IssueLabelService.prototype, "deleteIssueLabel", async () => undefined);
  const reader = new QueryObserver(client, query);
  const unsubscribe = reader.subscribe(() => {});
  t.after(unsubscribe);
  const writer = new MutationObserver(client, projectLabelMutationOptions(client, scope));

  const deleting = writer.mutate({ type: "delete", labelId: "group" });
  await readStarted.promise;
  assert.deepEqual(client.getQueryData(query.queryKey), []);
  const movedChild = { ...child, parent: null };
  canonical.resolve([movedChild]);
  await deleting;

  assert.deepEqual(client.getQueryData(query.queryKey), [movedChild]);
  assert.equal(writer.getCurrentResult().isSuccess, true);
});
