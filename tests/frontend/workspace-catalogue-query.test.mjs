import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const identity = await import("../../apps/web/core/queries/identity.ts");
const catalogue = await import("../../apps/web/core/queries/workspaces.ts");
const alpha = { id: "workspace-a", slug: "alpha", name: "Alpha" };
const beta = { id: "workspace-b", slug: "beta", name: "Beta" };
const key = catalogue.workspaceKeys.list("member");

function workspaceSession(t, workspaces = [alpha, beta]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(identity.identityKeys.currentUser(), { id: "member" });
  if (workspaces) client.setQueryData(key, workspaces);
  t.after(() => client.clear());
  return client;
}

function update(client, slug, change) {
  return new MutationObserver(client, catalogue.updateWorkspaceOptions(client, "member", slug)).mutate(change);
}

await test("workspace refresh replaces the full catalogue, removing revoked membership", async (t) => {
  const client = workspaceSession(t);
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => [beta]);
  await client.fetchQuery({ ...catalogue.workspacesOptions("member"), staleTime: 0 });
  assert.deepEqual(client.getQueryData(key), [beta]);
  assert.equal(
    catalogue.getWorkspaceDestination([beta], { workspace: { last_workspace_slug: "alpha" } }),
    "/create-workspace"
  );
  assert.equal(catalogue.getWorkspaceDestination([beta], { workspace: { fallback_workspace_slug: "beta" } }), "/beta");
  assert.equal(
    catalogue.getWorkspaceDestination([beta], {
      workspace: { last_workspace_slug: "alpha", fallback_workspace_slug: "beta" },
    }),
    "/beta"
  );
  assert.equal(
    catalogue.getWorkspaceDestination([alpha, beta], {
      workspace: { last_workspace_slug: "alpha", fallback_workspace_slug: "beta" },
    }),
    "/alpha"
  );
});

await test("concurrent saves of different workspaces retain both canonical server responses", async (t) => {
  const client = workspaceSession(t);
  const first = Promise.withResolvers();
  const entered = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "updateWorkspace", async (slug) => {
    if (slug === "alpha") {
      entered.resolve();
      return first.promise;
    }
    return { ...beta, name: "Canonical Beta" };
  });
  const slow = update(client, "alpha", { name: "Draft Alpha" });
  await entered.promise;
  await update(client, "beta", { name: "Draft Beta" });
  first.resolve({ ...alpha, name: "Canonical Alpha" });
  await slow;
  assert.deepEqual(
    client.getQueryData(key).map((workspace) => workspace.name),
    ["Canonical Alpha", "Canonical Beta"]
  );
});

await test("a failed workspace save preserves the prior catalogue", async (t) => {
  const client = workspaceSession(t);
  t.mock.method(WorkspaceService.prototype, "updateWorkspace", async () => {
    throw new Error("offline");
  });
  await assert.rejects(update(client, "alpha", { name: "Unsaved" }), /offline/);
  assert.deepEqual(client.getQueryData(key), [alpha, beta]);
});

[false, true].forEach((replaceSession) => {
  void test(`late workspace saves preserve a replacement ${replaceSession ? "session" : "query"}`, async (t) => {
    const client = workspaceSession(t);
    const request = Promise.withResolvers();
    const entered = Promise.withResolvers();
    t.mock.method(WorkspaceService.prototype, "updateWorkspace", async () => {
      entered.resolve();
      return request.promise;
    });
    const save = update(client, "alpha", { name: "Old draft" });
    await entered.promise;
    if (replaceSession) identity.advanceIdentitySession(client);
    client.removeQueries({ queryKey: key, exact: true });
    client.setQueryData(key, [{ ...alpha, name: "Replacement" }]);
    request.resolve({ ...alpha, name: "Late response" });
    await save;
    assert.equal(client.getQueryData(key)[0].name, "Replacement");
  });
});

await test("queued writes are rejected before transport when the account session changes", async (t) => {
  const client = workspaceSession(t);
  const request = Promise.withResolvers();
  const entered = Promise.withResolvers();
  const transport = t.mock.method(WorkspaceService.prototype, "updateWorkspace", async () => {
    entered.resolve();
    return request.promise;
  });
  const first = update(client, "alpha", { name: "First" });
  await entered.promise;
  const queued = assert.rejects(update(client, "alpha", { name: "Queued" }), /session has changed/);
  identity.advanceIdentitySession(client);
  request.resolve(alpha);
  await Promise.all([first, queued]);
  assert.equal(transport.mock.callCount(), 1);
});

await test("creation during initial loading cancels the old read and loads the whole authoritative list", async (t) => {
  const client = workspaceSession(t, null);
  const initial = Promise.withResolvers();
  let initialSignal;
  let reads = 0;
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async (signal) => {
    reads += 1;
    if (reads === 1) {
      initialSignal = signal;
      return initial.promise;
    }
    return [alpha, beta];
  });
  t.mock.method(WorkspaceService.prototype, "createWorkspace", async () => beta);
  const observer = new QueryObserver(client, catalogue.workspacesOptions("member"));
  const unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  await new MutationObserver(client, catalogue.createWorkspaceOptions(client, "member")).mutate({ name: "Beta" });
  initial.resolve([alpha]);
  assert.equal(initialSignal.aborted, true);
  assert.equal(reads, 2);
  assert.deepEqual(client.getQueryData(key), [alpha, beta]);
});

await test("deletion removes only the successfully deleted workspace", async (t) => {
  const client = workspaceSession(t);
  t.mock.method(WorkspaceService.prototype, "deleteWorkspace", async () => {});
  await new MutationObserver(client, catalogue.deleteWorkspaceOptions(client, "member", "alpha")).mutate();
  assert.deepEqual(client.getQueryData(key), [beta]);
});

await test("workspace transport receives cancellation and cannot repopulate a cleared session", async (t) => {
  const client = workspaceSession(t);
  const response = Promise.withResolvers();
  let requestSignal;
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async (signal) => {
    requestSignal = signal;
    return response.promise;
  });
  const loading = client.fetchQuery({ ...catalogue.workspacesOptions("member"), staleTime: 0 });
  const rejected = assert.rejects(loading);
  identity.advanceIdentitySession(client);
  client.clear();
  response.resolve([alpha]);
  await rejected;
  assert.equal(requestSignal.aborted, true);
  assert.equal(client.getQueryData(key), undefined);
});

await test("failed deletion retains membership and a retry can remove it", async (t) => {
  const client = workspaceSession(t);
  let unavailable = true;
  t.mock.method(WorkspaceService.prototype, "deleteWorkspace", async () => {
    if (unavailable) throw new Error("offline");
  });
  const deletion = new MutationObserver(client, catalogue.deleteWorkspaceOptions(client, "member", "alpha"));
  await assert.rejects(deletion.mutate(), /offline/);
  assert.deepEqual(client.getQueryData(key), [alpha, beta]);
  unavailable = false;
  await deletion.mutate();
  assert.deepEqual(client.getQueryData(key), [beta]);
});
