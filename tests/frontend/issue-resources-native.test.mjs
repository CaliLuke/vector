import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createIssueQueryClient } from "./helpers/issue-query-fixtures.mjs";

registerWebModuleHooks();

const { webRequire } = await import("./helpers/react.mjs");
const { QueryClient, MutationObserver } = webRequire("@tanstack/react-query");
const {
  issueAttachmentsOptions,
  issueAttachmentMutationOptions,
  issueLinksOptions,
  issueLinkMutationOptions,
  issueRelationsOptions,
  issueRelationMutationOptions,
  issueResourceKeys,
} = await import("../../apps/web/core/queries/issue-resources.ts");
const { IssueAttachmentService } = await import("../../apps/web/core/services/issue/issue_attachment.service.ts");
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const { IssueRelationService } = await import("../../apps/web/core/services/issue/issue_relation.service.ts");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { discussionKey } = await import("../../apps/web/core/queries/issue-discussion.ts");
const { issueAttachmentUploadScopeKey } = await import("../../apps/web/core/hooks/use-issue-resources.ts");
const { forgetItemReads, isItemAccessDenied, retryItemAccess } =
  await import("../../apps/web/core/queries/work-item-access.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  membershipRole: 15,
  membershipRevision: "membership-1",
  projectId: "project-1",
  projectMembershipRole: 20,
  projectRole: 20,
  serviceType: "issues",
  issueId: "issue-1",
};
const itemAccessScope = (value = scope) => ({
  accountId: value.accountId,
  generation: value.sessionGeneration,
  workspaceSlug: value.workspaceSlug,
  projectId: value.projectId,
  issueId: value.issueId,
  serviceType: value.serviceType,
});

function fixture(t, active = scope) {
  const { client, rolesKey: roles } = createIssueQueryClient(t, QueryClient, active);
  return { client, roles };
}

await test("native attachment reads forward abort signals and reject a replaced query result", async (t) => {
  const { client } = fixture(t);
  const pending = Promise.withResolvers();
  let receivedSignal;
  t.mock.method(IssueAttachmentService.prototype, "getIssueAttachments", (_workspace, _project, _issue, options) => {
    receivedSignal = options.signal;
    return pending.promise;
  });

  const key = issueResourceKeys.collection(scope, "attachments");
  const read = client.fetchQuery(issueAttachmentsOptions(client, scope));
  const rejected = assert.rejects(read);
  client.removeQueries({ queryKey: key, exact: true });
  client.setQueryData(key, []);
  pending.resolve([{ id: "private-attachment", issue_id: scope.issueId }]);

  await rejected;
  assert.ok(receivedSignal instanceof AbortSignal);
  assert.deepEqual(client.getQueryData(key), []);
});

await test("attachment reads preserve authorization status for shared item-access handling", async (t) => {
  const denial = Object.assign(new Error("Not visible"), { response: { status: 404 } });
  t.mock.method(IssueAttachmentService.prototype, "get", async () => {
    throw denial;
  });
  const service = new IssueAttachmentService(scope.serviceType);

  await assert.rejects(service.getIssueAttachments(scope.workspaceSlug, scope.projectId, scope.issueId), (error) => {
    assert.equal(error, denial);
    assert.equal(error.response.status, 404);
    return true;
  });
});

await test("a cancelled stale attachment 403 cannot deny a succeeding collection read", async (t) => {
  const { client } = fixture(t);
  const stale = Promise.withResolvers();
  let calls = 0;
  t.mock.method(IssueAttachmentService.prototype, "getIssueAttachments", () => {
    calls += 1;
    return calls === 1 ? stale.promise : Promise.resolve([{ id: "fresh", issue_id: scope.issueId }]);
  });
  const key = issueResourceKeys.collection(scope, "attachments");
  const options = issueAttachmentsOptions(client, scope);
  const firstRead = client.fetchQuery(options);
  const cancelled = assert.rejects(firstRead);
  await new Promise(setImmediate);
  await client.cancelQueries({ queryKey: key, exact: true });
  await cancelled;

  assert.deepEqual(await client.fetchQuery(options), [{ id: "fresh", issue_id: scope.issueId }]);
  stale.reject(Object.assign(new Error("Access changed"), { response: { status: 403 } }));
  await new Promise(setImmediate);

  assert.equal(isItemAccessDenied(client, itemAccessScope(scope)), false);
  assert.deepEqual(client.getQueryData(key), [{ id: "fresh", issue_id: scope.issueId }]);
});

await test("guest project scope can read issue links through the issue endpoint", async (t) => {
  const guestScope = { ...scope, projectMembershipRole: 5, projectRole: 5 };
  const { client } = fixture(t, guestScope);
  let request;
  t.mock.method(IssueService.prototype, "get", async (path, options) => {
    request = { path, options };
    return {
      data: {
        id: guestScope.issueId,
        project_id: guestScope.projectId,
        issue_link: [{ id: "link-1", issue_id: guestScope.issueId }],
      },
    };
  });

  assert.deepEqual(await client.fetchQuery(issueLinksOptions(client, guestScope)), [
    { id: "link-1", issue_id: guestScope.issueId },
  ]);
  assert.match(request.path, /\/issues\/issue-1\/$/);
  assert.equal(request.options.params.expand, "issue_link");
  assert.equal(request.options.params.fields, "id,project_id,issue_link");
  assert.ok(request.options.signal instanceof AbortSignal);
});

await test("a visible-parent denial is shared only with its matching service-type scope", async (t) => {
  const { client } = fixture(t);
  const denial = Object.assign(new Error("Not visible"), { response: { status: 404 } });
  t.mock.method(IssueService.prototype, "get", async () => {
    throw denial;
  });
  const issueKey = issueResourceKeys.collection(scope, "links");
  const epicScope = { ...scope, serviceType: "epics" };

  await assert.rejects(client.fetchQuery(issueLinksOptions(client, scope)), (error) => error === denial);

  assert.equal(isItemAccessDenied(client, itemAccessScope(scope)), true);
  assert.equal(isItemAccessDenied(client, itemAccessScope(epicScope)), false);
  assert.equal(client.getQueryData(issueKey), undefined);
  retryItemAccess(client, itemAccessScope(scope));
  assert.equal(isItemAccessDenied(client, itemAccessScope(scope)), false);
});

await test("confirmed item denial prevents an accepted write from reconciling resource data", async (t) => {
  const { client } = fixture(t);
  const key = issueResourceKeys.collection(scope, "links");
  const original = [{ id: "old", issue_id: scope.issueId, url: "https://old.test" }];
  client.setQueryData(key, original);
  const pending = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "createIssueLink", () => pending.promise);
  const mutation = new MutationObserver(client, issueLinkMutationOptions(client, scope));
  const write = mutation.mutate({ kind: "create", data: { url: "https://new.test" } });

  await new Promise(setImmediate);
  forgetItemReads(client, itemAccessScope(scope));
  const replacement = [{ id: "replacement", issue_id: scope.issueId, url: "https://replacement.test" }];
  client.setQueryData(key, replacement);
  pending.resolve({ id: "new", issue_id: scope.issueId, url: "https://new.test" });
  await write;

  assert.deepEqual(client.getQueryData(key), replacement);
  assert.equal(isItemAccessDenied(client, itemAccessScope(scope)), true);
});

await test("link writes reconcile only the captured collection after acceptance", async (t) => {
  const { client } = fixture(t);
  const key = issueResourceKeys.collection(scope, "links");
  client.setQueryData(key, [{ id: "old", issue_id: scope.issueId, url: "https://old.test" }]);
  const pending = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "createIssueLink", () => pending.promise);
  const mutation = new MutationObserver(client, issueLinkMutationOptions(client, scope));
  const write = mutation.mutate({ kind: "create", data: { url: "https://new.test" } });

  await new Promise(setImmediate);
  client.removeQueries({ queryKey: key, exact: true });
  client.setQueryData(key, [{ id: "replacement", issue_id: scope.issueId }]);
  pending.resolve({ id: "new", issue_id: scope.issueId, url: "https://new.test" });
  await write;

  assert.deepEqual(client.getQueryData(key), [{ id: "replacement", issue_id: scope.issueId }]);
});

await test("an accepted resource write cannot refresh a replacement session's activity query", async (t) => {
  const { client } = fixture(t);
  const pending = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "createIssueLink", () => pending.promise);
  const currentActivityKey = discussionKey(
    { ...scope, generation: scope.sessionGeneration, projectMembershipRole: scope.projectMembershipRole },
    "activity"
  );
  client.setQueryData(currentActivityKey, [{ id: "old-activity" }]);
  const mutation = new MutationObserver(client, issueLinkMutationOptions(client, scope));
  const write = mutation.mutate({ kind: "create", data: { url: "https://new.test" } });
  await new Promise(setImmediate);

  advanceIdentitySession(client);
  const replacementActivityKey = discussionKey(
    { ...scope, generation: scope.sessionGeneration + 1, projectMembershipRole: scope.projectMembershipRole },
    "activity"
  );
  client.setQueryData(replacementActivityKey, [{ id: "replacement-activity" }]);
  pending.resolve({ id: "new", issue_id: scope.issueId, url: "https://new.test" });
  await write;

  assert.equal(client.getQueryState(currentActivityKey).isInvalidated, false);
  assert.equal(client.getQueryState(replacementActivityKey).isInvalidated, false);
  assert.deepEqual(client.getQueryData(replacementActivityKey), [{ id: "replacement-activity" }]);
});

await test("a links read started during an accepted write cannot overwrite its canonical result", async (t) => {
  const { client } = fixture(t);
  const pendingWrite = Promise.withResolvers();
  const pendingRead = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "createIssueLink", () => pendingWrite.promise);
  t.mock.method(IssueService.prototype, "get", () => pendingRead.promise);
  const mutation = new MutationObserver(client, issueLinkMutationOptions(client, scope));
  const write = mutation.mutate({ kind: "create", data: { url: "https://new.test" } });
  await new Promise(setImmediate);

  const key = issueResourceKeys.collection(scope, "links");
  const read = client.fetchQuery(issueLinksOptions(client, scope));
  const cancelledRead = assert.rejects(read);
  pendingWrite.resolve({ id: "new", issue_id: scope.issueId, url: "https://new.test" });
  await write;
  pendingRead.resolve({
    data: { id: scope.issueId, project_id: scope.projectId, issue_link: [{ id: "stale", issue_id: scope.issueId }] },
  });
  await cancelledRead;

  assert.deepEqual(client.getQueryData(key), undefined);
});

await test("attachment mutations use the durable upload service and update the current collection", async (t) => {
  const { client } = fixture(t);
  const key = issueResourceKeys.collection(scope, "attachments");
  client.setQueryData(key, []);
  const attachment = { id: "attachment-1", issue_id: scope.issueId, attributes: { name: "file.pdf" } };
  let progress;
  t.mock.method(
    IssueAttachmentService.prototype,
    "uploadIssueAttachment",
    async (_workspace, _project, _issue, _file, onProgress) => {
      onProgress({ progress: 0.5 });
      return attachment;
    }
  );
  const mutation = new MutationObserver(client, issueAttachmentMutationOptions(client, scope));
  const result = await mutation.mutate({
    kind: "upload",
    file: new File(["content"], "file.pdf", { type: "application/pdf" }),
    onProgress: (value) => {
      progress = value;
    },
  });

  assert.deepEqual(result, { kind: "upload", saved: attachment });
  assert.equal(progress, 50);
  assert.deepEqual(client.getQueryData(key), [attachment]);
});

await test("attachment progress keys cannot cross account, session, or item replacement", () => {
  const original = issueAttachmentUploadScopeKey(scope);
  for (const replacement of [
    { ...scope, accountId: "bob" },
    { ...scope, sessionGeneration: scope.sessionGeneration + 1 },
    { ...scope, projectId: "project-2" },
    { ...scope, issueId: "issue-2" },
    { ...scope, serviceType: "epics" },
  ]) {
    assert.notEqual(issueAttachmentUploadScopeKey(replacement), original);
  }
});

await test("accepted relation writes invalidate inverse relations within the same actor and service", async (t) => {
  const { client } = fixture(t);
  const peerScope = { ...scope, issueId: "issue-2" };
  const sourceKey = issueResourceKeys.collection(scope, "relations");
  const peerKey = issueResourceKeys.collection(peerScope, "relations");
  const otherActorKey = issueResourceKeys.collection({ ...scope, accountId: "bob" }, "relations");
  const epicKey = issueResourceKeys.collection({ ...peerScope, serviceType: "epics" }, "relations");
  for (const key of [sourceKey, peerKey, otherActorKey, epicKey]) client.setQueryData(key, {});
  t.mock.method(IssueRelationService.prototype, "createIssueRelations", async () => [
    { id: "issue-2", project_id: "project-1" },
  ]);
  const mutation = new MutationObserver(client, issueRelationMutationOptions(client, scope));

  await mutation.mutate({ kind: "create", relationType: "relates_to", issueIds: ["issue-2"] });

  assert.equal(client.getQueryState(sourceKey).isInvalidated, true);
  assert.equal(client.getQueryState(peerKey).isInvalidated, true);
  assert.equal(client.getQueryState(otherActorKey).isInvalidated, false);
  assert.equal(client.getQueryState(epicKey).isInvalidated, false);
});

await test("epic relation reads use the epic endpoint and keep the epic service scope", async (t) => {
  const epicScope = { ...scope, serviceType: "epics" };
  const { client } = fixture(t, epicScope);
  let request;
  t.mock.method(IssueService.prototype, "get", async (path, options) => {
    request = { path, options };
    return {
      data: {
        id: epicScope.issueId,
        project_id: epicScope.projectId,
        issue_relation: [{ id: "related", project_id: epicScope.projectId, relation_type: "blocking" }],
        issue_related: [{ id: "reverse", project_id: epicScope.projectId, relation_type: "blocked_by" }],
      },
    };
  });

  await client.fetchQuery(issueRelationsOptions(client, epicScope));

  assert.match(request.path, /\/epics\/issue-1\/$/);
  assert.equal(request.options.params.expand, "issue_relation,issue_related");
  assert.equal(request.options.params.fields, "id,project_id,issue_relation,issue_related");
  assert.ok(request.options.signal instanceof AbortSignal);
  assert.deepEqual(client.getQueryData(issueResourceKeys.collection(epicScope, "relations")), {
    relates_to: [],
    duplicate: [],
    blocked_by: [],
    blocking: [
      { id: "related", project_id: epicScope.projectId, relation_type: "blocking" },
      { id: "reverse", project_id: epicScope.projectId, relation_type: "blocked_by" },
    ],
  });
});
