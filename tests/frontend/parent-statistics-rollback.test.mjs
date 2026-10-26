import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { parentCatalogueFixture, parentScope, parentRow } = await import("./helpers/parent-catalogue-fixture.mjs");
const { captureParentStatistics } = await import("../../apps/web/core/queries/parent-statistics.ts");
const { parentCatalogueKeys } = await import("../../apps/web/core/queries/cycle-modules.ts");
const updates = { pathUpdates: [{ path: ["total_issues"], value: 1 }], assigneeUpdates: [], labelUpdates: [] };
async function fixture(t) {
  const { client } = await parentCatalogueFixture(t),
    key = parentCatalogueKeys.detail("cycle", parentScope, "c1"),
    row = parentRow("c1", { version: 2, total_issues: 1, progress: [] });
  client.setQueryData(key, row);
  return { client, key, row, effect: captureParentStatistics(client, "cycle", parentScope, "c1", key) };
}
await test("native statistics rollback restores only its own current optimistic value", async (t) => {
  const { client, key, effect } = await fixture(t);
  const undo = effect.applyDistribution(updates);
  assert.equal(client.getQueryData(key).total_issues, 2);
  assert.equal(typeof undo, "function");
  undo();
  assert.equal(client.getQueryData(key).total_issues, 1);
});
await test("a newer canonical update with structural sharing defeats an older rollback receipt", async (t) => {
  const { client, key, effect } = await fixture(t);
  const undo = effect.applyDistribution(updates),
    optimistic = client.getQueryData(key);
  client.setQueryData(key, structuredClone(optimistic));
  assert.equal(
    client.getQueryData(key),
    optimistic,
    "Query structural sharing deliberately keeps the same data reference"
  );
  undo();
  assert.equal(
    client.getQueryData(key).total_issues,
    2,
    "canonical acceptance increments native Query's data revision even with equal data"
  );
});
await test("rollback cannot resurrect a removed query or touch an equal replacement query", async (t) => {
  const { client, key, row, effect } = await fixture(t);
  const undo = effect.applyDistribution(updates);
  client.removeQueries({ queryKey: key, exact: true });
  undo();
  assert.equal(client.getQueryData(key), undefined);
  client.setQueryData(key, { ...row, total_issues: 9 });
  undo();
  assert.equal(client.getQueryData(key).total_issues, 9);
});
