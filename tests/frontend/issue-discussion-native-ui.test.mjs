import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
globalThis.discussionUi = { React, toasts: [], clears: 0 };
const stubs = {
  "./sort-root": "export const ActivitySortRoot=()=>null;",
  "./filter-root": "export const ActivityFilterRoot=()=>null;",
  "@/components/editor/lite-text": `const R=globalThis.discussionUi.React; export const LiteTextEditor=R.forwardRef(function Editor(p,ref){ R.useImperativeHandle(ref,()=>({isEditorReadyToDiscard:()=>true,clearEditor:()=>globalThis.discussionUi.clears++,focus(){},setEditorValue(){}}));return p.editable?R.createElement(R.Fragment,null,R.createElement('input',{'aria-label':'Comment',value:p.initialValue??'',onChange:e=>p.onChange(null,e.target.value)}),p.onEnterKeyPress?R.createElement('button',{type:'button',disabled:p.isSubmitting,onClick:p.onEnterKeyPress},'Send'):null,p.onAssetUpload?R.createElement('button',{type:'button',onClick:()=>p.onAssetUpload('asset-1')},'Upload'):null):R.createElement('p',{'data-comment':p.id},p.initialValue);});`,
  "@/hooks/use-work-item-detail": `export const useWorkItemDetail=()=>({issue:{id:"issue",project_id:"p1",sequence_id:1}});`,
  "@/hooks/store/use-project": `export const useProject=()=>({getProjectById:()=>({id:'p1',identifier:'ACME'})});`,
  "@/components/relations": `export const useTimeLineRelationOptions=()=>({});`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=t=>globalThis.discussionUi.toasts.push(t);`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { IssueActivityService } = await import("../../apps/web/core/services/issue/issue_activity.service.ts");
const { IssueCommentService } = await import("../../apps/web/core/services/issue/issue_comment.service.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { workspaceKeys } = await import("../../apps/web/core/queries/workspaces.ts");
const { EActivityFilterType, E_SORT_ORDER } = await import("../../packages/constants/dist/index.mjs");
const { useIssueDiscussion } = await import("../../apps/web/core/hooks/use-issue-discussion.ts");
const { useWorkItemCommentOperations } =
  await import("../../apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx");
let Feed, Composer, FullActivity;
const defaultFilters = [EActivityFilterType.ACTIVITY, EActivityFilterType.COMMENT];
function Subject({ issueId = "i1", filters = defaultFilters, sortOrder = E_SORT_ORDER.ASC, compose = false }) {
  const discussion = useIssueDiscussion({ workspaceSlug: "acme", projectId: "p1", issueId });
  const operations = useWorkItemCommentOperations("acme", "p1", issueId, discussion);
  return React.createElement(
    React.Fragment,
    null,
    compose
      ? React.createElement(Composer, {
          key: issueId,
          workspaceSlug: "acme",
          projectId: "p1",
          entityId: issueId,
          activityOperations: operations,
        })
      : null,
    React.createElement(Feed, {
      workspaceSlug: "acme",
      projectId: "p1",
      issueId,
      isIntakeIssue: false,
      selectedFilters: filters,
      sortOrder,
      activityOperations: operations,
      discussion,
    })
  );
}
const comment = {
  id: "c1",
  project: "p1",
  workspace: "w1",
  issue: "i1",
  actor: "alice",
  created_by: "alice",
  actor_detail: { id: "alice", display_name: "Alice" },
  created_at: "2026-10-25T02:00:00Z",
  comment_html: "Canonical comment",
  comment_reactions: [],
};
async function fixture(t) {
  const f = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/",
    workspaceRole: 15,
    projectRoles: { p1: 15 },
  });
  f.client.setQueryData(workspaceKeys.list("alice"), [{ id: "w1", slug: "acme" }]);
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => []);
  t.mock.method(ProjectService.prototype, "getProject", async () => ({ id: "p1", identifier: "ACME" }));
  ({ IssueActivityCommentRoot: Feed } =
    await import("../../apps/web/core/components/issues/issue-detail/issue-activity/activity-comment-root.tsx"));
  ({ CommentCreate: Composer } = await import("../../apps/web/core/components/comments/comment-create.tsx"));
  ({ IssueActivity: FullActivity } =
    await import("../../apps/web/core/components/issues/issue-detail/issue-activity/root.tsx"));
  globalThis.discussionUi.toasts = [];
  globalThis.discussionUi.clears = 0;
  t.after(settleQuery);
  return {
    ...f,
    renderElement: f.render,
    render: async (props = {}) => {
      await f.render(React.createElement(React.StrictMode, null, React.createElement(Subject, props)));
      await settleQuery();
    },
  };
}
async function enter(f, value) {
  await act(async () => {
    const input = f.environment.container.querySelector('input[aria-label="Comment"]');
    assert.ok(input, "the comment composer is mounted");
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
await test("actual native discussion renders canonical comments and activity in selected chronological order", async (t) => {
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => [
    {
      id: "a1",
      project: "p1",
      issue: "i1",
      field: "name",
      new_value: "Earlier name",
      created_at: "2026-10-25T01:00:00Z",
      workspace_detail: { slug: "acme" },
      actor_detail: { id: "alice", display_name: "Alice" },
    },
  ]);
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => [comment]);
  const f = await fixture(t);
  await f.render();
  assert.ok(
    f.environment.container.textContent.indexOf("Earlier name") <
      f.environment.container.textContent.indexOf("Canonical comment")
  );
  await f.render({ sortOrder: E_SORT_ORDER.DESC });
  assert.ok(
    f.environment.container.textContent.indexOf("Canonical comment") <
      f.environment.container.textContent.indexOf("Earlier name")
  );
  await f.render({ filters: [EActivityFilterType.COMMENT] });
  assert.match(f.environment.container.textContent, /Canonical comment/);
  assert.doesNotMatch(f.environment.container.textContent, /Earlier name/);
});
await test("actual native composer retains a failed draft then renders the canonical saved comment", async (t) => {
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => []);
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => []);
  let fail = true;
  t.mock.method(IssueCommentService.prototype, "createIssueComment", async (_ws, _p, _i, data) => {
    if (fail) throw Error("offline");
    return { ...comment, comment_html: data.comment_html };
  });
  const f = await fixture(t);
  await f.render({ compose: true });
  await enter(f, "<p>Draft</p>");
  await act(async () => f.environment.container.querySelector("button").click());
  await settleQuery();
  assert.equal(f.environment.container.querySelector("input").value, "<p>Draft</p>");
  assert.equal(globalThis.discussionUi.clears, 0);
  fail = false;
  await act(async () => f.environment.container.querySelector("button").click());
  await settleQuery();
  assert.match(f.environment.container.textContent, /<p>Draft<\/p>/);
  assert.equal(f.environment.container.querySelector("input").value, "<p></p>");
  assert.equal(globalThis.discussionUi.clears, 1);
});
await test("old accepted comment cannot clear or notify a replacement item's actual composer", async (t) => {
  const write = Promise.withResolvers();
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => []);
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => []);
  t.mock.method(IssueCommentService.prototype, "createIssueComment", () => write.promise);
  const f = await fixture(t);
  await f.render({ compose: true });
  await enter(f, "Old draft");
  await act(async () => f.environment.container.querySelector("button").click());
  await f.render({ compose: true, issueId: "i2" });
  await enter(f, "Replacement draft");
  await act(async () => {
    write.resolve(comment);
    await write.promise;
  });
  await settleQuery();
  assert.equal(f.environment.container.querySelector("input").value, "Replacement draft");
  assert.equal(globalThis.discussionUi.clears, 0);
  assert.deepEqual(globalThis.discussionUi.toasts, []);
});
await test("failed canonical refresh hides cached discussion and Retry recovers the actual observer", async (t) => {
  let fail = false;
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => {
    if (fail) throw Error("offline");
    return [];
  });
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => [comment]);
  const f = await fixture(t);
  await f.render();
  assert.match(f.environment.container.textContent, /Canonical comment/);
  fail = true;
  await act(async () => {
    await f.client.invalidateQueries({ predicate: (q) => q.queryKey[2] === "issue-discussion" });
  });
  await settleQuery();
  assert.match(f.environment.container.textContent, /Could not load discussion/);
  assert.doesNotMatch(f.environment.container.textContent, /Canonical comment/);
  fail = false;
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Retry").click()
  );
  await settleQuery();
  assert.match(f.environment.container.textContent, /Canonical comment/);
});

await test("the actual activity root replaces a pending editor on same-item session replacement", async (t) => {
  const write = Promise.withResolvers();
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => []);
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => []);
  t.mock.method(IssueCommentService.prototype, "createIssueComment", () => write.promise);
  const f = await fixture(t);
  await f.render();
  await f.renderElement(
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(FullActivity, { workspaceSlug: "acme", projectId: "p1", issueId: "i1" })
    )
  );
  await settleQuery();
  await enter(f, "Old session draft");
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Send").click()
  );
  const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
  await act(async () => advanceIdentitySession(f.client));
  await settleQuery();
  assert.equal(f.environment.container.querySelector("input").value, "<p></p>");
  await enter(f, "Replacement session draft");
  await act(async () => {
    write.resolve(comment);
    await write.promise;
  });
  await settleQuery();
  assert.equal(f.environment.container.querySelector("input").value, "Replacement session draft");
  assert.equal(globalThis.discussionUi.clears, 0);
  assert.deepEqual(globalThis.discussionUi.toasts, []);
});

await test("a canonical refresh error keeps the actual activity editor draft while disabling submission until Retry", async (t) => {
  let fail = false;
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => {
    if (fail) throw Error("offline");
    return [];
  });
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => []);
  const f = await fixture(t);
  await f.renderElement(React.createElement(FullActivity, { workspaceSlug: "acme", projectId: "p1", issueId: "i1" }));
  await settleQuery();
  await enter(f, "Unsent draft");
  fail = true;
  await act(async () => f.client.invalidateQueries({ predicate: (q) => q.queryKey[2] === "issue-discussion" }));
  await settleQuery();
  assert.equal(f.environment.container.querySelector("input").value, "Unsent draft");
  const send = () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Send");
  assert.equal(send().disabled, true);
  fail = false;
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Retry").click()
  );
  await settleQuery();
  assert.equal(f.environment.container.querySelector("input").value, "Unsent draft");
  assert.equal(send().disabled, false);
});

await test("accepted comment remains successful when attachment dispatch loses verification and retries without reposting", async (t) => {
  const response = Promise.withResolvers(),
    started = Promise.withResolvers();
  let posts = 0,
    assets = 0;
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => []);
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => []);
  t.mock.method(IssueCommentService.prototype, "createIssueComment", () => {
    posts++;
    started.resolve();
    return response.promise;
  });
  const { FileService } = await import("../../apps/web/core/services/file.service.ts");
  t.mock.method(FileService.prototype, "updateBulkProjectAssetsUploadStatus", async () => {
    assets++;
  });
  const f = await fixture(t);
  await f.renderElement(React.createElement(FullActivity, { workspaceSlug: "acme", projectId: "p1", issueId: "i1" }));
  await settleQuery();
  await enter(f, "Posted once");
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Upload").click()
  );
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Send").click()
  );
  await started.promise;
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const identity = f.client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  await act(async () => {
    identity.setState({ status: "error", error: new Error("verification unavailable") });
    response.resolve(comment);
    await response.promise;
  });
  await settleQuery();
  assert.equal(posts, 1);
  assert.equal(assets, 0);
  assert.equal(f.environment.container.querySelector("input").value, "<p></p>");
  assert.match(f.environment.container.textContent, /Your comment was posted/);
  await act(async () => identity.setState({ status: "success", error: null }));
  await settleQuery();
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")]
      .find((button) => button.textContent === "Retry attachments")
      .click()
  );
  await settleQuery();
  assert.equal(posts, 1);
  assert.equal(assets, 1);
  assert.doesNotMatch(f.environment.container.textContent, /Your comment was posted/);
});

await test("one Retry rebuilds fresh permission and discussion options after membership revision recovery", async (t) => {
  t.mock.method(IssueActivityService.prototype, "getIssueActivities", async () => []);
  t.mock.method(IssueCommentService.prototype, "getIssueComments", async () => [comment]);
  const f = await fixture(t);
  await f.render();
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => ({
    is_active: true,
    role: 15,
    updated_at: "recovered",
  }));
  const membership = f.client
    .getQueryCache()
    .find({ queryKey: permissionKeys.workspaceMembership("alice", "acme"), exact: true });
  await act(async () => membership.setState({ status: "error", error: new Error("temporary permission error") }));
  await settleQuery();
  assert.match(f.environment.container.textContent, /Could not load discussion/);
  await act(async () =>
    [...f.environment.container.querySelectorAll("button")].find((button) => button.textContent === "Retry").click()
  );
  await settleQuery();
  assert.match(f.environment.container.textContent, /Canonical comment/);
  assert.equal(
    f.client.getQueryState(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "recovered")).status,
    "success"
  );
});
