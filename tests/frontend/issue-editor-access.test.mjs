import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import path from "node:path";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { createRouterFixture, routerModuleUrl } = await fromRoot("tests/frontend/helpers/router.mjs");
const { createReactDomTestEnvironment, waitForReactCondition } = await fromRoot("tests/frontend/helpers/react-dom.mjs");
const { React, act } = await fromRoot("tests/frontend/helpers/react.mjs");
const { setup, state } = await fromRoot("tests/frontend/helpers/issue-form-harness.mjs");
const { changeFormControl: change } = await fromRoot("tests/frontend/helpers/native-forms.mjs");
const { enableTypeFields } = await fromRoot("tests/frontend/helpers/issue-custom-fields.mjs");
const { resolvePackageStub } = await fromRoot("tests/frontend/helpers/module-stubs.mjs");
const stubs = {
  // The native Query mutation-owner boundary is covered separately.
  "@/hooks/use-issue-list-mutation-owner": "export const useIssueListMutationOwner=()=>()=>undefined;",
  "@/hooks/context/use-issue-modal":
    "export const useIssueModal=()=>({allowedProjectIds:['first'],selectedParentIssue:null});",
  "@/hooks/store/use-issue-detail":
    "export const useIssueDetail=()=>({fetchIssue:async()=>{},rootIssueStore:{issues:{addIssue:()=>{}}}});",
  "@/hooks/use-workspace-drafts":
    "export const useWorkspaceDraftMutations=()=>({createDraft:async()=>undefined,acceptDraftReceipt:()=>{},completePublication:async()=>{}});",
  "@/hooks/use-issue-list-type": "export const useIssueListType=()=> 'PROJECT';",
  "@/hooks/use-issues-actions": "export const useIssuesActions=()=>({});",
  "@plane/ui":
    "const R=globalThis.issueFormReact;export const EModalPosition={TOP:'top'};export const EModalWidth={XXXXL:'wide'};export const ModalCore=({children,isOpen})=>isOpen?R.createElement('section',{'aria-label':'Work item editor'},children):null;",
  "./item-edit-confirmation":
    "const R=globalThis.issueFormReact;export const ItemEditRecovery=()=>null;export const ItemEditConfirmation=({review,onDecide})=>R.createElement('section',{'aria-label':'Preservation review'},review.description_html,R.createElement('button',{type:'button',onClick:()=>onDecide(true)},'Confirm save'));",
};
registerHooks({
  resolve(specifier, context, next) {
    if (specifier === "@tanstack/react-router") return next(routerModuleUrl, context);
    return context.parentURL?.endsWith("/issue-modal/base.tsx") ||
      context.parentURL?.endsWith("/use-item-edit-reconciliation.ts")
      ? resolvePackageStub(specifier, stubs, context, next)
      : next(specifier, context);
  },
});
// Initialize browser feature detection before the modal imports React DOM.
const bootstrapEnvironment = await createReactDomTestEnvironment();
const { CreateUpdateIssueModalBase } = await fromRoot("apps/web/core/components/issues/issue-modal/base.tsx");
const { workItemEditsService } = await fromRoot("apps/web/core/services/work-item-edits.service.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
await bootstrapEnvironment.unmount();
async function verifyPermissionLoss(timing) {
  await test(`actual editor hides preserved description after permission loss ${timing}`, async (t) => {
    const env = await setup(t);
    enableTypeFields([]);
    const schema = state.schema;
    state.schema = async (input) => ({ ...(await schema(input)), revision: 4 });
    const response = Promise.withResolvers();
    let commits = 0,
      previews = 0;
    t.mock.method(workItemEditsService, "source", async () => ({
      item: { id: "existing", project_id: "first", type_id: "task", name: "Saved", description_html: "<p>Saved</p>" },
      source_version: "v1",
    }));
    t.mock.method(workItemEditsService, "preview", () => {
      previews++;
      return response.promise;
    });
    t.mock.method(workItemEditsService, "commit", async () => {
      commits++;
      return { id: "existing", project_id: "first" };
    });
    const router = createRouterFixture("/workspace/projects/first/issues/");
    await router.render(
      env,
      React.createElement(CreateUpdateIssueModalBase, {
        isOpen: true,
        onClose: () => {},
        data: { id: "existing", project_id: "first" },
        withDraftIssueWrapper: false,
      })
    );
    await change(env.container.querySelector('[name="name"]'), "Local");
    // Start submission without awaiting the intentional review/confirmation boundary.
    await act(async () => {
      env.container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    });
    assert.equal(previews, 1);
    if (timing === "during confirmation") {
      await act(async () =>
        response.resolve({ id: "review", preserved_properties: 1, description_html: "Private preserved values" })
      );
      assert.match(env.container.textContent, /Private preserved values/);
    }
    await act(async () => {
      env.client.setQueryData(permissionKeys.workspaceMembership("workspace-query-test-user", "workspace"), {
        is_active: false,
        role: 20,
        updated_at: "v2",
      });
      if (timing === "during preview")
        response.resolve({ id: "review", preserved_properties: 1, description_html: "Private preserved values" });
    });
    assert.doesNotMatch(env.container.textContent, /Private preserved values/);
    assert.equal(env.container.querySelector('[aria-label="Preservation review"]'), null);
    assert.equal(commits, 0);
  });
}

await verifyPermissionLoss("during preview");
await verifyPermissionLoss("during confirmation");

await test("an accepted existing-item edit submits its source version and acknowledges its published receipt once", async (t) => {
  const env = await setup(t);
  enableTypeFields([]);
  const schema = state.schema;
  state.schema = async (input) => ({ ...(await schema(input)), revision: 4 });
  const commits = [],
    submitted = [];
  let closed = 0;
  t.mock.method(workItemEditsService, "source", async () => ({
    item: { id: "existing", project_id: "first", type_id: "task", name: "Saved", description_html: "<p>Saved</p>" },
    source_version: "v1",
  }));
  t.mock.method(workItemEditsService, "preview", async (_workspace, _command, beforeSend) => {
    beforeSend();
    return { id: "review", preserved_properties: 0, description_html: "" };
  });
  t.mock.method(workItemEditsService, "commit", async (_workspace, command, beforeSend) => {
    beforeSend();
    commits.push(command);
    return { id: "existing", project_id: "first", sequence_id: 1, name: "Local" };
  });
  const router = createRouterFixture("/workspace/projects/first/issues/");
  await router.render(
    env,
    React.createElement(CreateUpdateIssueModalBase, {
      isOpen: true,
      onClose: () => {
        closed++;
      },
      onSubmit: (receipt) => {
        submitted.push(receipt);
      },
      data: { id: "existing", project_id: "first" },
      withDraftIssueWrapper: false,
    })
  );
  await change(env.container.querySelector('[name="name"]'), "Local");
  await act(async () =>
    env.container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await waitForReactCondition(() => closed === 1, "an accepted edit should complete its close callback");
  assert.equal(commits.length, 1);
  assert.equal(commits[0].expected_source_version, "v1");
  assert.equal(commits[0].changes.name, "Local");
  assert.equal(submitted.length, 1);
  assert.equal(submitted[0].id, "existing");
});
