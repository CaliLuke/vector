import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { MutationObserver, QueryClient } = webRequire("@tanstack/react-query");
const { EUserWorkspaceRoles } = webRequire("@plane/types");
const { workspaceDraftKeys, workspaceDraftMutationOptions } =
  await import("../../apps/web/core/queries/workspace-drafts.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { WorkspaceDraftService } = await import("../../apps/web/core/services/issue/workspace_draft.service.ts");

const scope = {
  accountId: "owner",
  sessionGeneration: 0,
  workspaceSlug: "default",
  membershipRole: EUserWorkspaceRoles.ADMIN,
  membershipRevision: "membership-v1",
};
const draft = { id: "draft", name: "Original", created_by: "owner", updated_by: "owner", is_draft: true };

function setup(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  client.setQueryData(permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug), {
    is_active: true,
    role: scope.membershipRole,
    updated_at: scope.membershipRevision,
  });
  const key = workspaceDraftKeys.list(scope);
  client.setQueryData(key, {
    pages: [{ results: [draft], total_count: 1, next_page_results: false }],
    pageParams: ["50:0:0"],
  });
  return { client, key };
}

await test("a failed workspace draft edit preserves the accepted list record for retry", async (t) => {
  const { client, key } = setup(t);
  const failure = new Error("draft update failed");
  t.mock.method(WorkspaceDraftService.prototype, "updateIssue", async () => {
    throw failure;
  });
  const mutation = new MutationObserver(client, workspaceDraftMutationOptions(client, scope));

  await assert.rejects(mutation.mutate({ kind: "update", issueId: "draft", data: { name: "Unsaved" } }), failure);

  assert.equal(client.getQueryData(key).pages[0].results[0].name, "Original");
});

await test("a failed draft deletion leaves the row available for another attempt", async (t) => {
  const { client, key } = setup(t);
  const failure = new Error("draft delete failed");
  t.mock.method(WorkspaceDraftService.prototype, "deleteIssue", async () => {
    throw failure;
  });
  const mutation = new MutationObserver(client, workspaceDraftMutationOptions(client, scope));

  await assert.rejects(mutation.mutate({ kind: "delete", issueId: "draft" }), failure);

  assert.deepEqual(client.getQueryData(key).pages[0].results, [draft]);
});
