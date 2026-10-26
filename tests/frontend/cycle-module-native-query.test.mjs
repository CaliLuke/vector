import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { MutationObserver } = webRequire("@tanstack/react-query");
const {
  parentCatalogueFixture,
  parentScope: scope,
  parentRow,
} = await import("./helpers/parent-catalogue-fixture.mjs");
const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
const { CycleArchiveService } = await import("../../apps/web/core/services/cycle_archive.service.ts");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const {
  parentCatalogueKeys,
  projectCycleOptions,
  projectModuleOptions,
  projectCycleUniverseOptions,
  cycleMutationOptions,
  moduleMutationOptions,
} = await import("../../apps/web/core/queries/cycle-modules.ts");
const tick = () => new Promise((resolve) => setImmediate(resolve));
await test("native cycle/module collections own canonical rows and errors without a normalized store", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  const rows = [parentRow("one")];
  t.mock.method(CycleService.prototype, "getWithParams", async () => rows);
  t.mock.method(ModuleService.prototype, "getModules", async () => {
    throw new Error("offline");
  });
  assert.deepEqual(await client.fetchQuery(projectCycleOptions(scope)), rows);
  const options = projectModuleOptions(scope);
  await assert.rejects(client.fetchQuery(options), /offline/);
  assert.equal(client.getQueryState(options.queryKey).status, "error");
});
await test("an old read cannot commit after identical permission query replacement", async (t) => {
  const { client, rolesKey } = await parentCatalogueFixture(t),
    read = Promise.withResolvers();
  t.mock.method(CycleService.prototype, "getWithParams", () => read.promise);
  const pending = client.fetchQuery(projectCycleOptions(scope));
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { p1: 20 });
  read.resolve([parentRow("private")]);
  await assert.rejects(pending, /access changed/);
  assert.equal(client.getQueryData(parentCatalogueKeys.list("cycle", scope)), undefined);
});
await test("grouping universe keeps authorized archived parents and rejects partial failure", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  const archived = parentRow("old", { archived_at: "2026-01-08" });
  t.mock.method(CycleService.prototype, "getWithParams", async () => [parentRow("active")]);
  const archive = t.mock.method(CycleArchiveService.prototype, "getArchivedCycles", async () => [archived]);
  const options = projectCycleUniverseOptions(scope);
  assert.deepEqual(
    (await client.fetchQuery(options)).map((row) => row.id),
    ["active", "old"]
  );
  archive.mock.mockImplementation(async () => {
    throw new Error("archive unavailable");
  });
  await assert.rejects(client.fetchQuery({ ...options, staleTime: 0 }), /archive unavailable/);
  assert.equal(client.getQueryState(options.queryKey).status, "error");
});
await test("archive/restore move canonical native rows; failure keeps the original collections", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  const row = parentRow("one"),
    active = parentCatalogueKeys.list("cycle", scope),
    archived = parentCatalogueKeys.list("cycle", scope, "archived"),
    universe = parentCatalogueKeys.list("cycle", scope, "universe");
  client.setQueryData(active, [row]);
  client.setQueryData(archived, []);
  client.setQueryData(universe, [row]);
  let fail = true;
  t.mock.method(CycleArchiveService.prototype, "archiveCycle", async () => {
    if (fail) throw new Error("offline");
    return { archived_at: "2026-01-08" };
  });
  t.mock.method(CycleArchiveService.prototype, "restoreCycle", async () => {});
  const mutation = new MutationObserver(client, cycleMutationOptions(client, scope));
  await assert.rejects(mutation.mutate({ type: "archive", id: row.id }), /offline/);
  assert.deepEqual(client.getQueryData(active), [row]);
  assert.deepEqual(client.getQueryData(archived), []);
  fail = false;
  await mutation.mutate({ type: "archive", id: row.id });
  assert.deepEqual(client.getQueryData(active), []);
  assert.equal(client.getQueryData(archived)[0].archived_at, "2026-01-08");
  assert.equal(client.getQueryData(universe)[0].id, "one");
  await mutation.mutate({ type: "restore", id: row.id });
  assert.equal(client.getQueryData(active)[0].archived_at, null);
  assert.deepEqual(client.getQueryData(archived), []);
});
await test("deleting one parent preserves every other cached detail", async (t) => {
  const { client } = await parentCatalogueFixture(t),
    a = parentRow("a"),
    b = parentRow("b");
  const ak = parentCatalogueKeys.detail("module", scope, "a"),
    bk = parentCatalogueKeys.detail("module", scope, "b");
  client.setQueryData(parentCatalogueKeys.list("module", scope), [a, b]);
  client.setQueryData(ak, a);
  client.setQueryData(bk, b);
  t.mock.method(ModuleService.prototype, "deleteModule", async () => {});
  await new MutationObserver(client, moduleMutationOptions(client, scope)).mutate({ type: "delete", id: "a" });
  assert.equal(client.getQueryData(ak), null);
  assert.deepEqual(client.getQueryData(bk), b);
});
async function verifyAcceptedWriteReconciliation(t, denied) {
  const { client } = await parentCatalogueFixture(t);
  const key = parentCatalogueKeys.list("module", scope);
  client.setQueryData(key, []);
  const write = Promise.withResolvers();
  t.mock.method(ModuleService.prototype, "createModule", () => write.promise);
  const pending = new MutationObserver(client, moduleMutationOptions(client, scope)).mutate({
    type: "create",
    data: { name: "new" },
  });
  await tick();
  client
    .getQueryCache()
    .find({ queryKey: identityKeys.currentUser(), exact: true })
    .setState({ status: "error", error: denied ? { response: { status: 403 } } : new Error("verification offline") });
  write.resolve(parentRow("new"));
  const result = await pending;
  assert.equal(result.captured.isCurrent(), !denied);
  assert.equal(client.getQueryData(key).length, denied ? 0 : 1);
}
await test("accepted writes survive transient identity failure but explicit denial blocks reconciliation", async (t) => {
  await t.test("transient verification failure", (subtest) => verifyAcceptedWriteReconciliation(subtest, false));
  await t.test("explicit denial", (subtest) => verifyAcceptedWriteReconciliation(subtest, true));
});
await test("queued writes reverify session and never touch a replacement actor", async (t) => {
  const { client } = await parentCatalogueFixture(t),
    write = Promise.withResolvers(),
    key = parentCatalogueKeys.list("module", scope);
  client.setQueryData(key, []);
  let calls = 0;
  t.mock.method(ModuleService.prototype, "createModule", () => {
    calls++;
    return write.promise;
  });
  const options = moduleMutationOptions(client, scope),
    a = new MutationObserver(client, options).mutate({ type: "create", data: { name: "a" } }),
    b = new MutationObserver(client, options).mutate({ type: "create", data: { name: "b" } }),
    rejected = assert.rejects(b, /editing access/);
  await tick();
  advanceIdentitySession(client);
  client.setQueryData(identityKeys.currentUser(), { id: "bob" });
  write.resolve(parentRow("a"));
  await a;
  await rejected;
  assert.equal(calls, 1);
  assert.deepEqual(client.getQueryData(key), []);
});
