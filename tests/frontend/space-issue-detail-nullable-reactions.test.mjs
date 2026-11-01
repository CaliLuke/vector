import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";
registerSpaceModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/propel/toast")
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent('export const TOAST_TYPE = { ERROR: "error" }; export const setToast = () => {};')}`,
      };
    return nextResolve(specifier, context);
  },
});
const require = createRequire(new URL("../../apps/space/package.json", import.meta.url));
const { QueryClient, MutationObserver } = require("@tanstack/react-query");
const { SitesIssueService } = require("@plane/services");
const {
  voteMutationOptions,
  reactionMutationOptions,
  commentMutationOptions,
  commentReactionMutationOptions,
  commentsOptions,
} = await import("../../apps/space/queries/issue-actions.ts");
const { issueOptions } = await import("../../apps/space/queries/issues.ts");
function setup(t) {
  const client = new QueryClient();
  t.after(() => client.clear());
  for (const anchor of ["A", "B"]) {
    client.setQueryData(issueOptions(anchor, "issue").queryKey, {
      id: "issue",
      vote_items: null,
      reaction_items: null,
    });
    client.setQueryData(["public", anchor, "issues"], { results: [] });
  }
  return client;
}
await test("a successful vote invalidates only its public page, including a nullable vote list", async (t) => {
  const client = setup(t);
  t.mock.method(SitesIssueService.prototype, "addVote", async (anchor, issueId, data) => {
    assert.deepEqual([anchor, issueId, data], ["A", "issue", { vote: 1 }]);
  });
  await new MutationObserver(client, voteMutationOptions(client, "A", "issue")).mutate(1);
  assert.equal(client.getQueryState(issueOptions("A", "issue").queryKey).isInvalidated, true);
  assert.equal(client.getQueryState(["public", "A", "issues"]).isInvalidated, true);
  assert.equal(client.getQueryState(issueOptions("B", "issue").queryKey).isInvalidated, false);
});
await test("failed vote removal preserves the existing votes and rejects", async (t) => {
  const client = setup(t);
  const original = [{ vote: 1, actor_details: { id: "actor" } }];
  client.setQueryData(issueOptions("A", "issue").queryKey, { vote_items: original });
  t.mock.method(SitesIssueService.prototype, "removeVote", async () => {
    throw new Error("offline");
  });
  await assert.rejects(new MutationObserver(client, voteMutationOptions(client, "A", "issue")).mutate(null), /offline/);
  assert.deepEqual(client.getQueryData(issueOptions("A", "issue").queryKey).vote_items, original);
});
await test("reaction changes accept null collections and route removal to the selected reaction", async (t) => {
  const client = setup(t);
  const calls = [];
  t.mock.method(SitesIssueService.prototype, "addReaction", async (...args) => calls.push(args));
  t.mock.method(SitesIssueService.prototype, "removeReaction", async (...args) => calls.push(args));
  const observer = new MutationObserver(client, reactionMutationOptions(client, "A", "issue"));
  await observer.mutate({ reaction: "128077", remove: false });
  await observer.mutate({ reaction: "128077", remove: true });
  assert.deepEqual(calls, [
    ["A", "issue", { reaction: "128077" }],
    ["A", "issue", "128077"],
  ]);
});
await test("failed comment updates preserve published content and remain observable to the form", async (t) => {
  const client = setup(t);
  const comments = [{ id: "comment", comment_html: "original" }];
  client.setQueryData(commentsOptions("A", "issue").queryKey, comments);
  t.mock.method(SitesIssueService.prototype, "updateComment", async () => {
    throw new Error("offline");
  });
  await assert.rejects(
    new MutationObserver(client, commentMutationOptions(client, "A", "issue")).mutate({
      kind: "update",
      id: "comment",
      data: { comment_html: "draft" },
    }),
    /offline/
  );
  assert.deepEqual(client.getQueryData(commentsOptions("A", "issue").queryKey), comments);
});

await test("public comment reads forward cancellation and retain the cached page", async (t) => {
  const client = setup(t);
  const options = commentsOptions("A", "issue");
  const existing = [{ id: "comment", comment_html: "Published" }];
  client.setQueryData(options.queryKey, existing);
  const response = Promise.withResolvers();
  let signal;
  t.mock.method(SitesIssueService.prototype, "listComments", async (anchor, issueId, requestSignal) => {
    assert.deepEqual([anchor, issueId], ["A", "issue"]);
    signal = requestSignal;
    return response.promise;
  });
  const pending = client.fetchQuery({ ...options, staleTime: 0 });
  await client.cancelQueries({ queryKey: options.queryKey });
  assert.deepEqual(await pending, existing);
  assert.equal(signal.aborted, true);
  response.resolve([{ id: "late", comment_html: "Late response" }]);
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(client.getQueryData(options.queryKey), existing);
});

for (const kind of ["add", "delete"]) {
  void test(`public comment ${kind} refreshes only the selected page`, async (t) => {
    const client = setup(t);
    const original = [{ id: "comment", comment_html: "Published" }];
    for (const anchor of ["A", "B"]) client.setQueryData(commentsOptions(anchor, "issue").queryKey, original);
    const writes = [];
    t.mock.method(SitesIssueService.prototype, "addComment", async (...args) => {
      writes.push(args);
      return original[0];
    });
    t.mock.method(SitesIssueService.prototype, "removeComment", async (...args) => {
      writes.push(args);
    });
    const change = kind === "add" ? { kind, data: { comment_html: "Published" } } : { kind, id: "comment" };
    const result = await new MutationObserver(client, commentMutationOptions(client, "A", "issue")).mutate(change);
    assert.deepEqual(writes, [["A", "issue", kind === "add" ? change.data : "comment"]]);
    assert.deepEqual(result, kind === "add" ? original[0] : undefined);
    assert.equal(client.getQueryState(commentsOptions("A", "issue").queryKey).isInvalidated, true);
    assert.equal(client.getQueryState(commentsOptions("B", "issue").queryKey).isInvalidated, false);
  });
}

await test("public comment reaction changes serialize per comment and refresh only their anchor", async (t) => {
  const client = setup(t);
  const original = [{ id: "comment", reactions: [] }];
  for (const anchor of ["A", "B"]) client.setQueryData(commentsOptions(anchor, "issue").queryKey, original);
  const first = Promise.withResolvers();
  const writes = [];
  t.mock.method(SitesIssueService.prototype, "addCommentReaction", async (...args) => {
    writes.push(args);
    await first.promise;
  });
  t.mock.method(SitesIssueService.prototype, "removeCommentReaction", async (...args) => {
    writes.push(args);
  });
  const add = new MutationObserver(client, commentReactionMutationOptions(client, "A", "issue", "comment"));
  const remove = new MutationObserver(client, commentReactionMutationOptions(client, "A", "issue", "comment"));
  const adding = add.mutate({ reaction: "128077", remove: false });
  await new Promise((resolve) => setImmediate(resolve));
  const removing = remove.mutate({ reaction: "128077", remove: true });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(remove.getCurrentResult().isPaused, true);
  assert.deepEqual(writes, [["A", "comment", { reaction: "128077" }]]);
  first.resolve();
  await Promise.all([adding, removing]);
  assert.deepEqual(writes, [
    ["A", "comment", { reaction: "128077" }],
    ["A", "comment", "128077"],
  ]);
  assert.equal(client.getQueryState(commentsOptions("A", "issue").queryKey).isInvalidated, true);
  assert.equal(client.getQueryState(commentsOptions("B", "issue").queryKey).isInvalidated, false);
});
