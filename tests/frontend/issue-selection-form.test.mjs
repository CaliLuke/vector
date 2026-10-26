import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { pathToFileURL, fileURLToPath } from "node:url";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { settleQuery } from "./helpers/query-router.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const formSource = fileURLToPath(
  import.meta.resolve("../../apps/web/core/components/core/modals/existing-issues-list-modal.tsx")
);
const formObserver = observeNativeForms([formSource]);
const reactUrl = pathToFileURL(webRequire.resolve("react")).href;
const stubs = {
  "@plane/ui": `import React from ${JSON.stringify(reactUrl)};
    export const EModalPosition = {CENTER:'center'}; export const EModalWidth = {XXL:'xxl'};
    export function ModalCore({children,handleClose}) { return React.createElement('section',{role:'dialog'},children,React.createElement('button',{onClick:handleClose},'Dismiss')); }
    export const Loader = ({children}) => React.createElement('div',{'aria-label':'Searching'},children);
    Loader.Item = () => React.createElement('span');
    export const ToggleSwitch = ({value,onChange}) => React.createElement('button',{role:'switch','aria-checked':value,onClick:onChange},'Workspace search');`,
  "@plane/i18n": `export const useTranslation = () => ({t: key => key});`,
  "@/hooks/use-platform-os": `export const usePlatformOS = () => ({isMobile:false});`,
  "@/components/issues/issue-detail/issue-identifier": `import React from ${JSON.stringify(reactUrl)}; export const IssueIdentifier = ({issueSequenceId}) => React.createElement('span',null,'ISSUE-'+issueSequenceId);`,
  "./issue-search-modal-empty-state": `import React from ${JSON.stringify(reactUrl)}; export const IssueSearchModalEmptyState = () => React.createElement('p',null,'No matches');`,
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (stubs[specifier])
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    return nextResolve(specifier, context);
  },
});
const { ExistingIssuesListModal } =
  await import("../../apps/web/core/components/core/modals/existing-issues-list-modal.tsx");
const { ProjectService } = await import("../../apps/web/core/services/project/index.ts");
const issue = (id, name) => ({
  id,
  name,
  project_id: "project",
  project__identifier: "TEST",
  sequence_id: id,
  state__color: "#fff",
  type_id: "type",
});
const alpha = issue("1", "Alpha task");
const beta = issue("2", "Beta task");

function canonicalSearchRow(row, projectId, workspaceSearch) {
  if (workspaceSearch || row.project_id === projectId) return row;
  return { ...row, project_id: projectId };
}

async function fixture(t, props = {}, fetch = async () => [alpha, beta]) {
  const query = await createQueryDomFixture(t, {
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: 3, gcTime: Infinity } },
  });
  const { env, client } = query;
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "acme",
    projectRoles: { project: 20, other: 20 },
  });
  const requests = t.mock.method(ProjectService.prototype, "projectIssuesSearch", async (...args) =>
    (await fetch(...args)).map((row) => canonicalSearchRow(row, args[1], args[2].workspace_search))
  );
  const handleOnSubmit = t.mock.fn(async () => {});
  const handleClose = t.mock.fn();
  const base = {
    workspaceSlug: "acme",
    projectId: "project",
    isOpen: true,
    searchParams: { cycle: true },
    handleOnSubmit,
    handleClose,
    ...props,
  };
  const render = async (extra = {}) => {
    await query.render(React.createElement(ExistingIssuesListModal, { ...base, ...extra }));
    await settleQuery();
  };
  await render();
  const button = (name) => [...env.container.querySelectorAll("button")].find((node) => node.textContent === name);
  const option = (name) =>
    [...env.container.querySelectorAll('[role="option"]')].find((node) => node.textContent.includes(name));
  return {
    env,
    client,
    requests,
    handleOnSubmit,
    handleClose,
    render,
    button,
    option,
    select: (name) =>
      act(async () => {
        const row = option(name);
        row.dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true }));
        row.dispatchEvent(new MouseEvent("mouseup", { button: 0, bubbles: true }));
        row.click();
      }),
    submit: () => act(async () => button("issue.select.add_selected").click()),
    async search(text) {
      await changeFormControl(env.container.querySelector('input[role="combobox"]'), text);
      await act(async () => new Promise((resolve) => setTimeout(resolve, 550)));
      await settleQuery();
    },
  };
}

await test("failed selection submissions retain the selection and retry only on request", async (t) => {
  const submit = t.mock.fn(async () => {});
  submit.mock.mockImplementationOnce(async () => {
    throw new Error("offline");
  });
  const f = await fixture(t, { handleOnSubmit: submit });
  await f.select("Alpha task");
  await f.submit();
  assert.equal(submit.mock.callCount(), 1);
  assert.equal(f.handleClose.mock.callCount(), 0);
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /try again/i);
  assert.equal(f.option("Alpha task").getAttribute("aria-selected"), "true");
  assert.equal(formObserver.getForms(formSource).at(-1)?.state.isSubmitSuccessful, false);
  assert.equal(f.button("issue.select.add_selected").disabled, false);
  await f.submit();
  assert.equal(submit.mock.callCount(), 2);
  assert.deepEqual(submit.mock.calls[1].arguments[0], [alpha]);
  assert.equal(f.handleClose.mock.callCount(), 1);
  assert.equal(formObserver.getForms(formSource).at(-1)?.state.isSubmitSuccessful, true);
});

await test("select all uses visible membership and preserves selections from other searches", async (t) => {
  const f = await fixture(t, {}, async (_workspace, _project, params) => (params.search === "beta" ? [beta] : [alpha]));
  await f.select("Alpha task");
  await f.search("beta");
  assert.ok(f.button("issue.select.select_all"));
  await act(async () => f.button("issue.select.select_all").click());
  assert.equal(f.option("Beta task").getAttribute("aria-selected"), "true");
  await act(async () => f.button("issue.select.deselect_all").click());
  await f.submit();
  assert.deepEqual(f.handleOnSubmit.mock.calls[0].arguments[0], [alpha]);
});

await test("pending submission cannot close twice and completion from an old scope cannot close a new dialog", async (t) => {
  const pending = Promise.withResolvers();
  const submit = t.mock.fn(() => pending.promise);
  const f = await fixture(t, { handleOnSubmit: submit });
  await f.select("Alpha task");
  await f.submit();
  await act(async () => f.button("Dismiss").click());
  assert.equal(f.handleClose.mock.callCount(), 0);
  assert.equal(submit.mock.callCount(), 1);
  await f.render({ projectId: "other" });
  assert.equal(f.option("Alpha task").getAttribute("aria-selected"), "false");
  await act(async () => pending.resolve());
  assert.equal(f.handleClose.mock.callCount(), 0);
});

await test("search errors are visible and retry does not lose the search term", async (t) => {
  const f = await fixture(t, {}, async (_workspace, _project, params) => {
    if (params.search === "broken") throw new Error("offline");
    return [alpha];
  });
  await f.search("broken");
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /could not search/i);
  f.requests.mock.mockImplementation(async () => [beta]);
  await act(async () => f.button("Retry search").click());
  await settleQuery();
  assert.ok(f.option("Beta task"));
  assert.equal(f.env.container.querySelector('input[role="combobox"]').value, "broken");
});

await test("switching scope aborts stale searches and reopening starts with no selection", async (t) => {
  const pending = Promise.withResolvers();
  const f = await fixture(t, {}, (_workspace, project) =>
    project === "project" ? pending.promise : Promise.resolve([beta])
  );
  const signal = f.requests.mock.calls[0].arguments[3];
  await f.render({ projectId: "other" });
  assert.equal(signal.aborted, true);
  await act(async () => pending.resolve([alpha]));
  await settleQuery();
  assert.equal(f.option("Alpha task"), undefined);
  await f.select("Beta task");
  await f.render({ projectId: "other", isOpen: false });
  assert.equal(f.env.container.querySelector('[role="dialog"]'), null);
  await f.render({ projectId: "other" });
  assert.equal(f.option("Beta task").getAttribute("aria-selected"), "false");
});

await test("missing project scope makes no request and explains why search is unavailable", async (t) => {
  const f = await fixture(t, { projectId: undefined });
  assert.equal(f.requests.mock.callCount(), 0);
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /workspace and project are required/i);
  assert.equal(f.button("issue.select.add_selected").disabled, true);
});
await test("failed permission verification hides cached results and blocks an existing selection", async (t) => {
  const f = await fixture(t);
  await f.select("Alpha task");
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  const key = permissionKeys.workspaceMembership("alice", "acme");
  const saved = f.client.getQueryData(key);
  await act(async () => {
    await assert.rejects(
      f.client.fetchQuery({
        queryKey: key,
        queryFn: async () => {
          throw new Error("offline");
        },
        retry: false,
      })
    );
  });
  await settleQuery();
  assert.equal(f.option("Alpha task"), undefined);
  assert.equal(f.button("issue.select.add_selected").disabled, true);
  assert.equal(f.handleOnSubmit.mock.callCount(), 0);
  await act(async () => f.client.setQueryData(key, saved));
  await settleQuery();
  assert.equal(f.option("Alpha task").getAttribute("aria-selected"), "true");
  assert.equal(f.button("issue.select.add_selected").disabled, false);
});
formObserver.restore();
