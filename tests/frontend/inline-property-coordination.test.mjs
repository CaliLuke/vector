import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedWorkspaceQuery } from "./helpers/workspace-query-fixture.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.inlineCoordinationReact = React;
const stubs = {
  "./use-published-item-edit": `export function usePublishedItemEdit(){const R=globalThis.inlineCoordinationReact;R.useEffect(()=>{globalThis.inlineCoordinationFixture.opened++;},[]);return {source:{},edits:{}};}`,
  "./inline-property-form": `export const InlinePropertyForm=()=>null;`,
  "@/components/issues/issue-modal/item-edit-confirmation": `export const ItemEditConfirmation=()=>null;export const ItemEditRecovery=()=>null;`,
  "@plane/propel/button": `export const Button=({variant,...props})=>globalThis.inlineCoordinationReact.createElement("button",props);`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { InlinePropertyEditor } =
  await import("../../apps/web/core/components/work-item-types/inline-property-editor.tsx");
const { useItemTextEditor } = await import("../../apps/web/core/components/issues/use-item-edit-coordination.ts");
async function fixture(t) {
  const { env, client } = await createQueryDomFixture(t, {});
  await seedWorkspaceQuery(client);
  const f = { env, client, opened: 0, entries: [] };
  globalThis.inlineCoordinationFixture = f;
  function Text() {
    const editor = useItemTextEditor("workspace/project/item", () => {});
    React.useLayoutEffect(() => {
      f.editor = editor;
    });
    return React.createElement("output", null, editor.locked ? "locked" : "editable");
  }
  f.render = () =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(Text),
        ...f.entries.map((id) =>
          React.createElement(
            "div",
            { key: id, "data-editor": id },
            React.createElement(InlinePropertyEditor, {
              scope: { workspaceSlug: "workspace", projectId: "project", issueId: "item" },
              onClose: () => {
                f.entries = f.entries.filter((x) => x !== id);
                void f.render();
              },
              onSaved: () => {},
              onAccessLost: () => {},
            })
          )
        )
      )
    );
  await f.render();
  return f;
}
await test("inline editor waits for failed title autosaves and loads its source only after they finish", async (t) => {
  const f = await fixture(t);
  await act(async () => f.editor.report("error"));
  f.entries = ["full"];
  await f.render();
  assert.equal(f.opened, 0);
  assert.match(f.env.container.textContent, /Waiting for other edits/);
  await act(async () => f.editor.report("submitted"));
  assert.equal(f.opened, 1);
  assert.equal(f.editor.canEdit(), false);
  f.entries = [];
  await f.render();
  assert.equal(f.editor.canEdit(), true);
});
await test("canceling an initial load releases the lease to a waiting peek editor", async (t) => {
  const f = await fixture(t);
  f.entries = ["full", "peek"];
  await f.render();
  assert.equal(f.opened, 1);
  assert.match(f.env.container.querySelector('[data-editor="full"]').textContent, /Loading current values/);
  assert.match(f.env.container.querySelector('[data-editor="peek"]').textContent, /Waiting for other edits/);
  await act(async () => f.env.container.querySelector('[data-editor="full"] button').click());
  assert.equal(f.opened, 2);
  assert.equal(f.env.container.querySelector('[data-editor="full"]'), null);
  assert.match(f.env.container.querySelector('[data-editor="peek"]').textContent, /Loading current values/);
});
