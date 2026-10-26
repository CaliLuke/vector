import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { parentCatalogueFixture, parentScope, parentRow } = await import("./helpers/parent-catalogue-fixture.mjs");
const { CycleArchiveService } = await import("../../apps/web/core/services/cycle_archive.service.ts");
const { projectCycleOptions, parentCatalogueKeys } = await import("../../apps/web/core/queries/cycle-modules.ts");
await test("opening archived cycles directly loads its native collection independently of active cycles", async (t) => {
  const { client } = await parentCatalogueFixture(t),
    row = parentRow("archive", { archived_at: "2026-01-08" });
  t.mock.method(CycleArchiveService.prototype, "getArchivedCycles", async () => [row]);
  const result = await client.fetchQuery(projectCycleOptions(parentScope, true));
  assert.deepEqual(result, [row]);
  assert.equal(client.getQueryData(parentCatalogueKeys.list("cycle", parentScope)), undefined);
});
await test("canonical empty archive is successful only for its exact project", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  t.mock.method(CycleArchiveService.prototype, "getArchivedCycles", async () => []);
  assert.deepEqual(await client.fetchQuery(projectCycleOptions(parentScope, true)), []);
  assert.equal(
    client.getQueryData(parentCatalogueKeys.list("cycle", { ...parentScope, projectId: "other" }, "archived")),
    undefined
  );
});
await test("failed archive reads expose error and a real retry replaces it with canonical empty", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  let fail = true;
  t.mock.method(CycleArchiveService.prototype, "getArchivedCycles", async () => {
    if (fail) throw new Error("offline");
    return [];
  });
  const options = projectCycleOptions(parentScope, true);
  await assert.rejects(client.fetchQuery(options), /offline/);
  assert.equal(client.getQueryState(options.queryKey).status, "error");
  assert.equal(client.getQueryData(options.queryKey), undefined);
  fail = false;
  assert.deepEqual(await client.fetchQuery(options), []);
});
