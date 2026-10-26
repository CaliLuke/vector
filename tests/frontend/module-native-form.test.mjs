import { changeFormControl as change, submitForm as submit } from "./helpers/native-forms.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
registerWebModuleHooks();
const nativeForms = observeNativeForms(["apps/web/core/components/modules/form.tsx"]);
globalThis.moduleFormReact = React;
const stubs = {
  "@/hooks/use-creation-permissions": `export const useCreationPermissions = () => ({projectsWithCreatePermissions:{first:true,second:true}});`,
  "@plane/i18n": `export const useTranslation = () => ({t:key=>key});`,
  "@plane/propel/button": `const React=globalThis.moduleFormReact;export const Button=({loading,variant,size,children,...props})=>React.createElement("button",props,children);`,
  "@plane/ui": `const React=globalThis.moduleFormReact;
    export const EModalPosition={TOP:"top"};export const EModalWidth={XXL:"xxl"};export const ModalCore=({children})=>React.createElement("div",{role:"dialog"},children);
    export const Input=({hasError,...props})=>React.createElement("input",props);
    export const TextArea=({hasError,...props})=>React.createElement("textarea",props);
    export const CustomSelect=Object.assign(({value,onChange,children,disabled})=>React.createElement("select",{value,disabled,onChange:event=>onChange(event.target.value)},children),{Option:({value,children})=>React.createElement("option",{value},children)});`,
  "@/components/dropdowns/project/dropdown": `const React=globalThis.moduleFormReact;export const ProjectDropdown=({value,onChange,disabled})=>React.createElement("select",{"aria-label":"Project",value,disabled,onChange:event=>onChange(event.target.value)},["first","second"].map(id=>React.createElement("option",{key:id,value:id},id)));`,
  "@/components/dropdowns/member/dropdown": `const React=globalThis.moduleFormReact;export const MemberDropdown=({value,onChange,multiple,disabled})=>React.createElement("input",{"aria-label":multiple?"Members":"Lead",value:multiple?value.join(","):value??"",disabled,onChange:event=>onChange(multiple?event.target.value.split(","):event.target.value)});`,
  "@/components/dropdowns/date-range": `const React=globalThis.moduleFormReact;export const DateRangeDropdown=({value})=>React.createElement("output",null,JSON.stringify(value));`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
await test("module edits preserve dates and unsaved values across refetches and failed saves", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ModuleForm } = await import("../../apps/web/core/components/modules/form.tsx");
  let fail = true,
    closed = 0;
  const saved = [];
  const data = {
    id: "module",
    project_id: "first",
    name: "Original",
    description: "",
    status: "backlog",
    lead_id: null,
    member_ids: [],
    start_date: "2030-01-01",
    target_date: "2030-02-01",
  };
  const props = {
    workspaceSlug: "acme",
    projectId: "first",
    data,
    handleClose: () => closed++,
    handleFormSubmit: async (value) => {
      saved.push(value);
      if (fail) throw new Error("offline");
      props.handleClose();
    },
  };
  await environment.render(React.createElement(ModuleForm, props));
  await change(environment.container.querySelector('[name="name"]'), "Draft");
  await environment.render(React.createElement(ModuleForm, { ...props, data: { ...data, name: "Refetched" } }));
  await submit(environment.container);
  assert.equal(closed, 0);
  assert.equal(
    nativeForms.getForms("apps/web/core/components/modules/form.tsx").at(-1).state.isSubmitSuccessful,
    false
  );
  assert.equal(environment.container.querySelector('[name="name"]').value, "Draft");
  assert.equal(saved[0].start_date, "2030-01-01");
  assert.equal(saved[0].target_date, "2030-02-01");
  fail = false;
  await submit(environment.container);
  assert.equal(closed, 1);
  assert.equal(nativeForms.getForms("apps/web/core/components/modules/form.tsx").at(-1).state.isSubmitSuccessful, true);
  assert.equal(saved[1].name, "Draft");
});

await test("changing a new module's project clears members from the previous project", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ModuleForm } = await import("../../apps/web/core/components/modules/form.tsx");
  const saved = [];
  await environment.render(
    React.createElement(ModuleForm, {
      workspaceSlug: "acme",
      projectId: "first",
      handleClose: () => {},
      handleFormSubmit: async (value) => saved.push(value),
    })
  );
  await change(environment.container.querySelector('[name="name"]'), "Module");
  await change(environment.container.querySelector('[aria-label="Lead"]'), "old-lead");
  await change(environment.container.querySelector('[aria-label="Members"]'), "old-member");
  await change(environment.container.querySelector('[aria-label="Project"]'), "second");
  await submit(environment.container);
  assert.equal(saved[0].project_id, "second");
  assert.equal(saved[0].lead_id, null);
  assert.deepEqual(saved[0].member_ids, []);
});

nativeForms.restore();
