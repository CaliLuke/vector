import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const identity = await import("../../apps/web/core/queries/identity.ts");
const links = await import("../../apps/web/core/queries/mcp-link.ts");
const { mcpLinkService } = await import("../../apps/web/core/services/mcp-link.service.ts");
const { authService } = await import("../../apps/web/core/services/auth.service.ts");

const pendingStatus = {
  issuer: "https://issuer.example",
  connected: false,
  subject: null,
  pending: { id: "flow-1", subject: "subject-1", expires_at: "2026-10-27T12:00:00Z" },
};

function accountClient(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "alice" });
  client.setQueryData(links.mcpLinkKey("alice"), pendingStatus);
  t.after(() => client.clear());
  return client;
}

await test("MCP link reads cannot publish a subject after the account session changes", async (t) => {
  const client = accountClient(t);
  const pending = Promise.withResolvers();
  t.mock.method(mcpLinkService, "status", () => pending.promise);
  const request = client.fetchQuery(links.mcpLinkOptions("alice"));
  const rejected = assert.rejects(request, /session changed/);
  identity.advanceIdentitySession(client);
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "bob" });
  pending.resolve({ ...pendingStatus, pending: { ...pendingStatus.pending, subject: "new-private-subject" } });
  await rejected;
  assert.equal(client.getQueryData(links.mcpLinkKey("alice")).pending.subject, "subject-1");
  assert.equal(client.getQueryData(links.mcpLinkKey("bob")), undefined);
});

await test("MCP confirmation rechecks the account after obtaining CSRF and before sending a write", async (t) => {
  const client = accountClient(t);
  const csrfStarted = Promise.withResolvers();
  const csrf = Promise.withResolvers();
  t.mock.method(authService, "requestCSRFToken", () => {
    csrfStarted.resolve();
    return csrf.promise;
  });
  let writes = 0;
  t.mock.method(mcpLinkService, "post", async () => {
    writes++;
    return {};
  });
  const mutation = new MutationObserver(
    client,
    links.mcpLinkChangeOptions(client, "alice", (id, guard) => mcpLinkService.confirm(id, guard))
  );
  const result = assert.rejects(mutation.mutate("flow-1"), /Reload its status/);
  await csrfStarted.promise;
  identity.advanceIdentitySession(client);
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "bob" });
  csrf.resolve({ csrf_token: "test-csrf" });
  await result;
  assert.equal(writes, 0);
});

await test("a lost confirmation response reloads canonical status without retrying the write", async (t) => {
  const client = accountClient(t);
  let serverStatus = pendingStatus;
  t.mock.method(mcpLinkService, "status", async () => serverStatus);
  await client.fetchQuery(links.mcpLinkOptions("alice"));
  const query = new QueryObserver(client, { ...links.mcpLinkOptions("alice"), staleTime: Infinity });
  const unsubscribe = query.subscribe(() => {});
  t.after(unsubscribe);
  await query.refetch();
  let writes = 0;
  const mutation = new MutationObserver(
    client,
    links.mcpLinkChangeOptions(client, "alice", async () => {
      writes++;
      serverStatus = { issuer: pendingStatus.issuer, connected: true, subject: "subject-1", pending: null };
      throw new Error("Response lost");
    })
  );
  await assert.rejects(mutation.mutate(), /Response lost/);
  assert.equal(writes, 1);
  assert.equal(client.getQueryData(links.mcpLinkKey("alice")).connected, true);
  assert.equal(client.getQueryData(links.mcpLinkKey("alice")).pending, null);
});

await test("removed connection queries cannot authorize a queued write", async (t) => {
  const client = accountClient(t);
  let writes = 0;
  const options = links.mcpLinkChangeOptions(client, "alice", async () => {
    writes++;
  });
  client.removeQueries({ queryKey: links.mcpLinkKey("alice"), exact: true });
  client.setQueryData(links.mcpLinkKey("alice"), pendingStatus);
  await assert.rejects(new MutationObserver(client, options).mutate(), /session changed/);
  assert.equal(writes, 0);
});

await test("MCP connection errors discard transport details and credentials", async (t) => {
  t.mock.method(authService, "requestCSRFToken", async () => ({ csrf_token: "private-token-marker" }));
  t.mock.method(mcpLinkService, "post", async () => {
    throw Object.assign(new Error("private-error-marker"), { config: { data: "private-body-marker" } });
  });
  await assert.rejects(
    mcpLinkService.confirm("flow-1", () => {}),
    (error) => {
      assert.match(error.message, /Reload its status/);
      assert.doesNotMatch(String(error), /private-/);
      assert.equal(error.config, undefined);
      assert.equal(error.cause, undefined);
      return true;
    }
  );
});
