import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

// Query must initialize with a browser environment so its real polling scheduler runs.
const env = await createReactDomTestEnvironment();
test.after(() => env.unmount());
registerWebModuleHooks();
const { QueryClient, QueryObserver, focusManager } = await import("@tanstack/react-query");
const { projectIssueScope, seedIssueQueryScope } = await import("./helpers/issue-query-fixtures.mjs");
const { issueListOptions } = await import("../../apps/web/core/queries/issue-lists.ts");
const { tablePropertyValuesOptions } = await import("../../apps/web/core/queries/work-item-table-values.ts");
const { hasCustomPropertyFilter } = await import("../../apps/web/core/queries/work-item-clock.ts");
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const midnight = Date.parse("2026-10-27T00:00:00Z");

function clockClient(t) {
  t.mock.timers.enable({ apis: ["Date", "setInterval"], now: midnight - 1000 });
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity, retry: false } } });
  client.mount();
  seedIssueQueryScope(client, projectIssueScope());
  focusManager.setFocused(true);
  t.after(() => {
    client.clear();
    client.unmount();
    focusManager.setFocused(undefined);
  });
  return client;
}

function observe(t, client, options) {
  const observer = new QueryObserver(client, { ...options, gcTime: Infinity });
  let waiting;
  const unsubscribe = observer.subscribe((result) => {
    if (!result.isFetching && result.isSuccess) waiting?.resolve(result.data);
    if (result.isError) waiting?.reject(result.error);
  });
  t.after(unsubscribe);
  return {
    async current() {
      const result = observer.getCurrentResult();
      if (result.isSuccess && !result.isFetching) return result.data;
      waiting = Promise.withResolvers();
      return waiting.promise;
    },
    async advance(ms) {
      waiting = Promise.withResolvers();
      t.mock.timers.tick(ms);
      return waiting.promise;
    },
    async focus() {
      waiting = Promise.withResolvers();
      focusManager.setFocused(true);
      return waiting.promise;
    },
    unsubscribe,
  };
}

function formulaTable(t) {
  const client = clockClient(t);
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "p1",
    generation: 0,
    membershipRole: 15,
    membershipRevision: "membership-1",
    projectRole: 20,
    membershipProjectRole: 20,
  };
  const item = { id: "item", updated_at: "unchanged" };
  t.mock.method(IssueService.prototype, "retrieveIssues", async () => [
    {
      ...item,
      project_id: "p1",
      properties_revision: 7,
      property_values: { today: { kind: "Formula", status: "value", value: new Date().toISOString().slice(0, 10) } },
      properties_evaluated_at: new Date().toISOString(),
    },
  ]);
  const reader = observe(t, client, tablePropertyValuesOptions(scope, 7, [item], true, true));
  return { client, scope, item, reader };
}

await test("visible formula cells re-read server values at UTC midnight and later minute boundaries", async (t) => {
  const { scope, item, reader } = formulaTable(t);
  assert.equal((await reader.current()).item.property_values.today.value, "2026-10-26");
  const updated = await reader.advance(1000);
  assert.equal(updated.item.property_values.today.value, "2026-10-27");
  assert.equal(updated.item.properties_evaluated_at, "2026-10-27T00:00:00.000Z");
  assert.equal((await reader.advance(60_000)).item.properties_evaluated_at, "2026-10-27T00:01:00.000Z");
  assert.equal(tablePropertyValuesOptions(scope, 7, [item], true, false).refetchInterval, false);
  assert.equal(tablePropertyValuesOptions(scope, 7, [item], false, true).refetchInterval, false);
});

await test("formula tables refresh on focus and remount even when their cache is less than 30 seconds old", async (t) => {
  await Promise.all(
    ["focus", "remount"].map((action) =>
      t.test(action, async (subtest) => {
        const { client, scope, item, reader } = formulaTable(subtest);
        assert.equal((await reader.current()).item.property_values.today.value, "2026-10-26");
        focusManager.setFocused(false);
        if (action === "remount") reader.unsubscribe();
        subtest.mock.timers.tick(1000);
        const result =
          action === "focus"
            ? await reader.focus()
            : await observe(subtest, client, tablePropertyValuesOptions(scope, 7, [item], true, true)).current();
        assert.equal(result.item.property_values.today.value, "2026-10-27");
        assert.equal(result.item.properties_evaluated_at, "2026-10-27T00:00:00.000Z");
      })
    )
  );
});

await test("clock refresh replaces formula-filtered membership and counts without an item edit", async (t) => {
  const client = clockClient(t);
  const input = {
    workspaceSlug: "acme",
    projectId: "p1",
    richFilters: { and: [{ not: { customproperty_elapsed__lt: 1 } }] },
    displayFilters: { layout: "list", group_by: null, sub_group_by: null },
  };
  const read = {
    scope: projectIssueScope(),
    pagination: { canGroup: false, perPageCount: 20 },
    request: { family: "project", input, isReadCurrent: () => true },
  };
  const transport = t.mock.method(IssueService.prototype, "getIssues", async () => {
    const results = Date.now() >= midnight ? [{ id: "due", project_id: "p1" }] : [];
    return { results, total_results: results.length, count: results.length, total_count: results.length };
  });
  const reader = observe(t, client, issueListOptions(read));
  const before = await reader.current();
  assert.deepEqual(before.response.results, []);
  const after = await reader.advance(1000);
  assert.deepEqual(
    after.response.results.map((row) => row.id),
    ["due"]
  );
  assert.equal(after.response.total_results, 1);
  assert.notEqual(after.generation, before.generation);
  reader.unsubscribe();
  const requests = transport.mock.callCount();
  t.mock.timers.tick(120_000);
  assert.equal(transport.mock.callCount(), requests, "unmounted lists stop refreshing");
  const ordinary = issueListOptions({ ...read, request: { ...read.request, input: { ...input, richFilters: {} } } });
  assert.equal(ordinary.refetchInterval, false);
});

await test("clock refresh finds nested property filters but ignores field values that resemble keys", () => {
  assert.equal(
    hasCustomPropertyFilter({ or: [{ priority__exact: "high" }, { not: { customproperty_age__gte: 1 } }] }),
    true
  );
  assert.equal(hasCustomPropertyFilter({ name__exact: "customproperty_age" }), false);
  assert.equal(hasCustomPropertyFilter({ type_id__in: ["task"] }), false);
});
