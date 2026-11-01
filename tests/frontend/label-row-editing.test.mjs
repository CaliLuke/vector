import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.labelRowReact = React;
registerWebTestModules({
  "@plane/ui": `export {CustomMenu} from ${JSON.stringify(new URL("../../packages/ui/src/dropdowns/custom-menu.tsx", import.meta.url).href)};
    export {DragHandle} from ${JSON.stringify(new URL("../../packages/ui/src/drag-handle.tsx", import.meta.url).href)};
    export {Input} from ${JSON.stringify(new URL("../../packages/ui/src/form-fields/input.tsx", import.meta.url).href)};`,
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error'};export const setToast=toast=>globalThis.labelRowToasts.push(toast);",
  "react-color": "export const TwitterPicker=()=>null;",
  "./label-drag-n-drop-HOC": "export const LabelDndHOC=({children})=>children(false,false,{current:null});",
});

const label = { id: "alpha", name: "Alpha", color: "#123456", parent: null, project_id: "p1", sort_order: 1 };
const menuItem = (text) =>
  Array.from(document.querySelectorAll('[role="menuitem"]')).find((node) => node.textContent === text);
const button = (text) => Array.from(document.querySelectorAll("button")).find((node) => node.textContent === text);

async function setup(env, kind, isEditable = true) {
  await env.render(null);
  globalThis.labelRowToasts = [];
  const state = { request: Promise.withResolvers(), updates: [], updating: [] };
  const Component =
    kind === "group"
      ? (await import("../../apps/web/core/components/labels/project-setting-label-group.tsx")).ProjectSettingLabelGroup
      : (await import("../../apps/web/core/components/labels/project-setting-label-item.tsx")).ProjectSettingLabelItem;
  await env.render(
    React.createElement(Component, {
      label,
      labelChildren: [],
      isUpdating: false,
      isEditable,
      isChild: false,
      isLastChild: true,
      setIsUpdating: (value) => state.updating.push(value),
      handleLabelDelete: () => {},
      onDrop: () => {},
      labelOperationsCallbacks: {
        createLabel: () => {
          throw Error("Editing must not create a label");
        },
        updateLabel: (id, data) => {
          state.updates.push({ id, data });
          return state.request.promise;
        },
      },
    })
  );
  const edit = async () => {
    await act(async () => env.container.querySelector('[aria-haspopup="menu"]').click());
    assert.ok(menuItem("Edit label"));
    await act(async () => menuItem("Edit label").click());
  };
  return { ...state, state, edit };
}

async function verifyEditing(parent, env, kind) {
  await parent.test(
    `${kind} label edits retain failed drafts, close on cancel and release the owner only after acceptance`,
    async () => {
      const { state, edit } = await setup(env, kind);
      await edit();
      assert.equal(env.container.querySelector('input[name="name"]').value, "Alpha");
      assert.deepEqual(state.updating, [true]);
      await act(async () => button("cancel").click());
      assert.equal(Boolean(env.container.querySelector('input[name="name"]')), false);
      assert.deepEqual(state.updating, [true, false]);
      await edit();
      const input = env.container.querySelector('input[name="name"]');
      await changeFormControl(input, "Renamed");
      await submitForm(env.container);
      assert.equal(input.disabled, true);
      assert.equal(button("cancel").disabled, true);
      assert.deepEqual(state.updates, [{ id: "alpha", data: { name: "Renamed", color: "#123456" } }]);
      await act(async () => state.request.reject(Error("offline")));
      assert.equal(input.disabled, false);
      assert.equal(input.value, "Renamed");
      assert.equal(globalThis.labelRowToasts.length, 1);
      assert.deepEqual(state.updating, [true, false, true]);
      state.request = Promise.withResolvers();
      await submitForm(env.container);
      await act(async () => state.request.resolve({ ...label, name: "Renamed" }));
      assert.equal(Boolean(env.container.querySelector('input[name="name"]')), false);
      assert.equal(state.updates.length, 2);
      assert.deepEqual(state.updating, [true, false, true, false]);
    }
  );
}

await test("label row editing uses native menus and forms in one browser document", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await verifyEditing(t, env, "group");
  await verifyEditing(t, env, "item");
  await t.test("read-only label groups expose no mutation menu", async () => {
    await setup(env, "group", false);
    assert.equal(Boolean(env.container.querySelector('[aria-haspopup="menu"]')), false);
    assert.equal(Boolean(env.container.querySelector('input[name="name"]')), false);
  });
});
