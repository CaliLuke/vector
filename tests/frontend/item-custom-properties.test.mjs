import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.detailFieldReact = React;
const stubs = {
  "@/hooks/use-work-item-detail": `export const useWorkItemDetail=()=>({issue:{id:"issue",project_id:"p",updated_at:"v1"}});`,
  "@/hooks/use-workspaces": `export const useWorkspaceBySlug=()=>({data:{id:"workspace"}});`,
  "@/components/editor/rich-text": `export function RichTextEditor(props){globalThis.detailRichTextProps=props;return globalThis.detailFieldReact.createElement("article",null,props.initialValue);}`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { ItemCustomProperties } =
  await import("../../apps/web/core/components/work-item-types/item-custom-properties.tsx");
const { workItemSchemaService } = await import("../../apps/web/core/services/work-item-schema.service.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { refreshItemSchema } = await import("../../apps/web/core/queries/work-item-detail-schema.ts");
const definition = (id, kind) => ({
  id,
  title: id,
  kind,
  description: "",
  configuration: { active: true, format: "", options: [], expression: "" },
});
await test("detail properties preserve attachment order, readonly rich text, and hide cached private values after denial", async (t) => {
  const { env, client } = await createQueryDomFixture(t, {
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const memberKey = permissionKeys.workspaceMembership("alice", "acme");
  client.setQueryData(memberKey, { is_active: true, role: 15, updated_at: "v1" });
  client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "v1"), { p: 5 });
  // Keep permission reads deterministic while exercising the real permission hooks.
  client.setQueryDefaults(["permissions"], { refetchOnMount: false, refetchOnWindowFocus: false });
  // The hook options specify refetchOnMount, so serve the native read transports too.
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => client.getQueryData(memberKey));
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({ p: 5 }));
  let fail = false;
  t.mock.method(workItemSchemaService, "schema", async () => {
    if (fail) throw { status: 403 };
    return {
      enabled: true,
      item_updated_at: "server-v1",
      evaluated_at: "2026-10-25T12:00:00Z",
      properties: [definition("Notes", "Rich text"), definition("Effort", "Number"), definition("Result", "Formula")],
      stored_values: { Notes: "Private notes", Effort: 0 },
      computed: { Result: { status: "value", value: 2 } },
    };
  });
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(ItemCustomProperties, { workspaceSlug: "acme", projectId: "p", issueId: "i" })
    )
  );
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.deepEqual(
    [...document.querySelectorAll("dt")].map((node) => node.textContent),
    ["Notes", "Effort", "Result"]
  );
  assert.match(document.body.textContent, /Private notes/);
  assert.equal(globalThis.detailRichTextProps.editable, false);
  assert.equal(globalThis.detailRichTextProps.onChange, undefined);
  assert.equal(document.querySelector("input,textarea,select"), null);
  assert.equal(document.querySelector("time").dateTime, "2026-10-25T12:00:00Z");
  fail = true;
  await act(async () => {
    await refreshItemSchema(client, "acme", "i");
  });
  assert.doesNotMatch(document.body.textContent, /Private notes/);
  assert.match(document.body.textContent, /Custom properties are no longer available with your current access/);
  fail = false;
  await act(async () => {
    document.querySelector("button").click();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.match(document.body.textContent, /Private notes/);
  await act(async () => client.setQueryData(memberKey, { is_active: false, role: 15, updated_at: "v2" }));
  assert.doesNotMatch(document.body.textContent, /Private notes/);
});
