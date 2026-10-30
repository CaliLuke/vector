import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.typeRemovalDialogReact = React;
// The dialog shell needs browser layout; keep its real fields and native form.
const controlsURL = new URL("../../apps/web/core/components/work-item-types/controls.tsx", import.meta.url).href;
const stubs = {
  "./controls": `export { Field, inputStyle } from ${JSON.stringify(controlsURL)};
export function PreviewDialog(props) { return globalThis.typeRemovalDialogReact.createElement("section", {"aria-label":props.title}, props.children, props.footer); }`,
};
registerHooks({
  resolve: (specifier, context, next) =>
    context.parentURL?.endsWith("/type-removal-dialog.tsx")
      ? resolvePackageStub(specifier, stubs, context, next)
      : next(specifier, context),
});

const { WorkspaceTypeDeletion } =
  await import("../../apps/web/core/components/work-item-types/workspace-type-deletion.tsx");
const { workItemTypeRemovalService } = await import("../../apps/web/core/services/work-item-type-removal.service.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const button = (name) => [...document.querySelectorAll("button")].find((node) => node.textContent === name);
const impact = { id: "review", items: 3, drafts: 2, projects: 1, preserved: 4, invalid_items: 0, blocked_reason: "" };
async function screen(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(identityKeys.currentUser(), { id: "owner" });
  const original = Object.getOwnPropertyDescriptors(Object.getPrototypeOf(workItemTypeRemovalService));
  const commits = [];
  let closes = 0;
  let types = [
    { id: "bug", name: "Bug", active: true, usage: 3 },
    { id: "task", name: "Task", active: true, usage: 0 },
  ];
  workItemTypeRemovalService.preview = async () => impact;
  workItemTypeRemovalService.commit = async (_slug, command) => {
    commits.push(command);
  };
  const render = (show) =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        show
          ? React.createElement(WorkspaceTypeDeletion, {
              type: { id: "bug", name: "Bug" },
              persistence: {
                types,
                propertyRemoval: { scope: "acme", refreshRevision: async () => 4 },
                remove: () => {
                  throw new Error("Used types need a reviewed replacement");
                },
              },
              revision: 3,
              onClose: () => {
                closes++;
              },
            })
          : null
      )
    );
  t.after(async () => {
    Object.defineProperties(workItemTypeRemovalService, { preview: original.preview, commit: original.commit });
    await env.unmount(() => client.clear());
  });
  await render(true);
  return {
    env,
    client,
    commits,
    render,
    setTypes: async (next) => {
      types = next;
      await render(true);
    },
    get closes() {
      return closes;
    },
  };
}
await test("deletion requires impact review before the confirmed write", async (t) => {
  const f = await screen(t);
  assert.equal(button("Review deletion").disabled, true);
  await changeFormControl(document.querySelector("select"), "task");
  await submitForm(document.body);
  assert.match(document.body.textContent, /3 work items and 2 drafts/);
  assert.equal(f.commits.length, 0);
  assert.equal(button("Delete type").disabled, false);
  await submitForm(document.body);
  assert.equal(f.commits.length, 1);
  assert.equal(f.commits[0].review_id, "review");
  assert.equal(f.closes, 1, document.body.textContent);
});
await test("reopened safe rejection keeps replacement through refreshed review", async (t) => {
  const f = await screen(t);
  workItemTypeRemovalService.commit = async () => {
    throw { response: { status: 409, data: { review_again: true, message: "Refresh required" } } };
  };
  await changeFormControl(document.querySelector("select"), "task");
  await submitForm(document.body);
  await submitForm(document.body);
  await f.render(false);
  await f.render(true);
  assert.equal(document.querySelector("select").value, "task");
  assert.ok(button("Refresh review"));
  await act(async () => {
    button("Refresh review").click();
  });
  await submitForm(document.body);
  assert.equal(document.querySelector("select").value, "task");
  assert.equal(button("Delete type").disabled, false);
  assert.equal(document.querySelector('[role="alert"]'), null);
});
await test("blocked replacement never exposes an enabled deletion action", async (t) => {
  const f = await screen(t);
  workItemTypeRemovalService.preview = async () => ({
    ...impact,
    id: "",
    blocked_reason: "Required values need repair",
    invalid_items: 2,
  });
  await changeFormControl(document.querySelector("select"), "task");
  await submitForm(document.body);
  assert.match(document.body.textContent, /Required values need repair/);
  assert.equal(button("Review deletion").disabled, true);
  assert.equal(button("Delete type"), undefined);
  await submitForm(document.body);
  assert.equal(f.commits.length, 0);
});

await test("successful catalog refresh keeps the used-type flow mounted until acknowledgement", async (t) => {
  const f = await screen(t);
  let finish;
  const response = new Promise((resolve) => {
    finish = resolve;
  });
  workItemTypeRemovalService.commit = async () => response;
  await changeFormControl(document.querySelector("select"), "task");
  await submitForm(document.body);
  await submitForm(document.body);
  await f.setTypes([{ id: "task", name: "Task", active: true, usage: 5 }]);
  await act(async () => {
    finish();
  });
  assert.equal(f.closes, 1);
  assert.doesNotMatch(document.body.textContent, /unused type/);
});
