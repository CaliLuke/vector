import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { fileURLToPath, pathToFileURL } from "node:url";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { settleQuery } from "./helpers/query-router.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const source = fileURLToPath(
  import.meta.resolve("../../apps/web/core/components/core/modals/bulk-delete-issues-modal.tsx")
);
const forms = observeNativeForms([source]);
const reactUrl = pathToFileURL(webRequire.resolve("react")).href;
const state = { route: {}, actions: {}, toasts: [] };
globalThis.bulkDeleteFormState = state;
const stubs = {
  "@tanstack/react-router": "export const useParams = () => globalThis.bulkDeleteFormState.route;",
  "@/hooks/use-issues-actions": "export const useIssuesActions = () => globalThis.bulkDeleteFormState.actions;",
  "@plane/propel/toast":
    "export const TOAST_TYPE={SUCCESS:'success'}; export const setToast = value => globalThis.bulkDeleteFormState.toasts.push(value);",
  "@plane/i18n": "export const useTranslation = () => ({t:key=>key});",
  "@plane/ui": `import React from ${JSON.stringify(reactUrl)};
    export const EModalPosition={CENTER:'center'}, EModalWidth={XXL:'xxl'};
    export const ModalCore=({children,handleClose})=>React.createElement('section',{role:'dialog'},children,React.createElement('button',{onClick:handleClose},'Dismiss'));
    export const Loader=({children})=>React.createElement('div',null,children); Loader.Item=()=>null;`,
  "@/components/empty-state/simple-empty-state-root": `import React from ${JSON.stringify(reactUrl)}; export const SimpleEmptyState=()=>React.createElement('p',null,'No matches');`,
  "@/components/issues/issue-detail/issue-identifier": "export const IssueIdentifier=()=>null;",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.endsWith(".webp?url"))
      return { shortCircuit: true, url: 'data:text/javascript,export default "image";' };
    if (stubs[specifier])
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    return nextResolve(specifier, context);
  },
});
const { BulkDeleteIssuesModal } =
  await import("../../apps/web/core/components/core/modals/bulk-delete-issues-modal.tsx");
const { ProjectService } = await import("../../apps/web/core/services/project/index.ts");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const row = { id: "issue", name: "Selected task", project_id: "project", project__identifier: "PRJ", sequence_id: 1 };
async function fixture(t, removeIssues = async () => {}, fetch = async () => [row]) {
  const query = await createQueryDomFixture(t, { defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const { env, client } = query;
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(client, { userId: "alice", workspaceSlug: "acme", projectRoles: { project: 20 } });
  state.route = { workspaceSlug: "acme", projectId: "project" };
  state.actions = { removeIssues: t.mock.fn(removeIssues) };
  state.toasts = [];
  const requests = t.mock.method(ProjectService.prototype, "projectIssuesSearch", fetch);
  const onClose = t.mock.fn();
  const render = async (isOpen = true) => {
    await query.render(React.createElement(BulkDeleteIssuesModal, { isOpen, onClose }));
    await settleQuery();
  };
  await render();
  const button = (name) => [...env.container.querySelectorAll("button")].find((node) => node.textContent === name);
  return {
    env,
    client,
    render,
    button,
    onClose,
    requests,
    actions: state.actions,
    async select() {
      await act(async () => {
        const option = env.container.querySelector('[role="option"]');
        option.dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true }));
        option.dispatchEvent(new MouseEvent("mouseup", { button: 0, bubbles: true }));
        option.click();
      });
    },
    async submit() {
      await act(async () => button("Delete selected work items").click());
    },
    async search(value) {
      await changeFormControl(env.container.querySelector('input[type="text"]'), value);
      await act(async () => new Promise((resolve) => setTimeout(resolve, 550)));
      await settleQuery();
    },
  };
}

await test("bulk delete retains selection and unsuccessful form state after rejection, then retries once", async (t) => {
  const action = t.mock.fn(async () => {});
  action.mock.mockImplementationOnce(async () => {
    throw new Error("offline");
  });
  const f = await fixture(t, action);
  await f.select();
  await f.submit();
  assert.equal(f.onClose.mock.callCount(), 0);
  assert.equal(f.env.container.querySelector('input[type="checkbox"]').checked, true);
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /try again/i);
  assert.equal(forms.getForms(source).at(-1).state.isSubmitSuccessful, false);
  await f.submit();
  assert.equal(action.mock.callCount(), 2);
  assert.deepEqual(action.mock.calls[1].arguments, ["project", ["issue"]]);
  assert.equal(f.onClose.mock.callCount(), 1);
  assert.equal(state.toasts.length, 1);
  assert.equal(forms.getForms(source).at(-1).state.isSubmitSuccessful, true);
});

await test("pending bulk deletion blocks dismissal and duplicate submits; old dialog completion cannot close its replacement", async (t) => {
  const pending = Promise.withResolvers();
  const f = await fixture(t, () => pending.promise);
  await f.select();
  await f.submit();
  await act(async () => f.button("Dismiss").click());
  await act(async () =>
    f.env.container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  assert.equal(f.actions.removeIssues.mock.callCount(), 1);
  assert.equal(f.onClose.mock.callCount(), 0);
  await f.render(false);
  await f.render();
  assert.equal(f.env.container.querySelector('input[type="checkbox"]').checked, false);
  await act(async () => pending.resolve());
  assert.equal(f.onClose.mock.callCount(), 0);
  assert.equal(state.toasts.length, 0);
});

await test("session replacement clears bulk selection and suppresses pending success feedback", async (t) => {
  const pending = Promise.withResolvers();
  const f = await fixture(t, () => pending.promise);
  await f.select();
  await f.submit();
  await act(async () => advanceIdentitySession(f.client));
  await settleQuery();
  assert.equal(f.env.container.querySelector('input[type="checkbox"]').checked, false);
  await act(async () => pending.resolve());
  assert.equal(f.onClose.mock.callCount(), 0);
  assert.equal(state.toasts.length, 0);
});

await test("empty results retain the selected deletion footer and search failures retry without losing selection", async (t) => {
  const f = await fixture(
    t,
    async () => {},
    async (_workspace, _project, params) => {
      if (params.search === "empty") return [];
      if (params.search === "broken") throw new Error("offline");
      return [row];
    }
  );
  await f.select();
  await f.search("empty");
  assert.equal(f.button("Delete selected work items").disabled, false);
  await f.search("broken");
  assert.equal(f.button("Delete selected work items").disabled, true);
  f.requests.mock.mockImplementation(async () => [row]);
  await act(async () => f.button("Retry search").click());
  await settleQuery();
  assert.equal(f.env.container.querySelector('input[type="checkbox"]').checked, true);
  await f.submit();
  assert.deepEqual(f.actions.removeIssues.mock.calls[0].arguments, ["project", ["issue"]]);
});
forms.restore();
