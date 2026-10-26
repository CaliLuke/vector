import { registerWebTestModules, reactModuleUrl as reactUrl } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";
import test from "node:test";

import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

const stubs = {
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@/hooks/use-work-item-detail": `export const useWorkItemDetailScope=()=>({scope:{accountId:"alice",sessionGeneration:0,workspaceSlug:"workspace",membershipRole:20,membershipRevision:"m1",projectId:"child-project",projectMembershipRole:5,projectRole:20,serviceType:"issues"},isReady:true});export const useWorkItemDetail=(...args)=>{globalThis.parentFixture.readArgs=args;return {issue:globalThis.parentFixture.parentReadIssue};};`,
  "@/queries/issue-details": `export const captureIssueSubItemsReconciliation=(_client,scope,ids)=>{globalThis.parentFixture.parentQueries=[scope,ids];return {refresh:async()=>globalThis.parentFixture.calls.push(["refresh-parent-queries"])};};`,
  "@/providers/work-item-detail-record": `export const useWorkItemDetailRecord=()=>globalThis.parentFixture.record;`,
  "./issue-detail-widgets/sub-issues/helper": `export const useSubIssueOperations=(...scope)=>{globalThis.parentFixture.scope=scope;return {removeSubIssue:async (...args)=>globalThis.parentFixture.calls.push(["unlink",...args])};};`,
  "@/components/issues/issue-detail/parent-select": `import React from ${JSON.stringify(reactUrl)};
    export const IssueParentSelect=(props)=>{globalThis.parentFixture.parentSelectProps=props;return React.createElement("div",null,
      React.createElement("button",{onClick:()=>props.handleParentIssue("new-parent","new-project")},"Select parent"),
      React.createElement("button",{onClick:()=>props.handleRemoveSubIssue()},"Remove parent"));};`,
};
registerWebTestModules(stubs);
const { IssueParentSelectRoot } = await import("../../apps/web/core/components/issues/parent-select-root.tsx");
const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");

await test("cross-project parent changes update the child and scope sub-item actions to the parent project", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient();
  t.after(() => env.unmount());
  t.after(() => client.clear());
  const calls = [];
  const child = { id: "child", project_id: "child-project", parent_id: "parent" };
  const parent = { id: "parent", project_id: "parent-project" };
  globalThis.parentFixture = {
    record: {
      workspaceSlug: "workspace",
      projectId: "child-project",
      issueId: "child",
      serviceType: "issues",
      issue: { ...child, parent },
    },
    calls,
  };
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(IssueParentSelectRoot, {
        workspaceSlug: "workspace",
        projectId: "child-project",
        issueId: "child",
        issueOperations: {
          update: async (...args) => calls.push(["update", ...args]),
          fetch: async (...args) => calls.push(["fetch", ...args]),
        },
      })
    )
  );
  await act(async () => document.querySelectorAll("button")[0].click());
  assert.deepEqual(calls, [
    ["update", "workspace", "child-project", "child", { parent_id: "new-parent" }],
    ["refresh-parent-queries"],
    ["fetch", "workspace", "child-project", "child", false],
  ]);
  assert.deepEqual(globalThis.parentFixture.parentQueries[1], ["parent", "new-parent"]);
  calls.length = 0;
  await act(async () => document.querySelectorAll("button")[1].click());
  assert.deepEqual(calls, [["unlink", "workspace", "child-project", "parent", "child"]]);
  assert.deepEqual(globalThis.parentFixture.scope, ["workspace", "parent-project", "parent", "issues"]);
});

await test("parent_id fallback keeps the relation while waiting for the canonical parent read", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient();
  t.after(() => env.unmount());
  t.after(() => client.clear());
  const child = { id: "child", project_id: "child-project", parent_id: "parent", parent: {} };
  globalThis.parentFixture = {
    record: {
      workspaceSlug: "workspace",
      projectId: "child-project",
      issueId: "child",
      serviceType: "issues",
      issue: child,
    },
    parentReadIssue: undefined,
    calls: [],
  };
  const render = () =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(IssueParentSelectRoot, {
          workspaceSlug: "workspace",
          projectId: "child-project",
          issueId: "child",
          issueOperations: { update: async () => {}, fetch: async () => {} },
        })
      )
    );

  await render();
  assert.deepEqual(globalThis.parentFixture.readArgs.slice(0, 3), ["workspace", "child-project", "parent"]);
  assert.equal(globalThis.parentFixture.parentSelectProps.hasParentRelation, true);
  assert.equal(globalThis.parentFixture.parentSelectProps.workItemLink, undefined);
  assert.doesNotMatch(globalThis.parentFixture.parentSelectProps.workItemLink ?? "", /undefined/);

  globalThis.parentFixture.parentReadIssue = {
    id: "parent",
    project_id: "parent-project",
    sequence_id: 17,
    name: "Parent task",
  };
  await render();
  assert.equal(
    globalThis.parentFixture.parentSelectProps.workItemLink,
    "/workspace/projects/parent-project/issues/parent"
  );
  assert.equal(globalThis.parentFixture.parentSelectProps.parentIssue.sequence_id, 17);
});
