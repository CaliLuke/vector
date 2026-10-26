import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { joinProjectOptions, leaveProjectOptions, leaveWorkspaceOptions } =
  await import("../../apps/web/core/queries/permission-mutations.ts");

function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "user" });
  const workspaceKey = permissionKeys.workspaceMembership("user", "acme");
  const rolesKey = permissionKeys.workspaceProjectRoles("user", "acme", 20, "revision-1");
  client.setQueryData(workspaceKey, { is_active: true, role: 20, updated_at: "revision-1" });
  client.setQueryData(rolesKey, { first: 15, second: 5 });
  return { client, workspaceKey, rolesKey };
}

await test("accepted join stays successful when permission refresh fails and never guesses the joined role", async (t) => {
  const { client, rolesKey } = fixture(t);
  const read = new QueryObserver(client, {
    queryKey: rolesKey,
    queryFn: async () => {
      throw new Error("offline");
    },
    staleTime: Infinity,
    retry: false,
  });
  const unsubscribe = read.subscribe(() => {});
  t.after(unsubscribe);
  const join = t.mock.method(userService, "joinProject", async () => ({ message: "accepted" }));
  const mutation = new MutationObserver(client, joinProjectOptions(client, "user", "acme", "third"));
  await mutation.mutate();
  assert.equal(mutation.getCurrentResult().status, "success");
  assert.deepEqual(client.getQueryData(rolesKey), { first: 15, second: 5 });
  assert.equal(client.getQueryState(rolesKey).status, "error");
  assert.equal(join.mock.callCount(), 1);
});

await test("void project leave removes only the departed project after acceptance", async (t) => {
  const { client, rolesKey } = fixture(t);
  t.mock.method(userService, "leaveProject", async () => {});
  const mutation = new MutationObserver(client, leaveProjectOptions(client, "user", "acme", "first"));
  await mutation.mutate();
  assert.equal(mutation.getCurrentResult().status, "success");
  assert.deepEqual(client.getQueryData(rolesKey), { second: 5 });
});

await test("rejected project leave preserves access and a retry performs the primary write again", async (t) => {
  const { client, rolesKey } = fixture(t);
  let rejected = true;
  const leave = t.mock.method(userService, "leaveProject", async () => {
    if (rejected) throw new Error("not accepted");
  });
  const mutation = new MutationObserver(client, leaveProjectOptions(client, "user", "acme", "first"));
  await assert.rejects(mutation.mutate(), /not accepted/);
  assert.deepEqual(client.getQueryData(rolesKey), { first: 15, second: 5 });
  rejected = false;
  await mutation.mutate();
  assert.equal(leave.mock.callCount(), 2);
  assert.deepEqual(client.getQueryData(rolesKey), { second: 5 });
});

await test("void workspace leave immediately marks the captured membership inactive", async (t) => {
  const { client, workspaceKey } = fixture(t);
  t.mock.method(userService, "leaveWorkspace", async () => {});
  await new MutationObserver(client, leaveWorkspaceOptions(client, "user", "acme")).mutate();
  assert.equal(client.getQueryData(workspaceKey).is_active, false);
  assert.equal(client.getQueryData(workspaceKey).role, null);
});

async function verifyLateLeave(t, transition) {
  const { client, rolesKey } = fixture(t);
  let accept;
  let started;
  const requestStarted = new Promise((resolve) => {
    started = resolve;
  });
  t.mock.method(userService, "leaveProject", () => {
    started();
    return new Promise((resolve) => {
      accept = resolve;
    });
  });
  const write = new MutationObserver(client, leaveProjectOptions(client, "user", "acme", "first")).mutate();
  await requestStarted;
  if (transition === "same-account session") advanceIdentitySession(client);
  else client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { first: 20, replacement: 15 });
  accept();
  await write;
  assert.deepEqual(client.getQueryData(rolesKey), { first: 20, replacement: 15 });
}

await test("late leave preserves a restarted same-account session", (t) => verifyLateLeave(t, "same-account session"));
await test("late leave preserves a replacement permission query", (t) => verifyLateLeave(t, "query replacement"));
