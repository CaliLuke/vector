import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "./helpers/module-hooks.mjs";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
const state = { assignments: [], notifications: [], fail: true };
globalThis.parentQuickAddTest = state;
const stubs = {
  "@/hooks/use-projects": `export const useProjectDetail=(workspaceSlug,projectId)=>({project:{id:projectId,identifier:"P1",name:"Project"}});`,
  "@/providers/parent-issue-preferences": `export const useOptionalParentIssuePreferences=()=>({requestInput:{kind:"cycle",workspaceSlug:"acme",projectId:"p1",viewId:"c1",parentStatistics:{isCurrent:()=>false},isScopeCurrent:()=>true}});`,
  "../../../create-issue-toast-action-items": `export const notifyIssueCreated=value=>globalThis.parentQuickAddTest.notifications.push(value);`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { QuickAddIssueForm } =
  await import("../../apps/web/core/components/issues/issue-layouts/quick-add/form/root.tsx");
const { IssueService } = await import("../../apps/web/core/services/issue/index.ts");
await test("real inline form retries an unfinished parent assignment without creating a second work item", async (t) => {
  state.assignments = [];
  state.notifications = [];
  state.fail = true;
  t.mock.method(IssueService.prototype, "addIssueToCycle", async (...args) => {
    state.assignments.push(args);
    if (state.fail) throw new Error("assignment offline");
  });
  const view = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/cycles/c1",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  let creates = 0;
  await view.render(
    React.createElement(QuickAddIssueForm, {
      workspaceSlug: "acme",
      projectId: "p1",
      layout: "list",
      isEpic: false,
      onClose() {},
      onCreate: async (_project, data) => {
        creates++;
        return { ...data, id: "accepted-issue", project_id: "p1" };
      },
    })
  );
  const input = view.environment.container.querySelector("input[name=name]");
  await changeFormControl(input, "A durable work item");
  await submitForm(view.environment.container);
  await settleQuery();
  assert.equal(creates, 1);
  assert.equal(state.assignments.length, 1);
  assert.deepEqual(state.assignments[0], ["acme", "p1", "c1", { issues: ["accepted-issue"] }]);
  assert.match(view.environment.container.querySelector("[role=alert]").textContent, /created.*Retry/i);
  assert.equal(input.disabled, true, "the accepted title cannot become a new creation command");
  assert.equal(state.notifications.length, 0);
  state.fail = false;
  const retry = [...view.environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Retry finishing work item"
  );
  assert.ok(retry);
  await act(async () => retry.click());
  await settleQuery();
  assert.equal(creates, 1);
  assert.equal(state.assignments.length, 2);
  assert.deepEqual(state.assignments[1], ["acme", "p1", "c1", { issues: ["accepted-issue"] }]);
  assert.equal(
    state.notifications.length,
    1,
    `retry should finish the current form: ${view.environment.container.textContent}`
  );
  assert.equal(input.value, "");
  assert.equal(input.disabled, false);
});

for (const layout of ["list", "kanban"]) {
  void test(`${layout} group creation honors restrictions and submits its group defaults through the real form`, async (t) => {
    const view = await createPermissionUiFixture(t, {
      path: "/acme/projects/p1/cycles/c1",
      workspaceRole: 15,
      projectRoles: { p1: 20 },
    });
    t.mock.method(IssueService.prototype, "addIssueToCycle", async () => {});
    const { GroupQuickAdd } = await import("../../apps/web/core/components/issues/issue-layouts/quick-add/root.tsx");
    const writes = [];
    const props = {
      layout,
      enabled: true,
      creationDisabled: false,
      groupBy: "state",
      isEpic: false,
      prePopulate: () => ({ state_id: "active", priority: "high" }),
      quickAddCallback: async (projectId, data) => {
        writes.push({ projectId, data });
        return { ...data, id: "group-item", project_id: projectId };
      },
    };
    await view.render(React.createElement(GroupQuickAdd, { ...props, creationDisabled: true }));
    assert.equal(view.environment.container.querySelector("button"), null);
    await view.render(React.createElement(GroupQuickAdd, { ...props, enabled: false }));
    assert.equal(view.environment.container.querySelector("button"), null);
    await view.render(React.createElement(GroupQuickAdd, { ...props, subGroupBy: "type_id" }));
    assert.equal(view.environment.container.querySelector("button"), null);
    await view.render(React.createElement(GroupQuickAdd, props));
    const trigger = view.environment.container.querySelector("button");
    assert.ok(trigger, "permitted groups expose their native creation control");
    await act(async () => trigger.click());
    await changeFormControl(view.environment.container.querySelector("input[name=name]"), "Grouped work");
    await submitForm(view.environment.container);
    await settleQuery();
    assert.equal(writes.length, 1);
    assert.equal(writes[0].projectId, "p1");
    assert.equal(writes[0].data.state_id, "active");
    assert.equal(writes[0].data.priority, "high");
    assert.equal(writes[0].data.name, "Grouped work");
  });
}
