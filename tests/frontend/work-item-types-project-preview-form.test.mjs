import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl, submitForm, waitForSubmitCompletion } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";

const source = "apps/web/core/components/work-item-types/project-preview.tsx";
const workItemsSource = "apps/web/core/components/work-item-types/work-item-preview.tsx";
const stubs = {
  "@plane/propel/button": `const React=globalThis.projectTypesReact;export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
  "./controls": `const React=globalThis.projectTypesReact;export const inputStyle="";export const TypeIcon=()=>null;export const PropertyIcon=()=>null;export const Field=()=>null;export const Confirmation=()=>null;export const EmptyState=({title,description})=>React.createElement("p",null,title,description);export const PreviewDialog=({title,description,children,footer,onClose})=>React.createElement("section",{"aria-label":title},React.createElement("button",{type:"button","aria-label":"Close import",onClick:onClose}),React.createElement("p",null,description),children,footer);`,
  "./property-input": `const React=globalThis.projectTypesReact;export const PropertyInput=({property,value,disabled,onChange})=>React.createElement("input",{"aria-label":property.name,value:Array.isArray(value)?value.join(","):String(value??""),disabled,onChange:event=>onChange(event.target.value)});`,
  "@plane/ui": `const React=globalThis.projectTypesReact;export const ToggleSwitch=({label,value,disabled,onChange})=>React.createElement("input",{type:"checkbox","aria-label":label,checked:value,disabled,onChange:event=>onChange(event.target.checked)});export const CustomSearchSelect=()=>null;export const Form=({onSubmit,...props})=>React.createElement("form",{...props,onSubmit:event=>{event.preventDefault();event.stopPropagation();void onSubmit();}});`,
};
const nativeForms = registerWorkItemTypesFormHarness({
  sourceFiles: [source, workItemsSource],
  stubs,
  reactGlobal: "projectTypesReact",
  react: React,
});
const { ProjectTypesPreview } = await import("../../apps/web/core/components/work-item-types/project-preview.tsx");
const { WorkItemsPreview } = await import("../../apps/web/core/components/work-item-types/work-item-preview.tsx");
const { WorkItemTypesPreviewProvider, useTypePreview } =
  await import("../../apps/web/core/components/work-item-types/preview-provider.tsx");

function Screen() {
  const { content, notice } = useTypePreview();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(ProjectTypesPreview),
    React.createElement("output", { "data-testid": "projects" }, JSON.stringify(content.projects)),
    React.createElement("output", { "data-testid": "notice" }, notice)
  );
}

function WorkItemsScreen() {
  const { content, notice } = useTypePreview();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(WorkItemsPreview),
    React.createElement("output", { "data-testid": "work-items" }, JSON.stringify(content.items)),
    React.createElement("output", { "data-testid": "notice" }, notice)
  );
}
const click = (node) => act(async () => node.click());

await test("project import selection is native, deduplicated, and reset on reopen or scope change", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(Screen)));
  const button = (name, scope = environment.container) =>
    [...scope.querySelectorAll("button")].find((node) => node.textContent === name);
  const projects = () => JSON.parse(environment.container.querySelector('[data-testid="projects"]').textContent);

  await click(button("Import from workspace"));
  const dialog = () => environment.container.querySelector('section[aria-label="Import from workspace"]');
  const add = () => [...dialog().querySelectorAll("button")].find((node) => node.textContent.startsWith("Add"));
  assert.equal(add().disabled, true);
  assert.deepEqual(nativeForms.getForms(source).at(-1).state.values.selectedTypeIds, []);

  const incident = dialog().querySelector('input[type="checkbox"]');
  assert.ok(incident);
  await click(incident);
  assert.equal(add().textContent, "Add (1)");
  await click(add());
  assert.deepEqual(
    projects()
      .find((project) => project.id === "platform")
      .types.slice(-1),
    ["incident"]
  );
  assert.equal(nativeForms.getForms(source).at(-1).state.isSubmitSuccessful, true);
  assert.deepEqual(nativeForms.getForms(source).at(-1).state.values.selectedTypeIds, []);

  await click(button("Import from workspace"));
  assert.equal(add().disabled, true);
  const projectPicker = environment.container.querySelector("select");
  await changeFormControl(projectPicker, "mobile");
  assert.equal(add().disabled, true);
  assert.deepEqual(nativeForms.getForms(source).at(-1).state.values.selectedTypeIds, []);

  const epic = [...dialog().querySelectorAll('input[type="checkbox"]')].find((input) =>
    input.closest("label").textContent.includes("Epic")
  );
  await click(epic);
  assert.equal(add().textContent, "Add (1)");
  await click(add());
  assert.deepEqual(
    projects()
      .find((project) => project.id === "mobile")
      .types.slice(-1),
    ["epic"]
  );
  assert.match(
    environment.container.querySelector('[data-testid="notice"]').textContent,
    /1 types imported into Mobile App/
  );
});

const clickText = async (scope, selector, text) => {
  const node = [...scope.querySelectorAll(selector)].find((element) => element.textContent.includes(text));
  assert.ok(node, `expected a ${selector} containing ${text}`);
  await click(node);
};

await test("sample property drafts stay with their work item when selection changes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(
    React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(WorkItemsScreen))
  );

  const itemValues = (id) =>
    JSON.parse(environment.container.querySelector('[data-testid="work-items"]').textContent).find(
      (item) => item.id === id
    ).values;

  await clickText(environment.container, "tbody button", "Keep the session when switching devices");
  const severity = environment.container.querySelector('input[aria-label="Severity"]');
  await changeFormControl(severity, "critical");
  await clickText(environment.container, "button", "Sample work items");
  await clickText(environment.container, "tbody button", "A more reliable sign-in experience");
  const summary = environment.container.querySelector('input[aria-label="Customer impact"]');
  await changeFormControl(summary, "Epic-specific sample draft");
  await clickText(environment.container, "button", "Sample work items");
  await clickText(environment.container, "tbody button", "Keep the session when switching devices");

  assert.equal(environment.container.querySelector('input[aria-label="Severity"]').value, "critical");
  assert.equal(itemValues("WEB-142").severity, "major", "editing is a draft until sample values are saved");
  const form = nativeForms.getForms(workItemsSource).at(-1);
  assert.ok(form);
  await submitForm(environment.container);
  await waitForSubmitCompletion(form);

  assert.equal(itemValues("WEB-142").severity, "critical");
  assert.match(itemValues("WEB-100").summary, /Reduce interruptions/, "the other item's base value stays intact");
  const savedItems = JSON.parse(environment.container.querySelector('[data-testid="work-items"]').textContent);
  assert.match(savedItems.find((item) => item.id === "WEB-142").activity[0], /updated custom properties/);
  assert.match(
    environment.container.querySelector('[data-testid="notice"]').textContent,
    /sample property values saved/i
  );
});

await test("required sample properties reject invalid drafts without losing their values", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(
    React.createElement(WorkItemTypesPreviewProvider, null, React.createElement(WorkItemsScreen))
  );
  const itemButton = [...environment.container.querySelectorAll("tbody button")].find((button) =>
    button.textContent.includes("Keep the session when switching devices")
  );
  await click(itemButton);
  const severity = environment.container.querySelector('input[aria-label="Severity"]');
  await changeFormControl(severity, "");
  const form = nativeForms.getForms(workItemsSource).at(-1);

  await submitForm(environment.container);
  await waitForSubmitCompletion(form);
  assert.equal(
    environment.container.querySelector('[data-testid="work-items"]').textContent.includes('"severity":"major"'),
    true
  );
  assert.equal(environment.container.querySelector('input[aria-label="Severity"]').value, "");
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Severity/);
  assert.match(environment.container.querySelector('[data-testid="notice"]').textContent, /Fill in Severity/);

  await changeFormControl(environment.container.querySelector('input[aria-label="Severity"]'), "critical");
  await submitForm(environment.container);
  await waitForSubmitCompletion(form);
  assert.equal(
    JSON.parse(environment.container.querySelector('[data-testid="work-items"]').textContent).find(
      (item) => item.id === "WEB-142"
    ).values.severity,
    "critical"
  );
});
