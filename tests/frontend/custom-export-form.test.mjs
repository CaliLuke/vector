import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { submitForm } from "./helpers/native-forms.mjs";

registerWebModuleHooks();
globalThis.customExportReact = React;
globalThis.customExportToasts = [];
globalThis.customExportGeneration = 0;
const stubs = {
  "@tanstack/react-query": "export const useQueryClient = () => ({});",
  "@/hooks/use-identity-generation": "export const useIdentityGeneration = () => globalThis.customExportGeneration;",
  "@/hooks/use-work-item-type-catalog": `export const useWorkItemTypeCatalog=()=>({identities:{enabled:globalThis.customExportEnabled}});`,
  "@/hooks/use-workspaces": `export const useCurrentWorkspace=()=>({data:{id:"workspace"}});`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails=()=>({projects:globalThis.customExportProjects??[{id:"project",workspace:"workspace",name:"Project",identifier:"PRJ"}]});`,
  "@/hooks/use-permissions": `export const useWorkspacePermissions=()=>({can:()=>true});`,
  "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:{id:"user"}});`,
  "@/hooks/use-creation-permissions": `export const useCreationPermissions=()=>({canPerformAnyCreateAction:true,projectsWithCreatePermissions:globalThis.customExportCreatePermissions??{project:true}});`,
  "@/services/project/project-export.service": `export class ProjectExportService {csvExport(slug,payload){return globalThis.customExportSubmit(slug,payload);}}`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error",WARNING:"warning"};export const setToast=toast=>globalThis.customExportToasts.push(toast);`,
  "@plane/propel/button": `const React=globalThis.customExportReact;export const Button=({loading,variant,size,...props})=>React.createElement("button",{...props,disabled:loading});`,
  "@plane/ui": `const React=globalThis.customExportReact;
    export const CustomSearchSelect=({value,onChange,options,disabled,label})=>React.createElement('div',null,React.createElement('output',{'aria-label':'Selected export projects'},label),React.createElement('select',{multiple:true,'aria-label':'Export projects',value,disabled,onChange:event=>onChange([...event.target.selectedOptions].map(option=>option.value))},...(options??[]).map(option=>React.createElement('option',{key:option.value,value:option.value},option.query))));
    const Context=React.createContext(null);
    export const CustomSelect=({value,onChange,disabled,children})=>React.createElement(Context.Provider,{value:{value,onChange,disabled}},children);
    CustomSelect.Option=({value,children})=>{const current=React.useContext(Context);return React.createElement('button',{type:'button','aria-label':'Format '+value.provider,'aria-pressed':current.value.provider===value.provider,disabled:current.disabled,onClick:()=>current.onChange(value)},children);};
    export const ToggleSwitch=({value,onChange,...props})=>React.createElement("button",{...props,type:"button",role:"switch","aria-checked":value,onClick:()=>onChange(!value)});`,
  "@plane/utils/common": `export const cn=(...values)=>values.filter(Boolean).join(" ");`,
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    return stubs[specifier] ? createModuleStub(stubs[specifier]) : nextResolve(specifier, context);
  },
});

await test("export form requires an explicit custom-property choice and preserves it after failure", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const submissions = [];
  globalThis.customExportEnabled = true;
  let fail = false;
  globalThis.customExportSubmit = async (slug, payload) => {
    submissions.push({ slug, ...payload });
    if (fail) throw new Error("offline");
  };
  const { ExportForm } = await import("../../apps/web/core/components/exporter/export-form.tsx");
  await env.render(
    React.createElement(ExportForm, { workspaceSlug: "demo", provider: "csv", mutateServices: () => {} })
  );

  const toggle = env.container.querySelector('[aria-label="Include custom properties"]');
  assert.equal(toggle.getAttribute("aria-checked"), "false");
  await submitForm(env.container);
  assert.equal(submissions[0].include_custom_properties, false);
  assert.equal(submissions[0].slug, "demo");

  await act(async () => toggle.click());
  fail = true;
  await submitForm(env.container);
  assert.equal(submissions[1].include_custom_properties, true);
  assert.equal(toggle.getAttribute("aria-checked"), "true");

  fail = false;
  await submitForm(env.container);
  assert.equal(submissions[2].include_custom_properties, true);
  await act(async () => toggle.click());
  await submitForm(env.container);
  assert.equal(submissions[3].include_custom_properties, false);

  await act(async () => toggle.click());
  globalThis.customExportEnabled = false;
  await env.render(
    React.createElement(ExportForm, { workspaceSlug: "demo", provider: "csv", mutateServices: () => {} })
  );
  assert.equal(env.container.querySelector('[aria-label="Include custom properties"]'), null);
  await submitForm(env.container);
  assert.equal(submissions[4].include_custom_properties, false);
});

void test("an accepted export remains successful when refreshing its list fails", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  globalThis.customExportToasts = [];
  let submissions = 0;
  globalThis.customExportSubmit = async () => {
    submissions++;
  };
  const refresh = Promise.withResolvers();
  const { ExportForm } = await import("../../apps/web/core/components/exporter/export-form.tsx");
  await env.render(
    React.createElement(ExportForm, { workspaceSlug: "demo", provider: "csv", mutateServices: () => refresh.promise })
  );
  await submitForm(env.container);
  assert.equal(submissions, 1);
  assert.equal(globalThis.customExportToasts[0].type, "success");
  await act(async () => refresh.reject(new Error("refresh failure")));
  assert.deepEqual(
    globalThis.customExportToasts.map((toast) => toast.type),
    ["success", "warning"]
  );
  assert.match(globalThis.customExportToasts[1].message, /export was accepted/);
  assert.equal(submissions, 1);
});

void test("a completed export cannot publish feedback or refresh after a session replacement", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  globalThis.customExportToasts = [];
  const pending = Promise.withResolvers();
  globalThis.customExportSubmit = () => pending.promise;
  const refresh = t.mock.fn();
  const { ExportForm } = await import("../../apps/web/core/components/exporter/export-form.tsx");
  const render = () =>
    env.render(React.createElement(ExportForm, { workspaceSlug: "demo", provider: "csv", mutateServices: refresh }));
  await render();
  await submitForm(env.container);
  globalThis.customExportGeneration++;
  await render();
  await act(async () => pending.resolve());
  assert.deepEqual(globalThis.customExportToasts, []);
  assert.equal(refresh.mock.callCount(), 0);
});

void test("export project and format choices remain scoped and survive a failed request", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  globalThis.customExportProjects = [
    { id: "one", workspace: "workspace", name: "One", identifier: "ONE" },
    { id: "two", workspace: "workspace", name: "Two", identifier: "TWO" },
    { id: "foreign", workspace: "other", name: "Foreign", identifier: "OTHER" },
    { id: "archived", workspace: "workspace", name: "Archived", identifier: "OLD", archived_at: "2026-01-01" },
    { id: "forbidden", workspace: "workspace", name: "Forbidden", identifier: "NO" },
  ];
  globalThis.customExportCreatePermissions = { one: true, two: true, foreign: true, archived: true };
  globalThis.customExportEnabled = true;
  t.after(() => {
    delete globalThis.customExportProjects;
    delete globalThis.customExportCreatePermissions;
  });
  const submissions = [];
  let fail = true;
  globalThis.customExportSubmit = async (slug, payload) => {
    submissions.push({ slug, ...payload });
    if (fail) throw new Error("offline");
  };
  const { ExportForm } = await import("../../apps/web/core/components/exporter/export-form.tsx");
  await env.render(React.createElement(ExportForm, { workspaceSlug: "demo", mutateServices: () => {} }));
  const projects = env.container.querySelector('[aria-label="Export projects"]');
  assert.deepEqual(
    [...projects.options].map((option) => option.value),
    ["one", "two"]
  );
  await act(async () => {
    for (const option of projects.options) option.selected = true;
    projects.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await act(async () => env.container.querySelector('[aria-label="Format xlsx"]').click());
  assert.equal(env.container.querySelector('[aria-label="Selected export projects"]').textContent, "ONE, TWO");
  await submitForm(env.container);
  assert.deepEqual(submissions, [
    {
      slug: "demo",
      provider: "xlsx",
      project: ["one", "two"],
      multiple: true,
      rich_filters: {},
      include_custom_properties: false,
    },
  ]);
  assert.deepEqual(
    [...projects.selectedOptions].map((option) => option.value),
    ["one", "two"]
  );
  assert.equal(env.container.querySelector('[aria-label="Format xlsx"]').getAttribute("aria-pressed"), "true");
  fail = false;
  await submitForm(env.container);
  assert.deepEqual(submissions[1], submissions[0]);
});
