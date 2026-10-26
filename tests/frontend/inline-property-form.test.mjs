import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
globalThis.inlineReact = React;
const stubs = {
  "@plane/propel/button": `export const Button=({loading,variant,...props})=>globalThis.inlineReact.createElement("button",props);`,
  "@/hooks/use-workspaces": `export const useWorkspaceBySlug=()=>({data:{id:"workspace"}});`,
  "./use-inline-property-preview": `export const useInlinePropertyPreview=()=>globalThis.inlineFixture.preview;`,
  "./item-property-input": `export function ItemPropertyInput(props){const R=globalThis.inlineReact;globalThis.inlineFixture.inputs[props.property.id]=props;if(props.property.kind==="Rich text")return props.renderRichText(props.property,{});return R.createElement("label",null,props.property.title,R.createElement("input",{value:props.value??"",ref:props.onControlMount,"data-field":props.property.id,onChange:e=>props.onChange(e.target.value)}),props.error&&R.createElement("span",{role:"alert"},props.error));}`,
  "./item-property-value": `export const ItemPropertyValue=props=>globalThis.inlineReact.createElement("span",null,String(props.schema.stored_values[props.property.id]??""));`,
  "@/components/editor/rich-text": `export const RichTextEditor=()=>null;`,
  "@/components/issues/issue-modal/components/item-rich-text-field": `export function WorkItemRichTextField(props){const R=globalThis.inlineReact,f=globalThis.inlineFixture;f.notifyEditor=props.onEditorStateChange;R.useLayoutEffect(()=>{f.editorMounts++;props.editors.set(props.propertyId,{isEditorReadyToDiscard:()=>f.editorReady});props.onEditorStateChange();return()=>props.editors.delete(props.propertyId);},[props.editors,props.propertyId]);return R.createElement("article",null,"Rich text editor");}`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { InlinePropertyForm } = await import("../../apps/web/core/components/work-item-types/inline-property-form.tsx");
const field = (id) => ({ id, title: id, kind: "Text", configuration: { format: "Single line" } });
async function fixture(t, properties = [field("one"), field("two")]) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const f = {
    env,
    inputs: {},
    preview: {},
    commands: [],
    closed: 0,
    closeReady: undefined,
    denied: 0,
    editorReady: true,
    editorMounts: 0,
    snapshot: {
      source: {
        source_version: "v1",
        item: { id: "item", project_id: "project", custom_properties: { one: "saved", two: "saved two" } },
      },
      schema: {
        enabled: true,
        revision: 7,
        properties,
        stored_values: { one: "different read" },
      },
    },
  };
  f.edit = {
    source: { isFetching: false },
    busy: false,
    conflict: false,
    save: async (command) => {
      f.commands.push(command);
    },
    reload: async () => f.fresh,
  };
  globalThis.inlineFixture = f;
  f.render = () =>
    env.render(
      React.createElement(InlinePropertyForm, {
        scope: { workspaceSlug: "acme", projectId: "project", issueId: "item" },
        edit: f.edit,
        snapshot: f.snapshot,
        onClose: () => f.closed++,
        onCloseReadyChange: (ready) => {
          f.closeReady = ready;
        },
        onAccessLost: () => f.denied++,
      })
    );
  f.change = (id, value) => act(async () => f.inputs[id].onChange(value));
  f.button = (name) => [...env.container.querySelectorAll("button")].find((b) => b.textContent === name);
  f.click = (name) => act(async () => f.button(name).click());
  f.submit = () =>
    act(async () =>
      env.container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
  await f.render();
  return f;
}

await test("inline edits use canonical source and submit only changed properties", async (t) => {
  const f = await fixture(t);
  assert.equal(f.inputs.one.value, "saved");
  await f.change("one", "local");
  await f.submit();
  assert.deepEqual(f.commands, [
    {
      mode: "edit",
      item_id: "item",
      project_id: "project",
      revision: 7,
      expected_source_version: "v1",
      changes: { custom_properties: { one: "local" } },
    },
  ]);
});

await test("validation retains edits and focuses the invalid required sibling", async (t) => {
  const f = await fixture(t);
  await f.change("one", "local");
  f.edit.save = async () => {
    f.edit.cause = { response: { data: { fields: { custom_properties: { two: "Required" } } } } };
    throw f.edit.cause;
  };
  f.edit.error = "Correct the invalid fields.";
  await f.submit();
  await act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  assert.equal(f.inputs.one.value, "local");
  assert.equal(document.activeElement.dataset.field, "two");
  assert.match(f.env.container.textContent, /Required/);
  assert.equal(f.button("Save properties").disabled, false);
});

await test("slow reload locks controls and keeps explicit changes over fresh untouched values", async (t) => {
  const f = await fixture(t);
  await f.change("one", "local");
  f.edit.conflict = true;
  let resolve;
  f.edit.reload = () =>
    new Promise((done) => {
      resolve = done;
    });
  await f.render();
  await f.click("Reload current values");
  assert.equal(f.env.container.querySelector("fieldset").disabled, true);
  assert.equal(f.button("Cancel").disabled, true);
  assert.equal(f.closeReady, false);
  const fresh = structuredClone(f.snapshot);
  fresh.source.source_version = "v2";
  fresh.source.item.custom_properties = { one: "changed remotely", two: "fresh second" };
  f.edit.conflict = false;
  await act(async () => resolve(fresh));
  assert.equal(f.inputs.one.value, "local");
  assert.equal(f.inputs.two.value, "fresh second");
  assert.match(f.env.container.textContent, /changed remotely/);
  await f.submit();
  assert.equal(f.commands[0].expected_source_version, "v2");
  assert.deepEqual(f.commands[0].changes.custom_properties, { one: "local" });
});

await test("disabled rollout after reload preserves the visible draft and prevents saving", async (t) => {
  const f = await fixture(t);
  await f.change("one", "local");
  f.edit.conflict = true;
  f.fresh = structuredClone(f.snapshot);
  f.fresh.schema.enabled = false;
  await f.render();
  await f.click("Reload current values");
  assert.equal(f.inputs.one.value, "local");
  assert.match(f.env.container.textContent, /Edited fields are no longer available/);
  assert.equal(f.button("Save properties"), undefined);
  await f.submit();
  assert.deepEqual(f.commands, []);
});

await test("definite preview denial hides private fields while network failure retains them", async (t) => {
  const f = await fixture(t);
  await f.change("one", "local");
  f.preview = { error: true, cause: new Error("Offline") };
  await f.render();
  assert.equal(f.inputs.one.value, "local");
  assert.equal(f.denied, 0);
  f.preview = { error: true, cause: { response: { status: 403 } } };
  await f.render();
  assert.equal(f.env.container.querySelector("input"), null);
  assert.equal(f.denied, 1);
});

await test("rich-text processing blocks save, cancel and reload without remounting the editor", async (t) => {
  const f = await fixture(t, [field("one"), { id: "rich", title: "Details", kind: "Rich text", configuration: {} }]);
  assert.equal(f.editorMounts, 1);
  f.editorReady = false;
  await act(async () => f.notifyEditor());
  assert.equal(f.button("Save properties").disabled, true);
  assert.equal(f.button("Cancel").disabled, true);
  assert.equal(f.closeReady, false);
  await f.submit();
  assert.deepEqual(f.commands, []);
  assert.match(f.env.container.textContent, /An editor is still processing changes/);
  f.edit.conflict = true;
  await f.render();
  assert.equal(f.button("Reload current values").disabled, true);
  f.editorReady = true;
  await act(async () => f.notifyEditor());
  assert.doesNotMatch(f.env.container.textContent, /An editor is still processing changes/);
  assert.equal(f.button("Reload current values").disabled, false);
  assert.equal(f.button("Cancel").disabled, false);
  assert.equal(f.closeReady, true);
  f.edit.conflict = false;
  await f.render();
  assert.equal(f.button("Save properties").disabled, false);
  assert.equal(f.editorMounts, 1);
});
