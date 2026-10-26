import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";
import { registerAdminModuleHooks } from "./helpers/web-module-hooks.mjs";

registerAdminModuleHooks();
const require = createRequire(new URL("../../apps/admin/package.json", import.meta.url));
const { QueryClient, MutationObserver, InfiniteQueryObserver, QueryObserver } = require("@tanstack/react-query");
const { InstanceService, InstanceWorkspaceService, UserService } = require("@plane/services");
const {
  instanceInfoOptions,
  instanceConfigurationsOptions,
  currentUserOptions,
  updateInstanceOptions,
  updateConfigurationsOptions,
  disableEmailOptions,
} = await import("../../apps/admin/queries/instance.ts");
const { instanceWorkspacesOptions } = await import("../../apps/admin/queries/workspaces.ts");

function clientFor(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  t.after(() => client.clear());
  return client;
}

await test("saving instance details replaces the instance and preserves public configuration", async (t) => {
  const client = clientFor(t);
  client.setQueryData(instanceInfoOptions.queryKey, { instance: { name: "before" }, config: { login: true } });
  t.mock.method(InstanceService.prototype, "update", async (patch) => ({ name: patch.name }));
  await new MutationObserver(client, updateInstanceOptions(client)).mutate({ name: "after" });
  assert.deepEqual(client.getQueryData(instanceInfoOptions.queryKey), {
    instance: { name: "after" },
    config: { login: true },
  });
});

await test("configuration saves merge newly returned keys and preserve unrelated settings", async (t) => {
  const client = clientFor(t);
  const initial = [
    { key: "EMAIL_HOST", value: "mail" },
    { key: "ENABLE_SMTP", value: "1" },
  ];
  client.setQueryData(instanceConfigurationsOptions.queryKey, initial);
  t.mock.method(InstanceService.prototype, "updateConfigurations", async () => [{ key: "LLM_MODEL", value: "model" }]);
  await new MutationObserver(client, updateConfigurationsOptions(client)).mutate({ LLM_MODEL: "model" });
  const observer = new QueryObserver(client, instanceConfigurationsOptions);
  assert.deepEqual(observer.getCurrentResult().data, { EMAIL_HOST: "mail", ENABLE_SMTP: "1", LLM_MODEL: "model" });
});

await test("failed email disable rejects and preserves the enabled setting", async (t) => {
  const client = clientFor(t);
  client.setQueryData(instanceConfigurationsOptions.queryKey, [{ key: "ENABLE_SMTP", value: "1" }]);
  t.mock.method(InstanceService.prototype, "disableEmail", async () => {
    throw new Error("offline");
  });
  await assert.rejects(new MutationObserver(client, disableEmailOptions(client)).mutate(), /offline/);
  assert.equal(client.getQueryData(instanceConfigurationsOptions.queryKey)[0].value, "1");
});

await test("workspace pagination keeps earlier pages and uses the returned cursor", async (t) => {
  const client = clientFor(t);
  const cursors = [];
  t.mock.method(InstanceWorkspaceService.prototype, "list", async (cursor) => {
    cursors.push(cursor);
    return { results: [{ id: cursor ? "second" : "first" }], next_page_results: !cursor, next_cursor: "next" };
  });
  await client.fetchInfiniteQuery(instanceWorkspacesOptions);
  const observer = new InfiniteQueryObserver(client, instanceWorkspacesOptions);
  await observer.fetchNextPage();
  assert.deepEqual(cursors, [undefined, "next"]);
  assert.deepEqual(
    observer.getCurrentResult().data.pages.flatMap((page) => page.results.map((item) => item.id)),
    ["first", "second"]
  );
  assert.equal(observer.getCurrentResult().hasNextPage, false);
});

await test("clearing the admin session cancels a late private response and a late save cannot recreate it", async (t) => {
  const client = clientFor(t);
  const user = Promise.withResolvers();
  const save = Promise.withResolvers();
  t.mock.method(UserService.prototype, "adminDetails", () => user.promise);
  t.mock.method(InstanceService.prototype, "updateConfigurations", () => save.promise);
  client.setQueryData(instanceConfigurationsOptions.queryKey, [{ key: "EMAIL_HOST_PASSWORD", value: "private" }]);
  const request = client.fetchQuery(currentUserOptions);
  const rejection = assert.rejects(request);
  const mutation = new MutationObserver(client, updateConfigurationsOptions(client)).mutate({
    EMAIL_HOST_PASSWORD: "changed",
  });
  await new Promise((resolve) => setImmediate(resolve));
  client.clear();
  user.resolve({ id: "old-user" });
  save.resolve([{ key: "EMAIL_HOST_PASSWORD", value: "changed" }]);
  await Promise.all([rejection, mutation]);
  assert.equal(client.getQueryCache().getAll().length, 0);
});
