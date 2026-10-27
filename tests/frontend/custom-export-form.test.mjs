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
const stubs = {
  "@/hooks/use-work-item-type-catalog": `export const useWorkItemTypeCatalog=()=>({identities:{enabled:globalThis.customExportEnabled}});`,
  "@/hooks/use-workspaces": `export const useCurrentWorkspace=()=>({data:{id:"workspace"}});`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails=()=>({projects:[{id:"project",workspace:"workspace",name:"Project",identifier:"PRJ"}]});`,
  "@/hooks/use-permissions": `export const useWorkspacePermissions=()=>({can:()=>true});`,
  "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:{id:"user"}});`,
  "@/hooks/use-creation-permissions": `export const useCreationPermissions=()=>({canPerformAnyCreateAction:true,projectsWithCreatePermissions:{project:true}});`,
  "@/services/project/project-export.service": `export class ProjectExportService {csvExport(slug,payload){return globalThis.customExportSubmit(slug,payload);}}`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=()=>{};`,
  "@plane/propel/button": `const React=globalThis.customExportReact;export const Button=({loading,variant,size,...props})=>React.createElement("button",{...props,disabled:loading});`,
  "@plane/ui": `const React=globalThis.customExportReact;
    export const CustomSearchSelect=()=>null;
    const Select=()=>null; Select.Option=()=>null; export const CustomSelect=Select;
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
