import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider, skipToken } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { waitForQueryState } from "./helpers/query-state.mjs";
import { waitForMutation } from "./helpers/wait-for-mutation.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const {
  applyProjectUserPropertiesPatches,
  projectUserPropertiesKeys,
  projectUserPropertiesOptions,
  patchProjectUserPropertiesOptions,
} = await import("../../apps/web/core/queries/project-user-properties.ts");
const { usePatchProjectUserProperties, useProjectUserProperties } =
  await import("../../apps/web/core/hooks/use-project-user-properties.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");

function createClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
}

function response(overrides = {}) {
  return {
    preferences: {
      pages: { block_display: true },
      navigation: { default_tab: "issues", hide_in_more_menu: ["cycles"] },
    },
    rich_filters: { and: [] },
    display_filters: { layout: "list", order_by: "created_at", show_empty_groups: false },
    display_properties: { assignee: true, labels: true, key: true },
    sort_order: 25,
    ...overrides,
  };
}

await test("navigation intents apply after other preference fields in the same patch", () => {
  const updated = applyProjectUserPropertiesPatches(response(), [
    {
      preferences: { navigation: { default_tab: "pages" } },
      navigationAction: { type: "hide-tab", tabKey: "modules" },
    },
  ]);

  assert.deepEqual(updated.preferences.navigation, {
    default_tab: "pages",
    hide_in_more_menu: ["cycles", "modules"],
  });
});

const scope = { accountId: "alice", workspaceSlug: "acme", projectId: "project-a" };

function seed(client, properties = response()) {
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  client.setQueryData(projectUserPropertiesKeys.detail(scope), properties);
}

function Probe({ access, onRender }) {
  const query = useProjectUserProperties(scope.workspaceSlug, scope.projectId, access);
  const mutation = usePatchProjectUserProperties(scope.workspaceSlug, scope.projectId, access.canAccess);
  onRender({ query, mutate: mutation.mutateAsync });
  return React.createElement("output", null, JSON.stringify(query.data));
}

function ObserverPair({ cacheOnlyFirst, onRender }) {
  const observers = cacheOnlyFirst
    ? [
        { access: { canAccess: true, fetchEnabled: false }, key: "cache-only" },
        { access: { canAccess: true, fetchEnabled: true }, key: "active" },
      ]
    : [
        { access: { canAccess: true, fetchEnabled: true }, key: "active" },
        { access: { canAccess: true, fetchEnabled: false }, key: "cache-only" },
      ];
  return React.createElement(
    React.Fragment,
    null,
    ...observers.map(({ access, key }) =>
      React.createElement(Probe, { key, access, onRender: (state) => onRender(key, state) })
    )
  );
}

async function renderProbe(t, client, access = { canAccess: true, fetchEnabled: true }) {
  const environment = await createReactDomTestEnvironment();
  let current;
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(Probe, {
        access,
        onRender: (state) => {
          current = state;
        },
      })
    )
  );
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  return { environment, current: () => current };
}

async function setupQueuedPreferenceWrites(t, initial = response()) {
  const client = createClient();
  seed(client, initial);
  const ui = await renderProbe(t, client);
  const firstStarted = Promise.withResolvers();
  const firstWrite = Promise.withResolvers();
  const secondStarted = Promise.withResolvers();
  const secondWrite = Promise.withResolvers();
  const sentPatches = [];
  let canonicalResponse = initial;
  let writes = 0;

  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", (_slug, _project, patch) => {
    sentPatches.push(patch);
    writes++;
    if (writes === 1) {
      firstStarted.resolve();
      return firstWrite.promise;
    }
    secondStarted.resolve();
    return secondWrite.promise;
  });
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => canonicalResponse);

  return {
    client,
    ui,
    firstStarted,
    firstWrite,
    secondStarted,
    secondWrite,
    sentPatches,
    setCanonicalResponse: (saved) => {
      canonicalResponse = saved;
    },
  };
}

async function assertLateAcceptedPatchKeepsReplacement(t, replaceQuery) {
  const client = createClient();
  seed(client);
  const ui = await renderProbe(t, client);
  const started = Promise.withResolvers();
  const saved = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", () => {
    started.resolve();
    return saved.promise;
  });

  let pending;
  await act(async () => {
    pending = ui.current().mutate({ sort_order: 50 });
    await started.promise;
  });
  const replacement = response({ sort_order: 77 });
  await act(async () => replaceQuery(client, replacement));
  await act(async () => {
    saved.resolve(response({ sort_order: 50 }));
    await pending;
  });

  assert.deepEqual(client.getQueryData(projectUserPropertiesKeys.detail(scope)), replacement);
}

await test("project user properties key and request require the complete caller scope", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  seed(client);
  const service = ProjectService.prototype;
  let args;
  t.mock.method(service, "getProjectUserProperties", async (...request) => {
    args = request;
    return response();
  });

  const options = projectUserPropertiesOptions(client, scope, { canAccess: true, fetchEnabled: true });
  const loaded = await client.fetchQuery({ ...options, staleTime: 0 });

  assert.deepEqual(options.queryKey, ["project-user-properties", "alice", "acme", "project-a"]);
  assert.deepEqual(loaded, response());
  assert.equal(args[0], "acme");
  assert.equal(args[1], "project-a");
  assert.ok(args[2] instanceof AbortSignal);
  assert.equal(
    projectUserPropertiesOptions(client, { ...scope, accountId: "bob" }, { canAccess: false, fetchEnabled: true })
      .queryFn,
    skipToken
  );
});

await test("cache-only observers see cached and pending values without fetching", async (t) => {
  const client = createClient();
  seed(client);
  let readCalls = 0;
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => {
    readCalls++;
    return response();
  });
  const ui = await renderProbe(t, client, { canAccess: true, fetchEnabled: false });
  const saved = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", () => {
    started.resolve();
    return saved.promise;
  });

  assert.deepEqual(ui.current().query.data, response());
  let pending;
  await act(async () => {
    pending = ui.current().mutate({ display_properties: { labels: false } });
    await started.promise;
    await waitForMutation(client, ({ state }) => state.status === "pending" && state.variables?.display_properties);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(ui.current().query.data.display_properties.labels, false);
  assert.equal(readCalls, 0);

  await act(async () => {
    saved.resolve(response({ display_properties: { assignee: true, labels: false, key: true } }));
    await pending;
  });
  assert.equal(readCalls, 0);
});

async function assertActiveObserverRefetchesWithCacheOnlyObserver(t, cacheOnlyFirst) {
  const client = createClient();
  seed(client);
  let reads = 0;
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => {
    reads++;
    return response({ sort_order: reads });
  });
  const environment = await createReactDomTestEnvironment();
  const renders = new Map();
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(ObserverPair, {
        cacheOnlyFirst,
        onRender: (key, state) => renders.set(key, state),
      })
    )
  );
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });

  assert.equal(reads, 0);
  await act(async () => {
    await client.invalidateQueries({ queryKey: projectUserPropertiesKeys.detail(scope), exact: true });
  });

  assert.equal(reads, 1);
  assert.equal(client.getQueryState(projectUserPropertiesKeys.detail(scope)).status, "success");
  assert.equal(client.getQueryData(projectUserPropertiesKeys.detail(scope)).sort_order, 1);
  assert.equal(renders.get("active").query.isError, false);
  assert.equal(renders.get("cache-only").query.isError, false);
}

await test("active preference reads survive a cache-only observer mounted afterward", async (t) => {
  await assertActiveObserverRefetchesWithCacheOnlyObserver(t, false);
});

await test("active preference reads survive a cache-only observer mounted first", async (t) => {
  await assertActiveObserverRefetchesWithCacheOnlyObserver(t, true);
});

await test("mutation options created before the detail query can use a later loaded query", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  const options = patchProjectUserPropertiesOptions(client, scope, true);
  const loaded = response();
  client.setQueryData(projectUserPropertiesKeys.detail(scope), loaded);
  const saved = response({ sort_order: 40 });
  let request;
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", async (...args) => {
    request = args;
    return saved;
  });

  const mutation = client.getMutationCache().build(client, options);
  const result = await mutation.execute({ sort_order: 40 });

  assert.equal(request[0], scope.workspaceSlug);
  assert.equal(request[1], scope.projectId);
  assert.deepEqual(result, saved);
  assert.deepEqual(client.getQueryData(projectUserPropertiesKeys.detail(scope)), saved);
});

await test("project preference service forwards its cancellation signal to the HTTP transport", async (t) => {
  const service = new ProjectService();
  const controller = new AbortController();
  let request;
  t.mock.method(service, "get", async (...args) => {
    request = args;
    return { data: response() };
  });

  await service.getProjectUserProperties("acme", "project-a", controller.signal);

  assert.deepEqual(request, [
    "/api/workspaces/acme/projects/project-a/user-properties/",
    { signal: controller.signal },
  ]);
});

await test("project user properties read forwards cancellation and discards its late response", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  seed(client);
  const started = Promise.withResolvers();
  const result = Promise.withResolvers();
  let signal;
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", (_slug, _project, requestSignal) => {
    signal = requestSignal;
    started.resolve();
    return result.promise;
  });
  const options = projectUserPropertiesOptions(client, scope, { canAccess: true, fetchEnabled: true });
  const request = client.fetchQuery({ ...options, staleTime: 0 }).catch((error) => error);
  await started.promise;
  await client.cancelQueries({ queryKey: options.queryKey, exact: true });
  result.resolve(response());
  await request;

  assert.ok(signal instanceof AbortSignal);
  assert.equal(signal.aborted, true);
  assert.deepEqual(client.getQueryData(options.queryKey), response());
});

await test("project preference patches compose pending nested fields and failed writes roll back by removal", async (t) => {
  const client = createClient();
  seed(client);
  const ui = await renderProbe(t, client);
  const firstStarted = Promise.withResolvers();
  const firstResponse = Promise.withResolvers();
  const secondStarted = Promise.withResolvers();
  const savedResponses = [];
  let serverResponse = response();
  let call = 0;
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", (_slug, _project, patch) => {
    savedResponses.push(patch);
    call++;
    if (call === 1) {
      firstStarted.resolve();
      return firstResponse.promise.then((saved) => {
        serverResponse = saved;
        return saved;
      });
    }
    secondStarted.resolve();
    if (call > 2) return Promise.reject(new Error("write rejected"));
    serverResponse = response({ preferences: patch.preferences });
    return Promise.resolve(serverResponse);
  });
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => serverResponse);

  let first;
  let second;
  await act(async () => {
    first = ui.current().mutate({ preferences: { navigation: { default_tab: "list" } } });
    await firstStarted.promise;
  });
  await act(async () => {
    second = ui.current().mutate({ preferences: { pages: { block_display: false } } });
    await waitForMutation(client, ({ state }) => state.status === "pending" && state.variables?.preferences?.pages);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.deepEqual(JSON.parse(ui.environment.container.querySelector("output").textContent).preferences, {
    pages: { block_display: false },
    navigation: { default_tab: "list", hide_in_more_menu: ["cycles"] },
  });

  await act(async () => {
    firstResponse.resolve(
      response({
        preferences: {
          pages: { block_display: true },
          navigation: { default_tab: "list", hide_in_more_menu: ["cycles"] },
        },
      })
    );
    await Promise.all([first, secondStarted.promise]);
  });
  assert.deepEqual(savedResponses[1].preferences, {
    pages: { block_display: false },
    navigation: { default_tab: "list", hide_in_more_menu: ["cycles"] },
  });
  await act(async () => second);
  assert.deepEqual(client.getQueryData(projectUserPropertiesKeys.detail(scope)).preferences, {
    pages: { block_display: false },
    navigation: { default_tab: "list", hide_in_more_menu: ["cycles"] },
  });

  await act(async () => {
    await assert.rejects(ui.current().mutate({ display_properties: { labels: false } }), /write rejected/);
  });
  assert.equal(ui.current().query.data.display_properties.labels, true);
  assert.equal(client.getQueryData(projectUserPropertiesKeys.detail(scope)).display_properties.labels, true);
});

await test("a failed calendar layout intent is not carried into a queued accepted calendar patch", async (t) => {
  const initial = response({
    display_filters: {
      layout: "calendar",
      calendar: { layout: "month", show_weekends: false },
    },
  });
  const { client, ui, firstStarted, firstWrite, secondStarted, secondWrite, sentPatches, setCanonicalResponse } =
    await setupQueuedPreferenceWrites(t, initial);

  let first;
  let second;
  await act(async () => {
    first = ui
      .current()
      .mutate({ display_filters: { calendar: { layout: "week" } } })
      .catch((error) => error);
    await firstStarted.promise;
  });
  await act(async () => {
    second = ui.current().mutate({ display_filters: { calendar: { show_weekends: true } } });
    await waitForMutation(client, ({ state }) => state.status === "pending" && state.variables?.display_filters);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.deepEqual(ui.current().query.data.display_filters.calendar, { layout: "week", show_weekends: true });
  await act(async () => {
    firstWrite.reject(new Error("calendar layout write rejected"));
    assert.match((await first).message, /calendar layout write rejected/);
    await secondStarted.promise;
    assert.deepEqual(sentPatches[1].display_filters.calendar, { layout: "month", show_weekends: true });
    const acceptedResponse = response({
      display_filters: {
        ...initial.display_filters,
        calendar: sentPatches[1].display_filters.calendar,
      },
    });
    setCanonicalResponse(acceptedResponse);
    secondWrite.resolve(acceptedResponse);
    await second;
  });

  assert.deepEqual(client.getQueryData(projectUserPropertiesKeys.detail(scope)).display_filters.calendar, {
    layout: "month",
    show_weekends: true,
  });
});

await test("a later queued patch is not persisted by an earlier write that succeeds", async (t) => {
  const { client, ui, firstStarted, firstWrite, secondStarted, secondWrite, sentPatches, setCanonicalResponse } =
    await setupQueuedPreferenceWrites(t);
  const accepted = response({
    preferences: {
      pages: { block_display: true },
      navigation: { default_tab: "list", hide_in_more_menu: ["cycles"] },
    },
  });
  let first;
  let second;
  await act(async () => {
    first = ui.current().mutate({ preferences: { navigation: { default_tab: "list" } } });
    second = ui
      .current()
      .mutate({ preferences: { pages: { block_display: false } } })
      .catch((error) => error);
    await firstStarted.promise;
    await waitForMutation(client, ({ state }) => state.status === "pending" && state.variables?.preferences?.pages);
  });

  assert.deepEqual(sentPatches[0].preferences, {
    pages: { block_display: true },
    navigation: { default_tab: "list", hide_in_more_menu: ["cycles"] },
  });

  await act(async () => {
    setCanonicalResponse(accepted);
    firstWrite.resolve(accepted);
    await first;
    await secondStarted.promise;
    secondWrite.reject(new Error("page preference write rejected"));
    assert.match((await second).message, /page preference write rejected/);
  });

  assert.deepEqual(client.getQueryData(projectUserPropertiesKeys.detail(scope)).preferences, {
    pages: { block_display: true },
    navigation: { default_tab: "list", hide_in_more_menu: ["cycles"] },
  });
});

await test("a failed earlier nested patch is removed before the queued patch is sent", async (t) => {
  const { client, ui, firstStarted, firstWrite, secondStarted, secondWrite, sentPatches, setCanonicalResponse } =
    await setupQueuedPreferenceWrites(t);

  let first;
  let second;
  await act(async () => {
    first = ui
      .current()
      .mutate({ preferences: { navigation: { default_tab: "board" } } })
      .catch((error) => error);
    await firstStarted.promise;
  });
  await act(async () => {
    second = ui.current().mutate({ preferences: { pages: { block_display: false } } });
    await waitForMutation(client, ({ state }) => state.status === "pending" && state.variables?.preferences?.pages);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(
    JSON.parse(ui.environment.container.querySelector("output").textContent).preferences.navigation.default_tab,
    "board"
  );

  await act(async () => {
    firstWrite.reject(new Error("navigation write rejected"));
    assert.match((await first).message, /navigation write rejected/);
    await secondStarted.promise;
    const acceptedResponse = response({ preferences: sentPatches[1].preferences });
    setCanonicalResponse(acceptedResponse);
    secondWrite.resolve(acceptedResponse);
    await second;
  });

  assert.deepEqual(sentPatches[1].preferences, {
    pages: { block_display: false },
    navigation: { default_tab: "issues", hide_in_more_menu: ["cycles"] },
  });
  assert.deepEqual(client.getQueryData(projectUserPropertiesKeys.detail(scope)).preferences, {
    pages: { block_display: false },
    navigation: { default_tab: "issues", hide_in_more_menu: ["cycles"] },
  });
});

await test("a late accepted patch cannot replace a new session's query instance", async (t) => {
  await assertLateAcceptedPatchKeepsReplacement(t, (client, replacement) => {
    advanceIdentitySession(client);
    client.setQueryData(projectUserPropertiesKeys.detail(scope), replacement);
  });
});

await test("a late accepted patch cannot replace a recreated exact query", async (t) => {
  await assertLateAcceptedPatchKeepsReplacement(t, (client, replacement) => {
    client.removeQueries({ queryKey: projectUserPropertiesKeys.detail(scope), exact: true });
    client.setQueryData(projectUserPropertiesKeys.detail(scope), replacement);
  });
});

await test("a queued patch does not send after its captured query is replaced", async (t) => {
  const client = createClient();
  seed(client);
  const ui = await renderProbe(t, client);
  const started = Promise.withResolvers();
  const saved = Promise.withResolvers();
  let calls = 0;
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", () => {
    calls++;
    started.resolve();
    return saved.promise;
  });

  let first;
  let second;
  await act(async () => {
    first = ui.current().mutate({ sort_order: 40 });
    second = ui
      .current()
      .mutate({ display_properties: { labels: false } })
      .catch((error) => error);
    await started.promise;
    await waitForMutation(client, ({ state }) => state.status === "pending" && state.variables?.display_properties);
  });
  const replacement = response({ sort_order: 77 });
  await act(async () => {
    client.removeQueries({ queryKey: projectUserPropertiesKeys.detail(scope), exact: true });
    client.setQueryData(projectUserPropertiesKeys.detail(scope), replacement);
    saved.resolve(response({ sort_order: 40 }));
    await first;
    assert.match((await second).message, /scope changed/);
  });

  assert.equal(calls, 1);
  assert.deepEqual(client.getQueryData(projectUserPropertiesKeys.detail(scope)), replacement);
});

await test("accepted patch stays successful when its follow-up query refresh fails", async (t) => {
  const client = createClient();
  seed(client);
  const ui = await renderProbe(t, client);
  const saved = response({ sort_order: 90 });
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", async () => saved);
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => {
    throw new Error("refresh failed");
  });

  let mutationResult;
  await act(async () => {
    mutationResult = await ui.current().mutate({ sort_order: 90 });
  });
  await waitForQueryState(client, projectUserPropertiesKeys.detail(scope), (state) => state?.status === "error");

  assert.equal(mutationResult.sort_order, 90);
  assert.equal(client.getQueryData(projectUserPropertiesKeys.detail(scope)).sort_order, 90);
  assert.equal(client.getQueryState(projectUserPropertiesKeys.detail(scope)).status, "error");
});

await test("disabling project preferences hides cached fields without discarding them", async (t) => {
  const client = createClient();
  seed(client);
  const ui = await renderProbe(t, client, { canAccess: false, fetchEnabled: false });

  assert.equal(ui.current().query.data, undefined);
  assert.equal(client.getQueryData(projectUserPropertiesKeys.detail(scope)).sort_order, 25);
  await assert.rejects(patchProjectUserPropertiesOptions(client, scope, false).mutationFn({ sort_order: 10 }), {
    message: "The account or project preferences scope changed.",
  });
});
