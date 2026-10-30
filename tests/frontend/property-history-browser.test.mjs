import { settleQuery as settle } from "./helpers/query-router.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.historyReact = React;
const stubs = {
  "./controls": `export const inputStyle="";export const PreviewDialog=props=>globalThis.historyReact.createElement("section",{role:"dialog"},props.title,props.children,props.footer);`,
  "@/hooks/use-work-item-detail": `export const useWorkItemDetail=()=>({issue:{id:"issue",project_id:"p"}});`,
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>({can:()=>true,isPending:false,isError:false,workspaceRole:15,projectRole:15,membershipQuery:{data:{updated_at:"v1"}}});`,
  "./property-history-restore": `export const PropertyHistoryRestore=props=>globalThis.historyReact.createElement("div",null,globalThis.historyReact.createElement("article",null,props.version.value),globalThis.historyReact.createElement("button",{onClick:props.onAccessLost},"Deny access"));`,
  "@/hooks/use-workspaces": `export const useWorkspaceBySlug=()=>({data:{id:"workspace"}});`,
  "@/components/editor/rich-text": `export function RichTextEditor(props){globalThis.historyEditor=props;return globalThis.historyReact.createElement("article",null,props.initialValue);}`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { PropertyHistoryBrowser } =
  await import("../../apps/web/core/components/work-item-types/property-history-browser.tsx");
const { workItemPropertyHistoryService: service } =
  await import("../../apps/web/core/services/work-item-property-history.service.ts");
const { workItemEditsService: edits } = await import("../../apps/web/core/services/work-item-edits.service.ts");
const { identityKeys, getIdentitySessionGeneration } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { refreshPropertyHistory } = await import("../../apps/web/core/queries/work-item-property-history.ts");
const revision = (id, field = "removed") => ({
  id,
  property_id: field,
  property_title: "Historical notes",
  source: "change",
  created_at: "2026-10-25T12:00:00Z",
  actor: { id: "author", display_name: "Author" },
});

function button(label) {
  return [...document.querySelectorAll("button")].find((node) => node.textContent === label);
}
async function fixture(t, editable = false) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  client.setQueryData(permissionKeys.workspaceMembership("alice", "acme"), {
    is_active: true,
    role: 15,
    updated_at: "v1",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "v1"), { p: editable ? 15 : 5 });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "p",
    issueId: "item",
    membershipRole: 15,
    membershipRevision: "v1",
    projectRole: editable ? 15 : 5,
    generation: getIdentitySessionGeneration(client),
  };
  let newest = "v2",
    denied = false;
  const cursors = [],
    restored = [];
  t.mock.method(service, "catalog", async () => [
    { property_id: "removed", title: "Historical notes", version_count: 3 },
    { property_id: "active", title: "Current notes", version_count: 1 },
  ]);
  t.mock.method(service, "versions", async (_scope, field, before) => {
    if (denied) throw { status: 403 };
    cursors.push(before);
    return {
      results: before ? [revision("v0", field)] : [revision(newest, field), revision("v1", field)],
      next_cursor: before ? null : "cursor-uuid",
    };
  });
  t.mock.method(service, "detail", async (_scope, field, id) => ({ ...revision(id, field), value: `Content ${id}` }));
  const editSource = t.mock.method(edits, "source", async () => {
    throw new Error("Browsing must not open an editor");
  });
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(PropertyHistoryBrowser, {
        scope,
        editable,
        restorable: ["active"],
        onRestore: (value) => restored.push(value),
      })
    )
  );
  await settle();
  return {
    env,
    client,
    scope,
    cursors,
    restored,
    editSource,
    setNewest: (value) => {
      newest = value;
    },
    deny: () => {
      denied = true;
    },
  };
}
await test("viewer browses retained deleted-field revisions with immutable cursor pagination and no editor", async (t) => {
  const f = await fixture(t);
  assert.match(document.body.textContent, /Historical notes/);
  assert.match(document.body.textContent, /Content v2/);
  assert.equal(globalThis.historyEditor.editable, false);
  assert.equal(button("Restore this revision"), undefined);
  await act(async () => button("Older").click());
  await settle();
  assert.match(document.body.textContent, /Content v0/);
  assert.equal(f.cursors.at(-1), "cursor-uuid");
  await act(async () => button("Newer").click());
  await settle();
  assert.match(document.body.textContent, /Content v2/);
  assert.equal(f.editSource.mock.callCount(), 0);
});
await test("refresh retains selected immutable revision and failed revision reads hide old content", async (t) => {
  const f = await fixture(t);
  await act(async () => document.querySelectorAll("nav li button")[1].click());
  await settle();
  assert.match(document.body.textContent, /Content v1/);
  f.setNewest("v3");
  await act(async () => refreshPropertyHistory(f.client, f.scope));
  await settle();
  assert.match(document.body.textContent, /Content v1/);
  f.deny();
  await act(async () => refreshPropertyHistory(f.client, f.scope));
  await settle();
  assert.doesNotMatch(document.body.textContent, /Content v1/);
  assert.match(document.body.textContent, /Could not load revisions/);
});
await test("only active attached rich-text fields expose restore, and choosing another field resets revision", async (t) => {
  const f = await fixture(t, true);
  assert.equal(button("Restore this revision"), undefined);
  await act(async () => {
    const select = document.querySelector("select");
    select.value = "active";
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await settle();
  await act(async () => button("Restore this revision").click());
  assert.equal(f.restored[0].property_id, "active");
  assert.equal(f.restored[0].id, "v2");
});

await test("denied restore replaces the entire dialog and removes Back navigation to cached history", async (t) => {
  const f = await fixture(t, true);
  const { ItemPropertyHistory } =
    await import("../../apps/web/core/components/work-item-types/item-property-history.tsx");
  const { workItemSchemaService } = await import("../../apps/web/core/services/work-item-schema.service.ts");
  t.mock.method(workItemSchemaService, "schema", async () => ({
    properties: [{ id: "active", kind: "Rich text", configuration: { active: true } }],
  }));
  await f.env.render(
    React.createElement(
      QueryClientProvider,
      { client: f.client },
      React.createElement(ItemPropertyHistory, { workspaceSlug: "acme", projectId: "p", issueId: "item" })
    )
  );
  await act(async () => button("Property history").click());
  await settle();
  await act(async () => {
    const select = document.querySelector("select");
    select.value = "active";
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await settle();
  await act(async () => button("Restore this revision").click());
  await settle();
  assert.match(document.body.textContent, /Content v2/);
  await act(async () => button("Deny access").click());
  await settle();
  assert.doesNotMatch(document.body.textContent, /Content v2/);
  assert.match(document.body.textContent, /no longer available/);
  assert.equal(button("Back to history"), undefined);
  assert.equal(document.querySelector("nav"), null);
});

await test("archived full-screen peek keeps history readable while editing stays disabled", async (t) => {
  const f = await fixture(t, true);
  const { ItemPropertyHistory } =
    await import("../../apps/web/core/components/work-item-types/item-property-history.tsx");
  globalThis.archivedPeekHistory = ItemPropertyHistory;
  const viewStubs = {
    "@/providers/issue-detail-ui-state":
      "export const useIssueDetailUiState=()=>({setPeekIssue:()=>{},isAnyModalOpen:false});",
    "@/providers/work-item-detail-record":
      "export const useWorkItemDetailRecord=()=>({issueId:'item',issue:{id:'item',project_id:'p',archived_at:'2026-10-26'}});",
    "@/hooks/use-keypress": "export default ()=>{};",
    "@/hooks/use-peek-overview-outside-click": "export default ()=>{};",
    "../use-text-save-status":
      "export const useTextSaveStatus=()=>({status:'saved',setTitleStatus:()=>{},setDescriptionStatus:()=>{}});",
    "../issue-detail/issue-activity": "export const IssueActivity=()=>null;",
    "../issue-detail-widgets": `const React=globalThis.historyReact;export const IssueDetailWidgets=({disabled})=>React.createElement('button',{disabled,'data-widget':true},'Add sub-item');`,
    "./error": "export const IssuePeekOverviewError=()=>null;",
    "./loader": "export const IssuePeekOverviewLoader=()=>null;",
    "./issue-detail": "export const PeekOverviewIssueDetails=()=>null;",
    "./header": `const React=globalThis.historyReact;export const IssuePeekOverviewHeader=({setPeekMode})=>React.createElement('button',{onClick:()=>setPeekMode('full-screen')},'Full screen');`,
    "./properties": `const React=globalThis.historyReact;export const PeekOverviewProperties=props=>React.createElement(globalThis.archivedPeekHistory,props);`,
  };
  const hooks = registerHooks({
    resolve: (specifier, context, next) =>
      context.parentURL?.endsWith("/peek-overview/view.tsx")
        ? resolvePackageStub(specifier, viewStubs, context, next)
        : next(specifier, context),
  });
  t.after(() => {
    hooks.deregister();
    delete globalThis.archivedPeekHistory;
  });
  const { IssueView } = await import("../../apps/web/core/components/issues/peek-overview/view.tsx");
  await f.env.render(
    React.createElement(
      QueryClientProvider,
      { client: f.client },
      React.createElement(IssueView, {
        workspaceSlug: "acme",
        projectId: "p",
        issueId: "item",
        is_archived: true,
        issueOperations: {},
      })
    )
  );
  assert.equal(document.querySelector("[data-widget]").disabled, true);
  await act(async () => button("Full screen").click());
  const history = button("Property history");
  assert.ok(history);
  assert.equal(
    history.closest(".pointer-events-none"),
    null,
    "read-only history must not have an inert sidebar ancestor"
  );
  assert.equal(
    document.querySelector("[data-widget]").disabled,
    true,
    "full-screen widgets must preserve archived restrictions"
  );
  await act(async () => history.click());
  await settle();
  assert.ok(document.querySelector('[role="dialog"]'));
  assert.match(document.body.textContent, /Content v2/);
  assert.equal(button("Restore this revision"), undefined, "archived history cannot offer restoration");
  assert.equal(f.editSource.mock.callCount(), 0);
});
