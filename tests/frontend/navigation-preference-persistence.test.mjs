import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { advanceIdentitySession, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const preferences = await import("../../apps/web/core/queries/navigation-preferences.ts");

function createClient(t, userId = "preference-user") {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity },
      mutations: { retry: false, gcTime: Infinity },
    },
  });
  client.setQueryData(identityKeys.currentUser(), { id: userId });
  t.after(async () => {
    await client.cancelQueries();
    client.clear();
  });
  return { client, userId };
}

function waitForQueryState(client, queryKey, predicate) {
  const queryCache = client.getQueryCache();
  const matches = () => queryCache.find({ queryKey, exact: true });
  const existing = matches()?.state;
  if (predicate(existing)) return Promise.resolve(existing);
  return new Promise((resolve, reject) => {
    let timeout;
    const unsubscribe = queryCache.subscribe((event) => {
      if (event.type !== "updated" || event.query.queryHash !== matches()?.queryHash) return;
      if (!predicate(event.query.state)) return;
      clearTimeout(timeout);
      unsubscribe();
      resolve(event.query.state);
    });
    timeout = setTimeout(() => {
      unsubscribe();
      reject(new Error(`Query did not reach the expected state: ${JSON.stringify(queryKey)}`));
    }, 3000);
  });
}

await test("sidebar PATCH acknowledgement merges submitted values and survives a failed refresh", async (t) => {
  const { client, userId } = createClient(t);
  const slug = "acme";
  const query = preferences.sidebarNavigationOptions(userId, slug);
  const queryKey = preferences.navigationPreferenceKeys.sidebar(userId, slug);
  const refreshStarted = Promise.withResolvers();
  const initialRead = Promise.withResolvers();
  let readCount = 0;
  let initialSignal;
  t.mock.method(WorkspaceService.prototype, "fetchSidebarNavigationPreferences", async (_slug, signal) => {
    readCount += 1;
    if (readCount === 1) {
      initialSignal = signal;
      return initialRead.promise;
    }
    refreshStarted.resolve();
    throw new Error("canonical refresh unavailable");
  });
  const savedPatch = [{ key: "stickies", is_pinned: true, sort_order: 4 }];
  t.mock.method(WorkspaceService.prototype, "updateBulkSidebarPreferences", async (receivedSlug, receivedPatch) => {
    assert.equal(receivedSlug, slug);
    assert.deepEqual(receivedPatch, savedPatch);
    return { message: "Saved" };
  });

  const observedQuery = new QueryObserver(client, query);
  const unsubscribe = observedQuery.subscribe(() => {});
  t.after(unsubscribe);
  const initialRequest = client.fetchQuery({ ...query, staleTime: 0 }).catch((error) => error);
  await waitForQueryState(client, queryKey, (state) => state?.fetchStatus === "fetching");

  const mutation = new MutationObserver(client, preferences.patchSidebarNavigationOptions(client, userId, slug));
  await mutation.mutate(savedPatch);
  await refreshStarted.promise;
  await waitForQueryState(client, queryKey, (state) => state?.fetchStatus === "idle" && Boolean(state.error));
  await initialRequest;

  assert.equal(initialSignal.aborted, true, "the superseded read is cancelled before committing the accepted patch");
  assert.deepEqual(client.getQueryData(queryKey), {
    stickies: { is_pinned: true, sort_order: 4 },
  });
  assert.notDeepEqual(client.getQueryData(queryKey), { message: "Saved" });
});

await test("project navigation mutation replaces the query with canonical properties and preserves unrelated fields", async (t) => {
  const { client, userId } = createClient(t);
  const slug = "acme";
  const queryKey = preferences.navigationPreferenceKeys.workspaceProperties(userId, slug);
  const prior = {
    navigation_control_preference: "ACCORDION",
    navigation_project_limit: 4,
    rich_filters: { and: [] },
    display_filters: { show_empty_groups: true },
    display_properties: { sub_work_items: true },
  };
  const canonical = {
    ...prior,
    navigation_project_limit: 8,
    display_filters: { show_empty_groups: false },
    display_properties: { sub_work_items: false },
  };
  client.setQueryData(queryKey, prior);
  t.mock.method(WorkspaceService.prototype, "patchWorkspaceFilters", async (receivedSlug, patch) => {
    assert.equal(receivedSlug, slug);
    assert.deepEqual(patch, { navigation_project_limit: 8 });
    return canonical;
  });

  const mutation = new MutationObserver(client, preferences.patchWorkspaceNavigationOptions(client, userId, slug));
  await mutation.mutate({ navigation_project_limit: 8 });

  assert.deepEqual(client.getQueryData(queryKey), canonical);
});

await test("failed project preference write remains failed and can be retried", async (t) => {
  const { client, userId } = createClient(t);
  const slug = "acme";
  const queryKey = preferences.navigationPreferenceKeys.workspaceProperties(userId, slug);
  const prior = {
    navigation_control_preference: "TABBED",
    navigation_project_limit: 3,
    rich_filters: { and: [] },
    display_filters: { show_empty_groups: true },
    display_properties: { sub_work_items: true },
  };
  const canonical = { ...prior, navigation_project_limit: 9 };
  client.setQueryData(queryKey, prior);
  let unavailable = true;
  t.mock.method(WorkspaceService.prototype, "patchWorkspaceFilters", async () => {
    if (unavailable) throw new Error("offline");
    return canonical;
  });
  const mutation = new MutationObserver(client, preferences.patchWorkspaceNavigationOptions(client, userId, slug));

  await assert.rejects(mutation.mutate({ navigation_project_limit: 9 }), /offline/);
  assert.deepEqual(client.getQueryData(queryKey), prior);
  unavailable = false;
  await mutation.mutate({ navigation_project_limit: 9 });
  assert.deepEqual(client.getQueryData(queryKey), canonical);
});

await test("old-session queued preference writes never reach transport", async (t) => {
  const { client, userId } = createClient(t);
  const slug = "acme";
  const queryKey = preferences.navigationPreferenceKeys.workspaceProperties(userId, slug);
  const prior = {
    navigation_control_preference: "ACCORDION",
    navigation_project_limit: 5,
    rich_filters: { and: [] },
    display_filters: { show_empty_groups: true },
    display_properties: { sub_work_items: true },
  };
  client.setQueryData(queryKey, prior);
  const started = Promise.withResolvers();
  const pending = Promise.withResolvers();
  let calls = 0;
  t.mock.method(WorkspaceService.prototype, "patchWorkspaceFilters", async () => {
    calls += 1;
    started.resolve();
    return pending.promise;
  });
  const mutation = new MutationObserver(client, preferences.patchWorkspaceNavigationOptions(client, userId, slug));
  const firstWrite = mutation.mutate({ navigation_project_limit: 6 });
  await started.promise;
  const queuedWrite = assert.rejects(mutation.mutate({ navigation_project_limit: 7 }), /session has changed/i);
  advanceIdentitySession(client);
  pending.resolve({ ...prior, navigation_project_limit: 6 });

  await Promise.all([firstWrite, queuedWrite]);
  assert.equal(calls, 1);
  assert.deepEqual(client.getQueryData(queryKey), prior);
});

await test("late preference responses do not cancel a replacement query or overwrite its result", async (t) => {
  const { client, userId } = createClient(t);
  const slug = "acme";
  const query = preferences.workspacePropertiesOptions(userId, slug);
  const queryKey = preferences.navigationPreferenceKeys.workspaceProperties(userId, slug);
  const prior = {
    navigation_control_preference: "ACCORDION",
    navigation_project_limit: 4,
    rich_filters: { and: [] },
    display_filters: { show_empty_groups: true },
    display_properties: { sub_work_items: true },
  };
  const replacement = { ...prior, navigation_project_limit: 2 };
  client.setQueryData(queryKey, prior);
  const mutationStarted = Promise.withResolvers();
  const patchResponse = Promise.withResolvers();
  const replacementReadStarted = Promise.withResolvers();
  const replacementReadResponse = Promise.withResolvers();
  let replacementSignal;
  t.mock.method(WorkspaceService.prototype, "patchWorkspaceFilters", () => {
    mutationStarted.resolve();
    return patchResponse.promise;
  });
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceFilters", (_slug, signal) => {
    replacementSignal = signal;
    replacementReadStarted.resolve();
    return replacementReadResponse.promise;
  });

  const mutation = new MutationObserver(client, preferences.patchWorkspaceNavigationOptions(client, userId, slug));
  const save = mutation.mutate({ navigation_project_limit: 10 });
  await mutationStarted.promise;
  client.removeQueries({ queryKey, exact: true });
  const replacementQuery = client.fetchQuery({ ...query, staleTime: 0 });
  await replacementReadStarted.promise;
  patchResponse.resolve({ ...prior, navigation_project_limit: 10 });
  await save;

  assert.equal(replacementSignal.aborted, false);
  replacementReadResponse.resolve(replacement);
  await replacementQuery;
  assert.deepEqual(client.getQueryData(queryKey), replacement);
});
