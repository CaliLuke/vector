import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const identity = await import("../../apps/web/core/queries/identity.ts");
const session = await import("../../apps/web/core/queries/identity-session.ts");
const currentUserSettings = await import("../../apps/web/core/queries/current-user-settings.ts");
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
const { authService } = await import("../../apps/web/core/services/auth.service.ts");

function accountClient(t) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false, gcTime: Infinity } },
  });
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "alice", first_name: "Alice" });
  t.after(() => client.clear());
  return client;
}

function authorizedItem(client, permissionKeys) {
  const membershipKey = permissionKeys.workspaceMembership("alice", "acme");
  const rolesKey = permissionKeys.workspaceProjectRoles("alice", "acme", 15, "membership-rev");
  const itemKey = ["work-item", "detail", "alice", "acme", "p1", "i1"];
  client.setQueryData(membershipKey, {
    is_active: true,
    role: 15,
    updated_at: "membership-rev",
  });
  client.setQueryData(rolesKey, { p1: 15 });
  client.setQueryData(itemKey, { id: "i1" });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "p1",
    membershipRole: 15,
    membershipRevision: "membership-rev",
    projectRole: 15,
  };
  return { membershipKey, rolesKey, itemKey, scope };
}

await test("current-user reads forward cancellation and cannot restore a cleared session", async (t) => {
  const client = accountClient(t);
  let signal;
  const pending = Promise.withResolvers();
  t.mock.method(userService, "currentUser", async (requestSignal) => {
    signal = requestSignal;
    return pending.promise;
  });
  const request = client.fetchQuery({ ...identity.currentUserOptions(), staleTime: 0 });
  const cancelled = assert.rejects(request);
  identity.advanceIdentitySession(client);
  client.clear();
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "bob" });
  pending.resolve({ id: "alice" });
  await cancelled;
  assert.equal(signal.aborted, true);
  assert.equal(client.getQueryData(identity.currentUserOptions().queryKey).id, "bob");
});

await test("current-user writes use the server record and preserve it after a failed save", async (t) => {
  const client = accountClient(t);
  const update = t.mock.method(userService, "updateUser", async () => ({ id: "alice", first_name: "Canonical" }));
  const mutation = new MutationObserver(client, identity.updateCurrentUserOptions(client, "alice"));
  await mutation.mutate({ first_name: "  Canonical  " });
  assert.equal(client.getQueryData(identity.currentUserOptions().queryKey).first_name, "Canonical");
  update.mock.mockImplementation(async () => {
    throw new Error("offline");
  });
  await assert.rejects(mutation.mutate({ first_name: "Unsaved" }), /offline/);
  assert.equal(client.getQueryData(identity.currentUserOptions().queryKey).first_name, "Canonical");
});

await test("cached identity data in an error query cannot start a scoped read", async (t) => {
  const client = accountClient(t);
  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(
    client.fetchQuery({ ...identity.currentUserOptions(), staleTime: 0 }),
    /identity refresh failed/
  );
  assert.equal(client.getQueryState(identity.identityKeys.currentUser()).status, "error");
  assert.equal(client.getQueryData(identity.identityKeys.currentUser()).id, "alice");
  assert.equal(identity.isIdentityScopeCurrent(client, identity.captureIdentityScope(client, "alice")), true);
  assert.equal(identity.isIdentityScopeVerified(client, identity.captureIdentityScope(client, "alice")), false);

  const settingsRead = t.mock.method(userService, "currentUserSettings", async () => ({ id: "alice" }));
  await assert.rejects(
    client.fetchQuery({ ...currentUserSettings.currentUserSettingsOptions("alice"), staleTime: 0 }),
    /before settings could load/
  );
  assert.equal(settingsRead.mock.callCount(), 0);
});

await test("accepted item receipts retain lineage reconciliation through an identity refresh error", async (t) => {
  const client = accountClient(t);
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  const { captureItemLineageScope, captureItemReadScope } =
    await import("../../apps/web/core/queries/work-item-read-scope.ts");
  const { itemKey, scope } = authorizedItem(client, permissionKeys);
  const acceptedReceiptGuard = captureItemLineageScope(client, scope, itemKey);
  const newReadGuard = captureItemReadScope(client, scope, itemKey);

  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(client.fetchQuery({ ...identity.currentUserOptions(), staleTime: 0 }));
  assert.equal(acceptedReceiptGuard(), true);
  assert.equal(newReadGuard(), false);
  identity.advanceIdentitySession(client);
  client.clear();
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "bob" });
  assert.equal(
    acceptedReceiptGuard(),
    false,
    "an accepted receipt cannot reconcile into a replacement account session"
  );
});

await test("accepted item lineage never survives a retained permission 403", async (t) => {
  const client = accountClient(t);
  const { permissionKeys, workspaceMembershipOptions, workspaceProjectRolesOptions } =
    await import("../../apps/web/core/queries/permissions.ts");
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { captureItemLineageScope } = await import("../../apps/web/core/queries/work-item-read-scope.ts");
  const { membershipKey, rolesKey, itemKey, scope } = authorizedItem(client, permissionKeys);
  const membershipGuard = captureItemLineageScope(client, scope, itemKey);
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
    throw { response: { status: 403 } };
  });
  await assert.rejects(
    client.fetchQuery({ ...workspaceMembershipOptions("alice", "acme"), staleTime: 0 }),
    (error) => error?.response?.status === 403
  );
  assert.equal(
    client.getQueryData(membershipKey).is_active,
    true,
    "TanStack retains prior permission data on refresh error"
  );
  assert.equal(membershipGuard(), false, "a cached membership cannot override an explicit denial");

  client.setQueryData(membershipKey, { is_active: true, role: 15, updated_at: "membership-rev" });
  const roleGuard = captureItemLineageScope(client, scope, itemKey);
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => {
    throw { response: { status: 403 } };
  });
  await assert.rejects(
    client.fetchQuery({ ...workspaceProjectRolesOptions("alice", "acme", 15, "membership-rev"), staleTime: 0 }),
    (error) => error?.response?.status === 403
  );
  assert.deepEqual(client.getQueryData(rolesKey), { p1: 15 });
  assert.equal(roleGuard(), false, "a cached project role cannot override an explicit denial");
});

await test("a read rejects its late result if identity verification fails while it is in flight", async (t) => {
  const client = accountClient(t);
  const pending = Promise.withResolvers();
  const readStarted = Promise.withResolvers();
  const settingsRead = t.mock.method(userService, "currentUserSettings", async () => {
    readStarted.resolve();
    return pending.promise;
  });
  const request = client.fetchQuery({ ...currentUserSettings.currentUserSettingsOptions("alice"), staleTime: 0 });
  await readStarted.promise;

  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(
    client.fetchQuery({ ...identity.currentUserOptions(), staleTime: 0 }),
    /identity refresh failed/
  );
  pending.resolve({ id: "alice", email: "private@example.com" });
  await assert.rejects(request, /while settings loaded/);
  assert.equal(settingsRead.mock.callCount(), 1);
  assert.equal(client.getQueryState(currentUserSettings.currentUserSettingsOptions("alice").queryKey).status, "error");
  assert.equal(client.getQueryData(currentUserSettings.currentUserSettingsOptions("alice").queryKey), undefined);
});

await test("an accepted current-user write reconciles while a refresh leaves cached identity in error", async (t) => {
  const client = accountClient(t);
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  const update = t.mock.method(userService, "updateUser", async () => {
    started.resolve();
    return pending.promise;
  });
  const mutation = new MutationObserver(client, identity.updateCurrentUserOptions(client, "alice"));
  const save = mutation.mutate({ first_name: "Edited" });
  await started.promise;

  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(
    client.fetchQuery({ ...identity.currentUserOptions(), staleTime: 0 }),
    /identity refresh failed/
  );
  pending.resolve({ id: "alice", first_name: "Canonical response" });
  await save;

  assert.equal(update.mock.callCount(), 1);
  assert.equal(client.getQueryData(identity.identityKeys.currentUser()).first_name, "Canonical response");
  assert.equal(client.getQueryState(identity.identityKeys.currentUser()).status, "success");
});

await test("a new current-user write is rejected while cached identity is unverified", async (t) => {
  const client = accountClient(t);
  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(
    client.fetchQuery({ ...identity.currentUserOptions(), staleTime: 0 }),
    /identity refresh failed/
  );
  const update = t.mock.method(userService, "updateUser", async () => ({ id: "alice", first_name: "Wrong" }));
  const mutation = new MutationObserver(client, identity.updateCurrentUserOptions(client, "alice"));
  await assert.rejects(mutation.mutate({ first_name: "Wrong" }), /unverified/);
  assert.equal(update.mock.callCount(), 0);
});

await test("a late current-user save cannot replace the next session even for the same user ID", async (t) => {
  const client = accountClient(t);
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(userService, "updateUser", async () => {
    started.resolve();
    return pending.promise;
  });
  const mutation = new MutationObserver(client, identity.updateCurrentUserOptions(client, "alice"));
  const save = mutation.mutate({ first_name: "Old draft" });
  await started.promise;
  identity.advanceIdentitySession(client);
  client.clear();
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "alice", first_name: "New session" });
  pending.resolve({ id: "alice", first_name: "Old draft" });
  await save;
  assert.equal(client.getQueryData(identity.currentUserOptions().queryKey).first_name, "New session");
});

await test("a queued current-user write is rejected before HTTP after a session transition", async (t) => {
  const client = accountClient(t);
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  const update = t.mock.method(userService, "updateUser", async () => {
    started.resolve();
    return pending.promise;
  });
  const first = new MutationObserver(client, identity.updateCurrentUserOptions(client, "alice"));
  const second = new MutationObserver(client, identity.updateCurrentUserOptions(client, "alice"));
  const save = first.mutate({ first_name: "First" });
  await started.promise;
  const queued = assert.rejects(second.mutate({ first_name: "Queued" }), /session has changed/);
  identity.advanceIdentitySession(client);
  pending.resolve({ id: "alice", first_name: "First" });
  await save;
  await queued;
  assert.equal(update.mock.callCount(), 1);
});

await test("sign-out retains private state until the server confirms the session ended", async (t) => {
  const client = accountClient(t);
  const request = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(authService, "requestCSRFToken", async () => ({ csrf_token: "test-csrf" }));
  t.mock.method(authService, "post", async () => {
    started.resolve();
    return request.promise;
  });
  t.mock.method(userService, "currentUser", async () => null);
  const mutation = new MutationObserver(
    client,
    session.endSessionOptions(
      client,
      "alice",
      () => {
        identity.advanceIdentitySession(client);
        client.clear();
      },
      "sign-out"
    )
  );
  const logout = mutation.mutate();
  await started.promise;
  assert.equal(client.getQueryData(identity.currentUserOptions().queryKey).id, "alice");
  request.resolve({ status: 200 });
  await logout;
  assert.equal(client.getQueryCache().getAll().length, 0);
  assert.equal(identity.getIdentitySessionGeneration(client), 1);
});

await test("sign-out remains available with cached identity after a failed refresh", async (t) => {
  const client = accountClient(t);
  const currentUser = t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(
    client.fetchQuery({ ...identity.currentUserOptions(), staleTime: 0 }),
    /identity refresh failed/
  );
  const post = t.mock.method(authService, "post", async () => ({ status: 200 }));
  t.mock.method(authService, "requestCSRFToken", async () => ({ csrf_token: "test-csrf" }));
  currentUser.mock.mockImplementation(async () => null);
  let resetCount = 0;
  const mutation = new MutationObserver(
    client,
    session.endSessionOptions(client, "alice", () => resetCount++, "sign-out")
  );
  await mutation.mutate();
  assert.equal(post.mock.callCount(), 1);
  assert.equal(resetCount, 1);
});

await test("a late sign-out cannot reset a replacement session", async (t) => {
  const client = accountClient(t);
  const response = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(authService, "requestCSRFToken", async () => ({ csrf_token: "test-csrf" }));
  t.mock.method(authService, "post", async () => {
    started.resolve();
    return response.promise;
  });
  t.mock.method(userService, "currentUser", async () => null);
  let resetCount = 0;
  const mutation = new MutationObserver(
    client,
    session.endSessionOptions(client, "alice", () => resetCount++, "sign-out")
  );
  const logout = mutation.mutate();
  await started.promise;
  identity.advanceIdentitySession(client);
  client.clear();
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "bob", first_name: "Bob" });
  response.resolve({ status: 200 });
  await logout;
  assert.equal(resetCount, 0);
  assert.equal(client.getQueryData(identity.currentUserOptions().queryKey).id, "bob");
});

await test("a redirect that leaves the account signed in is a failed sign-out", async (t) => {
  const client = accountClient(t);
  t.mock.method(authService, "requestCSRFToken", async () => ({ csrf_token: "test-csrf" }));
  t.mock.method(authService, "post", async () => ({ status: 200 }));
  t.mock.method(userService, "currentUser", async () => ({ id: "alice" }));
  const mutation = new MutationObserver(
    client,
    session.endSessionOptions(client, "alice", () => client.clear(), "sign-out")
  );
  await assert.rejects(mutation.mutate(), /Unable to sign out/);
  assert.equal(client.getQueryData(identity.currentUserOptions().queryKey).id, "alice");
});

await test("current-user transport distinguishes signed out from server failure", async (t) => {
  const get = t.mock.method(userService, "get", async () => ({ status: 401, data: { detail: "Not authenticated" } }));
  assert.equal(await userService.currentUser(), null);
  get.mock.mockImplementation(async () => ({ status: 503, data: { detail: "Unavailable" } }));
  await assert.rejects(userService.currentUser(), (error) => error.status === 503);
});

await test("an invalid successful account response does not confirm sign-out", async (t) => {
  t.mock.method(userService, "get", async () => ({ status: 200, data: "<html>Sign in</html>" }));
  await assert.rejects(userService.currentUser(), /invalid account response/);
});

await test("identity services preserve network errors for failed reads and saves", async (t) => {
  const offline = new Error("Network unavailable");
  t.mock.method(userService, "get", async () => {
    throw offline;
  });
  t.mock.method(userService, "patch", async () => {
    throw offline;
  });
  await assert.rejects(userService.getCurrentUserProfile(), (error) => error === offline);
  await assert.rejects(userService.currentUserSettings(), (error) => error === offline);
  await assert.rejects(userService.currentUserEmailNotificationSettings(), (error) => error === offline);
  await assert.rejects(userService.updateUser({ display_name: "Alice" }), (error) => error === offline);
  await assert.rejects(userService.updateCurrentUserProfile({ language: "en" }), (error) => error === offline);
});

await test("an old subscription mutation cannot cancel or overwrite a replacement query", async (t) => {
  const client = accountClient(t);
  const { issueSubscriptionOptions, updateIssueSubscriptionOptions } =
    await import("../../apps/web/core/queries/issue-subscription.ts");
  const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
  const scope = { userId: "alice", workspaceSlug: "acme", projectId: "p1", issueId: "i1", serviceType: "issues" };
  const options = issueSubscriptionOptions(scope);
  client.setQueryData(options.queryKey, false);
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "subscribeToIssueNotifications", async () => {
    started.resolve();
    return pending.promise;
  });
  const mutation = new MutationObserver(client, updateIssueSubscriptionOptions(client, scope));
  const save = mutation.mutate(true);
  await started.promise;
  client.removeQueries({ queryKey: options.queryKey, exact: true });
  const replacement = Promise.withResolvers();
  let signal;
  const read = client.fetchQuery({
    ...options,
    queryFn: (context) => {
      signal = context.signal;
      return replacement.promise;
    },
  });
  pending.resolve();
  await save;
  assert.equal(signal.aborted, false);
  replacement.resolve(false);
  await read;
  assert.equal(client.getQueryData(options.queryKey), false);
});
