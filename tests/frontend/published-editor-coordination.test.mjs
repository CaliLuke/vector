import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedWorkspaceQuery } from "./helpers/workspace-query-fixture.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.editorCoordinationReact = React;
const stubs = {
  "@tanstack/react-router": "export const useParams=()=>({workspaceSlug:'workspace',projectId:'project'});",
  "./provider": "export const IssueModalProvider=({children})=>children;",
  "./base":
    "const R=globalThis.editorCoordinationReact;export const CreateUpdateIssueModalBase=({modalTitle,onClose})=>R.createElement('section',{'aria-label':modalTitle},R.createElement('button',{onClick:onClose},'Close '+modalTitle));",
  "@plane/ui":
    "const R=globalThis.editorCoordinationReact;export const EModalPosition={TOP:'top'};export const EModalWidth={XXXXL:'wide'};export const ModalCore=({children})=>R.createElement('aside',{},children);",
};
registerHooks({
  resolve(specifier, context, next) {
    return context.parentURL?.endsWith("/issue-modal/modal.tsx")
      ? resolvePackageStub(specifier, stubs, context, next)
      : next(specifier, context);
  },
});
const { CreateUpdateIssueModal } = await import("../../apps/web/core/components/issues/issue-modal/modal.tsx");
const { useItemTextEditor } = await import("../../apps/web/core/components/issues/use-item-edit-coordination.ts");
async function fixture(t) {
  const env = await createReactDomTestEnvironment(),
    client = new QueryClient();
  await seedWorkspaceQuery(client);
  let editor;
  function ExistingText() {
    const current = useItemTextEditor("workspace/project/item", () => {});
    React.useLayoutEffect(() => {
      editor = current;
    });
    return React.createElement("output", {}, current.locked ? "locked" : "editable");
  }
  const render = (modals = []) =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(ExistingText),
        ...modals.map((props) =>
          React.createElement(CreateUpdateIssueModal, {
            isOpen: true,
            data: { id: "item", project_id: "project" },
            onClose: () => {},
            ...props,
          })
        )
      )
    );
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  await render();
  return { ...env, render, editor: () => editor };
}
await test("all published modal entries wait for unsaved text before mounting the source form", async (t) => {
  const f = await fixture(t);
  await act(async () => f.editor().report("submitting"));
  await f.render([{ key: "edit", modalTitle: "Published update" }]);
  assert.equal(f.container.querySelector("section"), null);
  assert.match(f.container.textContent, /Waiting for title/);
  await act(async () => f.editor().report("error"));
  assert.equal(f.container.querySelector("section"), null);
  await act(async () => f.editor().report("submitted"));
  assert.ok(f.container.querySelector('[aria-label="Published update"]'));
  assert.equal(f.container.querySelector("output").textContent, "locked");
  await f.render();
  assert.equal(f.container.querySelector("output").textContent, "editable");
});
await test("two modal entries cannot own the same published item at once", async (t) => {
  const f = await fixture(t);
  await f.render([
    { key: "first", modalTitle: "First" },
    { key: "second", modalTitle: "Second" },
  ]);
  assert.equal(f.container.querySelectorAll("section").length, 1);
  assert.ok(f.container.querySelector('[aria-label="First"]'));
  await f.render([{ key: "second", modalTitle: "Second" }]);
  assert.equal(f.container.querySelectorAll("section").length, 1);
  assert.ok(f.container.querySelector('[aria-label="Second"]'));
});
await test("creation and private drafts do not wait for published-item autosaves", async (t) => {
  const f = await fixture(t);
  await act(async () => f.editor().report("submitting"));
  await f.render([
    { key: "create", modalTitle: "Create", data: { project_id: "project" } },
    { key: "draft", modalTitle: "Draft", isDraft: true },
  ]);
  assert.ok(f.container.querySelector('[aria-label="Create"]'));
  assert.ok(f.container.querySelector('[aria-label="Draft"]'));
  assert.equal(f.container.querySelector("output").textContent, "editable");
});
