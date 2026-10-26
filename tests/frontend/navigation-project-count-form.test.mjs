import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

registerWebModuleHooks();
globalThis.navigationFormReact = React;
globalThis.navigationProjectPreferences = {
  navigationMode: "ACCORDION",
  limitedProjectsCount: 4,
  showLimitedProjects: true,
  failSave: true,
  savedCounts: [],
};
globalThis.navigationWorkspaceSlug = "acme";
const sourcePath = "apps/web/core/components/navigation/customize-navigation-dialog.tsx";
const nativeForms = observeNativeForms([sourcePath]);
const stubs = {
  "@tanstack/react-router": `export const useParams=()=>({workspaceSlug:globalThis.navigationWorkspaceSlug});`,
  "lucide-react": `export const GripVertical=()=>null;export const X=()=>null;export const FileArchive=()=>null;export const FileAudio=()=>null;export const FileCode=()=>null;export const FileImage=()=>null;export const FileSpreadsheet=()=>null;export const FileText=()=>null;export const FileVideo=()=>null;export const Link2=()=>null;export const Mail=()=>null;`,
  "react-icons/fa6": `export const FaChrome=()=>null;export const FaDribbble=()=>null;export const FaFacebook=()=>null;export const FaFigma=()=>null;export const FaGithub=()=>null;export const FaInstagram=()=>null;export const FaLinkedin=()=>null;export const FaTwitter=()=>null;export const FaYoutube=()=>null;`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/button": `const React=globalThis.navigationFormReact;export const Button=({children,loading,...props})=>React.createElement("button",props,children);`,
  "@plane/ui": `const React=globalThis.navigationFormReact;export const EModalPosition={CENTER:"center"};export const EModalWidth={XXL:"xxl"};export const ModalCore=({children,handleClose})=>{globalThis.navigationModalHandleClose=handleClose;return React.createElement("div",{role:"dialog"},children)};export const Checkbox=({checked,onChange})=>React.createElement("input",{type:"checkbox",checked,onChange});export const Sortable=({data,render})=>React.createElement(React.Fragment,null,data.map(item=>React.createElement(React.Fragment,{key:item.key},render(item))));`,
  "@/services/user.service":
    "export class UserService { currentUser() { throw new Error('unexpected identity request'); } } export default new UserService();",
  "@/services/workspace.service":
    "export class WorkspaceService { workspaceMemberMe() { throw new Error('unexpected membership request'); } getWorkspaceUserProjectsRole() { throw new Error('unexpected roles request'); } }",
  "@/hooks/use-navigation-preferences": `export const usePersonalNavigationPreferences=()=>({preferences:{items:{stickies:{enabled:false,sort_order:0},your_work:{enabled:true,sort_order:1},drafts:{enabled:true,sort_order:2}}},togglePersonalItem:()=>{},updatePersonalItemOrder:()=>{}});export const useProjectNavigationPreferences=()=>({preferences:globalThis.navigationProjectPreferences,updateNavigationMode:mode=>{globalThis.navigationProjectPreferences.navigationMode=mode;},updateShowLimitedProjects:show=>{globalThis.navigationProjectPreferences.showLimitedProjects=show;},updateLimitedProjectsCount:async(count)=>{const state=globalThis.navigationProjectPreferences;state.savedCounts.push(count);state.onSaveStarted?.();if(state.deferredSave)await state.deferredSave;if(state.failSave)throw new Error("offline");state.limitedProjectsCount=count;}});export const useWorkspaceNavigationPreferences=()=>({preferences:{items:{}},toggleWorkspaceItem:()=>{},updateWorkspaceItemOrder:()=>{}});`,
  "@/components/workspace/sidebar/helper": `export const getSidebarNavigationItemIcon=()=>null;`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

async function createPermissionClient(workspaceSlugs) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await Promise.all(
    workspaceSlugs.map((workspaceSlug) => seedPermissionQueries(client, { userId: "alice", workspaceSlug }))
  );
  return client;
}

const withPermissions = (client, element) => React.createElement(QueryClientProvider, { client }, element);

await test("project-count draft validates, survives a failed save, and reopens at the saved value", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = await createPermissionClient(["acme", "beta"]);
  t.after(async () => {
    await client.cancelQueries();
    client.clear();
  });
  const { CustomizeNavigationDialog } =
    await import("../../apps/web/core/components/navigation/customize-navigation-dialog.tsx");
  const prefs = globalThis.navigationProjectPreferences;
  const render = (isOpen) =>
    environment.render(
      withPermissions(client, React.createElement(CustomizeNavigationDialog, { isOpen, onClose: () => {} }))
    );

  await render(true);
  const mode = environment.container.querySelector('input[type="radio"][value="TABBED"]');
  await act(async () => mode.click());
  assert.equal(prefs.navigationMode, "TABBED", "navigation mode remains an immediate preference action");
  assert.deepEqual(prefs.savedCounts, []);
  const input = environment.container.querySelector('[name="projectCount"]');
  await changeFormControl(input, "");
  await act(async () => input.dispatchEvent(new FocusEvent("focusout", { bubbles: true })));
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /whole number of at least 1/);
  assert.deepEqual(prefs.savedCounts, []);

  await changeFormControl(input, "8");
  await act(async () => input.dispatchEvent(new FocusEvent("focusout", { bubbles: true })));
  const formApi = nativeForms.getForms(sourcePath).at(-1);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(input.value, "8");
  assert.equal(prefs.limitedProjectsCount, 4);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Could not save the project limit/);
  const retry = [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Retry");
  assert.ok(retry);
  assert.equal(retry.disabled, false, "a failed save remains retryable through the visible control");

  prefs.failSave = false;
  await act(async () => retry.click());
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.equal(prefs.limitedProjectsCount, 8);
  assert.deepEqual(prefs.savedCounts, [8, 8]);

  await render(false);
  await render(true);
  assert.equal(environment.container.querySelector('[name="projectCount"]').value, "8");
  await changeFormControl(environment.container.querySelector('[name="projectCount"]'), "11");
  globalThis.navigationWorkspaceSlug = "beta";
  globalThis.navigationProjectPreferences = {
    navigationMode: "ACCORDION",
    limitedProjectsCount: 2,
    showLimitedProjects: true,
    failSave: false,
    savedCounts: [],
  };
  await render(true);
  assert.equal(
    environment.container.querySelector('[name="projectCount"]').value,
    "2",
    "switching workspaces discards the previous workspace's unsaved count draft"
  );
});

await test("project-count save blocks dialog dismissal until persistence settles", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = await createPermissionClient(["acme"]);
  t.after(async () => {
    await client.cancelQueries();
    client.clear();
  });
  const prefs = globalThis.navigationProjectPreferences;
  prefs.failSave = false;
  prefs.savedCounts = [];
  prefs.limitedProjectsCount = 4;
  let resolveSave;
  let notifySaveStarted;
  const saveStarted = new Promise((resolve) => {
    notifySaveStarted = resolve;
  });
  prefs.deferredSave = new Promise((resolve) => {
    resolveSave = resolve;
  });
  prefs.onSaveStarted = notifySaveStarted;
  const closeCalls = [];
  const { CustomizeNavigationDialog } =
    await import("../../apps/web/core/components/navigation/customize-navigation-dialog.tsx");
  await environment.render(
    withPermissions(
      client,
      React.createElement(CustomizeNavigationDialog, { isOpen: true, onClose: () => closeCalls.push("closed") })
    )
  );

  const input = environment.container.querySelector('[name="projectCount"]');
  await changeFormControl(input, "9");
  await act(async () => {
    input.dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
    await saveStarted;
  });

  const closeButton = environment.container.querySelector('button[aria-label="close"]');
  assert.equal(closeButton.disabled, true);
  await act(async () => {
    globalThis.navigationModalHandleClose();
    closeButton.click();
  });
  assert.deepEqual(closeCalls, [], "neither modal dismissal path can discard an in-flight save");

  await act(async () => resolveSave());
  assert.equal(closeButton.disabled, false);
  await act(async () => globalThis.navigationModalHandleClose());
  assert.deepEqual(closeCalls, ["closed"], "dialog can close after persistence settles");
  delete prefs.deferredSave;
  delete prefs.onSaveStarted;
});

nativeForms.restore();
