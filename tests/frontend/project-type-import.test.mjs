import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { submitForm } from "./helpers/native-forms.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
registerWebModuleHooks();
const source = "apps/web/core/components/work-item-types/project-type-import.tsx";
const forms = observeNativeForms([source]);
globalThis.projectTypesReact = React;
globalThis.projectTypesTestService = {};
globalThis.projectPropertiesTestService = {
  catalog: async () => ({ revision: 1, properties: [], attachments: [], projects: [], members: [] }),
};
const stubs = {
  "@plane/propel/button": `const React=globalThis.projectTypesReact;export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
  "@tanstack/react-router": `const React=globalThis.projectTypesReact;export const Link=({children})=>React.createElement("a",null,children);export const useParams=()=>({workspaceSlug:"demo"});`,
  "./library": `const React=globalThis.projectTypesReact;export const TypeLibrary=({persistence})=>React.createElement("button",{onClick:()=>void persistence.save({id:"bug",name:"Bug"})},"Save workspace type");`,
  "@plane/ui": `const React=globalThis.projectTypesReact;export const CustomSearchSelect=({value,onChange,options,disabled})=>React.createElement("select",{"aria-label":"Default work item type",value,disabled,onChange:event=>onChange(event.target.value)},options.map(option=>React.createElement("option",{key:option.value,value:option.value,disabled:option.disabled},option.query)));export const ToggleSwitch=({label,value,onChange,disabled})=>React.createElement("button",{role:"switch","aria-label":label,"aria-checked":value,disabled,onClick:()=>onChange(!value)},label);export const Form=({onSubmit,children,...props})=>React.createElement("form",{...props,onSubmit:event=>{event.preventDefault();void onSubmit();}},children);export const FormErrors=({errors})=>errors.length?React.createElement("div",{role:"alert"},errors.filter(Boolean).join(" ")):null;`,
  "@/services/work-item-properties.service": `export const workItemPropertiesService=globalThis.projectPropertiesTestService;`,
  "@/services/work-item-types.service": `export const typeErrorMessage=error=>error.message;export const workItemTypesService=globalThis.projectTypesTestService;`,
  "./controls": `const React=globalThis.projectTypesReact;export const TypeIcon=()=>null;export const EmptyState=({title,description})=>React.createElement("p",null,title,description);export const PreviewDialog=({title,children,footer,onClose})=>React.createElement("section",{role:"dialog","aria-label":title},children,footer,React.createElement("button",{onClick:onClose},"Close"));`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { ProjectTypeImport } = await import("../../apps/web/core/components/work-item-types/project-type-import.tsx");
const bug = { id: "bug", name: "Bug", description: "Defects", active: true };
const story = { id: "story", name: "Story", description: "User stories", active: true };
const checkbox = (container, name) =>
  [...container.querySelectorAll("label")].find((node) => node.textContent.startsWith(name)).querySelector("input");
const button = (container, name) => [...container.querySelectorAll("button")].find((node) => node.textContent === name);
const settle = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
await test("failed import retains selections, reports a failed submission, and retries successfully", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  forms.clear();
  let fail = true,
    closed = 0;
  const saved = [];
  await env.render(
    React.createElement(ProjectTypeImport, {
      types: [bug, story],
      projectName: "Platform",
      canManage: true,
      onSave: async (ids) => {
        if (fail) throw new Error("Catalog changed. Review and retry.");
        saved.push(ids);
      },
      onClose: () => closed++,
    })
  );
  await act(async () => checkbox(env.container, "Bug").click());
  await submitForm(env.container);
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Catalog changed/);
  assert.equal(checkbox(env.container, "Bug").checked, true);
  assert.equal(closed, 0);
  assert.equal(forms.getForms(source).at(-1).state.isSubmitSuccessful, false);
  fail = false;
  await submitForm(env.container);
  assert.deepEqual(saved, [["bug"]]);
  assert.equal(closed, 1);
  assert.equal(forms.getForms(source).at(-1).state.isSubmitSuccessful, true);
});
await test("unavailable selections remain visible and can be cleared without discarding other choices", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const saved = [];
  const props = { projectName: "Platform", canManage: true, onSave: async (ids) => saved.push(ids), onClose: () => {} };
  await env.render(React.createElement(ProjectTypeImport, { ...props, types: [bug, story] }));
  await act(async () => {
    checkbox(env.container, "Bug").click();
    checkbox(env.container, "Story").click();
  });
  await env.render(React.createElement(ProjectTypeImport, { ...props, types: [{ ...bug, active: false }, story] }));
  assert.equal(checkbox(env.container, "Bug").disabled, false);
  await submitForm(env.container);
  assert.equal(saved.length, 0);
  await env.render(React.createElement(ProjectTypeImport, { ...props, types: [story] }));
  assert.match(env.container.textContent, /No longer available/);
  await act(async () => checkbox(env.container, "Bug").click());
  assert.equal(checkbox(env.container, "Story").checked, true);
  await submitForm(env.container);
  assert.deepEqual(saved, [["story"]]);
});
await test("pending import locks choices and dismissal, and read-only access prevents submission", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let release,
    closed = 0,
    calls = 0;
  const props = {
    types: [bug, { ...story, active: false }],
    projectName: "Platform",
    canManage: true,
    onSave: () => {
      calls++;
      return new Promise((resolve) => {
        release = resolve;
      });
    },
    onClose: () => closed++,
  };
  await env.render(React.createElement(ProjectTypeImport, props));
  assert.equal(checkbox(env.container, "Story").disabled, true);
  await act(async () => checkbox(env.container, "Bug").click());
  await submitForm(env.container);
  assert.equal(checkbox(env.container, "Bug").disabled, true);
  assert.equal(button(env.container, "Cancel").disabled, true);
  await act(async () => button(env.container, "Close").click());
  assert.equal(closed, 0);
  await act(async () => release());
  assert.equal(closed, 1);
  await env.render(React.createElement(ProjectTypeImport, { ...props, canManage: false }));
  await submitForm(env.container);
  assert.equal(calls, 1);
});
const { ProjectTypes } = await import("../../apps/web/core/components/work-item-types/project-types.tsx");
await test("project removal failures stay inside the dialog and retry refreshes availability", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
  t.after(() => env.unmount(() => client.clear()));
  let catalog = {
    enabled: true,
    can_manage: true,
    revision: 1,
    project_name: "Platform",
    types: [{ ...bug, imported: true, project_active: true, available: true }],
  };
  let fail = true;
  globalThis.projectTypesTestService.projectCatalog = async () => structuredClone(catalog);
  globalThis.projectTypesTestService.mutateProject = async (slug, project, input) => {
    assert.equal(slug, "demo");
    assert.equal(project, "platform");
    assert.equal(input.action, "remove");
    if (fail) throw new Error("Please review the latest catalog.");
    catalog = {
      ...catalog,
      revision: 2,
      types: [{ ...bug, imported: false, project_active: false, available: false }],
    };
    return structuredClone(catalog);
  };

  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(ProjectTypes, { workspaceSlug: "demo", projectId: "platform" })
    )
  );
  await settle();
  await act(async () => button(env.container, "Remove Bug").click());
  await act(async () => button(env.container, "Remove type").click());
  await settle();
  const dialog = env.container.querySelector('[role="dialog"]');
  assert.ok(dialog);
  assert.match(dialog.querySelector('[role="alert"]').textContent, /latest catalog/);
  assert.ok(button(env.container, "Remove Bug"));
  fail = false;
  await act(async () => button(env.container, "Remove type").click());
  await settle();
  assert.equal(env.container.querySelector('[role="dialog"]'), null);
  assert.equal(button(env.container, "Remove Bug"), undefined);
});
const { WorkspaceTypeLibrary } = await import("../../apps/web/core/components/work-item-types/workspace-library.tsx");
await test("late workspace type save cannot repopulate a cleared session cache", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } },
  });
  t.after(() => env.unmount(() => client.clear()));
  const key = ["workspace", "demo", "work-item-types"];
  const catalog = {
    enabled: true,
    can_manage: true,
    revision: 1,
    project_count: 1,
    types: [bug],
    properties: [],
    attachments: [],
    projects: [],
    members: [],
  };
  let resolveSave;
  const pendingSave = new Promise((resolve) => {
    resolveSave = resolve;
  });
  globalThis.projectPropertiesTestService.catalog = async () => catalog;
  globalThis.projectTypesTestService.mutate = () => pendingSave;
  client.setQueryData(key, catalog);
  await env.render(React.createElement(QueryClientProvider, { client }, React.createElement(WorkspaceTypeLibrary)));
  await settle();
  await act(async () => button(env.container, "Save workspace type").click());
  await env.render(React.createElement("div", null, "Signed out"));
  client.clear();
  await act(async () => resolveSave({ ...catalog, revision: 2 }));
  await settle();
  assert.equal(client.getQueryData(key), undefined);
  assert.equal(client.getQueryCache().getAll().length, 0);
});
forms.restore();
