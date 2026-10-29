import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
} from "../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";
import { EIssuesStoreType, EIssueServiceType } from "@plane/types";
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast": "export const TOAST_TYPE={};export const setToast=()=>{};",
  "@/hooks/use-estimates": "export const useProjectEstimates=()=>({});export const useWorkspaceEstimates=()=>({});",
  "@/components/issues/issue-detail/parent-select":
    "export const IssueParentSelect=props=>{globalThis.parentReassignment=props;return null;};",
});
const { IssueParentSelectRoot } = await import("../core/components/issues/parent-select-root.tsx");
const { useWorkItemDetail, useWorkItemSubItems } = await import("../core/hooks/use-work-item-detail.ts");
const { useIssueDetailWrites } = await import("../core/hooks/use-issue-detail-writes.ts");
const { useIssuesActions } = await import("../core/hooks/use-issues-actions.tsx");
const { WorkItemDetailRecordProvider } = await import("../core/providers/work-item-detail-record.tsx");
const { IssueService } = await import("../core/services/issue/issue.service.ts");
const identity = await import("../core/queries/identity.ts");
function Root({ oldProject, oldService }) {
  const detail = useWorkItemDetail("acme", "p1", "child");
  const writes = useIssueDetailWrites(
    detail.scope,
    "child",
    detail.isReady,
    useIssuesActions(EIssuesStoreType.PROJECT)
  );
  const old = useWorkItemSubItems("acme", oldProject, "old-parent", oldService);
  const next = useWorkItemSubItems("acme", "p3", "new-parent");
  // Keep the legacy fetch callback to detect accidental extra reads after canonical reconciliation.
  const operations = {
    ...writes,
    updateText: writes.update,
    fetch: async () => {
      await detail.query.refetch();
    },
  };
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(
      "output",
      null,
      JSON.stringify({
        parent: detail.issue?.parent_id,
        old: old.subItems?.map((i) => i.id),
        next: next.subItems?.map((i) => i.id),
      })
    ),
    detail.issue
      ? React.createElement(
          WorkItemDetailRecordProvider,
          {
            value: {
              workspaceSlug: "acme",
              projectId: "p1",
              issueId: "child",
              serviceType: "issues",
              issue: detail.issue,
            },
          },
          React.createElement(IssueParentSelectRoot, {
            workspaceSlug: "acme",
            projectId: "p1",
            issueId: "child",
            issueOperations: operations,
          })
        )
      : null
  );
}
async function setup(t, options = {}) {
  const view = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20, p3: 20 },
  });
  const old = { id: "old-parent", project_id: options.oldProject ?? "p2", sequence_id: 2, is_epic: !!options.oldEpic };
  const next = { id: "new-parent", project_id: "p3", sequence_id: 3 };
  let child = { id: "child", project_id: "p1", name: "Child", parent_id: old.id, parent: options.fallback ? {} : old };
  const reads = { child: 0, parents: [], writes: 0 };
  const transport = { patch: async () => {} };
  t.mock.method(IssueService.prototype, "retrieve", async (_w, p, id) => {
    if (id === old.id) {
      assert.equal(p, old.project_id);
      await options.parentRead?.();
      return old;
    }
    assert.equal(p, "p1");
    assert.equal(id, "child");
    reads.child++;
    return child;
  });
  t.mock.method(IssueService.prototype, "subIssues", async (_w, p, id) => {
    reads.parents.push([p, id]);
    assert.equal(p, id === old.id ? old.project_id : "p3");
    return { sub_issues: child.parent_id === id ? [child] : [] };
  });
  t.mock.method(IssueService.prototype, "patchIssue", async (_w, p, id, data) => {
    assert.equal(p, "p1");
    assert.equal(id, "child");
    reads.writes++;
    await transport.patch();
    child = { ...child, ...data, parent: data.parent_id === old.id ? old : next };
    return child;
  });
  await view.render(
    React.createElement(Root, {
      oldProject: old.project_id,
      oldService: options.oldEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES,
    })
  );
  await settleQuery();
  const state = () => JSON.parse(view.environment.container.querySelector("output").textContent);
  assert.deepEqual(state(), { parent: old.id, old: ["child"], next: [] });
  return {
    ...view,
    reads,
    transport,
    state,
    select: (id = next.id, projectId = next.project_id) =>
      globalThis.parentReassignment.handleParentIssue(id, projectId),
  };
}
void test("parent reassignment refreshes both parents in their actual projects", async (t) => {
  const view = await setup(t);
  await act(() => view.select());
  await settleQuery();
  assert.deepEqual(view.state(), { parent: "new-parent", old: [], next: ["child"] });
  assert.equal(view.reads.parents.filter(([p, id]) => p === "p2" && id === "old-parent").length, 2);
  assert.equal(view.reads.parents.filter(([p, id]) => p === "p3" && id === "new-parent").length, 2);
});
void test("canonical parent selection does not reread an already reconciled child", async (t) => {
  const view = await setup(t);
  const reads = view.reads.child;
  await act(() => view.select());
  await settleQuery();
  assert.equal(view.state().parent, "new-parent");
  assert.equal(view.reads.child, reads);
});
void test("a late parent selection cannot refetch the replacement session detail", async (t) => {
  const view = await setup(t);
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  let write;
  await act(() => {
    write = view.select();
  });
  await settleQuery();
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  const reads = view.reads.child;
  const parents = view.reads.parents.length;
  await act(async () => {
    pending.resolve();
    await write;
  });
  await settleQuery();
  assert.equal(view.reads.child, reads);
  assert.equal(view.reads.parents.length, parents);
});

void test("selecting the current parent does not repeat an unchanged write", async (t) => {
  const view = await setup(t);
  await act(() => view.select("old-parent", "p2"));
  await settleQuery();
  assert.equal(view.reads.writes, 0);
  assert.deepEqual(view.state(), { parent: "old-parent", old: ["child"], next: [] });
});
void test("a rejected parent write retains the relationship and does not refresh its readers", async (t) => {
  const view = await setup(t);
  view.transport.patch = async () => {
    throw new Error("parent rejected");
  };
  const reads = view.reads.child;
  const parents = view.reads.parents.length;
  await act(async () => assert.rejects(view.select(), /parent rejected/));
  await settleQuery();
  assert.equal(view.reads.child, reads);
  assert.equal(view.reads.parents.length, parents);
  assert.deepEqual(view.state(), { parent: "old-parent", old: ["child"], next: [] });
});
void test("parent_id fallback keeps the relationship while its canonical parent loads", async (t) => {
  const pending = Promise.withResolvers();
  const view = await setup(t, { oldProject: "p1", fallback: true, parentRead: () => pending.promise });
  assert.equal(globalThis.parentReassignment.hasParentRelation, true);
  assert.equal(globalThis.parentReassignment.workItemLink, undefined);
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(globalThis.parentReassignment.workItemLink, "/acme/projects/p1/issues/old-parent");
  assert.equal(globalThis.parentReassignment.parentIssue.sequence_id, 2);
  assert.equal(view.reads.writes, 0);
});

void test("changing an epic parent refreshes the old epic and new issue readers", async (t) => {
  const view = await setup(t, { oldEpic: true });
  await act(() => view.select());
  await settleQuery();
  assert.deepEqual(view.state(), { parent: "new-parent", old: [], next: ["child"] });
});
