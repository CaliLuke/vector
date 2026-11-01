import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules, reactModuleUrl } from "./helpers/web-test-modules.mjs";
const state = { calls: [], toasts: [], failRelation: false };
globalThis.widgetModalFlows = state;
const react = `import React from ${JSON.stringify(reactModuleUrl)};`;
registerWebTestModules({
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/ui":
    react +
    `export const CustomMenu=({children,disabled})=>React.createElement('fieldset',{disabled},children);CustomMenu.MenuItem=({children,onClick})=>React.createElement('button',{type:'button',onClick},children);`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:'error'};export const setToast=value=>globalThis.widgetModalFlows.toasts.push(value);`,
  "@/hooks/use-issue-link-operations": `export const useLinkOperations=()=>({});`,
  "@/hooks/use-issue-resources": `export const useIssueRelations=scope=>({create:async(relation,ids)=>{globalThis.widgetModalFlows.calls.push(['relate',scope,relation,ids]);if(globalThis.widgetModalFlows.failRelation)throw new Error('Relation offline');}});`,
  "./sub-issues/helper": `export const useSubIssueOperations=(...scope)=>({refreshSubIssues:parent=>globalThis.widgetModalFlows.calls.push(['refresh',scope,parent]),addSubIssue:async(...args)=>globalThis.widgetModalFlows.calls.push(['add',scope,...args])});`,
  "../issue-modal/modal":
    react +
    `export const CreateUpdateIssueModal=({isOpen,data,onSubmit,onClose})=>isOpen?React.createElement('section',{role:'dialog','data-flow':'create','data-parent':data.parent_id,'data-project':data.project_id},React.createElement('button',{type:'button',onClick:()=>void onSubmit({id:'created',parent_id:data.parent_id})},'Created child'),React.createElement('button',{type:'button',onClick:onClose},'Close create')):null;`,
  "../issue-detail/links/create-update-link-modal":
    react +
    `export const IssueLinkCreateUpdateModal=({isModalOpen,handleOnClose})=>isModalOpen?React.createElement('section',{role:'dialog','data-flow':'link'},React.createElement('button',{type:'button',onClick:handleOnClose},'Close link')):null;`,
  "@/components/core/modals/existing-issues-list-modal":
    react +
    `export function ExistingIssuesListModal({isOpen,handleClose,handleOnSubmit,searchParams,projectId}){
    const [error,setError]=React.useState('');
    const submit=async data=>{try{await handleOnSubmit(data);setError('');}catch(cause){setError(cause.message);}};
    return isOpen?React.createElement('section',{role:'dialog','data-flow':searchParams.issue_relation?'relation':'existing','data-parent':searchParams.issue_id,'data-project':projectId},
      error&&React.createElement('p',{role:'alert'},error),
      React.createElement('button',{type:'button',onClick:()=>void submit([{id:'picked',project_id:'p1'}])},'Accept selection'),
      React.createElement('button',{type:'button',onClick:()=>void submit([])},'Empty selection'),
      React.createElement('button',{type:'button',onClick:handleClose},'Close selection')):null;
  }`,
});
const { IssueDetailWidgetModals } =
  await import("../../apps/web/core/components/issues/issue-detail-widgets/issue-detail-widget-modals.tsx");
const { SubIssuesActionButton } =
  await import("../../apps/web/core/components/issues/issue-detail-widgets/sub-issues/quick-action-button.tsx");
const { IssueDetailUiStateProvider, useIssueDetailUiState } =
  await import("../../apps/web/core/providers/issue-detail-ui-state.tsx");
const { WorkItemDetailRecordProvider } = await import("../../apps/web/core/providers/work-item-detail-record.tsx");
const { EIssueServiceType } = await import("../../packages/types/dist/index.mjs");
const button = (label) => [...document.querySelectorAll("button")].find((item) => item.textContent === label);
const click = async (label) => {
  assert.ok(button(label), label);
  await act(async () => button(label).click());
};
async function fixture(t, serviceType = EIssueServiceType.ISSUES, hideWidgets, targetProject = "p1") {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  state.calls = [];
  state.toasts = [];
  state.failRelation = false;
  let current;
  function Controls() {
    const ui = useIssueDetailUiState(serviceType);
    React.useLayoutEffect(() => {
      current = ui;
    }, [ui]);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(
        "button",
        {
          onClick: () => {
            ui.setIssueLinkData({ id: "link" });
            ui.toggleIssueLinkModal(true);
          },
        },
        "Open link"
      ),
      React.createElement(
        "button",
        {
          onClick: () => {
            ui.setRelationKey("relates_to");
            ui.toggleRelationModal("root", "relates_to");
          },
        },
        "Open relation"
      )
    );
  }
  const record = {
    workspaceSlug: "acme",
    projectId: targetProject,
    issueId: "nested",
    serviceType,
    issue: { id: "nested", project_id: targetProject },
  };
  await env.render(
    React.createElement(
      IssueDetailUiStateProvider,
      null,
      React.createElement(Controls),
      React.createElement(
        WorkItemDetailRecordProvider,
        { value: record },
        React.createElement(SubIssuesActionButton, { issueId: "nested", issueServiceType: serviceType })
      ),
      React.createElement(IssueDetailWidgetModals, {
        workspaceSlug: "acme",
        projectId: "p1",
        issueId: "root",
        issueServiceType: serviceType,
        hideWidgets,
      })
    )
  );
  return { env, current: () => current };
}
for (const serviceType of [EIssueServiceType.ISSUES, EIssueServiceType.EPICS]) {
  void test(`${serviceType} existing sub-items attach to the selected nested parent`, async (t) => {
    const view = await fixture(t, serviceType);
    await click("common.add_existing");
    assert.equal(view.env.container.querySelector('[data-flow="existing"]').dataset.parent, "nested");
    await click("Accept selection");
    const call = state.calls.find((item) => item[0] === "add");
    assert.deepEqual(call, [
      "add",
      ["acme", "p1", "nested", serviceType],
      "acme",
      "p1",
      "nested",
      [{ id: "picked", project_id: "p1" }],
    ]);
    await click("Close selection");
    assert.equal(view.current().issueCrudOperationState.existing.parentIssueId, null);
    assert.equal(view.current().issueCrudOperationState.existing.projectId, null);
    assert.equal(view.current().isSubIssuesModalOpen, null);
    assert.equal(view.current().lastWidgetAction, "sub-work-items");
  });
}
await test("created nested items refresh their own parent and closing the editor retires its create flow", async (t) => {
  const view = await fixture(t);
  await click("common.create_new");
  assert.equal(view.env.container.querySelector('[data-flow="create"]').dataset.parent, "nested");
  await click("Created child");
  assert.deepEqual(state.calls, [["refresh", ["acme", "p1", "nested", EIssueServiceType.ISSUES], "nested"]]);
  await click("Close create");
  assert.equal(view.current().isCreateIssueModalOpen, false);
  assert.equal(view.current().issueCrudOperationState.create.parentIssueId, null);
  assert.equal(view.current().issueCrudOperationState.create.projectId, null);
  assert.equal(view.current().lastWidgetAction, "sub-work-items");
});
await test("relation selection retains failures, rejects empty selections and closes after an accepted retry", async (t) => {
  const view = await fixture(t);
  await click("Open relation");
  await click("Empty selection");
  assert.equal(state.calls.length, 0);
  assert.match(state.toasts.at(-1).message, /select at least one/);
  state.failRelation = true;
  await click("Accept selection");
  assert.match(view.env.container.querySelector('[role="alert"]').textContent, /Relation offline/);
  assert.ok(view.env.container.querySelector('[data-flow="relation"]'));
  state.failRelation = false;
  await click("Accept selection");
  assert.equal(Boolean(view.env.container.querySelector('[data-flow="relation"]')), false);
  assert.deepEqual(
    state.calls.map((call) => call.slice(2)),
    [
      ["relates_to", ["picked"]],
      ["relates_to", ["picked"]],
    ]
  );
  await click("Open relation");
  await click("Close selection");
  assert.equal(view.current().relationKey, null);
  assert.equal(view.current().lastWidgetAction, "relations");
});
await test("link dismissal clears retained link data", async (t) => {
  const view = await fixture(t);
  await click("Open link");
  assert.ok(view.env.container.querySelector('[data-flow="link"]'));
  await click("Close link");
  assert.equal(view.current().issueLinkData, null);
  assert.equal(view.current().lastWidgetAction, "links");
});

await test("hidden detail widgets keep their dialog flows unmounted", async (t) => {
  const view = await fixture(t, EIssueServiceType.ISSUES, ["links", "sub-work-items", "relations"]);
  await click("common.create_new");
  await click("Open link");
  await click("Open relation");
  assert.equal(view.env.container.querySelectorAll('[role="dialog"]').length, 0);
  assert.deepEqual(state.calls, []);
});

await test("nested attachments use the parent project when its peek differs from the outer item", async (t) => {
  const view = await fixture(t, EIssueServiceType.ISSUES, undefined, "p2");
  await click("common.add_existing");
  assert.equal(view.env.container.querySelector('[data-flow="existing"]').dataset.project, "p2");
  await click("Accept selection");
  assert.deepEqual(
    state.calls.find((call) => call[0] === "add"),
    [
      "add",
      ["acme", "p2", "nested", EIssueServiceType.ISSUES],
      "acme",
      "p2",
      "nested",
      [{ id: "picked", project_id: "p1" }],
    ]
  );
});
await test("nested creation keeps the selected parent project instead of the outer detail project", async (t) => {
  const view = await fixture(t, EIssueServiceType.ISSUES, undefined, "p2");
  await click("common.create_new");
  assert.equal(view.env.container.querySelector('[data-flow="create"]').dataset.project, "p2");
  await click("Created child");
  assert.deepEqual(state.calls, [["refresh", ["acme", "p2", "nested", EIssueServiceType.ISSUES], "nested"]]);
});
