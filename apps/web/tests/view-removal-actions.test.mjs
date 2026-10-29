import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  clickAndSettle as click,
  createPermissionUiFixture,
  settleQuery,
} from "../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";
globalThis.React = React;
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=value=>globalThis.removalToasts.push(value);",
  "@plane/propel/icons":
    "export const LinkIcon=()=>null,CopyIcon=()=>null,NewTabIcon=()=>null,EditIcon=()=>null,ArchiveIcon=()=>null,TrashIcon=()=>null;",
  "@/hooks/use-projects": "export const useWorkspaceProjectDetails=()=>({projects:[]});",
  "@/hooks/use-project-states": "export const useProjectStates=()=>({getStateById:()=>undefined});",
});
const { useCycleIssueMenuItems, useModuleIssueMenuItems } =
  await import("../core/components/issues/issue-layouts/quick-action-dropdowns/helper.tsx");
const identity = await import("../core/queries/identity.ts");
async function setup(t, kind, { unavailable = false } = {}) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  globalThis.removalToasts = [];
  const writes = [];
  const transport = { write: () => Promise.resolve() };
  let items;
  function Probe({ parentId, readOnly }) {
    const props = {
      issue: { id: "item", project_id: "p1", sequence_id: 1 },
      workspaceSlug: "acme",
      projectIdentifier: "P1",
      isEditingAllowed: !readOnly,
      isDeletingAllowed: false,
      setIssueToEdit() {},
      setCreateUpdateIssueModal() {},
      setDeleteIssueModal() {},
      [`${kind}Id`]: parentId,
      handleRemoveFromView: unavailable
        ? undefined
        : () => {
            writes.push(parentId);
            return transport.write();
          },
    };
    items = (kind === "cycle" ? useCycleIssueMenuItems : useModuleIssueMenuItems)(props);
    return React.createElement(
      "div",
      null,
      ...items
        .filter((item) => item.shouldRender !== false)
        .map((item) =>
          React.createElement(
            "button",
            { key: item.key, "data-action": item.key, disabled: item.disabled, onClick: item.action },
            item.title
          )
        )
    );
  }
  const render = async (parentId = "first", readOnly = false, copies = 1) => {
    await fixture.render(
      React.createElement(
        React.Fragment,
        null,
        ...Array.from({ length: copies }, (_, index) => React.createElement(Probe, { key: index, parentId, readOnly }))
      )
    );
    await settleQuery();
  };
  await render();
  return {
    ...fixture,
    writes,
    transport,
    render,
    action: () => items.find((item) => item.key === `remove-from-${kind}`).action,
    button: () => fixture.environment.container.querySelector(`[data-action="remove-from-${kind}"]`),
  };
}
for (const kind of ["cycle", "module"]) {
  void test(`${kind} removal prevents duplicate pending requests`, async (t) => {
    const v = await setup(t, kind);
    const pending = Promise.withResolvers();
    v.transport.write = () => pending.promise;
    const control = v.button();
    await act(() => {
      control.click();
      control.click();
    });
    await settleQuery();
    const writesBefore = [...v.writes];
    const disabled = v.button().disabled;
    await act(() => pending.resolve());
    await settleQuery();
    assert.deepEqual(writesBefore, ["first"]);
    assert.equal(disabled, true);
    assert.equal(v.button().disabled, false);
  });
  void test(`${kind} rejected removal reports one error and permits retry`, async (t) => {
    const v = await setup(t, kind);
    const pending = Promise.withResolvers();
    v.transport.write = () => pending.promise;
    await click(v.button());
    await act(() => pending.reject(new Error("Removal failed")));
    await settleQuery();
    assert.deepEqual(
      globalThis.removalToasts.map((value) => value.type),
      ["error"]
    );
    assert.equal(v.button().disabled, false);
    v.transport.write = () => Promise.resolve();
    await click(v.button());
    assert.deepEqual(v.writes, ["first", "first"]);
  });
  void test(`${kind} old-parent action cannot submit or report a late failure`, async (t) => {
    const v = await setup(t, kind);
    const pending = Promise.withResolvers();
    v.transport.write = () => pending.promise;
    const oldAction = v.action();
    await click(v.button());
    await v.render("second");
    await act(() => oldAction());
    const writesBefore = [...v.writes];
    await act(() => pending.reject(new Error("Old removal failed")));
    await settleQuery();
    assert.deepEqual(writesBefore, ["first"]);
    assert.deepEqual(globalThis.removalToasts, []);
    v.transport.write = () => Promise.resolve();
    await click(v.button());
    assert.deepEqual(v.writes, ["first", "second"]);
  });
  void test(`${kind} old editable action cannot run after read-only replacement`, async (t) => {
    const v = await setup(t, kind);
    const oldAction = v.action();
    await v.render("first", true);
    assert.equal(v.button(), null);
    await act(() => oldAction());
    await settleQuery();
    assert.deepEqual(v.writes, []);
  });
  void test(`${kind} unavailable removal cannot be invoked`, async (t) => {
    const v = await setup(t, kind, { unavailable: true });
    assert.equal(v.button()?.disabled ?? true, true);
  });
  void test(`${kind} late rejected removal is silent after session replacement`, async (t) => {
    const v = await setup(t, kind);
    const pending = Promise.withResolvers();
    v.transport.write = () => pending.promise;
    await click(v.button());
    await act(() => identity.advanceIdentitySession(v.client));
    await settleQuery();
    await act(() => pending.reject(new Error("Old removal failed")));
    await settleQuery();
    assert.deepEqual(globalThis.removalToasts, []);
  });
  void test(`${kind} two visible menus share pending removal ownership`, async (t) => {
    const v = await setup(t, kind);
    await v.render("first", false, 2);
    const pending = Promise.withResolvers();
    v.transport.write = () => pending.promise;
    const controls = v.environment.container.querySelectorAll(`[data-action="remove-from-${kind}"]`);
    assert.equal(controls.length, 2);
    await act(() => {
      controls[0].click();
      controls[1].click();
    });
    await settleQuery();
    const writesBefore = [...v.writes];
    const disabled = [...controls].map((control) => control.disabled);
    await act(() => pending.resolve());
    await settleQuery();
    assert.deepEqual(writesBefore, ["first"]);
    assert.deepEqual(disabled, [true, true]);
  });
  void test(`${kind} parent round trip does not revive a retired menu action`, async (t) => {
    const v = await setup(t, kind);
    const oldAction = v.action();
    await v.render("second");
    await v.render("first");
    await act(() => oldAction());
    await settleQuery();
    assert.deepEqual(v.writes, []);
    await click(v.button());
    assert.deepEqual(v.writes, ["first"]);
  });
  void test(`${kind} missing parent disables removal`, async (t) => {
    const v = await setup(t, kind);
    await v.render("");
    assert.equal(v.button().disabled, true);
    await click(v.button());
    assert.deepEqual(v.writes, []);
  });
  void test(`${kind} late rejected removal is silent after unmount`, async (t) => {
    const v = await setup(t, kind);
    const pending = Promise.withResolvers();
    v.transport.write = () => pending.promise;
    await click(v.button());
    await v.render(null, true, 0);
    await act(() => pending.reject(new Error("Unmounted removal failed")));
    await settleQuery();
    assert.deepEqual(globalThis.removalToasts, []);
  });
}
