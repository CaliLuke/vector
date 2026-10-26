import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { identityKeys, advanceIdentitySession, getIdentitySessionGeneration } =
  await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { publishedItemEditSourceOptions } = await import("../../apps/web/core/queries/published-item-edit.ts");
const { forgetItemReads } = await import("../../apps/web/core/queries/work-item-access.ts");
const { workItemEditsService: edits } = await import("../../apps/web/core/services/work-item-edits.service.ts");
const { workItemSchemaService: schema } = await import("../../apps/web/core/services/work-item-schema.service.ts");
function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  client.setQueryData(permissionKeys.workspaceMembership("alice", "acme"), {
    is_active: true,
    role: 15,
    updated_at: "v1",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "v1"), { p: 15 });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "p",
    issueId: "item",
    membershipRole: 15,
    membershipRevision: "v1",
    projectRole: 15,
    generation: getIdentitySessionGeneration(client),
  };
  const f = {
    client,
    scope,
    source: {
      item: { id: "item", sequence_id: 1, project_id: "p", type_id: "type", custom_properties: { field: "canonical" } },
      source_version: "v1",
    },
    schema: { type_id: "type", revision: 1, stored_values: { field: "another read" } },
  };
  t.mock.method(edits, "source", async () => structuredClone(f.source));
  t.mock.method(schema, "schema", async () => structuredClone(f.schema));
  return f;
}
await test("retained source stays fixed per intent and a new edit intent reads fresh values", async (t) => {
  const f = fixture(t);
  const options = publishedItemEditSourceOptions(f.scope, "first");
  const first = await f.client.fetchQuery(options);
  assert.equal(first.source.item.custom_properties.field, "canonical");
  assert.equal(first.schema.stored_values.field, "another read");
  f.source.source_version = "v2";
  assert.equal((await f.client.fetchQuery(options)).source.source_version, "v1");
  assert.equal(
    (await f.client.fetchQuery(publishedItemEditSourceOptions(f.scope, "second"))).source.source_version,
    "v2"
  );
});
await test("separate reads crossing a type change cannot initialize an editor", async (t) => {
  const f = fixture(t);
  f.schema.type_id = "other";
  await assert.rejects(f.client.fetchQuery(publishedItemEditSourceOptions(f.scope, "first")), /changed while loading/);
});
await test("a late source response from an earlier session cannot populate the current cache", async (t) => {
  const f = fixture(t),
    started = Promise.withResolvers(),
    response = Promise.withResolvers();
  t.mock.method(edits, "source", () => {
    started.resolve();
    return response.promise;
  });
  const options = publishedItemEditSourceOptions(f.scope, "first");
  const rejected = assert.rejects(f.client.fetchQuery(options));
  await started.promise;
  advanceIdentitySession(f.client);
  response.resolve(f.source);
  await rejected;
  assert.equal(f.client.getQueryData(options.queryKey), undefined);
});
await test("known item denial prevents a newly mounted editor from sending source reads", async (t) => {
  const f = fixture(t);
  forgetItemReads(f.client, f.scope);
  t.mock.method(edits, "source", async () => assert.fail("Source read after a known denial"));
  await assert.rejects(f.client.fetchQuery(publishedItemEditSourceOptions(f.scope, "first")), /access changed/);
});

await test("a late access denial cannot block a replacement account session", async (t) => {
  const f = fixture(t),
    started = Promise.withResolvers(),
    response = Promise.withResolvers();
  const { propertyHistoryCatalogOptions } = await import("../../apps/web/core/queries/work-item-property-history.ts");
  const { workItemPropertyHistoryService: history } =
    await import("../../apps/web/core/services/work-item-property-history.service.ts");
  const { itemAccessOptions } = await import("../../apps/web/core/queries/work-item-access.ts");
  t.mock.method(history, "catalog", () => {
    started.resolve();
    return response.promise;
  });
  const rejected = assert.rejects(f.client.fetchQuery(propertyHistoryCatalogOptions(f.scope)));
  await started.promise;
  advanceIdentitySession(f.client);
  response.reject({ response: { status: 403 } });
  await rejected;
  assert.equal(f.client.getQueryData(itemAccessOptions(f.scope).queryKey), undefined);
  assert.equal(
    f.client.getQueryData(
      itemAccessOptions({ ...f.scope, generation: getIdentitySessionGeneration(f.client) }).queryKey
    ),
    undefined
  );
});

await test("workspace invalidation cannot silently replace an open edit baseline", async (t) => {
  const f = fixture(t),
    options = publishedItemEditSourceOptions(f.scope, "first");
  await f.client.fetchQuery(options);
  const observer = new QueryObserver(f.client, options),
    unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  f.source.source_version = "v2";
  await f.client.invalidateQueries({ queryKey: ["workspace", "acme"] });
  assert.equal(observer.getCurrentResult().data.source.source_version, "v1");
  await observer.refetch();
  assert.equal(observer.getCurrentResult().data.source.source_version, "v2");
});
