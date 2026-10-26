import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, MutationObserver } from "@tanstack/react-query";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { currentUserOptions, advanceIdentitySession, identityKeys } =
  await import("../../apps/web/core/queries/identity.ts");
const { currentUserSettingsOptions, refreshCurrentUserSettings } =
  await import("../../apps/web/core/queries/current-user-settings.ts");
const { ProjectPublishService } = await import("../../apps/web/core/services/project/project-publish.service.ts");
const { projectPublishingOptions, projectPublishingMutationOptions } =
  await import("../../apps/web/core/queries/project-publishing.ts");
const { emailNotificationSettingsOptions, emailNotificationSettingsMutationOptions } =
  await import("../../apps/web/core/queries/user-settings.ts");
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
const publishing = {
  id: "publication",
  anchor: "public",
  is_comments_enabled: false,
  is_reactions_enabled: false,
  is_votes_enabled: false,
  view_props: { list: true, kanban: true },
};
function publishingScope(workspaceSlug) {
  return {
    accountId: "settings-user",
    sessionGeneration: 0,
    workspaceSlug,
    workspaceRole: 20,
    membershipRevision: "2026-10-24T00:00:00Z",
    projectRoles: [["project", 20]],
    projectId: "project",
    projectRole: 20,
  };
}
async function seedPublishingAccess(client, workspaceSlug) {
  client.setQueryData(currentUserOptions().queryKey, { id: "settings-user" });
  await seedPermissionQueries(client, {
    userId: "settings-user",
    workspaceSlug,
    workspaceRole: 20,
    projectRoles: { project: 20 },
  });
}
function clientFor(t) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false, gcTime: Infinity } },
  });
  t.after(() => client.clear());
  return client;
}
await test("publishing failures retain settings and successful changes affect only their project", async (t) => {
  const client = clientFor(t);
  await seedPublishingAccess(client, "first");
  await seedPublishingAccess(client, "other");
  const first = projectPublishingOptions(client, publishingScope("first")).queryKey;
  const other = projectPublishingOptions(client, publishingScope("other")).queryKey;
  client.setQueryData(first, publishing);
  client.setQueryData(other, publishing);
  let fail = true;
  t.mock.method(ProjectPublishService.prototype, "updatePublishSettings", async () => {
    if (fail) throw new Error("offline");
    return { ...publishing, is_comments_enabled: true };
  });
  t.mock.method(ProjectPublishService.prototype, "unpublishProject", async () => {});
  const mutation = new MutationObserver(client, projectPublishingMutationOptions(client, publishingScope("first")));
  const action = { type: "save", id: publishing.id, settings: { is_comments_enabled: true } };
  await assert.rejects(mutation.mutate(action), /offline/);
  assert.deepEqual(client.getQueryData(first), publishing);
  fail = false;
  await mutation.mutate(action);
  assert.equal(client.getQueryData(first).is_comments_enabled, true);
  assert.deepEqual(client.getQueryData(other), publishing);
  await mutation.mutate({ type: "unpublish", id: publishing.id });
  assert.equal(client.getQueryData(first).anchor, undefined);
  assert.equal(client.getQueryData(other).anchor, "public");
});

await test("current-user settings reject late responses after the identity generation changes", async (t) => {
  const client = clientFor(t);
  const userId = "settings-user";
  client.setQueryData(currentUserOptions().queryKey, { id: userId });
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  let signal;
  t.mock.method(userService, "currentUserSettings", async (bustCache, requestSignal) => {
    assert.equal(bustCache, false);
    signal = requestSignal;
    started.resolve();
    return pending.promise;
  });
  const options = currentUserSettingsOptions(userId);
  const request = client.fetchQuery({ ...options, staleTime: 0 });
  const rejected = assert.rejects(request, /account session changed while settings loaded/i);
  await started.promise;
  advanceIdentitySession(client);
  pending.resolve({ id: userId, workspace: { last_workspace_slug: "old-session" } });
  await rejected;
  assert.ok(signal instanceof AbortSignal);
  assert.equal(signal.aborted, false);
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("cancelling current-user settings aborts the request and discards a late result", async (t) => {
  const client = clientFor(t);
  const userId = "settings-user";
  client.setQueryData(currentUserOptions().queryKey, { id: userId });
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  let signal;
  t.mock.method(userService, "currentUserSettings", async (_bustCache, requestSignal) => {
    signal = requestSignal;
    started.resolve();
    return pending.promise;
  });
  const options = currentUserSettingsOptions(userId);
  const request = client.fetchQuery({ ...options, staleTime: 0 }).catch((error) => error);
  await started.promise;
  await client.cancelQueries({ queryKey: options.queryKey, exact: true });
  assert.equal(signal.aborted, true);
  pending.resolve({ id: userId, workspace: { last_workspace_slug: "late" } });
  await request;
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("current-user settings refresh uses a fresh query while retaining the cache-busting endpoint option", async (t) => {
  const client = clientFor(t);
  const userId = "settings-user";
  client.setQueryData(currentUserOptions().queryKey, { id: userId });
  const settings = { id: userId, workspace: { last_workspace_slug: "new-workspace" } };
  let request;
  t.mock.method(userService, "currentUserSettings", async (bustCache, signal) => {
    request = { bustCache, signal };
    return settings;
  });
  const result = await refreshCurrentUserSettings(client, userId, true);
  assert.deepEqual(result, settings);
  assert.equal(request.bustCache, true);
  assert.ok(request.signal instanceof AbortSignal);
  assert.deepEqual(client.getQueryData(identityKeys.settings(userId)), settings);
});

await test("current-user settings reject data returned for a different account", async (t) => {
  const client = clientFor(t);
  const userId = "settings-user";
  client.setQueryData(currentUserOptions().queryKey, { id: userId });
  t.mock.method(userService, "currentUserSettings", async () => ({ id: "other-user" }));
  const options = currentUserSettingsOptions(userId);
  await assert.rejects(client.fetchQuery({ ...options, staleTime: 0 }), /different account/i);
  assert.equal(client.getQueryData(options.queryKey), undefined);
});
await Promise.all(
  ["publishing", "email notifications"].map((resource) =>
    test(`late ${resource} saves cannot replace a new session's cached data`, async (t) => {
      const client = clientFor(t);
      const pending = Promise.withResolvers();
      const started = Promise.withResolvers();
      const userId = "settings-user";
      client.setQueryData(currentUserOptions().queryKey, { id: userId });
      if (resource === "publishing") await seedPublishingAccess(client, "first");
      const key =
        resource === "publishing"
          ? projectPublishingOptions(client, publishingScope("first")).queryKey
          : emailNotificationSettingsOptions(userId).queryKey;
      const original = resource === "publishing" ? publishing : { property_change: false };
      client.setQueryData(key, original);
      const network = () => {
        started.resolve();
        return pending.promise;
      };
      let options, action;
      if (resource === "publishing") {
        t.mock.method(ProjectPublishService.prototype, "updatePublishSettings", network);
        options = projectPublishingMutationOptions(client, publishingScope("first"));
        action = { type: "save", id: publishing.id, settings: { is_comments_enabled: true } };
      } else {
        t.mock.method(userService, "updateCurrentUserEmailNotificationSettings", network);
        options = emailNotificationSettingsMutationOptions(client, userId);
        action = { property_change: true };
      }
      const mutation = new MutationObserver(client, options);
      const saved = mutation.mutate(action);
      await started.promise;
      client.clear();
      client.setQueryData(currentUserOptions().queryKey, { id: userId });
      client.setQueryData(key, original);
      pending.resolve({ ...publishing, is_comments_enabled: true });
      await saved;
      assert.deepEqual(client.getQueryData(key), original);
    })
  )
);

await test("email preference mutation writes canonical data only to its captured identity query", async (t) => {
  const client = clientFor(t);
  const userId = "settings-user";
  const key = emailNotificationSettingsOptions(userId).queryKey;
  const original = { property_change: false, issue_completed: false };
  const canonical = { property_change: true, issue_completed: false };
  client.setQueryData(currentUserOptions().queryKey, { id: userId });
  client.setQueryData(key, original);
  t.mock.method(userService, "updateCurrentUserEmailNotificationSettings", async () => canonical);
  const mutation = new MutationObserver(client, emailNotificationSettingsMutationOptions(client, userId));
  await mutation.mutate({ property_change: true });
  assert.deepEqual(client.getQueryData(key), canonical);
  assert.deepEqual(client.getQueryData(emailNotificationSettingsOptions("other-user").queryKey), undefined);
});

await test("late email preference responses cannot write after the current account changes", async (t) => {
  const client = clientFor(t);
  const oldUserId = "old-user";
  const nextUserId = "next-user";
  const oldKey = emailNotificationSettingsOptions(oldUserId).queryKey;
  const nextKey = emailNotificationSettingsOptions(nextUserId).queryKey;
  const oldSettings = { property_change: false };
  const nextSettings = { property_change: true };
  client.setQueryData(currentUserOptions().queryKey, { id: oldUserId });
  client.setQueryData(oldKey, oldSettings);
  client.setQueryData(nextKey, nextSettings);
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(userService, "updateCurrentUserEmailNotificationSettings", () => {
    started.resolve();
    return pending.promise;
  });
  const mutation = new MutationObserver(client, emailNotificationSettingsMutationOptions(client, oldUserId));
  const response = mutation.mutate({ property_change: true });
  await started.promise;
  client.setQueryData(currentUserOptions().queryKey, { id: nextUserId });
  pending.resolve({ property_change: true });
  await response;
  assert.deepEqual(client.getQueryData(oldKey), oldSettings);
  assert.deepEqual(client.getQueryData(nextKey), nextSettings);
});

await test("late email preference saves do not cancel a replacement settings query", async (t) => {
  const client = clientFor(t);
  const userId = "settings-user";
  const key = emailNotificationSettingsOptions(userId).queryKey;
  client.setQueryData(currentUserOptions().queryKey, { id: userId });
  client.setQueryData(key, { property_change: false });
  const mutationPending = Promise.withResolvers();
  const mutationStarted = Promise.withResolvers();
  const queryPending = Promise.withResolvers();
  const queryStarted = Promise.withResolvers();
  let replacementSignal;
  t.mock.method(userService, "updateCurrentUserEmailNotificationSettings", () => {
    mutationStarted.resolve();
    return mutationPending.promise;
  });
  t.mock.method(userService, "currentUserEmailNotificationSettings", (_signal) => {
    // Query functions pass the AbortSignal as the first and only argument.
    replacementSignal = _signal;
    queryStarted.resolve();
    return queryPending.promise;
  });
  const mutation = new MutationObserver(client, emailNotificationSettingsMutationOptions(client, userId));
  const saving = mutation.mutate({ property_change: true });
  await mutationStarted.promise;
  client.removeQueries({ queryKey: key, exact: true });
  const replacementQuery = client.fetchQuery({ ...emailNotificationSettingsOptions(userId), staleTime: 0 });
  await queryStarted.promise;
  mutationPending.resolve({ property_change: true });
  await saving;
  assert.equal(replacementSignal.aborted, false);
  queryPending.resolve({ property_change: false });
  assert.deepEqual(await replacementQuery, { property_change: false });
  assert.deepEqual(client.getQueryData(key), { property_change: false });
});
