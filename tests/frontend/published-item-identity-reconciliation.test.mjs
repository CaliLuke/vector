import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { React } = await fromRoot("tests/frontend/helpers/react.mjs");
const { createReactDomTestEnvironment } = await fromRoot("tests/frontend/helpers/react-dom.mjs");
registerWebModuleHooks();
const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");
const { identityKeys } = await fromRoot("apps/web/core/queries/identity.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { currentParentCatalogueScope } = await import("../../apps/web/core/queries/cycle-module-scope.ts");
const { currentIssueListMutationOwner } = await import("../../apps/web/core/queries/issue-list-mutations.ts");
const { issueListKeys } = await import("../../apps/web/core/queries/issue-lists.ts");
const nativeFixture = {};
globalThis.nativeFixture = nativeFixture;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      context.parentURL?.endsWith("/use-item-edit-reconciliation.ts") &&
      specifier === "@/hooks/use-issue-list-mutation-owner"
    )
      return {
        shortCircuit: true,
        url: `data:text/javascript,export const useIssueListMutationOwner=()=>((workspaceSlug, projectId)=>globalThis.nativeFixture.owner(workspaceSlug, projectId));`,
      };
    return nextResolve(specifier, context);
  },
});
const { useItemEditReconciliation } =
  await import("../../apps/web/core/components/work-item-types/use-item-edit-reconciliation.ts");

await test("accepted published edit receipts reconcile exact source and destination Query owners after identity refresh failure", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const sourceScope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "source",
    issueId: "item-1",
    membershipRole: 15,
    membershipRevision: "revision-1",
    projectRole: 20,
    generation: 0,
  };
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  client.setQueryData(permissionKeys.workspaceMembership("alice", "acme"), {
    is_active: true,
    role: 15,
    updated_at: "revision-1",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "revision-1"), {
    source: 20,
    destination: 20,
  });
  const sourceKey = ["workspace", "acme", "item-edit-source", "alice", 0, "source", "item-1"];
  client.setQueryData(sourceKey, { item: { id: "item-1", project_id: "source" } });
  const listQueries = new Map();
  for (const projectId of ["source", "destination"]) {
    const access = currentParentCatalogueScope(client, "acme", projectId).project;
    const input = { workspaceSlug: "acme", projectId, richFilters: {}, displayFilters: {} };
    const read = {
      scope: access,
      pagination: { canGroup: false, perPageCount: 20 },
      request: { family: "project", input, isReadCurrent: () => true },
    };
    const key = issueListKeys.list(read);
    const query = client.getQueryCache().build(client, { queryKey: key });
    client.setQueryData(key, { generation: `g-${projectId}`, response: { results: [], total_count: 0 } });
    listQueries.set(projectId, { key, query });
  }
  globalThis.nativeFixture.owner = (workspaceSlug, projectId) =>
    currentIssueListMutationOwner(client, workspaceSlug, projectId, "issues", () => true);
  let reconciliation;
  function Probe() {
    const value = useItemEditReconciliation(sourceScope, sourceKey);
    React.useEffect(() => {
      reconciliation = value;
    }, [value]);
    return null;
  }
  t.after(async () => {
    delete globalThis.nativeFixture.owner;
    delete globalThis.nativeFixture;
    await environment.unmount(() => client.clear());
  });
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(Probe)));

  const captured = await reconciliation.captureListOwners("destination");
  const userQuery = client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  userQuery.setState({ ...userQuery.state, status: "error", error: new Error("identity refresh failed") });
  const moved = await reconciliation.reconcile(
    { id: "item-1", sequence_id: 7, project_id: "destination", type_id: "bug" },
    () => true,
    captured
  );

  assert.equal(moved, true);
  assert.equal(listQueries.get("source").query.state.isInvalidated, true);
  assert.equal(listQueries.get("destination").query.state.isInvalidated, true);
});
