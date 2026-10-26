import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { React, act, createPermissionUiFixture, registerPermissionUiStubs, settleQuery } = await fromRoot(
  "tests/frontend/helpers/permission-ui-fixture.mjs"
);

const uiStubs = {
  "@plane/propel/emoji-reaction": `
    const React=globalThis.reactionRosterTest.React;
    export function EmojiReactionGroup({reactions,onReactionClick}) {
      const names=reactions.flatMap((reaction)=>reaction.users ?? []).join(", ");
      return React.createElement('div', {'data-testid':'reaction-users'}, names, React.createElement('button', {onClick:()=>reactions[0] && onReactionClick(reactions[0].emoji), 'aria-label':'Toggle reaction'}));
    }
    export function EmojiReactionPicker({label}) { return React.createElement(React.Fragment, null, label); }
  `,
};
globalThis.reactionRosterTest = { React, toasts: [] };
registerPermissionUiStubs("/reactions/issue.tsx", uiStubs);
registerPermissionUiStubs("/reactions/picker.tsx", uiStubs);
registerPermissionUiStubs("/comments/comment-reaction.tsx", uiStubs);
registerPermissionUiStubs("/reactions/issue.tsx", {
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=value=>globalThis.reactionRosterTest.toasts.push(value);`,
});

const { IssueReaction } = await fromRoot("apps/web/core/components/issues/issue-detail/reactions/issue.tsx");
const { CommentReactions } = await fromRoot("apps/web/core/components/comments/comment-reaction.tsx");
const { useIssueDiscussion } = await fromRoot("apps/web/core/hooks/use-issue-discussion.ts");
const { useWorkItemCommentOperations } = await fromRoot(
  "apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx"
);
function NativeCommentReactions({ workspaceSlug, disabled }) {
  const discussion = useIssueDiscussion({ workspaceSlug, projectId: "project-1", issueId: "issue-1" });
  const activityOperations = useWorkItemCommentOperations(workspaceSlug, "project-1", "issue-1", discussion);
  const comment = discussion.comments.isSuccess ? discussion.comments.data[0] : undefined;
  return comment ? React.createElement(CommentReactions, { comment, activityOperations, disabled }) : null;
}
const { IssueReactionService } = await fromRoot("apps/web/core/services/issue/issue_reaction.service.ts");
const { IssueCommentService } = await fromRoot("apps/web/core/services/issue/issue_comment.service.ts");
const { IssueActivityService } = await fromRoot("apps/web/core/services/issue/issue_activity.service.ts");
const { IssueService } = await fromRoot("apps/web/core/services/issue/index.ts");
registerPermissionUiStubs("/issue-activity/helper.tsx", {
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=value=>globalThis.reactionRosterTest.toasts.push(value);`,
});
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { EUserWorkspaceRoles } = await fromRoot("packages/types/dist/index.mjs");

function deferred() {
  let resolve;
  const promise = new Promise((done) => (resolve = done));
  return { promise, resolve };
}

function memberRow(id, displayName) {
  return {
    id: `membership-${id}`,
    role: EUserWorkspaceRoles.MEMBER,
    is_active: true,
    member: { id, display_name: displayName, avatar_url: null },
  };
}

function reactionNames(fixture) {
  return fixture.environment.container.querySelector('[data-testid="reaction-users"]')?.textContent ?? "";
}

function memberScope() {
  return {
    is_active: true,
    role: EUserWorkspaceRoles.MEMBER,
    updated_at: "revision-1",
  };
}

async function createFixture(t, routePath = "/acme/projects/p1/issues/") {
  const fixture = await createPermissionUiFixture(t, {
    path: routePath,
    userId: "viewer",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": 15 },
  });
  fixture.client.setQueryData(permissionKeys.workspaceMembership("viewer", "acme"), memberScope());
  fixture.client.setQueryData(permissionKeys.workspaceMembership("viewer", "beta"), memberScope());
  fixture.client.setQueryData(permissionKeys.workspaceProjectRoles("viewer", "beta", 15, "revision-1"), {
    "project-1": 15,
  });
  t.mock.method(IssueService.prototype, "retrieve", async () => ({
    id: "issue-1",
    project_id: "project-1",
    sequence_id: 1,
    project_identifier: "P",
  }));
  return fixture;
}

function renderReaction(fixture, workspaceSlug, disabled = true) {
  return fixture.render(
    React.createElement(IssueReaction, {
      workspaceSlug,
      projectId: "project-1",
      issueId: "issue-1",
      disabled,
    })
  );
}
function renderCommentReaction(fixture, workspaceSlug, disabled = true) {
  return fixture.render(React.createElement(NativeCommentReactions, { workspaceSlug, disabled }));
}

await test("reaction labels use the scoped roster and preserve record attribution while reads resolve", async (t) => {
  const acmeRead = deferred();
  const betaRead = deferred();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", (slug) => {
    if (slug === "acme") return acmeRead.promise;
    if (slug === "beta") return betaRead.promise;
    throw new Error(`Unexpected workspace ${slug}`);
  });

  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/",
    userId: "viewer",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": 15 },
  });
  fixture.client.setQueryData(permissionKeys.workspaceMembership("viewer", "acme"), memberScope());
  fixture.client.setQueryData(permissionKeys.workspaceMembership("viewer", "beta"), memberScope());

  const reaction = {
    id: "reaction-1",
    actor: "actor-1",
    display_name: "Record fallback",
    reaction: "128077",
  };
  t.mock.method(IssueReactionService.prototype, "listIssueReactions", async () => [reaction]);
  fixture.client.setQueryData(permissionKeys.workspaceProjectRoles("viewer", "beta", 15, "revision-1"), {
    "project-1": 15,
  });

  await renderReaction(fixture, "acme");
  await settleQuery();
  assert.equal(reactionNames(fixture), "Record fallback");

  await act(async () => {
    acmeRead.resolve([memberRow("actor-1", "Acme actor")]);
    await new Promise((resolve) => setTimeout(resolve, 40));
  });
  await settleQuery();
  assert.equal(reactionNames(fixture), "Acme actor");

  await renderReaction(fixture, "beta");
  await settleQuery();
  assert.equal(reactionNames(fixture), "Record fallback");
  assert.doesNotMatch(reactionNames(fixture), /Acme actor/);

  await act(async () => {
    betaRead.resolve([memberRow("actor-1", "Beta actor")]);
    await new Promise((resolve) => setTimeout(resolve, 40));
  });
  await settleQuery();
  assert.equal(reactionNames(fixture), "Beta actor");
  assert.doesNotMatch(reactionNames(fixture), /Acme actor/);
});

await test("reaction toggles write within the scoped entity and report transport outcomes", async (t) => {
  globalThis.reactionRosterTest.toasts = [];
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => []);
  const fixture = await createFixture(t);
  const writes = [];
  const reaction = {
    id: "reaction-1",
    actor: "actor-1",
    display_name: "Member",
    reaction: "128077",
    issue: "issue-1",
    project: "project-1",
  };
  t.mock.method(IssueReactionService.prototype, "listIssueReactions", async () => [reaction]);
  t.mock.method(IssueReactionService.prototype, "createIssueReaction", async (...args) => {
    writes.push(["create", ...args]);
    return { ...reaction, id: "mine", actor: "viewer" };
  });
  t.mock.method(IssueReactionService.prototype, "deleteIssueReaction", async (...args) => {
    writes.push(["remove", ...args]);
    throw new Error("Transport failed");
  });
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => []);
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => [
    {
      id: "comment-1",
      issue: "issue-1",
      project: "project-1",
      comment_reactions: [{ ...reaction, comment: "comment-1" }],
    },
  ]);
  t.mock.method(IssueReactionService.prototype, "createIssueCommentReaction", async (...args) => {
    writes.push(["comment-create", ...args]);
    return { ...reaction, id: "mine", actor: "viewer", comment: "comment-1" };
  });
  t.mock.method(IssueReactionService.prototype, "deleteIssueCommentReaction", async (...args) => {
    writes.push(["comment-remove", ...args]);
    throw new Error("Transport failed");
  });

  await renderReaction(fixture, "acme", false);
  await settleQuery();
  await act(async () => fixture.environment.container.querySelector('[aria-label="Toggle reaction"]').click());
  await settleQuery();
  assert.deepEqual(writes, [["create", "acme", "project-1", "issue-1", { reaction: "128077" }]]);
  assert.deepEqual(globalThis.reactionRosterTest.toasts.at(-1), {
    title: "Success!",
    type: "success",
    message: "Reaction saved successfully",
  });

  await renderReaction(fixture, "acme", false);
  await settleQuery();
  await act(async () => fixture.environment.container.querySelector('[aria-label="Toggle reaction"]').click());
  await settleQuery();
  assert.deepEqual(writes.at(-1), ["remove", "acme", "project-1", "issue-1", "128077"]);
  assert.deepEqual(globalThis.reactionRosterTest.toasts.at(-1), {
    title: "Error!",
    type: "error",
    message: "Could not save reaction",
  });

  await renderCommentReaction(fixture, "acme", false);
  await settleQuery();
  await act(async () => fixture.environment.container.querySelector('[aria-label="Toggle reaction"]').click());
  await settleQuery();
  assert.deepEqual(writes.at(-1), ["comment-create", "acme", "project-1", "comment-1", { reaction: "128077" }]);

  await renderCommentReaction(fixture, "acme", false);
  await settleQuery();
  await act(async () => fixture.environment.container.querySelector('[aria-label="Toggle reaction"]').click());
  await settleQuery();
  assert.deepEqual(writes.at(-1), ["comment-remove", "acme", "project-1", "comment-1", "128077"]);
  assert.deepEqual(globalThis.reactionRosterTest.toasts.at(-1), {
    title: "Error!",
    type: "error",
    message: "Could not save reaction",
  });
});

await test("a failed reaction refresh hides cached attribution and Retry recovers the actual native control", async (t) => {
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => []);
  let fail = false;
  t.mock.method(IssueReactionService.prototype, "listIssueReactions", async () => {
    if (fail) throw { response: { status: 403 } };
    return [{ id: "r1", issue: "issue-1", actor: "actor-1", reaction: "128077", display_name: "Cached name" }];
  });
  const f = await createFixture(t);
  await renderReaction(f, "acme");
  await settleQuery();
  assert.match(reactionNames(f), /Cached name/);
  fail = true;
  await act(async () => f.client.invalidateQueries({ predicate: (q) => q.queryKey[2] === "issue-discussion" }));
  await settleQuery();
  assert.match(f.environment.container.textContent, /Could not load reactions/);
  assert.doesNotMatch(f.environment.container.textContent, /Cached name/);
  fail = false;
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Retry").click()
  );
  await settleQuery();
  assert.match(reactionNames(f), /Cached name/);
});
