import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createIssueQueryClient } from "./helpers/issue-query-fixtures.mjs";
registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { QueryClient, MutationObserver } = webRequire("@tanstack/react-query");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const {
  discussionKey,
  issueActivityOptions,
  issueCommentsOptions,
  issueReactionsOptions,
  refreshIssueActivity,
  captureIssueActivityRefreshOwner,
} = await import("../../apps/web/core/queries/issue-discussion.ts");
const { commentMutationOptions, reactionMutationOptions } =
  await import("../../apps/web/core/queries/issue-discussion-mutations.ts");
const { IssueActivityService } = await import("../../apps/web/core/services/issue/issue_activity.service.ts");
const { IssueCommentService } = await import("../../apps/web/core/services/issue/issue_comment.service.ts");
const { IssueReactionService } = await import("../../apps/web/core/services/issue/issue_reaction.service.ts");
const scope = {
  accountId: "alice",
  generation: 0,
  workspaceSlug: "acme",
  projectId: "p1",
  issueId: "i1",
  membershipRole: 15,
  membershipRevision: "m1",
  projectRole: 20,
  projectMembershipRole: 20,
  serviceType: "issues",
};
const comment = {
  id: "c1",
  issue: "i1",
  project: "p1",
  workspace: "w1",
  actor: "alice",
  created_by: "alice",
  comment_html: "<p>Original</p>",
  comment_reactions: [],
  created_at: "2026-10-25T00:00:00Z",
};
function fixture(t, active = scope) {
  const { client, membershipKey: membership, rolesKey: roles } = createIssueQueryClient(t, QueryClient, active);
  client.setQueryData(discussionKey(active, "comments"), [comment]);
  client.setQueryData(discussionKey(active, "reactions"), []);
  return { client, membership, roles };
}
async function beginCommentUpdate(t, client, write) {
  t.mock.method(IssueCommentService.prototype, "patchIssueComment", () => write.promise);
  const pending = new MutationObserver(client, commentMutationOptions(client, scope)).mutate({
    kind: "update",
    commentId: "c1",
    data: { comment_html: "saved" },
  });
  await new Promise(setImmediate);
  return { pending };
}

await test("canonical comments replace deleted rows and retain threading payload fields and nested reactions", async (t) => {
  const { client } = fixture(t);
  const reply = {
    ...comment,
    id: "reply",
    parent: "c1",
    comment_reactions: [{ id: "r1", actor: "alice", reaction: "128077" }],
  };
  let rows = [reply];
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => rows);
  assert.deepEqual(await client.fetchQuery(issueCommentsOptions(scope)), [reply]);
  rows = [];
  assert.deepEqual(await client.fetchQuery(issueCommentsOptions(scope)), []);
});
await test("late history cannot populate cache after same-value permission query replacement", async (t) => {
  const { client, roles } = fixture(t);
  const pending = Promise.withResolvers();
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", () => pending.promise);
  const read = client.fetchQuery(issueActivityOptions(scope));
  const rejected = assert.rejects(read, /access changed/);
  client.removeQueries({ queryKey: roles, exact: true });
  client.setQueryData(roles, { p1: 20 });
  pending.resolve([{ id: "secret" }]);
  await rejected;
  assert.equal(client.getQueryData(discussionKey(scope, "activity")), undefined);
});
await test("raw guest role changes invalidate a read even when workspace admin keeps effective role unchanged", async (t) => {
  const admin = { ...scope, membershipRole: 20, projectMembershipRole: 5 };
  const { client, roles } = fixture(t, admin);
  const pending = Promise.withResolvers();
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", () => pending.promise);
  const read = client.fetchQuery(issueActivityOptions(admin));
  const rejected = assert.rejects(read, /access changed/);
  client.setQueryData(roles, { p1: 15 });
  pending.resolve([]);
  await rejected;
});
await test("failed edits and failed reaction removals preserve acknowledged canonical data", async (t) => {
  const { client } = fixture(t);
  const reaction = { id: "r1", actor: "alice", reaction: "128077", issue: "i1" };
  client.setQueryData(discussionKey(scope, "reactions"), [reaction]);
  t.mock.method(IssueCommentService.prototype, "patchIssueComment", async () => {
    throw new Error("offline");
  });
  t.mock.method(IssueReactionService.prototype, "deleteIssueReaction", async () => {
    throw new Error("offline");
  });
  await assert.rejects(
    new MutationObserver(client, commentMutationOptions(client, scope)).mutate({
      kind: "update",
      commentId: "c1",
      data: { comment_html: "changed" },
    }),
    /offline/
  );
  await assert.rejects(
    new MutationObserver(client, reactionMutationOptions(client, scope, "reactions")).mutate({
      kind: "remove",
      reaction: "128077",
    }),
    /offline/
  );
  assert.equal(client.getQueryData(discussionKey(scope, "comments"))[0].comment_html, "<p>Original</p>");
  assert.deepEqual(client.getQueryData(discussionKey(scope, "reactions")), [reaction]);
});
await test("accepted comment survives transient verification error while new work remains blocked", async (t) => {
  const { client } = fixture(t);
  const write = Promise.withResolvers();
  t.mock.method(IssueCommentService.prototype, "patchIssueComment", () => write.promise);
  const observer = new MutationObserver(client, commentMutationOptions(client, scope));
  const result = observer.mutate({ kind: "update", commentId: "c1", data: { comment_html: "saved" } });
  await new Promise(setImmediate);
  const identity = client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  identity.setState({ status: "error", error: new Error("offline") });
  write.resolve({ ...comment, comment_html: "saved" });
  await result;
  assert.equal(client.getQueryData(discussionKey(scope, "comments"))[0].comment_html, "saved");
  await assert.rejects(observer.mutate({ kind: "delete", commentId: "c1" }), /access changed/);
});
await test("late accepted writes cannot resurrect a removed query or write into a new session", async (t) => {
  const { client } = fixture(t);
  const write = Promise.withResolvers();
  t.mock.method(IssueCommentService.prototype, "createIssueComment", () => write.promise);
  const result = new MutationObserver(client, commentMutationOptions(client, scope)).mutate({
    kind: "create",
    data: { comment_html: "saved" },
  });
  await new Promise(setImmediate);
  client.removeQueries({ queryKey: discussionKey(scope, "comments"), exact: true });
  advanceIdentitySession(client);
  client.setQueryData(discussionKey(scope, "comments"), []);
  write.resolve({ ...comment, id: "new" });
  assert.equal((await result).comment.id, "new");
  assert.deepEqual(client.getQueryData(discussionKey(scope, "comments")), []);
});
await test("comment reactions reconcile canonical comment records without a second reaction owner", async (t) => {
  const { client } = fixture(t);
  const reaction = { id: "r1", actor: "alice", reaction: "128077", comment: "c1", project: "p1" };
  t.mock.method(IssueReactionService.prototype, "createIssueCommentReaction", async () => reaction);
  t.mock.method(IssueReactionService.prototype, "deleteIssueCommentReaction", async () => undefined);
  const observer = new MutationObserver(client, reactionMutationOptions(client, scope, "comments"));
  await observer.mutate({ kind: "add", commentId: "c1", reaction: "128077" });
  assert.deepEqual(client.getQueryData(discussionKey(scope, "comments"))[0].comment_reactions, [reaction]);
  await observer.mutate({ kind: "remove", commentId: "c1", reaction: "128077" });
  assert.deepEqual(client.getQueryData(discussionKey(scope, "comments"))[0].comment_reactions, []);
});
await test("activity invalidation targets item/service and does not touch another item or epic", async (t) => {
  const { client } = fixture(t);
  client.setQueryData(discussionKey(scope, "activity"), []);
  const other = { ...scope, issueId: "other" },
    epic = { ...scope, serviceType: "epics" };
  client.setQueryData(discussionKey(other, "activity"), []);
  client.setQueryData(discussionKey(epic, "activity"), []);
  await refreshIssueActivity(client, scope, captureIssueActivityRefreshOwner(client, scope));
  assert.equal(client.getQueryState(discussionKey(scope, "activity")).isInvalidated, true);
  assert.equal(client.getQueryState(discussionKey(other, "activity")).isInvalidated, false);
  assert.equal(client.getQueryState(discussionKey(epic, "activity")).isInvalidated, false);
});

await test("a retired history read cannot release a replacement item's native pending state", async (t) => {
  const { client, membership } = fixture(t);
  const first = Promise.withResolvers(),
    second = Promise.withResolvers();
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", (_ws, _project, id) =>
    id === "i1" ? first.promise : second.promise
  );
  const replacement = { ...scope, issueId: "i2" };
  const old = client.fetchQuery(issueActivityOptions(scope));
  const rejected = assert.rejects(old, /access changed/);
  const next = client.fetchQuery(issueActivityOptions(replacement));
  client.setQueryData(membership, { is_active: false, role: 15, updated_at: "revoked" });
  first.resolve([{ id: "private" }]);
  await rejected;
  assert.equal(client.getQueryState(discussionKey(replacement, "activity")).fetchStatus, "fetching");
  const nextRejected = assert.rejects(next, /access changed/);
  second.resolve([]);
  await nextRejected;
  assert.equal(client.getQueryState(discussionKey(replacement, "activity")).fetchStatus, "idle");
});

await test("queued discussion writes recheck session before transport and keep the accepted predecessor successful", async (t) => {
  const { client } = fixture(t);
  const write = Promise.withResolvers();
  let calls = 0;
  t.mock.method(IssueCommentService.prototype, "createIssueComment", () => {
    calls++;
    return write.promise;
  });
  const first = new MutationObserver(client, commentMutationOptions(client, scope)).mutate({
    kind: "create",
    data: { comment_html: "one" },
  });
  const second = new MutationObserver(client, commentMutationOptions(client, scope)).mutate({
    kind: "create",
    data: { comment_html: "two" },
  });
  const rejected = assert.rejects(second, /access changed/);
  await new Promise(setImmediate);
  advanceIdentitySession(client);
  write.resolve({ ...comment, id: "saved" });
  await first;
  await rejected;
  assert.equal(calls, 1);
  assert.deepEqual(client.getQueryData(discussionKey(scope, "comments")), [comment]);
});
await test("a definite authorization error blocks accepted receipt reconciliation", async (t) => {
  const { client } = fixture(t);
  const write = Promise.withResolvers();
  const { pending } = await beginCommentUpdate(t, client, write);
  client
    .getQueryCache()
    .find({ queryKey: identityKeys.currentUser(), exact: true })
    .setState({ status: "error", error: { status: 403 } });
  write.resolve({ ...comment, comment_html: "saved" });
  await pending;
  assert.equal(client.getQueryData(discussionKey(scope, "comments"))[0].comment_html, "<p>Original</p>");
});
await test("native epic history forwards cancellation and the epic service endpoint", async (t) => {
  const { client } = fixture(t);
  let requested;
  t.mock.method(IssueCommentService.prototype, "get", async (path, options) => {
    requested = { path, options };
    return { data: [] };
  });
  await client.fetchQuery(issueCommentsOptions({ ...scope, serviceType: "epics" }));
  assert.match(requested.path, /\/epics\/i1\/history\/$/);
  assert.equal(requested.options.params.activity_type, "epic-comment");
  assert.ok(requested.options.signal instanceof AbortSignal);
});

await test("item collection denial retires editor feedback and accepted cache reconciliation", async (t) => {
  const { client } = fixture(t),
    write = Promise.withResolvers();
  const { captureDiscussionWrite } = await import("../../apps/web/core/queries/issue-discussion.ts");
  const editor = captureDiscussionWrite(client, scope, "comments");
  const { pending } = await beginCommentUpdate(t, client, write);
  client
    .getQueryCache()
    .find({ queryKey: discussionKey(scope, "comments"), exact: true })
    .setState({ status: "error", error: { status: 403 } });
  assert.equal(editor.isCurrent(), false);
  assert.equal(editor.isVerified(), false);
  write.resolve({ ...comment, comment_html: "saved" });
  assert.equal((await pending).captured.isCurrent(), false);
  assert.equal(client.getQueryData(discussionKey(scope, "comments"))[0].comment_html, "<p>Original</p>");
});
await test("queued reaction toggles resolve against the latest acknowledged native collection", async (t) => {
  const { client } = fixture(t);
  const calls = [];
  t.mock.method(IssueReactionService.prototype, "createIssueReaction", async () => {
    calls.push("add");
    return { id: "mine", actor: "alice", issue: "i1", reaction: "128077" };
  });
  t.mock.method(IssueReactionService.prototype, "deleteIssueReaction", async () => {
    calls.push("remove");
  });
  const observer = new MutationObserver(client, reactionMutationOptions(client, scope, "reactions"));
  await Promise.all([
    observer.mutate({ kind: "toggle", reaction: "128077" }),
    observer.mutate({ kind: "toggle", reaction: "128077" }),
  ]);
  assert.deepEqual(calls, ["add", "remove"]);
  assert.deepEqual(client.getQueryData(discussionKey(scope, "reactions")), []);
});

await test("guest reaction reads use the visible parent projection and retain authorization status", async (t) => {
  const guest = { ...scope, projectRole: 5, projectMembershipRole: 5 };
  const { client } = fixture(t, guest);
  let requested;
  const reaction = { id: "r1", issue: "i1", actor: "alice", reaction: "128077", display_name: "Alice" };
  t.mock.method(IssueReactionService.prototype, "get", async (path, options) => {
    requested = { path, options };
    return { data: { id: "i1", issue_reactions: [reaction] } };
  });
  const rows = await client.fetchQuery(issueReactionsOptions(guest));
  assert.deepEqual(rows, [reaction]);
  assert.match(requested.path, /\/issues\/i1\/$/);
  assert.deepEqual(requested.options.params, { fields: "id,issue_reactions", expand: "issue_reactions" });
  assert.ok(requested.options.signal instanceof AbortSignal);
  t.mock.method(IssueReactionService.prototype, "get", async () => {
    throw { response: { status: 403, data: { detail: "Not permitted" } } };
  });
  await assert.rejects(client.fetchQuery(issueReactionsOptions(guest)), (error) => error.response.status === 403);
  assert.equal(client.getQueryState(discussionKey(guest, "reactions")).status, "error");
});

await test("a read started during comment PATCH cannot overwrite the acknowledged canonical edit", async (t) => {
  const { client } = fixture(t),
    write = Promise.withResolvers(),
    read = Promise.withResolvers(),
    started = Promise.withResolvers();
  t.mock.method(IssueCommentService.prototype, "patchIssueComment", () => {
    started.resolve();
    return write.promise;
  });
  t.mock.method(IssueCommentService.prototype, "getIssueComments", () => read.promise);
  const saving = new MutationObserver(client, commentMutationOptions(client, scope)).mutate({
    kind: "update",
    commentId: "c1",
    data: { comment_html: "saved" },
  });
  await started.promise;
  const background = client.fetchQuery(issueCommentsOptions(scope)).catch(() => undefined);
  write.resolve({ ...comment, comment_html: "saved" });
  await saving;
  read.resolve([comment]);
  await background;
  assert.equal(client.getQueryData(discussionKey(scope, "comments"))[0].comment_html, "saved");
});
await test("a read started during comment reaction POST cannot erase its acknowledged reaction", async (t) => {
  const { client } = fixture(t),
    write = Promise.withResolvers(),
    read = Promise.withResolvers(),
    started = Promise.withResolvers();
  const reaction = { id: "r1", actor: "alice", reaction: "128077", comment: "c1", project: "p1" };
  t.mock.method(IssueReactionService.prototype, "createIssueCommentReaction", () => {
    started.resolve();
    return write.promise;
  });
  t.mock.method(IssueCommentService.prototype, "getIssueComments", () => read.promise);
  const saving = new MutationObserver(client, reactionMutationOptions(client, scope, "comments")).mutate({
    kind: "add",
    commentId: "c1",
    reaction: "128077",
  });
  await started.promise;
  const background = client.fetchQuery(issueCommentsOptions(scope)).catch(() => undefined);
  write.resolve(reaction);
  await saving;
  read.resolve([comment]);
  await background;
  assert.deepEqual(client.getQueryData(discussionKey(scope, "comments"))[0].comment_reactions, [reaction]);
});
await test("activity invalidation replaces an initial pending snapshot with a read started after the write", async (t) => {
  const { client } = fixture(t),
    old = Promise.withResolvers();
  let calls = 0;
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", () =>
    ++calls === 1 ? old.promise : Promise.resolve([{ id: "fresh" }])
  );
  const { QueryObserver } = webRequire("@tanstack/react-query");
  const observer = new QueryObserver(client, issueActivityOptions(scope));
  const unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  const refreshing = refreshIssueActivity(client, scope, captureIssueActivityRefreshOwner(client, scope));
  old.resolve([{ id: "old" }]);
  await refreshing;
  assert.equal(calls, 2);
  assert.deepEqual(client.getQueryData(discussionKey(scope, "activity")), [{ id: "fresh" }]);
});

await test("history denial blocks a concurrently pending sibling reaction read", async (t) => {
  const { client } = fixture(t),
    history = Promise.withResolvers(),
    reactions = Promise.withResolvers();
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", () => history.promise);
  t.mock.method(IssueReactionService.prototype, "listIssueReactions", () => reactions.promise);
  const historyRead = client.fetchQuery(issueActivityOptions(scope));
  const denied = assert.rejects(historyRead, (error) => error.response.status === 403);
  const sibling = client.fetchQuery(issueReactionsOptions(scope));
  const blocked = assert.rejects(sibling, /access changed/);
  history.reject({ response: { status: 403 } });
  await denied;
  reactions.resolve([{ id: "private", actor: "alice", issue: "i1", reaction: "128077" }]);
  await blocked;
  assert.deepEqual(client.getQueryData(discussionKey(scope, "reactions")), []);
});
await test("a confirmed sibling denial blocks accepted mutation reconciliation without failing its write", async (t) => {
  const { client } = fixture(t),
    write = Promise.withResolvers(),
    started = Promise.withResolvers();
  t.mock.method(IssueReactionService.prototype, "createIssueReaction", () => {
    started.resolve();
    return write.promise;
  });
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => {
    throw { response: { status: 403 } };
  });
  const saving = new MutationObserver(client, reactionMutationOptions(client, scope, "reactions")).mutate({
    kind: "add",
    reaction: "128077",
  });
  await started.promise;
  await assert.rejects(client.fetchQuery(issueActivityOptions(scope)));
  write.resolve({ id: "saved", actor: "alice", issue: "i1", reaction: "128077" });
  assert.equal((await saving).captured.isCurrent(), false);
  assert.deepEqual(client.getQueryData(discussionKey(scope, "reactions")), []);
});
await test("Retry retires a denied read before clearing access so its late success cannot replace recovery", async (t) => {
  const { client } = fixture(t),
    old = Promise.withResolvers();
  let calls = 0;
  const { retryItemAccess, isItemAccessDenied } = await import("../../apps/web/core/queries/work-item-access.ts");
  t.mock.method(IssueReactionService.prototype, "listIssueReactions", () =>
    ++calls === 1 ? old.promise : Promise.resolve([{ id: "fresh", actor: "alice", issue: "i1", reaction: "128077" }])
  );
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => {
    throw { response: { status: 403 } };
  });
  const obsolete = client.fetchQuery(issueReactionsOptions(scope)).catch(() => undefined);
  await assert.rejects(client.fetchQuery(issueActivityOptions(scope)));
  assert.equal(isItemAccessDenied(client, scope), true);
  retryItemAccess(client, scope);
  await client.fetchQuery(issueReactionsOptions(scope));
  old.resolve([{ id: "old", actor: "alice", issue: "i1", reaction: "128077" }]);
  await obsolete;
  assert.equal(isItemAccessDenied(client, scope), false);
  assert.equal(client.getQueryData(discussionKey(scope, "reactions"))[0].id, "fresh");
});
await test("a wrong-kind denial and eviction preserve a valid sibling service family", async (t) => {
  const { client } = fixture(t),
    epic = { ...scope, serviceType: "epics" };
  const { forgetItemReads, isItemAccessDenied } = await import("../../apps/web/core/queries/work-item-access.ts");
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => {
    throw { response: { status: 404 } };
  });
  await assert.rejects(client.fetchQuery(issueActivityOptions(epic)));
  forgetItemReads(client, epic);
  assert.equal(isItemAccessDenied(client, epic), true);
  assert.equal(isItemAccessDenied(client, scope), false);
  assert.deepEqual(client.getQueryData(discussionKey(scope, "comments")), [comment]);
});
await test("confirmed denial evicts only the exact actor/session/workspace/project/item family", async (t) => {
  const { client } = fixture(t);
  const { forgetItemReads } = await import("../../apps/web/core/queries/work-item-access.ts");
  const otherActor = { ...scope, accountId: "bob" },
    otherSession = { ...scope, generation: 1 },
    otherWorkspace = { ...scope, workspaceSlug: "beta" },
    otherProject = { ...scope, projectId: "p2" },
    otherItem = { ...scope, issueId: "i2" };
  client.setQueryData(discussionKey(otherActor, "comments"), ["actor"]);
  client.setQueryData(discussionKey(otherSession, "comments"), ["session"]);
  client.setQueryData(discussionKey(otherWorkspace, "comments"), ["workspace"]);
  client.setQueryData(discussionKey(otherProject, "comments"), ["project"]);
  client.setQueryData(discussionKey(otherItem, "comments"), ["item"]);
  forgetItemReads(client, scope);
  assert.equal(client.getQueryData(discussionKey(scope, "comments")), undefined);
  assert.deepEqual(client.getQueryData(discussionKey(otherActor, "comments")), ["actor"]);
  assert.deepEqual(client.getQueryData(discussionKey(otherSession, "comments")), ["session"]);
  assert.deepEqual(client.getQueryData(discussionKey(otherWorkspace, "comments")), ["workspace"]);
  assert.deepEqual(client.getQueryData(discussionKey(otherProject, "comments")), ["project"]);
  assert.deepEqual(client.getQueryData(discussionKey(otherItem, "comments")), ["item"]);
});

await test("an old actor activity refresh cannot cancel a replacement actor read", async (t) => {
  const { client } = fixture(t),
    pending = Promise.withResolvers();
  const owner = captureIssueActivityRefreshOwner(client, scope);
  const replacement = { ...scope, accountId: "bob" };
  client.setQueryData(identityKeys.currentUser(), { id: "bob" });
  client.setQueryData(permissionKeys.workspaceMembership("bob", "acme"), {
    is_active: true,
    role: 15,
    updated_at: "m1",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("bob", "acme", 15, "m1"), { p1: 20 });
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", () => pending.promise);
  const reading = client.fetchQuery(issueActivityOptions(replacement));
  await refreshIssueActivity(client, scope, owner);
  assert.equal(client.getQueryState(discussionKey(replacement, "activity")).fetchStatus, "fetching");
  assert.equal(client.getQueryState(discussionKey(replacement, "activity")).isInvalidated, false);
  pending.resolve([{ id: "bob-history" }]);
  assert.deepEqual(await reading, [{ id: "bob-history" }]);
});
await test("accepted activity refresh keeps same-lineage transient verification errors distinct from denial", async (t) => {
  const { client } = fixture(t);
  client.setQueryData(discussionKey(scope, "activity"), []);
  const owner = captureIssueActivityRefreshOwner(client, scope);
  const identity = client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  identity.setState({ status: "error", error: new Error("offline") });
  await refreshIssueActivity(client, scope, owner);
  assert.equal(client.getQueryState(discussionKey(scope, "activity")).isInvalidated, true);
});
