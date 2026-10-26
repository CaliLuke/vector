import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";
const source = "apps/web/core/components/work-item-types/property-editor.tsx";
const contextSource = "apps/web/core/components/work-item-types/context-editor.tsx";
const stubs = {
  "@plane/propel/button": `const React=globalThis.propertyEditorReact;export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
  "@plane/ui": `const React=globalThis.propertyEditorReact;export const Form=({onSubmit,children,...props})=>React.createElement("form",{...props,onSubmit:event=>{event.preventDefault();void onSubmit();}},children);export const ToggleSwitch=({label,value,onChange,disabled})=>React.createElement("input",{type:"checkbox","aria-label":label,checked:value,disabled,onChange:event=>onChange(event.target.checked)});`,
  "@/services/work-item-types.service": `export const typeErrorMessage=error=>error.message;`,
  "./controls": `const React=globalThis.propertyEditorReact;export const inputStyle="";export const PropertyIcon=()=>null;
 export const Field=({label,children})=>React.createElement("label",null,label,children(label));
 export const Choice=({label,value,options,onChange,disabled})=>React.createElement("select",{"aria-label":label,value,disabled,onChange:event=>onChange(event.target.value)},options.map(option=>React.createElement("option",{key:option},option)));
 export const PreviewDialog=({title,children,footer,onClose})=>React.createElement("section",{"aria-label":title},children,footer,React.createElement("button",{onClick:onClose},"Close"));`,
};
const forms = registerWorkItemTypesFormHarness({
  sourceFiles: [source, contextSource],
  stubs,
  reactGlobal: "propertyEditorReact",
  react: React,
});
const { PropertyEditor } = await import("../../apps/web/core/components/work-item-types/property-editor.tsx");
const { ContextEditor } = await import("../../apps/web/core/components/work-item-types/context-editor.tsx");
const { WorkItemTypesPreviewProvider } =
  await import("../../apps/web/core/components/work-item-types/preview-provider.tsx");
const { propertyForEditor, propertyToSave } =
  await import("../../apps/web/core/components/work-item-types/property-adapter.ts");
const base = {
  id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  name: "score",
  title: "Score",
  description: "",
  icon: "",
  kind: "Number",
  configuration: {
    active: true,
    required: false,
    multiple: false,
    format: "",
    content: "",
    expression: "",
    result: "",
    default_value: 0,
    options: [],
    levels: [],
  },
  contexts: [],
};
const button = (env, name) => [...env.container.querySelectorAll("button")].find((node) => node.textContent === name);
const wrap = (node) => React.createElement(WorkItemTypesPreviewProvider, null, node);
await test("canonical property round trip retains zero and explicit context inheritance", () => {
  const def = {
    ...base,
    contexts: [
      {
        id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
        name: "Optional",
        project_ids: [],
        type_ids: ["type"],
        override_keys: ["required", "default_value"],
        configuration: { ...base.configuration, required: false, default_value: null },
      },
    ],
  };
  assert.deepEqual(propertyToSave(propertyForEditor(def, true)), def);
  const renamed = propertyForEditor(def, true);
  renamed.name = "New label";
  assert.equal(propertyToSave(renamed).name, "score");
});
await test("property save waits, locks dismissal, retains failed draft and identity, then retries", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let rejectSave,
    closed = 0,
    fail = true;
  const submissions = [];
  const onSave = (value) => {
    submissions.push(value);
    return fail
      ? new Promise((_, reject) => {
          rejectSave = reject;
        })
      : Promise.resolve();
  };
  await env.render(
    wrap(
      React.createElement(PropertyEditor, {
        library: { types: [], properties: [] },
        members: [],
        onSave,
        onClose: () => closed++,
      })
    )
  );
  await changeFormControl(env.container.querySelector('input[id="Title"]'), "Risk");
  await submitForm(env.container);
  assert.equal(env.container.querySelector("fieldset").disabled, true);
  await act(async () => button(env, "Close").click());
  assert.equal(closed, 0);
  await act(async () => rejectSave(new Error("Revision changed")));
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Revision changed/);
  assert.equal(env.container.querySelector('input[id="Title"]').value, "Risk");
  assert.equal(forms.getForms(source).at(-1).state.isSubmitSuccessful, false);
  fail = false;
  await submitForm(env.container);
  assert.equal(closed, 1);
  assert.equal(submissions.length, 2);
  assert.equal(submissions[0].id, submissions[1].id);
  assert.equal(forms.getForms(source).at(-1).state.isSubmitSuccessful, true);
});
await test("changing an unused persisted number to cascading starts with two valid levels", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const property = propertyForEditor(base, false);
  let saved;
  await env.render(
    wrap(
      React.createElement(PropertyEditor, {
        property,
        library: { types: [], properties: [property] },
        onSave: async (value) => {
          saved = value;
        },
        onClose: () => {},
      })
    )
  );
  await changeFormControl(env.container.querySelector('select[aria-label="Property type"]'), "Cascading");
  assert.equal(env.container.querySelectorAll('input[aria-label$=" name"]').length, 2);
  await submitForm(env.container);
  assert.equal(saved.levels.length, 2);
  assert.notEqual(saved.levels[0].id, saved.levels[1].id);
  assert.match(saved.levels[0].id, /^[0-9a-f-]{36}$/);
});
await test("context persistence failure is visible from Default and leaves staged changes for retry", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let fail = true,
    closed = 0;
  const property = propertyForEditor(base, false);
  await env.render(
    wrap(
      React.createElement(ContextEditor, {
        property,
        library: { types: [], projects: [] },
        onSave: async () => {
          if (fail) throw new Error("Context revision changed");
        },
        onClose: () => closed++,
      })
    )
  );
  await act(async () => button(env, "Save changes").click());
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Context revision changed/);
  assert.equal(closed, 0);
  assert.equal(forms.getForms(contextSource).at(-1).state.isSubmitSuccessful, false);
  fail = false;
  await act(async () => button(env, "Save changes").click());
  assert.equal(closed, 1);
});
forms.restore();

await test("invalid numeric defaults are rejected before JSON can turn them into null", () => {
  for (const text of ["not a number", "1e999"]) {
    const editor = propertyForEditor(base, false);
    editor.contexts = [
      {
        id: "scope",
        name: "Scope",
        projects: ["project"],
        types: [],
        required: null,
        active: null,
        defaultValue: text,
      },
    ];
    assert.throws(() => JSON.stringify(propertyToSave(editor)), /finite numeric default/);
  }
});
