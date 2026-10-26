import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl as change, submitForm as submit } from "./helpers/native-forms.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
registerWebModuleHooks();
const cycleFormSource = "apps/web/core/components/cycles/form.tsx";
const viewFormSource = "apps/web/core/components/views/form.tsx";
const labelFormSource = "apps/web/core/components/issues/issue-detail/label/create-label.tsx";
const emailFormSource = "apps/web/core/components/core/modals/change-email-modal.tsx";
const formObserver = observeNativeForms([cycleFormSource, viewFormSource, labelFormSource, emailFormSource]);
const state = {};
globalThis.nativeFormWorkflows = state;
globalThis.nativeWorkflowReact = React;
const stubs = {
  "@/hooks/use-current-user": `export {useCurrentUser,useCurrentUserId} from "${new URL("../../apps/web/core/hooks/use-current-user.ts", import.meta.url).href}"; export const useSignOut = () => ({mutateAsync: () => globalThis.nativeFormWorkflows.user.signOut()});`,
  "@/hooks/use-creation-permissions": `export const useCreationPermissions = () => ({projectsWithCreatePermissions:{first:true,second:true}});`,
  "@/hooks/store/use-project": `export const useProject = () => ({getProjectById:()=>({cycle_view:true,module_view:true})});`,
  "@/hooks/use-platform-os": `export const usePlatformOS = () => ({isMobile:false});`,
  "@/hooks/use-local-storage": `export default () => ({setValue:()=>{}});`,
  "@/services/auth.service": `export const authService = {requestCSRFToken: async () => ({csrf_token:"fresh"})}; export const signOutCurrentSession=()=>globalThis.nativeFormWorkflows.user.signOut();`,
  "@/services/user.service": `export default {checkEmail: (...args)=>globalThis.nativeFormWorkflows.checkEmail(...args),generateEmailCode:(...args)=>globalThis.nativeFormWorkflows.generateCode(...args),verifyEmailCode:(...args)=>globalThis.nativeFormWorkflows.verifyCode(...args)};`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=()=>{};`,
  "@plane/ui": `export { Form, FormErrors } from "${new URL("../../packages/ui/src/form-fields/root.tsx", import.meta.url).href}";
    export { ModalForm } from "${new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href}";
    const React=globalThis.nativeWorkflowReact;
    export const EModalPosition={TOP:"top",CENTER:"center"}, EModalWidth={XXL:"xxl"};
    export const ModalCore=({children,handleClose})=>React.createElement("div",{role:"dialog"},children,React.createElement("button",{type:"button","aria-label":"Dismiss",onClick:handleClose},"Dismiss"));
    export const Input=({hasError,inputSize,...props})=>React.createElement("input",props);
    export const TextArea=({hasError,...props})=>React.createElement("textarea",props);
    export const InputColorPicker=({hasError,...props})=>React.createElement("input",{...props,onChange:event=>props.onChange(event.target.value)});`,
  "@/components/dropdowns/project/dropdown": `const React=globalThis.nativeWorkflowReact;export const ProjectDropdown=({value,onChange})=>React.createElement("select",{value,"aria-label":"Project",onChange:event=>onChange(event.target.value)},["first","second"].map(value=>React.createElement("option",{key:value,value},value)));`,
  "@/components/dropdowns/date-range": `const React=globalThis.nativeWorkflowReact;export const DateRangeDropdown=({value,onSelect})=>React.createElement("div",null,["from","to"].map(key=>React.createElement("input",{key,name:"date-"+key,value:value[key]?.toISOString().slice(0,10)??"",onChange:event=>onSelect({...value,[key]:new Date(event.target.value)})})));`,
  "@/components/issues/issue-layouts/filters": `export const DisplayFiltersSelection=()=>null;export const FiltersDropdown=({children})=>children;`,
  "@/components/work-item-filters/filters-hoc/project-level": `export const ProjectLevelWorkItemFiltersHOC=()=>null;`,
  "@/components/work-item-filters/filters-hoc/workspace-level": `const React=globalThis.nativeWorkflowReact;export const WorkspaceLevelWorkItemFiltersHOC=({updateFilters})=>React.createElement("button",{type:"button","aria-label":"Edit filters",onClick:()=>updateFilters({and:[{priority__in:"high"},{not:{state_group__in:"completed"}}]})},"Edit filters");`,
  "@/components/work-item-filters/filters-row": `export const WorkItemFiltersRow=()=>null;`,
  "../dropdowns/layout": `export const LayoutDropDown=()=>null;`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
async function setup(t) {
  const env = await createReactDomTestEnvironment();
  const router = createRouterFixture("/workspace/projects/first/issues/");
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false, gcTime: Infinity } },
  });
  t.after(async () => {
    await env.unmount();
    client.clear();
  });
  return {
    ...env,
    client,
    container: document.body,
    render: (element) => router.render(env, React.createElement(QueryClientProvider, { client }, element)),
  };
}
await test("cycle validation uses the selected project and failed creation retains the draft", async (t) => {
  const env = await setup(t);
  const { CycleCreateUpdateModal } = await import("../../apps/web/core/components/cycles/modal.tsx");
  const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
  const validations = [],
    creates = [];
  let closed = 0,
    fail = true;
  t.mock.method(CycleService.prototype, "validateDates", async (...args) => {
    validations.push(args);
    return { status: true };
  });
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { seedPermissionQueries } = await import("./helpers/permission-query-fixture.mjs");
  env.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(env.client, {
    userId: "alice",
    workspaceSlug: "workspace",
    workspaceRole: 15,
    projectRoles: { first: 20, second: 20 },
  });
  t.mock.method(CycleService.prototype, "create", async (...args) => {
    creates.push(args);
    if (fail) throw new Error("offline");
    return { id: "new-cycle", project_id: args[1], ...args[2] };
  });
  await env.render(
    React.createElement(CycleCreateUpdateModal, {
      isOpen: true,
      handleClose: () => closed++,
      workspaceSlug: "workspace",
      projectId: "first",
    })
  );
  await change(env.container.querySelector('[name="name"]'), "Draft cycle");
  await change(env.container.querySelector("select"), "second");
  await change(env.container.querySelector('[name="date-from"]'), "2030-01-01");
  await change(env.container.querySelector('[name="date-to"]'), "2030-01-02");
  await submit(env.container);
  assert.equal(closed, 0);
  assert.equal(creates[0][1], "second");
  assert.equal(validations[0][1], "second");
  assert.equal(env.container.querySelector('[name="name"]').value, "Draft cycle");
  assert.equal(env.container.querySelector('[role="alert"]').textContent, "offline");
  const [cycleForm] = formObserver.getForms(cycleFormSource);
  assert.ok(cycleForm, "the workflow mounted a real native form");
  assert.equal(cycleForm.state.isSubmitSuccessful, false);
  fail = false;
  await submit(env.container);
  assert.equal(closed, 1);
  assert.equal(creates.length, 2);
});
await test("view forms keep dirty values across refetches and failures", async (t) => {
  const env = await setup(t);
  const { ViewForm } = await import("../../apps/web/core/components/views/form.tsx");
  let fail = true;
  const saved = [];
  const props = {
    workspaceSlug: "workspace",
    data: { id: "view", name: "Original" },
    handleClose: () => {},
    handleFormSubmit: async (value) => {
      saved.push(value);
      if (fail) throw new Error("offline");
    },
  };
  await env.render(React.createElement(ViewForm, props));
  await change(env.container.querySelector('[name="name"]'), "Draft view");
  await act(async () => env.container.querySelector('[aria-label="Edit filters"]').click());
  await env.render(React.createElement(ViewForm, { ...props, data: { id: "view", name: "Refetched" } }));
  await submit(env.container);
  assert.equal(env.container.querySelector('[name="name"]').value, "Draft view");
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Could not save/);
  assert.equal(formObserver.getForms(viewFormSource).at(-1)?.state.isSubmitSuccessful, false);
  fail = false;
  await submit(env.container);
  assert.equal(saved[1].name, "Draft view");
  assert.deepEqual(saved[1].rich_filters, {
    and: [{ priority__in: "high" }, { not: { state_group__in: "completed" } }],
  });
  assert.equal(env.container.querySelector('[role="alert"]'), null);
});
await test("view filter edits participate in native form dirty state and reset", async (t) => {
  const env = await setup(t);
  const { ViewForm } = await import("../../apps/web/core/components/views/form.tsx");
  const original = { not: { priority__exact: "low" } };
  const saved = [];
  await env.render(
    React.createElement(ViewForm, {
      workspaceSlug: "workspace",
      data: { id: "view", name: "Original", rich_filters: original },
      handleClose: () => {},
      handleFormSubmit: async (value) => saved.push(value),
    })
  );
  const form = formObserver.getForms(viewFormSource).at(-1);
  assert.equal(form.state.isDirty, false);
  await act(async () => env.container.querySelector('[aria-label="Edit filters"]').click());
  assert.equal(form.state.isDirty, true);
  await submit(env.container);
  assert.deepEqual(saved[0].rich_filters, {
    and: [{ priority__in: "high" }, { not: { state_group__in: "completed" } }],
  });
  await act(async () => form.reset());
  assert.equal(form.state.isDirty, false);
  await submit(env.container);
  assert.deepEqual(saved[1].rich_filters, original);
});
await test("editing view metadata preserves malformed saved filter siblings", async (t) => {
  const env = await setup(t);
  const { ViewForm } = await import("../../apps/web/core/components/views/form.tsx");
  const original = { and: [{ priority__exact: "high" }], unsupported__exact: "kept" };
  const saved = [];
  await env.render(
    React.createElement(ViewForm, {
      workspaceSlug: "workspace",
      data: { id: "view", name: "Original", rich_filters: original },
      handleClose: () => {},
      handleFormSubmit: async (value) => saved.push(value),
    })
  );
  await change(env.container.querySelector('[name="name"]'), "Renamed view");
  await submit(env.container);
  assert.equal(saved[0].name, "Renamed view");
  assert.deepEqual(saved[0].rich_filters, original);
});
await test("retrying label attachment reuses the created label", async (t) => {
  const env = await setup(t);
  const { LabelCreate } = await import("../../apps/web/core/components/issues/issue-detail/label/create-label.tsx");
  let creates = 0,
    fail = true;
  const updates = [];
  await env.render(
    React.createElement(LabelCreate, {
      workspaceSlug: "workspace",
      projectId: "project",
      issueId: "issue",
      values: ["existing"],
      labelOperations: {
        createLabel: async () => {
          creates++;
          return { id: "created" };
        },
        updateIssue: async (...args) => {
          updates.push(args);
          if (fail) throw new Error("offline");
        },
      },
    })
  );
  await act(async () => env.container.querySelector("button").click());
  await change(env.container.querySelector('[name="name"]'), "New label");
  await submit(env.container);
  assert.equal(creates, 1);
  assert.ok(env.container.querySelector('[role="alert"]'));
  assert.equal(env.container.querySelector('[name="name"]').disabled, true);
  assert.equal(formObserver.getForms(labelFormSource).at(-1)?.state.isSubmitSuccessful, false);
  fail = false;
  await submit(env.container);
  assert.equal(creates, 1);
  assert.deepEqual(updates[1][3], { label_ids: ["existing", "created"] });
  assert.equal(env.container.querySelector("form"), null);
});
await test("email changes retry sign-out without verifying the email twice", async (t) => {
  const env = await setup(t);
  const { ChangeEmailModal } = await import("../../apps/web/core/components/core/modals/change-email-modal.tsx");
  let verifies = 0,
    signouts = 0,
    closed = 0;
  state.checkEmail = async () => ({ existing: false });
  state.generateCode = async () => {};
  state.verifyCode = async () => {
    verifies++;
  };
  state.user = {
    signOut: async () => {
      if (++signouts === 1) throw new Error("offline");
    },
  };
  await env.render(React.createElement(ChangeEmailModal, { isOpen: true, onClose: () => closed++ }));
  const submitEmail = async () => {
    await submit(env.container);
    await act(async () => {
      const form = formObserver.getForms(emailFormSource).at(-1);
      if (!form.state.isSubmitting) return;
      await new Promise((resolve) => {
        const subscription = form.store.subscribe(() => {
          if (!form.state.isSubmitting) {
            subscription.unsubscribe();
            resolve();
          }
        });
      });
    });
  };
  await change(env.container.querySelector('[name="email"]'), "new@example.com");
  await submitEmail();
  await change(env.container.querySelector('[name="code"]'), "123456");
  await submitEmail();
  assert.equal(verifies, 1);
  assert.equal(closed, 0);
  assert.match(env.container.textContent, /Retry sign out/);
  assert.equal(formObserver.getForms(emailFormSource).at(-1)?.state.isSubmitSuccessful, false);
  await submitEmail();
  assert.equal(verifies, 1);
  assert.equal(signouts, 2);
  assert.equal(closed, 1);
});

formObserver.restore();
