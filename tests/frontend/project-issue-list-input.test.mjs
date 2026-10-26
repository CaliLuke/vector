import { projectIssueScope } from "./helpers/issue-query-fixtures.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedIssueQueryScope } from "./helpers/issue-query-fixtures.mjs";

const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
registerWebModuleHooks();
const { QueryClient } = await import("@tanstack/react-query");
const { IssueService } = await fromRoot("apps/web/core/services/issue/index.ts");
const { issueListKeys } = await import("../../apps/web/core/queries/issue-lists.ts");
const { currentIssueListMutationOwner, patchIssueMutation } =
  await import("../../apps/web/core/queries/issue-list-mutations.ts");

const scope = projectIssueScope();
const issue = (overrides = {}) => ({ id: "item-1", project_id: "p1", type_id: "task", name: "Before", ...overrides });

function clientWithAccess() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedIssueQueryScope(client, scope);
  return client;
}

await test("an accepted work item PATCH reconciles its canonical receipt into the captured Query", async (t) => {
  const client = clientWithAccess();
  t.after(() => client.clear());
  const read = {
    scope,
    pagination: { canGroup: false, perPageCount: 20 },
    request: {
      family: "project",
      input: {
        workspaceSlug: "acme",
        projectId: "p1",
        richFilters: {},
        displayFilters: { layout: "list", order_by: "sort_order" },
      },
      isReadCurrent: () => true,
    },
  };
  const key = issueListKeys.list(read);
  const query = client.getQueryCache().build(client, { queryKey: key });
  client.setQueryData(key, { generation: "g1", response: { results: [issue()], total_count: 1 } });
  const saved = issue({ name: "After", type_id: "bug" });
  t.mock.method(IssueService.prototype, "patchIssue", async () => saved);
  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);

  const receipt = await patchIssueMutation(client, owner, "item-1", { name: "After" });

  assert.deepEqual(receipt, saved);
  assert.equal(client.getQueryCache().find({ queryKey: key, exact: true }), query);
  assert.equal(client.getQueryData(key).response.results[0].name, "After");
  assert.equal(query.state.isInvalidated, true);
});
