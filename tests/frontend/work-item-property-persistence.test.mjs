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
  "@plane/ui": `const React=globalThis.propertyEditorReact;export const Form=({onSubmit,children,...props})=>React.createElement("form",{...props,onSubmit:event=>{event.preventDefault();void onSubmit();}},children);function Menu({children}){return React.createElement("div",null,children)}Menu.MenuItem=({children,onClick})=>React.createElement("button",{onClick},children);export const CustomMenu=Menu;export const ToggleSwitch=({label,value,onChange,disabled})=>React.createElement("input",{type:"checkbox","aria-label":label,checked:value,disabled,onChange:event=>onChange(event.target.checked)});`,
  "@/hooks/use-work-item-table-catalog": `export const useWorkItemTableCatalog=()=>({supported:true,identities:{},fields:globalThis.propertyIconColumns});`,
  "./cascading-editor": `const React=globalThis.propertyEditorReact;export const CascadingEditor=({property,review,onClose})=>React.createElement("section",{"aria-label":"Cascade follow-up","data-property":property.id,"data-revision":review?.revision},property.name,React.createElement("button",{onClick:onClose},"Skip"));`,
  "@/services/work-item-types.service": `export const typeErrorMessage=error=>error.message;`,
  "./controls": `const React=globalThis.propertyEditorReact;export const inputStyle="";export const PropertyIcon=()=>null;export const TypeIcon=()=>null;export const Confirmation=()=>null;export const EmptyState=({children})=>React.createElement("div",null,children);
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
const bootstrap = await createReactDomTestEnvironment();
const { PropertyEditor } = await import("../../apps/web/core/components/work-item-types/property-editor.tsx");
const { ContextEditor } = await import("../../apps/web/core/components/work-item-types/context-editor.tsx");
const { WorkItemTypesPreviewProvider } =
  await import("../../apps/web/core/components/work-item-types/preview-provider.tsx");
const { propertyForEditor, propertyToSave } =
  await import("../../apps/web/core/components/work-item-types/property-adapter.ts");
await bootstrap.unmount();
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
await test("property and option icon selections survive failed save and serialize without adding colors", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const property = propertyForEditor(
    {
      ...base,
      kind: "Dropdown",
      configuration: {
        ...base.configuration,
        default_value: null,
        options: [{ id: "choice", name: "First", icon: "", parent: "", level: 0, next_label: "" }],
      },
    },
    false
  );
  let fail = true;
  const submissions = [];
  await env.render(
    wrap(
      React.createElement(PropertyEditor, {
        property,
        onClose: () => {},
        onSave: async (value) => {
          submissions.push(propertyToSave(value));
          if (fail) throw new Error("Save rejected");
        },
      })
    )
  );
  await act(async () => env.container.querySelector('[aria-label="Choose property icon"]').click());
  await changeFormControl(document.querySelector('[aria-label="Search icons"]'), "Database");
  await act(async () => document.querySelector('input[aria-label="Database"]').click());
  assert.equal(document.querySelector('[aria-label^="Color #"]'), null);
  await act(async () => document.querySelector('[aria-label="Close icon picker"]').click());
  await act(async () => env.container.querySelector('[aria-label="Choose icon for option 1"]').click());
  await changeFormControl(document.querySelector('[aria-label="Search icons"]'), "Bug");
  await act(async () => document.querySelector('input[value="bug"]').click());
  await act(async () => document.querySelector('[aria-label="Close icon picker"]').click());
  await submitForm(env.container);
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Save rejected/);
  assert.equal(submissions[0].icon, "Database");
  assert.equal(submissions[0].configuration.options[0].icon, "bug");
  await act(async () => env.container.querySelector('[aria-label="Choose property icon"]').click());
  assert.equal(document.querySelector('input[value="Database"]').checked, true);
  await act(async () => document.querySelector('[aria-label="Close icon picker"]').click());
  fail = false;
  await submitForm(env.container);
  assert.deepEqual(submissions[1], submissions[0]);
});

await test("used option appearance stays locked while unrelated saves retain unknown icon identifiers", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const property = propertyForEditor(
    {
      ...base,
      icon: "future-property",
      kind: "Dropdown",
      configuration: {
        ...base.configuration,
        default_value: null,
        options: [{ id: "choice", name: "First", icon: "future-option", parent: "", level: 0, next_label: "" }],
      },
    },
    true
  );
  let saved;
  await env.render(
    wrap(
      React.createElement(PropertyEditor, {
        property,
        onClose: () => {},
        onSave: async (value) => {
          saved = propertyToSave(value);
        },
      })
    )
  );
  assert.equal(env.container.querySelector('[aria-label="Choose icon for option 1"]').disabled, true);
  assert.equal(env.container.querySelector('[aria-label="Choose property icon"]').disabled, false);
  await changeFormControl(env.container.querySelector('textarea[id="Description"]'), "New description");
  await submitForm(env.container);
  assert.equal(saved.icon, "future-property");
  assert.equal(saved.configuration.options[0].icon, "future-option");
});

await test("saved property icons render in field labels and unknown identifiers use the kind fallback", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { PropertyIcon } = await import("../../apps/web/core/components/work-item-types/property-icon.tsx");
  await env.render(React.createElement(PropertyIcon, { kind: "Number", icon: "Database" }));
  assert.ok(env.container.querySelector("svg.lucide-database"));
  await env.render(React.createElement(PropertyIcon, { kind: "Number", icon: "future-icon" }));
  assert.ok(env.container.querySelector("svg.lucide-hash"));
  const kinds = [
    ["Email", "mail"],
    ["File", "paperclip"],
    ["Work item picker", "square-check"],
  ];
  await env.render(
    React.createElement(
      React.Fragment,
      null,
      ...kinds.map(([kind]) => React.createElement(PropertyIcon, { key: kind, kind }))
    )
  );
  for (const [kind, icon] of kinds) {
    assert.ok(env.container.querySelector(`svg.lucide-${icon}`), `${kind} has a visible field icon`);
    const definition = { ...base, kind, configuration: { ...base.configuration, default_value: null } };
    assert.deepEqual(propertyToSave(propertyForEditor(definition, true)), definition);
  }
});

await test("custom column selector shows saved property icons and kind fallbacks", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  globalThis.propertyIconColumns = [
    { key: "saved", name: "Saved", kind: "Number", icon: "Database" },
    { key: "empty", name: "Empty", kind: "Number", icon: "" },
    { key: "unknown", name: "Unknown", kind: "Date", icon: "future-icon" },
  ];
  t.after(() => delete globalThis.propertyIconColumns);
  const { CustomPropertyColumns } =
    await import("../../apps/web/core/components/issues/issue-layouts/filters/header/display-filters/custom-property-columns.tsx");
  await env.render(
    React.createElement(CustomPropertyColumns, {
      displayProperties: {},
      displayFilters: { layout: "spreadsheet" },
      isEpic: false,
      onChange: () => {},
    })
  );
  assert.ok(button(env, "Saved").querySelector("svg.lucide-database"));
  assert.ok(button(env, "Empty").querySelector("svg.lucide-hash"));
  assert.ok(button(env, "Unknown").querySelector("svg.lucide-calendar"));
});

await test("Cascading creation opens options only after acceptance using canonical identity and revision", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { TypeLibrary } = await import("../../apps/web/core/components/work-item-types/library.tsx");
  let rejectSave;
  let fail = true;
  const submissions = [];
  const persistence = {
    types: [],
    properties: [],
    attachments: [],
    projects: [],
    members: [],
    projectCount: 0,
    revision: 8,
    saveProperty: async (property, creating, revision) => {
      submissions.push({ property, creating, revision });
      if (fail)
        await new Promise((_, reject) => {
          rejectSave = reject;
        });
      return { property: { ...property, name: "Canonical area" }, revision: 9 };
    },
    propertyRemoval: { scope: "acme", accountId: "alice" },
  };
  await env.render(wrap(React.createElement(TypeLibrary, { persistence })));
  await act(async () => button(env, "Properties").click());
  await act(async () => button(env, "Add new property").click());
  await changeFormControl(env.container.querySelector('input[id="Title"]'), "Area");
  await changeFormControl(env.container.querySelector('select[aria-label="Property type"]'), "Cascading");
  await submitForm(env.container);
  assert.equal(env.container.querySelector('[aria-label="Cascade follow-up"]'), null);
  await act(async () => rejectSave(new Error("Creation rejected")));
  assert.equal(env.container.querySelector('[aria-label="Cascade follow-up"]'), null);
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Creation rejected/);
  fail = false;
  await submitForm(env.container);
  const followup = env.container.querySelector('[aria-label="Cascade follow-up"]');
  assert.ok(followup);
  assert.equal(followup.dataset.property, submissions[0].property.id);
  assert.equal(submissions[1].property.id, submissions[0].property.id);
  assert.equal(followup.dataset.revision, "9");
  assert.match(followup.textContent, /Canonical area/);
  assert.equal(submissions[1].revision, 8);
  await act(async () => button(env, "Skip").click());
  assert.equal(env.container.querySelector('[aria-label="Cascade follow-up"]'), null);
  assert.equal(submissions.length, 2);
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
