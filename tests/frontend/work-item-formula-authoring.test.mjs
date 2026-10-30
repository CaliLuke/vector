import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";
globalThis.formulaUseQuery = useQuery;
registerWorkItemTypesFormHarness({
  sourceFiles: [],
  reactGlobal: "formulaReact",
  react: React,
  stubs: {
    "@plane/propel/button": `const React=globalThis.formulaReact;export const Button=({variant,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
    "@/services/work-item-types.service": `export const typeErrorMessage=error=>error.message;`,
    "@/hooks/use-project-issue-search": `export const useProjectIssueSearch=(scope,project,params,enabled)=>globalThis.formulaUseQuery({queryKey:["samples",scope,project,params],queryFn:({signal})=>globalThis.formulaSearch(project,params,signal),enabled});`,
    "./controls": `const React=globalThis.formulaReact;export const inputStyle="";
      export const Field=({label,children})=>React.createElement("label",null,label,children(label));
      export const Choice=({label,value,options,onChange})=>React.createElement("select",{"aria-label":label,value,onChange:event=>onChange(event.target.value)},options.map(option=>React.createElement("option",{key:option},option)));`,
  },
});
const { FormulaEditor } = await import("../../apps/web/core/components/work-item-types/formula-editor.tsx");
const { FormulaPreview } = await import("../../apps/web/core/components/work-item-types/formula-preview.tsx");
const { formulaReferences } = await import("../../apps/web/core/components/work-item-types/formula-reference.ts");
const { newProperty } = await import("../../apps/web/core/components/work-item-types/preview-content.ts");
const property = () => ({ ...newProperty("Formula"), id: "formula", name: "Calculation", expression: "{{name}} + 1" });
const library = {
  types: [{ id: "task", name: "Task", properties: [] }],
  properties: [],
  projects: [
    { id: "p", name: "Project" },
    { id: "q", name: "Other" },
  ],
};
const control = (env, label) => env.container.querySelector(`[id="${label}"]`);
const button = (env, text) => [...env.container.querySelectorAll("button")].find((node) => node.textContent === text);
const settle = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 15));
  });
const valueResult = (value) => ({
  valid: true,
  active: true,
  evaluated_at: "2026-10-25T00:00:00Z",
  contexts: [],
  result: { status: "value", kind: "Number", value },
});
async function environment(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  return { env, render: (child) => env.render(React.createElement(QueryClientProvider, { client }, child)) };
}
function Editor() {
  const [current, setCurrent] = React.useState(property());
  return React.createElement(FormulaEditor, {
    property: current,
    library,
    onChange: (expression, result) => setCurrent({ ...current, expression, result }),
  });
}

await test("references use stable identifiers and must be available on every attached type", () => {
  const props = [
    { id: "shared", internalName: "original_effort", name: "Renamed effort", kind: "Number" },
    { id: "local", internalName: "local", name: "Local", kind: "Text" },
    { id: "formula", internalName: "self", name: "Self", kind: "Number" },
    { id: "other", internalName: "other", name: "Other formula", kind: "Formula" },
  ];
  const types = [
    { id: "task", properties: props.map((p) => p.id) },
    { id: "bug", properties: ["shared", "formula"] },
  ];
  const custom = formulaReferences("formula", "task", types, props).filter((field) =>
    props.some((p) => p.internalName === field.name)
  );
  assert.deepEqual(custom, [{ name: "original_effort", title: "Renamed effort", kind: "Number" }]);
});
await test("field insertion keeps the cursor stable, including identical replacements; examples choose their result type", async (t) => {
  const { env, render } = await environment(t);
  await render(React.createElement(Editor));
  const input = control(env, "Expression");
  input.setSelectionRange(0, 8);
  await changeFormControl(control(env, "Insert field"), "name");
  assert.equal(input.selectionStart, 8);
  await changeFormControl(input, "{{name}} + 12");
  assert.equal(input.selectionStart, input.value.length);
  input.setSelectionRange(11, input.value.length);
  await changeFormControl(control(env, "Insert field"), "estimate_point");
  assert.equal(input.value, "{{name}} + {{estimate_point}}");
  assert.equal(input.selectionStart, input.value.length);
  await act(async () => button(env, "TODAY()").click());
  assert.equal(input.value, "TODAY()");
  assert.equal(env.container.querySelector('[aria-label="Result type"]').value, "Date");
  await act(async () => button(env, 'IF({{priority}} = "URGENT", "Urgent", "Normal")').click());
  assert.equal(env.container.querySelector('[aria-label="Result type"]').value, "Text");
});
await test("edited expressions cancel old evaluations and distinguish validation from calculation errors", async (t) => {
  const { env, render } = await environment(t);
  const requests = [];
  const persistence = {
    scope: "workspace",
    accountId: "owner",
    revision: 4,
    preview: (input, signal) => new Promise((resolve) => requests.push({ input, signal, resolve })),
  };
  let current = property();
  const show = () =>
    render(
      React.createElement(FormulaPreview, {
        property: current,
        typeId: "task",
        projects: library.projects,
        persistence,
      })
    );
  await show();
  await act(async () => button(env, "Evaluate").click());
  assert.equal(requests.length, 1);
  current = { ...current, expression: "1/0" };
  await show();
  assert.equal(requests[0].signal.aborted, true);
  await act(async () => requests[0].resolve(valueResult(12345)));
  await settle();
  assert.doesNotMatch(env.container.textContent, /12345/);
  await act(async () => button(env, "Evaluate").click());
  await act(async () =>
    requests[1].resolve({ ...valueResult(0), result: { status: "error", error: "Division by zero" } })
  );
  await settle();
  assert.match(env.container.textContent, /Expression is valid/);
  assert.match(env.container.textContent, /Sample calculation: Division by zero/);
  current = { ...current, expression: "INVALID(" };
  await show();
  assert.doesNotMatch(env.container.textContent, /Division by zero/);
  await act(async () => button(env, "Evaluate").click());
  await act(async () =>
    requests[2].resolve({ valid: false, validation_error: "Expected a closing parenthesis", contexts: [] })
  );
  await settle();
  assert.match(env.container.querySelector('[role="alert"]').textContent, /closing parenthesis/);
  assert.doesNotMatch(env.container.textContent, /Expression is valid/);
});

await test("sample search is type-scoped and project/type changes discard the old selection", async (t) => {
  const { env, render } = await environment(t);
  const searches = [];
  const evaluations = [];
  globalThis.formulaSearch = async (project, params, signal) => {
    searches.push({ project, params, signal });
    return [
      { id: "sample", name: "Sample", type_id: "task", project__identifier: "P", sequence_id: 1 },
      { id: "wrong", name: "Wrong type", type_id: "bug" },
    ];
  };
  const persistence = {
    scope: "workspace",
    accountId: "owner",
    revision: 4,
    preview: async (input) => {
      evaluations.push(input);
      return valueResult(0);
    },
  };
  const types = [...library.types, { id: "bug", name: "Bug", properties: [] }];
  await render(
    React.createElement(FormulaEditor, {
      property: property(),
      library: { ...library, types },
      onChange: () => {},
      persistence,
    })
  );
  await changeFormControl(control(env, "Preview project"), "p");
  await changeFormControl(control(env, "Find a sample work item"), "Sample");
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 350));
  });
  await settle();
  assert.equal(searches[0].params.type_id, "task");
  assert.equal(searches[0].project, "p");
  assert.doesNotMatch(control(env, "Sample work item").textContent, /Wrong type/);
  await changeFormControl(control(env, "Sample work item"), "sample");
  await act(async () => button(env, "Evaluate").click());
  await settle();
  assert.equal(evaluations[0].issue_id, "sample");
  assert.match(env.container.textContent, /Expression is valid/);
  await changeFormControl(control(env, "Preview project"), "q");
  assert.equal(control(env, "Sample work item").value, "");
  assert.equal(control(env, "Find a sample work item").value, "");
  assert.doesNotMatch(env.container.textContent, /Expression is valid/);
  await act(async () => button(env, "Evaluate").click());
  await settle();
  assert.equal(evaluations[1].issue_id, undefined);
  assert.equal(evaluations[1].project_id, "q");
  await changeFormControl(control(env, "Reference type"), "bug");
  assert.equal(control(env, "Preview project").value, "");
  assert.equal(control(env, "Sample work item"), null);
  assert.doesNotMatch(env.container.textContent, /Expression is valid/);
});
