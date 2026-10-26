import { createProjectQueryDomFixture } from "./helpers/query-dom.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { settleQuery } from "./helpers/query-router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.propertyAccessReact = React;
const stubs = {
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>({isPending:false,isError:false,can:()=>true,workspaceRole:15,projectRole:15,membershipQuery:{data:{updated_at:"v1"}}});`,
  "@/hooks/use-work-item-detail": `export const useWorkItemDetail=()=>({issue:{id:"issue",project_id:"p",updated_at:"u1"}});`,
  "@/hooks/use-workspaces": `export const useWorkspaceBySlug=()=>({data:{id:"workspace"}});`,
  "./inline-property-editor": `export const InlinePropertyEditor=props=>globalThis.propertyAccessReact.createElement("button",{onClick:props.onAccessLost},"Deny inline edit");`,
  "./item-property-value": `export const ItemPropertyValue=props=>globalThis.propertyAccessReact.createElement("span",null,props.schema.stored_values[props.property.id]);`,
  "./property-history-browser": `export const HistoryLoadError=()=>null;export const PropertyHistoryBrowser=()=>globalThis.propertyAccessReact.createElement("article",null,"Private revision content");`,
  "./property-history-restore": `export const PropertyHistoryRestore=()=>null;`,
  "./controls": `export const PreviewDialog=props=>globalThis.propertyAccessReact.createElement("section",{role:"dialog"},props.children,props.footer);`,
  "@/components/editor/rich-text": `export const RichTextEditor=()=>null;`,
  "@plane/propel/button": `export const Button=({variant,...props})=>globalThis.propertyAccessReact.createElement("button",props);`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { ItemCustomProperties } =
  await import("../../apps/web/core/components/work-item-types/item-custom-properties.tsx");
const { workItemSchemaService: service } = await import("../../apps/web/core/services/work-item-schema.service.ts");
async function fixture(t) {
  const { env, client } = await createProjectQueryDomFixture(t, 15);
  const f = { env, client, reads: 0, denied: false };
  t.mock.method(service, "schema", async () => {
    f.reads++;
    if (f.denied) throw { response: { status: 403 } };
    return {
      enabled: true,
      properties: [
        { id: "notes", title: "Notes", kind: "Text", configuration: { format: "Single line", active: true } },
      ],
      stored_values: { notes: "Private current content" },
    };
  });
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(
        React.Fragment,
        null,
        React.createElement(
          "div",
          { "data-surface": "full" },
          React.createElement(ItemCustomProperties, { workspaceSlug: "acme", projectId: "p", issueId: "item" })
        ),
        React.createElement(
          "div",
          { "data-surface": "peek" },
          React.createElement(ItemCustomProperties, { workspaceSlug: "acme", projectId: "p", issueId: "item" })
        )
      )
    )
  );
  await settleQuery();
  f.click = async (surface, label) =>
    act(async () =>
      [...env.container.querySelector(`[data-surface="${surface}"]`).querySelectorAll("button")]
        .find((b) => b.textContent.trim() === label)
        .click()
    );
  await f.click("full", "Edit properties");
  await f.click("peek", "Property history");
  assert.match(env.container.textContent, /Private revision content/);
  return f;
}
function assertHidden(f) {
  assert.doesNotMatch(f.env.container.textContent, /Private current content|Private revision content|Deny inline edit/);
  assert.equal(f.env.container.querySelectorAll('[role="alert"]').length, 2);
}
await test("a definite inline denial hides both mounted surfaces and an open sibling history", async (t) => {
  const f = await fixture(t);
  const reads = f.reads;
  await f.click("full", "Deny inline edit");
  await settleQuery();
  assertHidden(f);
  assert.equal(f.reads, reads);
  await f.click("peek", "Retry");
  await settleQuery();
  assert.match(f.env.container.textContent, /Private current content/);
  assert.doesNotMatch(f.env.container.textContent, /Private revision content/);
});
await test("a background schema denial closes an editor and history without a refetch loop", async (t) => {
  const f = await fixture(t);
  f.denied = true;
  const reads = f.reads;
  await act(async () => f.client.invalidateQueries({ queryKey: ["workspace", "acme", "work-item-schema"] }));
  await settleQuery();
  assertHidden(f);
  assert.equal(f.reads, reads + 2);
});

await test("a denial from the history dialog's independent schema read hides both surfaces", async (t) => {
  const f = await fixture(t);
  f.denied = true;
  const reads = f.reads;
  await act(async () =>
    f.client.invalidateQueries({
      predicate: ({ queryKey: key }) => key[2] === "work-item-schema" && key[9] === undefined,
    })
  );
  await settleQuery();
  assertHidden(f);
  assert.equal(f.reads, reads + 1);
});

await test("a denied history catalog read hides the current properties in other mounted views", async (t) => {
  const f = await fixture(t);
  const { propertyHistoryCatalogOptions } = await import("../../apps/web/core/queries/work-item-property-history.ts");
  const { workItemPropertyHistoryService: history } =
    await import("../../apps/web/core/services/work-item-property-history.service.ts");
  const { getIdentitySessionGeneration } = await import("../../apps/web/core/queries/identity.ts");
  t.mock.method(history, "catalog", async () => {
    throw { response: { status: 404 } };
  });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "p",
    issueId: "item",
    membershipRole: 15,
    membershipRevision: "v1",
    projectRole: 15,
    generation: getIdentitySessionGeneration(f.client),
  };
  const reads = f.reads;
  await act(async () => assert.rejects(f.client.fetchQuery(propertyHistoryCatalogOptions(scope))));
  await settleQuery();
  assertHidden(f);
  assert.equal(f.reads, reads);
});
