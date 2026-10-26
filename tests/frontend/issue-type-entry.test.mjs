import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act } from "./helpers/react.mjs";
import { setup, state } from "./helpers/issue-form-harness.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
const stubs = {
  "@/providers/work-item-detail-record": `export const useWorkItemDetailRecord=()=>{const issueId=globalThis.issueFormWorkflow.currentIssueId??"first";const issue=globalThis.issueFormWorkflow.items[issueId];return {workspaceSlug:"workspace",projectId:issue.project_id,issueId:issue.id,serviceType:"issues",issue}};`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails=()=>({projects:[{id:"first",identifier:"PROJ"}]});`,
  "./issue-detail/issue-identifier":
    "export const IssueIdentifier=(props)=>{globalThis.issueFormWorkflow.identifierProps=props;return null;};",
  "@/components/work-item-types/controls":
    "const R=globalThis.issueFormReact;export const TypeIcon=({type})=>R.createElement('span',{'aria-label':type.icon});",
  "./issue-modal/modal":
    "const R=globalThis.issueFormReact;export const CreateUpdateIssueModal=({data,onClose,onSubmit})=>R.createElement('section',{'aria-label':'Update '+data.id},R.createElement('button',{onClick:onClose},'Close editor'),R.createElement('button',{onClick:()=>void Promise.resolve(onSubmit?.()).then(onClose)},'Save'));",
};
registerHooks({
  resolve(specifier, context, next) {
    return context.parentURL?.endsWith("/issues/issue-type-switcher.tsx")
      ? resolvePackageStub(specifier, stubs, context, next)
      : next(specifier, context);
  },
});
const { IssueTypeSwitcher } = await import("../../apps/web/core/components/issues/issue-type-switcher.tsx");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
async function fixture(t) {
  const env = await setup(t);
  const key = [
    "workspace",
    "workspace",
    "issue-discussion",
    "workspace-query-test-user",
    0,
    "first",
    "first",
    "issues",
    20,
    "v1",
    20,
    20,
    "activity",
  ];
  env.client.setQueryData(key, []);
  state.items = {
    first: { id: "first", project_id: "first", sequence_id: 1, type_id: "inactive" },
    second: { id: "second", project_id: "first", sequence_id: 2, type_id: "task" },
  };
  state.types = {
    enabled: true,
    types: [
      { id: "inactive", name: "Retired incident", icon: "incident", color: "#ff0000", active: false, available: false },
      { id: "task", name: "Task", icon: "task", color: "#0000ff", active: true, available: true },
    ],
  };
  const notices = [];
  const render = (overrides = {}) => {
    state.currentIssueId = overrides.issueId ?? "first";
    return env.render(
      React.createElement(IssueTypeSwitcher, {
        workspaceSlug: "workspace",
        issueId: "first",
        disabled: false,
        onEditorOpenChange: (value) => notices.push(value),
        ...overrides,
      })
    );
  };
  await render();
  return {
    ...env,
    render,
    notices,
    trigger: () => env.container.querySelector("button"),
    identifierProps: () => globalThis.issueFormWorkflow.identifierProps,
  };
}
await test("saved inactive types keep their identity and open the reviewed update entry", async (t) => {
  const f = await fixture(t);
  assert.equal(f.identifierProps().projectIdentifier, "PROJ");
  assert.equal(f.identifierProps().issueSequenceId, 1);
  assert.equal("issueId" in f.identifierProps(), false, "the identifier uses the already-loaded issue record");
  assert.equal(f.trigger().getAttribute("aria-label"), "Change work item type: Retired incident");
  assert.ok(f.container.querySelector('[aria-label="incident"]'));
  await act(async () => f.trigger().click());
  assert.ok(f.container.querySelector('[aria-label="Update first"]'));
  assert.equal(f.notices.at(-1), true);
  await act(async () => f.container.querySelector("section button").click());
  assert.equal(f.container.querySelector("section"), null);
  assert.equal(f.notices.at(-1), false);
});
await test("read-only and archived presentations retain the type without a switch action", async (t) => {
  const f = await fixture(t);
  await f.render({ disabled: true });
  assert.match(f.container.textContent, /Retired incident/);
  assert.equal(f.trigger(), null);
  await act(async () =>
    f.client.setQueryData(permissionKeys.workspaceProjectRoles("workspace-query-test-user", "workspace", 20, "v1"), {
      first: 5,
      second: 20,
    })
  );
  // A workspace admin still has effective admin rights; use a member role to check the viewer project boundary.
  await act(async () => {
    f.client.setQueryData(permissionKeys.workspaceMembership("workspace-query-test-user", "workspace"), {
      is_active: true,
      role: 15,
      updated_at: "viewer",
    });
    f.client.setQueryData(
      permissionKeys.workspaceProjectRoles("workspace-query-test-user", "workspace", 15, "viewer"),
      { first: 5 }
    );
  });
  await f.render();
  assert.match(f.container.textContent, /Retired incident/);
  assert.equal(f.trigger(), null);
});
await test("navigation and permission loss unmount the old type editor", async (t) => {
  const f = await fixture(t);
  await act(async () => f.trigger().click());
  await f.render({ issueId: "second" });
  assert.equal(f.container.querySelector("section"), null);
  assert.match(f.trigger().textContent, /Task/);
  await act(async () => f.trigger().click());
  await act(async () =>
    f.client.setQueryData(permissionKeys.workspaceMembership("workspace-query-test-user", "workspace"), {
      is_active: false,
      role: 20,
      updated_at: "revoked",
    })
  );
  assert.equal(f.container.querySelector("section"), null);
  assert.doesNotMatch(f.container.textContent, /Task/);
  assert.equal(f.notices.at(-1), false);
});

await test("accepted type editor closes after saving", async (t) => {
  const f = await fixture(t);
  await act(async () => f.trigger().click());
  await act(async () =>
    [...f.container.querySelectorAll("section button")].find((b) => b.textContent === "Save").click()
  );
  assert.equal(f.container.querySelector("section"), null);
});
