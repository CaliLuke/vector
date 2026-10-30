import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";
const forms = registerWorkItemTypesFormHarness({
  sourceFiles: [
    "apps/web/core/components/work-item-types/context-editor.tsx",
    "apps/web/core/components/work-item-types/cascading-editor.tsx",
  ],
  reactGlobal: "propertyEditReact",
  react: React,
  stubs: {
    "@plane/propel/button": `const React=globalThis.propertyEditReact;export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
    "@plane/ui": `const React=globalThis.propertyEditReact;export const Form=({onSubmit,children,...props})=>React.createElement("form",{...props,onSubmit:event=>{event.preventDefault();onSubmit();}},children);`,
    "@/services/work-item-types.service": `export const typeErrorMessage=error=>error.message;`,
    "./controls": `const React=globalThis.propertyEditReact;export const inputStyle="";export const Confirmation=()=>null;
      export const Field=({label,children})=>React.createElement("label",null,label,children(label));
      export const Choice=({label,value,options,onChange,disabled})=>React.createElement("select",{"aria-label":label,value,disabled,onChange:event=>onChange(event.target.value)},options.map(option=>React.createElement("option",{key:option},option)));
      export const PreviewDialog=({title,children,footer})=>React.createElement("section",{"aria-label":title},children,footer);`,
  },
});
const { ContextEditor } = await import("../../apps/web/core/components/work-item-types/context-editor.tsx");
const { CascadingEditor } = await import("../../apps/web/core/components/work-item-types/cascading-editor.tsx");
const { newProperty } = await import("../../apps/web/core/components/work-item-types/preview-content.ts");
const { WorkItemTypesPreviewProvider } =
  await import("../../apps/web/core/components/work-item-types/preview-provider.tsx");
const button = (env, name) => [...env.container.querySelectorAll("button")].find((node) => node.textContent === name);
const settle = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
const context = (options) => ({
  id: "scope",
  name: "Scoped",
  projects: ["project"],
  types: [],
  required: null,
  active: null,
  defaultValue: null,
  format: null,
  ...options,
});
const library = { projects: [{ id: "project", name: "Project" }], types: [{ id: "task", name: "Task" }] };
async function screen(t, Component, property, extra = {}) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const previews = [];
  const commits = [];
  const persistence = {
    scope: "workspace",
    accountId: "owner",
    refreshRevision: async () => 4,
    preview: async (input) => {
      previews.push(structuredClone(input));
      return { id: "review", items: 0, types: 1, projects: 0 };
    },
    commit: async (input) => {
      commits.push(input);
    },
  };
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(
        WorkItemTypesPreviewProvider,
        null,
        React.createElement(Component, {
          property,
          library,
          onClose: () => {},
          review: { revision: 4, persistence },
          ...extra,
        })
      )
    )
  );
  return { env, previews, commits };
}
await test("canceling a context review retains staged edits and keeps options inherited", async (t) => {
  const property = { ...newProperty("Cascading"), id: "area", name: "Area", contexts: [context()] };
  const { env, previews } = await screen(t, ContextEditor, property);
  await act(async () => button(env, "Scoped").click());
  await changeFormControl(env.container.querySelector('input[id="Context name"]'), "Renamed context");
  await act(async () => button(env, "Apply context").click());
  await act(async () => button(env, "Save changes").click());
  await settle();
  assert.equal(previews.length, 1);
  assert.equal(previews[0].definition.contexts[0].name, "Renamed context");
  assert.equal(previews[0].definition.contexts[0].override_keys.includes("options"), false);
  await act(async () => button(env, "Cancel").click());
  assert.equal(env.container.querySelector('input[id="Context name"]').value, "Renamed context");
  assert.ok(button(env, "Apply context"));
});
await test("invalid numeric context drafts stay editable without opening a review", async (t) => {
  const property = {
    ...newProperty("Number"),
    id: "effort",
    name: "Effort",
    contexts: [context({ defaultValue: "3" })],
  };
  const { env, previews } = await screen(t, ContextEditor, property);
  await act(async () => button(env, "Scoped").click());
  const form = forms.getForms("apps/web/core/components/work-item-types/context-editor.tsx").at(-1);
  // Imported drafts and programmatic input must validate too, even when the browser's number input rejects letters.
  await act(async () =>
    form.setFieldValue("activeContext", { ...form.state.values.activeContext, defaultValue: "abc" })
  );
  await act(async () => button(env, "Apply context").click());
  await act(async () => button(env, "Save changes").click());
  assert.equal(previews.length, 0);
  assert.match(env.container.querySelector('[role="alert"]').textContent, /finite numeric default/);
  assert.equal(form.state.values.activeContext.defaultValue, "abc");
  assert.ok(button(env, "Apply context"));
});
await test("canceling cascading review retains removals regardless of option array order", async (t) => {
  const property = {
    ...newProperty("Cascading"),
    id: "area",
    name: "Area",
    options: [
      { id: "child", name: "Child", parent: "root", level: 1, nextLabel: "" },
      { id: "root", name: "Root", parent: "", level: 0, nextLabel: "" },
    ],
  };
  const { env, previews } = await screen(t, CascadingEditor, property);
  await act(async () => env.container.querySelector('button[aria-label="Delete Root"]').click());
  await act(async () => button(env, "Save changes").click());
  await settle();
  assert.deepEqual(previews[0].definition.configuration.options, []);
  const changes = env.container.querySelector('[aria-label="Changed cascading choices"]');
  assert.match(changes.textContent, /Root → Child/);
  assert.match(changes.textContent, /removed or changed paths/);
  await act(async () => button(env, "Cancel").click());
  assert.equal(env.container.querySelector('input[aria-label="Option Root"]'), null);
  const form = forms.getForms("apps/web/core/components/work-item-types/cascading-editor.tsx").at(-1);
  assert.deepEqual(form.state.values.options, []);
});
await test("switching to inherited options clears the obsolete branch before adding children", async (t) => {
  const property = {
    ...newProperty("Cascading"),
    id: "area",
    name: "Area",
    options: [{ id: "base", name: "Base", parent: "", level: 0, nextLabel: "" }],
    contexts: [context({ options: [{ id: "custom", name: "Custom", parent: "", level: 0, nextLabel: "" }] })],
  };
  const { env } = await screen(t, CascadingEditor, property);
  await changeFormControl(env.container.querySelector("select"), "scope");
  await act(async () => env.container.querySelector('input[aria-label="Option Custom"]').focus());
  assert.equal(
    [...env.container.querySelectorAll("button")].filter((node) => node.textContent === "Add option").length,
    2
  );
  const inherited = [...env.container.querySelectorAll("label")]
    .find((node) => node.textContent.includes("Use base options"))
    .querySelector("input");
  await act(async () => inherited.click());
  assert.ok(env.container.querySelector('input[aria-label="Option Base"]'));
  assert.equal(
    [...env.container.querySelectorAll("button")].filter((node) => node.textContent === "Add option").length,
    1
  );
  assert.match(env.container.textContent, /Choose Level 1 first/);
  assert.equal(env.container.querySelector('input[aria-label="Option Base"]').readOnly, true);
  assert.equal(env.container.querySelector('button[aria-label="Delete Base"]').disabled, true);
  assert.equal(button(env, "Add option").disabled, true);
  await act(async () => inherited.click());
  assert.equal(env.container.querySelector('input[aria-label="Option Base"]').readOnly, false);
  await changeFormControl(env.container.querySelector('input[aria-label="Option Base"]'), "Context base");
  const form = forms.getForms("apps/web/core/components/work-item-types/cascading-editor.tsx").at(-1);
  assert.equal(form.state.values.contexts[0].options[0].name, "Context base");
  assert.equal(form.state.values.options[0].name, "Base");
});

await test("member defaults show names and retain multiple selected member identities", async (t) => {
  const property = {
    ...newProperty("Member picker"),
    id: "reviewers",
    name: "Reviewers",
    multiple: true,
    defaultValue: "member-a",
    contexts: [context({ defaultValue: "member-a" })],
  };
  const { env, previews } = await screen(t, ContextEditor, property, {
    members: [
      { id: "member-a", name: "Alex" },
      { id: "member-b", name: "Blair" },
    ],
  });
  await act(async () => button(env, "Scoped").click());
  const summary = env.container.querySelector('[id="context-summary-scope"]');
  assert.match(summary.textContent, /Project · All types · Default/);
  const select = env.container.querySelector('select[id="Default value"]');
  assert.equal(select.multiple, true);
  assert.equal(select.options[1].textContent, "Alex");
  assert.equal(select.options[1].selected, true);
  assert.equal(select.options[2].textContent, "Blair");
  await act(async () => {
    select.options[2].selected = true;
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await act(async () => button(env, "Apply context").click());
  await act(async () => button(env, "Save changes").click());
  await settle();
  assert.deepEqual(previews[0].definition.contexts[0].configuration.default_value, ["member-a", "member-b"]);
});
await test("single dropdown defaults show option names and save one identity", async (t) => {
  const property = {
    ...newProperty("Dropdown"),
    id: "severity",
    name: "Severity",
    options: [
      { id: "low", name: "Low", parent: "", level: 0 },
      { id: "high", name: "High", parent: "", level: 0 },
    ],
    contexts: [context({ defaultValue: "low" })],
  };
  const { env, previews } = await screen(t, ContextEditor, property);
  await act(async () => button(env, "Scoped").click());
  const select = env.container.querySelector('select[id="Default value"]');
  assert.equal(select.multiple, false);
  assert.equal(select.options[2].textContent, "High");
  await changeFormControl(select, "high");
  await act(async () => button(env, "Apply context").click());
  await act(async () => button(env, "Save changes").click());
  await settle();
  assert.equal(previews[0].definition.contexts[0].configuration.default_value, "high");
});

await test("cascade review distinguishes removed paths from scope-change candidates", async () => {
  const { cascadeChanges } = await import("../../apps/web/core/components/work-item-types/cascade-change-summary.tsx");
  const property = {
    ...newProperty("Cascading"),
    options: [
      { id: "root", name: "Root", parent: "", level: 0 },
      { id: "leaf", name: "Leaf", parent: "root", level: 1 },
    ],
    contexts: [],
  };
  const inheriting = { ...property, contexts: [context({ active: false })] };
  assert.deepEqual(cascadeChanges(property, inheriting), [], "behavior contexts retain the base catalog");
  const renamed = structuredClone(property);
  for (const option of renamed.options) option.name = "Renamed";
  assert.deepEqual(cascadeChanges(property, renamed), [], "labels alone do not invalidate selected paths");
  const removed = { ...property, options: [property.options[0]] };
  assert.deepEqual(cascadeChanges(property, removed)[0].paths, [{ id: "leaf", name: "Root → Leaf" }]);
  const independent = { ...property, contexts: [context({ options: property.options })] };
  assert.equal(cascadeChanges(property, independent)[0].scopeChanged, true);
  const renamedContext = structuredClone(independent);
  renamedContext.contexts[0].name = "Renamed context";
  assert.deepEqual(cascadeChanges(independent, renamedContext), []);
});
