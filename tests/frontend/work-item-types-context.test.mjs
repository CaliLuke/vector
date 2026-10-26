import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const nativeForms = observeNativeForms([
  "apps/web/core/components/work-item-types/context-editor.tsx",
  "apps/web/core/components/work-item-types/type-editor.tsx",
]);
globalThis.typeContextReact = React;
const stubs = {
  "@plane/propel/button": `const React=globalThis.typeContextReact; export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
  "@plane/ui": `const React=globalThis.typeContextReact;export const Form=({onSubmit,children,...props})=>React.createElement("form",{...props,onSubmit:event=>{event.preventDefault();onSubmit();}},children);`,
  "@/services/work-item-types.service": `export const typeErrorMessage=error=>error.message;`,
  "./controls": `const React=globalThis.typeContextReact;export const inputStyle="";export const TypeIcon=()=>null;
    export const Field=({label,children})=>React.createElement("label",null,label,children(label));
    export const Choice=({label,value,options,onChange,disabled})=>React.createElement("select",{"aria-label":label,value,disabled,onChange:event=>onChange(event.target.value)},options.map(option=>React.createElement("option",{key:option},option)));
    export const PreviewDialog=({title,children,footer})=>React.createElement("section",{"aria-label":title},children,footer);`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { ContextEditor } = await import("../../apps/web/core/components/work-item-types/context-editor.tsx");
const { WorkItemTypesPreviewProvider, useTypePreview } =
  await import("../../apps/web/core/components/work-item-types/preview-provider.tsx");
function Screen() {
  const { content } = useTypePreview();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(ContextEditor, {
      property: content.properties.find((property) => property.id === "severity"),
      onClose: () => {},
    }),
    React.createElement("output", null, JSON.stringify(content.properties.find((p) => p.id === "severity").contexts))
  );
}

await test("selecting and applying a property context retains its scope and unapplied edits", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(Screen)));
  const button = (name) =>
    [...environment.container.querySelectorAll("button")].find((node) => node.textContent === name);
  await act(async () => button("Mobile bugs").click());
  const name = environment.container.querySelector('input[id="Context name"]');
  assert.equal(name.value, "Mobile bugs");
  const checkbox = (label) =>
    [...environment.container.querySelectorAll("label")]
      .find((node) => node.textContent === label)
      .querySelector("input");
  assert.equal(checkbox("Mobile App").checked, true);
  assert.equal(checkbox("Bug").checked, true);
  assert.equal(environment.container.querySelector('select[aria-label="required"]').value, "No");
  await changeFormControl(name, "Mobile defects");
  await act(async () => button("Default").click());
  assert.equal(name.value, "Mobile defects");
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Apply this context/);
  await submitForm(environment.container);
  const contextForm = nativeForms.getForms("apps/web/core/components/work-item-types/context-editor.tsx").at(-1);
  assert.equal(contextForm.state.isSubmitSuccessful, true, "Enter applies the active context through Form");
  assert.equal(contextForm.state.values.contexts[0].name, "Mobile defects");
  assert.deepEqual(contextForm.state.values.contexts[0].projects, ["mobile"]);
  assert.equal(contextForm.state.values.activeContext.name, "Mobile defects");
  await act(async () => button("Default").click());
  await act(async () => button("Mobile defects").click());
  assert.equal(environment.container.querySelector('input[id="Context name"]').value, "Mobile defects");
  assert.equal(checkbox("Mobile App").checked, true);
  assert.equal(checkbox("Bug").checked, true);
  const attempts = contextForm.state.submissionAttempts;
  await act(async () => button("Save changes").click());
  assert.equal(contextForm.state.submissionAttempts, attempts + 1);
  assert.equal(JSON.parse(environment.container.querySelector("output").textContent)[0].name, "Mobile defects");
  assert.equal(contextForm.state.isSubmitSuccessful, true);
});

await test("final context save rejects a bypassed submission while active edits remain unapplied", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(Screen)));
  const button = (name) =>
    [...environment.container.querySelectorAll("button")].find((node) => node.textContent === name);
  const form = nativeForms.getForms("apps/web/core/components/work-item-types/context-editor.tsx").at(-1);
  const original = JSON.parse(environment.container.querySelector("output").textContent);
  const contextsButton = [...environment.container.querySelectorAll("button")].find(
    (node) => node.textContent === "Mobile bugs"
  );
  await act(async () => contextsButton.click());
  await changeFormControl(environment.container.querySelector('input[id="Context name"]'), "Unapplied change");
  assert.equal(button("Save changes").disabled, true);
  await act(async () => {
    await assert.rejects(form.handleSubmit({ action: "save" }), /Apply this context before saving/);
  });
  assert.equal(form.state.isSubmitSuccessful, false);
  assert.deepEqual(JSON.parse(environment.container.querySelector("output").textContent), original);
  assert.equal(button("Apply context").disabled, false);
});

await test("new contexts validate and stage native values before the final save", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(Screen)));
  const button = (name) =>
    [...environment.container.querySelectorAll("button")].find((node) => node.textContent === name);
  await act(async () => button("Add context").click());
  const form = nativeForms.getForms("apps/web/core/components/work-item-types/context-editor.tsx").at(-1);
  await act(async () => button("Apply context").click());
  assert.equal(form.state.isSubmitSuccessful, false);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /name/);
  await changeFormControl(environment.container.querySelector('input[id="Context name"]'), "Mobile triage");
  const mobileProject = [...environment.container.querySelectorAll("label")]
    .find((node) => node.textContent === "Mobile App")
    .querySelector("input");
  await act(async () => mobileProject.click());
  await act(async () => button("Apply context").click());
  assert.equal(form.state.isSubmitSuccessful, true);
  assert.equal(form.state.values.contexts.at(-1).name, "Mobile triage");
  assert.deepEqual(form.state.values.contexts.at(-1).projects, ["mobile"]);
});

await test("type creation awaits persistence and retains a failed draft for retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { TypeEditor } = await import("../../apps/web/core/components/work-item-types/type-editor.tsx");
  let fail = true;
  let closed = false;
  const submissions = [];
  await environment.render(
    React.createElement(
      WorkItemTypesPreviewProvider,
      null,
      React.createElement(TypeEditor, {
        types: [],
        onSave: async (value, creating) => {
          submissions.push({ value, creating });
          if (fail) throw new Error("The server could not save this type.");
        },
        onClose: () => {
          closed = true;
        },
      })
    )
  );
  const name = environment.container.querySelector('input[aria-label="Name"]');
  await changeFormControl(name, "  Support request  ");
  await changeFormControl(
    environment.container.querySelector('textarea[aria-label="Description"]'),
    "Help from the team"
  );
  await act(async () => environment.container.querySelector('[aria-label="Choose icon and color"]').click());
  await changeFormControl(document.querySelector('[aria-label="Search icons"]'), "Database");
  await act(async () => document.querySelector('input[aria-label="Database"]').click());
  await act(async () => document.querySelector('input[aria-label="Color #02b55c"]').click());
  assert.equal(document.querySelector('input[aria-label="Database"]').checked, true);
  assert.equal(document.querySelector('input[aria-label="Color #02b55c"]').checked, true);
  await act(async () => document.querySelector('[aria-label="Close icon picker"]').click());
  await submitForm(environment.container);
  assert.equal(closed, false);
  assert.equal(name.value, "  Support request  ");
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /server could not save/);
  assert.equal(submissions[0].value.name, "Support request");
  assert.equal(submissions[0].value.icon, "Database");
  assert.equal(submissions[0].value.color, "#02b55c");
  assert.equal(submissions[0].value.active, false);
  assert.equal(submissions[0].value.level, 0);
  assert.equal(submissions[0].creating, true);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/work-item-types/type-editor.tsx").at(-1).state.isSubmitSuccessful,
    false
  );
  fail = false;
  await submitForm(environment.container);
  assert.equal(submissions.length, 2);
  assert.equal(submissions[1].value.icon, "Database");
  assert.equal(submissions[1].value.color, "#02b55c");
  assert.equal(closed, true);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/work-item-types/type-editor.tsx").at(-1).state.isSubmitSuccessful,
    true
  );
});

await test("type icon sheet preserves an existing color, toggles closed, and reports empty searches", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { TypeEditor } = await import("../../apps/web/core/components/work-item-types/type-editor.tsx");
  await environment.render(
    React.createElement(
      WorkItemTypesPreviewProvider,
      null,
      React.createElement(TypeEditor, {
        type: {
          id: "existing",
          name: "Custom",
          description: "",
          icon: "Database",
          color: "#123456",
          active: true,
          mandatory: false,
          properties: [],
          level: 0,
        },
        onClose: () => {},
      })
    )
  );
  const trigger = environment.container.querySelector('[aria-label="Choose icon and color"]');
  await act(async () => trigger.click());
  assert.equal(document.querySelector('input[aria-label="Color #123456"]').checked, true);
  assert.equal(document.querySelector('input[aria-label="Database"]').checked, true);
  await act(async () => document.querySelector('input[aria-label="Color #02b55c"]').click());
  assert.ok(document.querySelector('input[aria-label="Color #123456"]'), "the original custom color stays available");
  await changeFormControl(document.querySelector('[aria-label="Search icons"]'), "no icon matches this");
  assert.match(document.querySelector("output").textContent, /No icons found/);
  await act(async () => trigger.click());
  assert.equal(document.querySelector('[aria-label="Search icons"]'), null);
  await act(async () => trigger.click());
  assert.equal(document.querySelector('[aria-label="Search icons"]').value, "");
  assert.equal(document.querySelector('input[aria-label="Database"]').checked, true);
});

nativeForms.restore();
