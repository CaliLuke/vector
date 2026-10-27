import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";

const modalCoreUrl = new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href;
const modalConstantsUrl = new URL("../../packages/ui/src/modals/constants.ts", import.meta.url).href;
const stubs = {
  "@plane/propel/button": `const React=globalThis.workItemTypeDraftReact;export const Button=({children,prependIcon,variant,size,...props})=>React.createElement("button",{type:"button",...props},children);`,
  "@plane/ui": `const React=globalThis.workItemTypeDraftReact;
    export const Form=({onSubmit,children,...props})=>React.createElement("form",{...props,onSubmit:async event=>{event.preventDefault();event.stopPropagation();await onSubmit();}},children);
    export { ModalCore } from ${JSON.stringify(modalCoreUrl)};
    export { EModalWidth } from ${JSON.stringify(modalConstantsUrl)};
    export const ToggleSwitch=({label,value,onChange,disabled})=>React.createElement("input",{type:"checkbox","aria-label":label,checked:value,disabled,onChange:event=>onChange(event.target.checked)});
    function Menu({children}){return React.createElement("div",null,children)}
    Menu.MenuItem=({children,onClick})=>React.createElement("button",{type:"button",onClick},children);
    export const CustomMenu=Menu;
    export const CustomSearchSelect=({value,onChange,options})=>React.createElement("select",{"aria-label":"Work item type",value,onChange:event=>onChange(event.target.value)},options.map(option=>React.createElement("option",{key:option.value,value:option.value},option.query)));`,
  "@/services/work-item-types.service": `export const typeErrorMessage=error=>error instanceof Error?error.message:"Something failed";`,
  "./type-icons": `const Icon=()=>null;export const typeIconComponents={task:Icon,bug:Icon,incident:Icon,story:Icon,epic:Icon,Database:Icon};`,
  "./type-editor": `export const TypeEditor=()=>null;`,
  "./property-editor": `export const PropertyEditor=()=>null;`,
  "./context-editor": `export const ContextEditor=()=>null;`,
  "./cascading-editor": `export const CascadingEditor=()=>null;`,
  "./property-input": `const React=globalThis.workItemTypeDraftReact;export const PropertyInput=({property,value,onChange})=>React.createElement("input",{type:"text","aria-label":property.name,value:Array.isArray(value)?value.join("/"):String(value??""),onChange:event=>onChange(event.target.value)});`,
};
const nativeForms = registerWorkItemTypesFormHarness({
  sourceFiles: [
    "apps/web/core/components/work-item-types/cascading-editor.tsx",
    "apps/web/core/components/work-item-types/library.tsx",
    "apps/web/core/components/work-item-types/delete-type-preview.tsx",
    "apps/web/core/components/work-item-types/attach-properties.tsx",
    "apps/web/core/components/work-item-types/work-item-preview.tsx",
    "apps/web/core/components/work-item-types/controls.tsx",
  ],
  stubs,
  reactGlobal: "workItemTypeDraftReact",
  react: React,
});

const environment = await createReactDomTestEnvironment();
const { WorkItemTypesPreviewProvider, useTypePreview } =
  await import("../../apps/web/core/components/work-item-types/preview-provider.tsx");
const { CascadingEditor } = await import("../../apps/web/core/components/work-item-types/cascading-editor.tsx");
const { TypeLibrary } = await import("../../apps/web/core/components/work-item-types/library.tsx");
const { AttachProperties } = await import("../../apps/web/core/components/work-item-types/attach-properties.tsx");
const { createPreviewContent } = await import("../../apps/web/core/components/work-item-types/preview-content.ts");
const { reorderAttachedProperties } =
  await import("../../apps/web/core/components/work-item-types/property-attachment-row.tsx");
const { WorkItemsPreview } = await import("../../apps/web/core/components/work-item-types/work-item-preview.tsx");

function CascadeScreen() {
  const { content } = useTypePreview();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(CascadingEditor, {
      property: content.properties.find((property) => property.id === "area"),
      onClose: () => {},
    }),
    React.createElement("output", null, JSON.stringify(content))
  );
}

function LibraryScreen() {
  const { content } = useTypePreview();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(TypeLibrary),
    React.createElement("output", null, JSON.stringify(content))
  );
}

function WorkItemsScreen() {
  const { content } = useTypePreview();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(WorkItemsPreview),
    React.createElement("output", null, JSON.stringify(content))
  );
}

const click = (node) => act(async () => node.click());
const button = (name, scope = document.body) =>
  [...scope.querySelectorAll("button")].find((node) => node.textContent.trim() === name);
const addButton = () =>
  [...document.body.querySelectorAll("button")].find((node) => /^Add(?: \(|$)/.test(node.textContent.trim()));
const saveWorkItem = () => document.body.querySelector('button[form="preview-work-item"]');
const readContent = () => JSON.parse(document.body.querySelector("output").textContent);

await test("cascading option removals stay in Form through cancel and preserve affected values after confirmation", async (t) => {
  t.after(async () => environment.render(null));
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(CascadeScreen)));

  await click(document.body.querySelector('button[aria-label="Delete Web"]'));
  await click(button("Save changes"));
  assert.ok(document.body.querySelector('[role="dialog"]'));
  const cascadeForm = nativeForms.getForms("apps/web/core/components/work-item-types/cascading-editor.tsx").at(-1);
  assert.equal(
    cascadeForm.state.values.options.some((option) => option.id === "web"),
    false
  );
  assert.equal(
    readContent()
      .properties.find((property) => property.id === "area")
      .options.some((option) => option.id === "web"),
    true
  );
  await click(button("Cancel"));
  assert.match(document.body.querySelector("h2").textContent, /Manage levels/);

  await click(button("Save changes"));
  assert.match(document.body.querySelector("h2").textContent, /Remove cascading values/);
  const confirmationLabel = [...document.body.querySelectorAll("label")].find((node) =>
    node.textContent.includes("Type Confirm to continue")
  );
  await changeFormControl(document.getElementById(confirmationLabel.htmlFor), "Confirm");
  await click(button("Remove and preserve values"));
  const saved = readContent();
  const affected = saved.items.find((item) => item.values.area?.includes("web"));
  assert.equal(affected, undefined);
  const preservedItem = saved.items.find((item) => item.id === "WEB-142");
  assert.match(preservedItem.description, /Product area/);
  assert.equal(cascadeForm.state.isSubmitSuccessful, true);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/work-item-types/controls.tsx").at(-1).state.isSubmitSuccessful,
    true
  );
});

await test("cascading impact confirmation only opens after native validation", async (t) => {
  t.after(async () => environment.render(null));
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(CascadeScreen)));
  await click(document.body.querySelector('button[aria-label="Delete Web"]'));
  await changeFormControl(document.body.querySelector('input[aria-label="Level 1 name"]'), "");
  await click([...document.body.querySelectorAll("button")].find((node) => node.textContent.trim() === "Save changes"));
  assert.equal(document.body.querySelector("h2").textContent.startsWith("Manage levels"), true);
  assert.match(document.body.querySelector('[role="alert"]').textContent, /Give every level a name/);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/work-item-types/cascading-editor.tsx").at(-1).state
      .isSubmitSuccessful,
    false
  );
});

await test("property attachment requires a selection and updates only after native validation", async (t) => {
  t.after(async () => environment.render(null));
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(LibraryScreen)));
  await click(document.body.querySelector('button[aria-label="Expand Task"]'));
  await click([...document.body.querySelectorAll("button")].find((node) => node.textContent === "Add properties"));
  assert.equal(addButton().disabled, true);
  const attachForm = nativeForms.getForms("apps/web/core/components/work-item-types/attach-properties.tsx").at(-1);
  assert.equal(attachForm.state.isSubmitSuccessful, false);
  const severity = [...document.body.querySelectorAll("label")].find((node) => node.textContent.includes("Severity"));
  await click(severity.querySelector("input"));
  await click(addButton());
  const content = readContent();
  assert.ok(content.types.find((type) => type.id === "task").properties.includes("severity"));
  assert.equal(attachForm.state.isSubmitSuccessful, true);
});

await test("property removal submits the permanent-removal choice through Confirmation Form", async (t) => {
  t.after(async () => environment.render(null));
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(LibraryScreen)));
  await click(
    [...document.body.querySelectorAll('[role="tab"]')].find((node) => node.textContent.includes("Properties"))
  );
  await click([...document.body.querySelectorAll("button")].find((node) => node.textContent === "Delete property"));
  const permanent = document.body.querySelector('input[aria-label="Remove values permanently"]');
  assert.ok(permanent);
  await click(permanent);
  const confirmationLabel = [...document.body.querySelectorAll("label")].find((node) =>
    node.textContent.includes("Type Confirm to continue")
  );
  await changeFormControl(document.getElementById(confirmationLabel.htmlFor), "Confirm");
  await click(document.body.querySelector('button[type="submit"]'));
  const saved = JSON.parse(document.body.querySelector("output").textContent);
  assert.equal(
    saved.properties.some((property) => property.id === "severity"),
    false
  );
  const affected = saved.items.find((item) => item.id === "WEB-142");
  assert.equal(affected.values.severity, undefined);
  assert.doesNotMatch(affected.description, /Severity:/);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/work-item-types/controls.tsx").at(-1).state.values.permanent,
    true
  );
  assert.equal(
    nativeForms.getForms("apps/web/core/components/work-item-types/controls.tsx").at(-1).state.isSubmitSuccessful,
    true
  );
});

await test("type replacement is a native required selection and preserves the affected item", async (t) => {
  t.after(async () => environment.render(null));
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(LibraryScreen)));
  await click([...document.body.querySelectorAll("button")].find((node) => node.textContent === "Delete type"));
  const form = nativeForms.getForms("apps/web/core/components/work-item-types/delete-type-preview.tsx").at(-1);
  await click(document.body.querySelector('button[type="submit"]'));
  assert.equal(form.state.isSubmitSuccessful, false);
  assert.match(document.body.querySelector('[role="alert"]').textContent, /replacement type/i);
  const replacementSelect = [...document.body.querySelectorAll("label")]
    .find((node) => node.textContent.includes("Replacement type"))
    .querySelector("select");
  await changeFormControl(replacementSelect, "story");
  assert.equal(form.state.values.replacement, "story");
  await click(document.body.querySelector('button[type="submit"]'));
  const saved = JSON.parse(document.body.querySelector("output").textContent);
  assert.equal(
    saved.types.some((type) => type.id === "bug"),
    false
  );
  assert.equal(saved.items.find((item) => item.id === "WEB-142").type, "story");
  assert.equal(form.state.values.replacement, "story");
  assert.equal(form.state.isSubmitSuccessful, true);
});

await test("work-item draft mode is a native value that skips publication-only required fields", async (t) => {
  t.after(async () => environment.render(null));
  await environment.render(
    React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(WorkItemsScreen))
  );
  await click([...document.body.querySelectorAll("button")].find((node) => node.textContent === "Add work item"));
  await changeFormControl(document.body.querySelector('select[aria-label="Work item type"]'), "bug");
  await changeFormControl(
    document.body.querySelector('input[placeholder="What needs to be done?"]'),
    "Investigate login issue"
  );
  await click(saveWorkItem());
  const workItemForm = nativeForms.getForms("apps/web/core/components/work-item-types/work-item-preview.tsx").at(-1);
  assert.equal(workItemForm.state.isSubmitSuccessful, false);
  assert.match(document.body.querySelector('[role="alert"]').textContent, /Severity is required/);
  const draft = document.body.querySelector('input[aria-label="Save as draft"]');
  await click(draft);
  await click(saveWorkItem());
  const saved = readContent();
  assert.equal(saved.items.at(-1).draft, true);
  assert.equal(workItemForm.state.values.draft, true);
  assert.equal(workItemForm.state.isSubmitSuccessful, true);
});

await test("attachment reordering preserves identities and no-op drops", () => {
  const ids = ["a", "b", "c"];
  assert.deepEqual(reorderAttachedProperties(ids, "a", "c", true), ["b", "c", "a"]);
  assert.deepEqual(reorderAttachedProperties(ids, "c", "a", false), ["c", "a", "b"]);
  assert.equal(reorderAttachedProperties(ids, "a", "b", false), ids);
  assert.equal(reorderAttachedProperties(ids, "foreign", "b", false), ids);
  assert.deepEqual(ids, ["a", "b", "c"]);
});

await test("attachment controls reorder only their own type and expose drag handles", async (t) => {
  t.after(async () => environment.render(null));
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(LibraryScreen)));
  await click(document.body.querySelector('button[aria-label="Expand Task"]'));
  const before = readContent();
  const task = before.types.find((type) => type.id === "task");
  const property = before.properties.find((item) => item.id === task.properties[0]);
  const handle = document.body.querySelector(`button[aria-label="Drag ${property.name} to reorder"]`);
  assert.ok(handle);
  assert.equal(handle.parentElement.draggable, true);
  await click(document.body.querySelector(`button[aria-label="Move ${property.name} down"]`));
  const after = readContent();
  assert.equal(after.types.find((type) => type.id === "task").properties[1], property.id);
  assert.deepEqual(
    after.types.filter((type) => type.id !== "task"),
    before.types.filter((type) => type.id !== "task")
  );
  assert.match(handle.parentElement.textContent, /Active/);
});

await test("attachment empty states distinguish a new library from an exhausted library and link to management", async (t) => {
  t.after(async () => environment.render(null));
  const content = createPreviewContent();
  let navigated = false;
  const props = {
    type: content.types[0],
    properties: [],
    onClose() {},
    onManageProperties() {
      navigated = true;
    },
  };
  await environment.render(
    React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(AttachProperties, props))
  );
  assert.match(document.body.textContent, /No properties yet/);
  assert.equal(addButton().disabled, true);
  await click(button("Manage properties"));
  assert.equal(navigated, true);
  await environment.render(
    React.createElement(
      WorkItemTypesPreviewProvider,
      null,
      React.createElement(AttachProperties, {
        ...props,
        properties: content.properties,
        type: { ...props.type, properties: content.properties.map((property) => property.id) },
      })
    )
  );
  assert.match(document.body.textContent, /All properties are attached/);
  assert.equal(addButton().disabled, true);
});

nativeForms.restore();
await environment.unmount();
