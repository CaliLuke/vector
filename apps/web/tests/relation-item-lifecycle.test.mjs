import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
  registerPermissionUiStubs,
} from "../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";
globalThis.React = React;
globalThis.relationToasts = [];
const dropdown = (exported, name) =>
  `export const ${exported}=props=>globalThis.renderRelationProperty(${JSON.stringify(name)},props);`;
globalThis.renderRelationProperty = (name, props) =>
  React.createElement(
    "button",
    {
      type: "button",
      disabled: props.disabled,
      "data-property": name,
      "data-value": JSON.stringify(props.value),
      onClick: () => props.onChange(name === "assignee" ? ["bob"] : "urgent"),
    },
    name
  );
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=value=>globalThis.relationToasts.push(value);",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/hooks/use-issue-peek-overview-redirection": "export default ()=>({handleRedirection:()=>{}});",
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails=()=>({projects:[{id:'p1',identifier:'P1'},{id:'p2',identifier:'P2'}]});",
  "@/components/issues/issue-detail/issue-identifier": "export const IssueIdentifier=()=>null;",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState=()=>({toggleCreateIssueModal:()=>{},toggleDeleteIssueModal:()=>{}});",
  "@/components/relations": "export const useTimeLineRelationOptions=()=>({relates_to:{i18n_label:'Related'}});",
  "@/components/dropdowns/state/dropdown": dropdown("StateDropdown", "state"),
  "@/components/dropdowns/priority": dropdown("PriorityDropdown", "priority"),
  "@/components/dropdowns/member/dropdown": dropdown("MemberDropdown", "assignee"),
  "@/components/issues/issue-modal/modal":
    "export const CreateUpdateIssueModal=props=>{globalThis.relationModal=props;return null;};",
  "@/components/epic-modal": "export const CreateUpdateEpicModal=props=>{globalThis.relationModal=props;return null;};",
});
const ui = `const React=globalThis.React;
export const Collapsible=({children})=>React.createElement('section',null,children);
export const ControlLink=({children,...props})=>React.createElement('a',props,children);
export const CustomMenu=({children})=>React.createElement('div',null,children);
CustomMenu.MenuItem=({children,...props})=>React.createElement('button',{type:'button',...props},children);
export const AlertModalCore=props=>React.createElement('button',{type:'button','data-delete-confirm':true,disabled:props.isSubmitting,onClick:props.handleSubmit},'Confirm delete');`;
for (const file of [
  "/relations/issue-list-item.tsx",
  "/issue-detail-widgets/relations/content.tsx",
  "/components/issues/delete-issue-modal.tsx",
])
  registerPermissionUiStubs(file, { "@plane/ui": ui });
const { RelationsCollapsibleContent } =
  await import("../core/components/issues/issue-detail-widgets/relations/content.tsx");
const { useRelationOperations } = await import("../core/components/issues/issue-detail-widgets/relations/helper.tsx");
const { useWorkItemDetail } = await import("../core/hooks/use-work-item-detail.ts");
const { IssueService } = await import("../core/services/issue/index.ts");
const { IssueRelationService } = await import("../core/services/issue/issue_relation.service.ts");
const identity = await import("../core/queries/identity.ts");
async function click(button) {
  assert.ok(button);
  await act(async () => {
    button.click();
    await Promise.resolve();
  });
  await settleQuery();
}

async function setup(
  t,
  { epic = false, operationsOnly = false, patchStatus = 200, readOnly = false, parentEpic = false } = {}
) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20 },
  });
  let row = {
    id: "related",
    project_id: "p2",
    name: "Related item",
    sequence_id: 2,
    state_id: "started",
    priority: "low",
    assignee_ids: [],
    created_by: "alice",
    is_epic: epic,
  };
  const parent = { id: "source", project_id: "p1", name: "Source", sequence_id: 1, is_epic: parentEpic };
  const paths = {
    source: `/api/workspaces/acme/projects/p1/${parentEpic ? "epics" : "issues"}/source/`,
    related: `/api/workspaces/acme/projects/p2/${epic ? "epics" : "issues"}/related/`,
  };
  const reads = [],
    writes = [];
  const transport = { patch: async () => {}, delete: async () => {}, remove: async () => {} };
  let operations;
  let linked = true;
  globalThis.relationToasts = [];
  globalThis.relationModal = undefined;
  t.mock.method(IssueService.prototype, "get", async (path) => {
    reads.push(path);
    if (path === paths.source)
      return {
        status: 200,
        data: {
          ...parent,
          issue_relation: row && linked ? [{ ...row, relation_type: "relates_to" }] : [],
          issue_related: [],
        },
      };
    if (path === paths.related && row) return { status: 200, data: row };
    throw { response: { data: { status: 404 } } };
  });
  t.mock.method(IssueService.prototype, "patch", async (path, data) => {
    writes.push(["patch", path, data]);
    assert.equal(path, paths.related);
    await transport.patch();
    row = { ...row, ...data };
    return patchStatus === 204 ? { status: 204 } : { status: 200, data: row };
  });
  t.mock.method(IssueService.prototype, "delete", async (path) => {
    writes.push(["delete", path]);
    assert.equal(path, paths.related);
    await transport.delete();
    row = undefined;
    return { status: 204 };
  });
  t.mock.method(IssueRelationService.prototype, "post", async (path, data) => {
    writes.push(["remove", path, data]);
    await transport.remove();
    linked = false;
    return { status: 200, data: {} };
  });
  function Probe({ onOperations }) {
    onOperations(useRelationOperations("acme", "p2", "related", epic ? "epics" : "issues"));
    const detail = useWorkItemDetail("acme", "p2", "related", epic ? "epics" : "issues");
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("output", null, detail.issue?.priority ?? "missing"),
      operationsOnly
        ? null
        : React.createElement(RelationsCollapsibleContent, {
            workspaceSlug: "acme",
            projectId: "p1",
            issueId: "source",
            disabled: readOnly,
            issueServiceType: parentEpic ? "epics" : "issues",
          })
    );
  }
  await fixture.render(
    React.createElement(Probe, {
      onOperations: (value) => {
        operations = value;
      },
    })
  );
  await settleQuery();
  const action = (text) =>
    [...fixture.environment.container.querySelectorAll("button")].find((button) => button.textContent.includes(text));
  return {
    ...fixture,
    paths,
    reads,
    writes,
    transport,
    action,
    click,
    property: () => fixture.environment.container.querySelector('[data-property="priority"]'),
    detail: () => fixture.environment.container.querySelector("output").textContent,
    operations: () => operations,
    row: () => row,
    accept: (data) => {
      row = { ...row, ...data };
      return row;
    },
  };
}
void test("related-item operations preserve rejected writes for their caller", async (t) => {
  const v = await setup(t, { operationsOnly: true });
  const expected = new Error("rejected");
  v.transport.patch = async () => {
    throw expected;
  };
  await act(async () => {
    await assert.rejects(v.operations().update({ priority: "urgent" }), (error) => error === expected);
  });
});
void test("relation property edits suppress duplicate pending writes", async (t) => {
  const v = await setup(t);
  const pending = Promise.withResolvers();
  v.transport.patch = () => pending.promise;
  await act(() => {
    v.property().click();
    v.property().click();
  });
  await settleQuery();
  const count = v.writes.length;
  const disabled = v.property().disabled;
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(count, 1);
  assert.equal(disabled, true);
  assert.equal(v.property().disabled, false);
});
for (const failed of [false, true])
  void test(`relation property completion after session replacement is silent (${failed ? "failure" : "success"})`, async (t) => {
    const v = await setup(t);
    const pending = Promise.withResolvers();
    v.transport.patch = () => pending.promise;
    await v.click(v.property());
    await act(() => identity.advanceIdentitySession(v.client));
    await settleQuery();
    await act(() => {
      if (failed) pending.reject(new Error("old write"));
      else pending.resolve();
    });
    await settleQuery();
    assert.deepEqual(globalThis.relationToasts, []);
  });
for (const epic of [false, true])
  void test(`accepted ${epic ? "epic" : "issue"} modal edits reconcile without another write`, async (t) => {
    const v = await setup(t, { epic });
    await v.click(v.action("common.actions.edit"));
    assert.ok(globalThis.relationModal);
    const accepted = v.accept({ priority: "urgent" });
    await act(() => globalThis.relationModal.onSubmit(accepted));
    await settleQuery();
    assert.deepEqual(v.writes, []);
    assert.equal(v.detail(), "urgent");
    assert.equal(v.property().getAttribute("data-value"), '"urgent"');
  });
for (const epic of [false, true])
  void test(`deleting a related ${epic ? "epic" : "issue"} clears its own project detail`, async (t) => {
    const v = await setup(t, { epic });
    await v.click(v.action("common.actions.delete"));
    await v.click(v.environment.container.querySelector("[data-delete-confirm]"));
    assert.deepEqual(v.writes, [["delete", v.paths.related]]);
    assert.equal(v.detail(), "missing");
    assert.equal(Boolean(v.property()), false);
  });
void test("an accepted relation property edit updates its visible relation row", async (t) => {
  const v = await setup(t);
  await v.click(v.property());
  assert.equal(v.detail(), "urgent");
  assert.equal(v.property().getAttribute("data-value"), '"urgent"');
});

for (const epic of [false, true])
  void test(`an accepted ${epic ? "epic" : "issue"} property edit with HTTP 204 refreshes detail and relation rows`, async (t) => {
    const v = await setup(t, { epic, patchStatus: 204 });
    await v.click(v.property());
    assert.equal(v.detail(), "urgent");
    assert.equal(v.property().getAttribute("data-value"), '"urgent"');
    assert.equal(v.writes.length, 1);
  });
void test("a rejected relation property save shows one error and retries", async (t) => {
  const v = await setup(t);
  v.transport.patch = async () => {
    throw new Error("save failed");
  };
  await v.click(v.property());
  assert.deepEqual(
    globalThis.relationToasts.map((x) => x.type),
    ["error"]
  );
  assert.equal(v.detail(), "low");
  assert.equal(v.property().disabled, false);
  v.transport.patch = async () => {};
  await v.click(v.property());
  assert.equal(v.writes.length, 2);
  assert.equal(v.detail(), "urgent");
  assert.deepEqual(
    globalThis.relationToasts.map((x) => x.type),
    ["error", "success"]
  );
});
void test("read-only relation rows expose no mutation controls", async (t) => {
  const v = await setup(t, { readOnly: true });
  assert.equal(v.property().disabled, true);
  for (const key of ["common.actions.edit", "common.actions.delete", "common.actions.remove_relation"])
    assert.equal(Boolean(v.action(key)), false);
  assert.deepEqual(v.writes, []);
});
void test("relation removal prevents duplicate pending requests and preserves the related item", async (t) => {
  const v = await setup(t);
  const pending = Promise.withResolvers();
  v.transport.remove = () => pending.promise;
  const button = v.action("common.actions.remove_relation");
  await act(() => {
    button.click();
    button.click();
  });
  await settleQuery();
  const count = v.writes.length;
  const disabled = button.disabled;
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(count, 1);
  assert.equal(disabled, true);
  assert.deepEqual(v.writes, [
    [
      "remove",
      "/api/workspaces/acme/projects/p1/issues/source/remove-relation/",
      { relation_type: "relates_to", related_issue: "related" },
    ],
  ]);
  assert.equal(Boolean(v.property()), false);
  assert.equal(v.detail(), "low");
});
void test("failed relation removal shows one error, keeps the row, and allows retry", async (t) => {
  const v = await setup(t);
  v.transport.remove = async () => {
    throw { response: { data: { detail: "denied" } } };
  };
  await v.click(v.action("common.actions.remove_relation"));
  assert.ok(v.property());
  assert.deepEqual(
    globalThis.relationToasts.map((x) => x.type),
    ["error"]
  );
  assert.equal(v.action("common.actions.remove_relation").disabled, false);
  v.transport.remove = async () => {};
  await v.click(v.action("common.actions.remove_relation"));
  assert.equal(v.writes.length, 2);
  assert.equal(Boolean(v.property()), false);
  assert.equal(v.detail(), "low");
});
for (const failed of [false, true])
  void test(`unmounted relation property completion is silent (${failed ? "failure" : "success"})`, async (t) => {
    const v = await setup(t);
    const pending = Promise.withResolvers();
    v.transport.patch = () => pending.promise;
    await v.click(v.property());
    await v.render(null);
    await act(() => {
      if (failed) pending.reject(new Error("late"));
      else pending.resolve();
    });
    await settleQuery();
    assert.deepEqual(globalThis.relationToasts, []);
  });
void test("an accepted edit from a retired modal cannot refresh a replacement session", async (t) => {
  const v = await setup(t);
  await v.click(v.action("common.actions.edit"));
  const acknowledge = globalThis.relationModal.onSubmit;
  await act(() => identity.advanceIdentitySession(v.client));
  await settleQuery();
  v.reads.length = 0;
  await act(() => acknowledge(v.accept({ priority: "urgent" })));
  await settleQuery();
  assert.deepEqual(v.writes, []);
  assert.deepEqual(v.reads, []);
  assert.deepEqual(globalThis.relationToasts, []);
});
void test("failed related-item deletion leaves the detail and relation row available for retry", async (t) => {
  const v = await setup(t);
  v.transport.delete = async () => {
    throw new Error("delete failed");
  };
  await v.click(v.action("common.actions.delete"));
  await v.click(v.environment.container.querySelector("[data-delete-confirm]"));
  assert.equal(v.detail(), "low");
  assert.ok(v.property());
  assert.deepEqual(
    globalThis.relationToasts.map((x) => x.type),
    ["error"]
  );
  v.transport.delete = async () => {};
  await v.click(v.environment.container.querySelector("[data-delete-confirm]"));
  assert.equal(v.detail(), "missing");
  assert.equal(Boolean(v.property()), false);
  assert.equal(v.writes.length, 2);
});

for (const epic of [false, true]) {
  void test(`accepted ${epic ? "epic" : "issue"} modal edits under an epic reconcile each own scope`, async (t) => {
    const v = await setup(t, { epic, parentEpic: true });
    await v.click(v.action("common.actions.edit"));
    await act(() => globalThis.relationModal.onSubmit(v.accept({ priority: "urgent" })));
    await settleQuery();
    assert.deepEqual(v.writes, []);
    assert.equal(v.detail(), "urgent");
    assert.equal(v.property().getAttribute("data-value"), '"urgent"');
  });
  void test(`deleting a related ${epic ? "epic" : "issue"} under an epic uses the related endpoint`, async (t) => {
    const v = await setup(t, { epic, parentEpic: true });
    await v.click(v.action("common.actions.delete"));
    await v.click(v.environment.container.querySelector("[data-delete-confirm]"));
    assert.deepEqual(v.writes, [["delete", v.paths.related]]);
    assert.equal(v.detail(), "missing");
    assert.equal(Boolean(v.property()), false);
  });
}
for (const retiredBy of ["unmount", "session"])
  void test(`related-item operations reject writes after ${retiredBy}`, async (t) => {
    const v = await setup(t, { operationsOnly: true });
    const operations = v.operations();
    if (retiredBy === "unmount") await v.render(null);
    else await act(() => identity.advanceIdentitySession(v.client));
    await settleQuery();
    await act(async () => {
      await assert.rejects(operations.update({ priority: "urgent" }));
      await assert.rejects(operations.remove());
    });
    assert.deepEqual(v.writes, []);
  });
