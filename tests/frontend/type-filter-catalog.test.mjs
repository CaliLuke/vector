import { filterEditorFixture } from "./helpers/filter-editor-fixture.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClientProvider, notifyManager } from "@tanstack/react-query";
import {
  displayFilterConditions,
  filterConditions,
  displayOperatorOptions,
  readFilterExpression,
  updateFilterConditionOperator,
  updateFilterConditionValue,
  workItemFiltersAdapter,
  writeFilterExpression,
} from "../../packages/shared-state/dist/index.js";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createProjectQueryDomFixture } from "./helpers/query-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
const stubs = {
  "@/hooks/use-current-user": `export const useCurrentUserId=()=>"alice"; export const useCurrentUser=()=>({data:{id:"alice"}});`,
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>globalThis.typeFilterFixture.permissions;`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const bootstrap = await createReactDomTestEnvironment();
const { useTypeFilterConfig } = await import("../../apps/web/core/hooks/work-item-filters/use-type-filter-config.tsx");
const { FiltersRow } = await import("../../apps/web/core/components/rich-filters/filters-row.tsx");
const { itemTypeIdentityOptions } = await import("../../apps/web/core/queries/work-item-type-identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { workItemTypesService: service } = await import("../../apps/web/core/services/work-item-types.service.ts");
await bootstrap.unmount();
function Harness({ initialExpression, projectId, allowed = true }) {
  const { config, isError, retry } = useTypeFilterConfig("acme", projectId, allowed);
  const [snapshot, setSnapshot] = React.useState(() => readFilterExpression(workItemFiltersAdapter, initialExpression));
  const { expression, committedConditionIds, error } = snapshot;
  const updateValue = React.useCallback(
    (id, value) => {
      const next = updateFilterConditionValue(expression, id, value).expression;
      const result = writeFilterExpression(workItemFiltersAdapter, next, committedConditionIds, error);
      if (!result.ok) throw new Error(result.error);
      globalThis.typeFilterFixture.changes.push(result.value);
      setSnapshot({ ...snapshot, expression: next, source: result.value });
    },
    [committedConditionIds, error, expression, setSnapshot, snapshot]
  );
  const editor = filterEditorFixture({
    expression: snapshot.expression,
    conditions: filterConditions(snapshot.expression),
    displayConditions: displayFilterConditions(snapshot.expression),
    configs: [config],
    conversionError: snapshot.error,
    hasActiveFilters: true,
    onUpdateConditionOperator: (id, operator, isNegation) => {
      const next = updateFilterConditionOperator(snapshot.expression, id, operator, isNegation, false);
      const result = writeFilterExpression(
        workItemFiltersAdapter,
        next,
        snapshot.committedConditionIds,
        snapshot.error
      );
      if (!result.ok) throw new Error(result.error);
      globalThis.typeFilterFixture.changes.push(result.value);
      setSnapshot({ ...snapshot, expression: next, source: result.value });
    },
    onUpdateConditionValue: updateValue,
  });
  React.useEffect(() => {
    Object.assign(globalThis.typeFilterFixture, { config, updateValue, snapshot });
  }, [config, snapshot, updateValue]);
  return React.createElement(
    React.Fragment,
    null,
    isError && React.createElement("button", { onClick: retry }, "Retry types"),
    React.createElement(FiltersRow, { editor })
  );
}
async function fixture(t, raw = { type_id__in: "paused,missing" }) {
  const f = await createProjectQueryDomFixture(t, 15);
  f.permissions = {
    isPending: false,
    isError: false,
    projectRole: 15,
    workspaceRole: 15,
    membershipQuery: { isSuccess: true, data: { updated_at: "v1" } },
    rolesQuery: { isError: false },
  };
  f.catalog = {
    enabled: true,
    revision: 3,
    types: [
      {
        id: "paused",
        name: "Paused incident",
        icon: "incident",
        color: "#ff0000",
        active: false,
        default: false,
        level: 0,
      },
      {
        id: "unimported",
        name: "Unimported story",
        icon: "story",
        color: "#0000ff",
        active: true,
        default: false,
        level: 0,
      },
    ],
  };
  f.changes = [];
  globalThis.typeFilterFixture = f;
  t.mock.method(service, "identities", async () => f.catalog);
  f.draw = (props = {}) =>
    f.env.render(
      React.createElement(
        QueryClientProvider,
        { client: f.client },
        React.createElement(Harness, { initialExpression: raw, projectId: "p", ...props })
      )
    );
  return f;
}
await test("type filters keep paused identities, show missing IDs, and support exclusion without changing saved values", async (t) => {
  const f = await fixture(t);
  await f.draw();
  assert.match(f.env.container.textContent, /Paused incident/);
  assert.match(f.env.container.textContent, /Unavailable \(missing\)/);
  const config = f.config;
  assert.deepEqual(
    (await config.supportedOperatorConfigsMap.get("in").getOptions()).map((x) => x.value),
    ["paused", "unimported"]
  );
  assert.deepEqual(
    displayOperatorOptions(config, ["paused"]).map((option) => option.value),
    ["exact", "not_exact", "in", "not_in"]
  );
  assert.equal(new Set(displayOperatorOptions(config, ["paused"]).map((entry) => entry.label)).size, 4);
  const operatorButton = [...f.env.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("is any of")
  );
  assert.ok(operatorButton, "the current operator remains available as a button");
  await act(async () => operatorButton.click());
  const excludedOperator = [...document.querySelectorAll('[role="option"]')].find((option) =>
    option.textContent.includes("is none of")
  );
  assert.ok(excludedOperator, "the exclusion operator is available in the menu");
  await act(async () => {
    excludedOperator.dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
    excludedOperator.click();
  });
  assert.deepEqual(f.changes.at(-1), { not: { type_id__in: "paused,missing" } });
});
await test("failed refresh scrubs names, retains the predicate, and retry restores authorized choices", async (t) => {
  const f = await fixture(t, { type_id__exact: "paused" });
  await f.draw();
  assert.match(f.env.container.textContent, /Paused incident/);
  const key = itemTypeIdentityOptions({
    accountId: "alice",
    generation: 0,
    workspaceSlug: "acme",
    projectId: "p",
    membershipRole: 15,
    membershipRevision: "v1",
    projectRole: 15,
  }).queryKey;
  t.mock.method(service, "identities", async () => {
    throw new Error("Access denied");
  });
  await act(async () => f.client.refetchQueries({ queryKey: key, exact: true }));
  assert.doesNotMatch(f.env.container.textContent, /Paused incident/);
  assert.match(f.env.container.textContent, /Invalid filter/);
  assert.match(f.env.container.textContent, /paused/);
  assert.deepEqual(f.changes, []);
  t.mock.method(service, "identities", async () => f.catalog);
  const retry = [...f.env.container.querySelectorAll("button")].find((x) => x.textContent === "Retry types");
  await act(async () => retry.click());
  assert.match(f.env.container.textContent, /Paused incident/);
  assert.deepEqual(f.changes, []);
});
await test("disabling a page catalog removes choices and labels but keeps raw predicates", async (t) => {
  const f = await fixture(t);
  await f.draw();
  await f.draw({ allowed: false });
  assert.doesNotMatch(f.env.container.textContent, /Paused incident/);
  assert.deepEqual(f.changes, []);
  assert.deepEqual(f.snapshot.source, { type_id__in: "paused,missing" });
});
for (const change of ["membership", "session", "query"]) {
  void test(`workspace identity request rejects delayed ${change} changes`, async (t) => {
    const f = await fixture(t);
    let resolve;
    t.mock.method(
      service,
      "identities",
      () =>
        new Promise((done) => {
          resolve = done;
        })
    );
    const options = itemTypeIdentityOptions({
      accountId: "alice",
      generation: 0,
      workspaceSlug: "acme",
      membershipRole: 15,
      membershipRevision: "v1",
    });
    const request = f.client.fetchQuery(options).catch((error) => error);
    assert.equal(typeof resolve, "function");
    if (change === "membership")
      f.client.setQueryData(permissionKeys.workspaceMembership("alice", "acme"), {
        is_active: false,
        role: 15,
        updated_at: "v2",
      });
    if (change === "session") advanceIdentitySession(f.client);
    if (change === "query") f.client.removeQueries({ queryKey: options.queryKey, exact: true });
    resolve(f.catalog);
    assert.ok((await request) instanceof Error, change);
    assert.equal(f.client.getQueryData(options.queryKey), undefined, change);
  });
}
await test("workspace filters remain usable when unrelated project roles fail", async (t) => {
  const f = await fixture(t);
  f.permissions.isError = true;
  f.permissions.rolesQuery = { isError: true };
  await f.draw({ projectId: undefined });
  assert.match(f.env.container.textContent, /Paused incident/);
  assert.doesNotMatch(f.env.container.textContent, /Retry types/);
});

await test("nested filter chips observe in-place value changes", async (t) => {
  const f = await fixture(t, { and: [{ priority__in: "high" }, { type_id__exact: "paused" }] });
  await f.draw();
  assert.match(f.env.container.textContent, /Paused incident/);
  const condition = f.snapshot.expression.children[1];
  await act(async () => f.updateValue(condition.id, "unimported"));
  assert.deepEqual(f.changes.at(-1), { and: [{ priority__in: "high" }, { type_id__exact: "unimported" }] });
  assert.match(f.env.container.textContent, /Unimported story/);
  assert.doesNotMatch(f.env.container.textContent, /Paused incident/);
});
