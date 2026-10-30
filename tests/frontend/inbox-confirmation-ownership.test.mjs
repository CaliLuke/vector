import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const state = { React, writes: [], navigations: [], toasts: [] };
globalThis.inboxConfirmationTest = state;
const context = "globalThis.inboxConfirmationTest";
const empty = (name) => `export const ${name}=()=>null;`;
registerWebTestModules(
  {
    "@tanstack/react-router": `export const useRouter=()=>({navigate:(to)=>{${context}.navigations.push(to);return Promise.resolve();}});`,
    "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
    "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=toast=>${context}.toasts.push(toast);`,
    "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:${context}.user});`,
    "@/hooks/use-permissions": `export const useProjectPermissions=()=>({can:()=>Boolean(${context}.user),projectRole:20});`,
    "@/hooks/use-projects": "export const useProjectDetail=()=>({data:{id:'project',identifier:'TEST'}});",
    "@/hooks/use-intake": `export const useIntakeDetail=()=>({data:${context}.detail});export const useIntakeMutation=()=>({mutateAsync:input=>{${context}.writes.push(input);return ${context}.save(input);}});`,
    "@/components/issues/issue-modal/modal": empty("CreateUpdateIssueModal"),
    "@/components/issues/issue-update-status": empty("NameDescriptionUpdateStatus"),
    "@/components/issues/work-item-selection-modal": empty("WorkItemSelectionModal"),
    "../inbox-issue-status": empty("InboxIssueStatus"),
    "../modals/snooze-issue-modal": empty("InboxIssueSnoozeModal"),
    "./issue-root": `export const InboxIssueMainContent=props=>${context}.React.createElement("output",{"data-editable":String(props.isEditable)});`,
    "./inbox-issue-mobile-header": empty("InboxIssueActionsMobileHeader"),
    "./intake-action-controls": `export const IntakeDesktopActions=({actions})=>${context}.React.createElement('div',null,${context}.React.createElement('button',{'data-action':'decline',onClick:actions.decline},'Decline'),${context}.React.createElement('button',{'data-action':'delete',onClick:actions.remove},'Delete'));`,
  },
  (specifier, moduleContext, nextResolve) => {
    if (specifier === "@plane/ui" && moduleContext.parentURL?.endsWith("/modals/issue-confirmation.tsx")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(`const React=${context}.React;export const AlertModalCore=props=>React.createElement('div',{role:'dialog'},props.content,React.createElement('button',{'data-action':'cancel',onClick:props.handleClose},'Cancel'),React.createElement('button',{'data-action':'confirm',disabled:props.isSubmitting,onClick:props.handleSubmit},'Confirm'));`)}`,
      };
    }
    return nextResolve(specifier, moduleContext);
  }
);
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { InboxContentRoot } = await import("../../apps/web/core/components/inbox/content/root.tsx");

const item = (id) => ({
  id: `intake-${id}`,
  status: -2,
  issue: { id, project_id: "project", sequence_id: 1, created_by: "alice" },
});
async function setup(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity, retry: false } } });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  state.user = { id: "alice" };
  state.detail = item("first");
  state.writes = [];
  state.navigations = [];
  state.toasts = [];
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  const render = (inboxIssueId = "first") =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(InboxContentRoot, {
          workspaceSlug: "acme",
          projectId: "project",
          inboxIssueId,
          isMobileSidebar: false,
          setIsMobileSidebar() {},
          navigation: { tab: "open", issueIds: ["first", "second"] },
        })
      )
    );
  const click = async (action) =>
    act(async () => {
      const button = env.container.querySelector(`[data-action="${action}"]`);
      assert.ok(button, `Missing ${action} action`);
      button.click();
    });
  await render();
  return { env, client, render, click };
}

await test("a pending decline cannot dismiss or close and navigate a replacement item", async (t) => {
  const { env, render, click } = await setup(t);
  const response = Promise.withResolvers();
  state.save = () => response.promise;
  await click("decline");
  await click("confirm");
  await click("cancel");
  assert.ok(env.container.querySelector('[role="dialog"]'));
  assert.equal(state.writes.length, 1);
  state.detail = item("second");
  await render("second");
  await click("decline");
  await act(async () => response.resolve());
  assert.ok(env.container.querySelector('[role="dialog"]'));
  assert.equal(env.container.querySelector('[data-action="confirm"]').disabled, false);
  assert.deepEqual(state.navigations, []);
  assert.deepEqual(state.toasts, []);
});

await test("accepted deletion still navigates when its receipt removes the detail header", async (t) => {
  const { render, click } = await setup(t);
  const response = Promise.withResolvers();
  state.save = () => response.promise;
  await click("delete");
  await click("confirm");
  state.detail = undefined;
  await render();
  await act(async () => response.resolve());
  assert.deepEqual(state.navigations, [{ href: "/acme/projects/project/intake" }]);
});

await test("an accepted deletion cannot navigate after its selected item or account session changes", async (t) => {
  const { client, render, click } = await setup(t);
  const response = Promise.withResolvers();
  state.save = () => response.promise;
  await click("delete");
  await click("confirm");
  await act(async () => advanceIdentitySession(client));
  state.detail = item("second");
  await render("second");
  await act(async () => response.resolve());
  assert.deepEqual(state.navigations, []);
  assert.deepEqual(state.toasts, []);
});

await test("a failed deletion retains the confirmation for one successful retry", async (t) => {
  const { env, click } = await setup(t);
  state.save = async () => {
    throw new Error("offline");
  };
  await click("delete");
  await click("confirm");
  assert.ok(env.container.querySelector('[role="alert"]'));
  assert.equal(env.container.querySelector('[data-action="confirm"]').disabled, false);
  assert.deepEqual(state.navigations, []);
  state.save = async () => ({});
  await click("confirm");
  assert.equal(state.writes.length, 2);
  assert.equal(env.container.querySelector('[role="dialog"]'), null);
  assert.deepEqual(state.navigations, [{ href: "/acme/projects/project/intake" }]);
});

await test("missing account and creator identifiers cannot enable intake text editing", async (t) => {
  const { env, render } = await setup(t);
  state.user = undefined;
  state.detail = { ...item("first"), issue: { ...item("first").issue, created_by: undefined } };
  await render();
  assert.equal(env.container.querySelector("output[data-editable]").getAttribute("data-editable"), "false");
});
