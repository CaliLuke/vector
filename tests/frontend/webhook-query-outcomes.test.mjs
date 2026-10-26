import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { MutationObserver, QueryClient } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const state = {
  lists: [],
  details: [],
  created: [],
  updated: [],
  deleted: [],
  regenerated: [],
  listResponses: {},
  listPending: null,
  detailResponses: {},
  createResponse: null,
  createPending: null,
  updateResponse: null,
  updatePending: null,
  deleteError: null,
  deletePending: null,
  regenerateResponse: null,
  regeneratePending: null,
  downloads: [],
  toasts: [],
  createPromise: Promise.resolve(),
};
globalThis.webhookQueryTest = state;
globalThis.webhookQueryReact = React;

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    const parentURL = context.parentURL ?? "";
    if (parentURL.endsWith("/create-webhook-modal.tsx")) {
      if (specifier === "./form")
        return createModuleStub(`const React = globalThis.webhookQueryReact;
          export const WebhookForm = ({onSubmit}) => React.createElement('button', {
            onClick: () => { globalThis.webhookQueryTest.createPromise = onSubmit({url:'https://hooks.example.test'}); }
          }, 'Create webhook');`);
      if (specifier === "./generated-hook-details")
        return createModuleStub(`const React = globalThis.webhookQueryReact;
          export const GeneratedHookDetails = ({secretKey, handleClose}) => React.createElement('div', null,
            React.createElement('output', null, secretKey),
            React.createElement('button', {onClick:handleClose}, 'Close generated webhook'));`);
    }
    const stubs = {
      "@/services/webhook.service": `export class WebhookService {
        async fetchWebhooksList(workspaceSlug, signal) {
          globalThis.webhookQueryTest.lists.push({workspaceSlug, signal});
          if (globalThis.webhookQueryTest.listPending) return globalThis.webhookQueryTest.listPending.promise;
          return globalThis.webhookQueryTest.listResponses[workspaceSlug] ?? [];
        }
        async fetchWebhookDetails(workspaceSlug, webhookId, signal) {
          globalThis.webhookQueryTest.details.push({workspaceSlug, webhookId, signal});
          return globalThis.webhookQueryTest.detailResponses[workspaceSlug + '/' + webhookId];
        }
        async createWebhook(workspaceSlug, data) {
          globalThis.webhookQueryTest.created.push({workspaceSlug, data});
          if (globalThis.webhookQueryTest.createPending) return globalThis.webhookQueryTest.createPending.promise;
          return globalThis.webhookQueryTest.createResponse;
        }
        async updateWebhook(workspaceSlug, webhookId, data) {
          globalThis.webhookQueryTest.updated.push({workspaceSlug, webhookId, data});
          if (globalThis.webhookQueryTest.updatePending) return globalThis.webhookQueryTest.updatePending.promise;
          if (globalThis.webhookQueryTest.updateError) throw globalThis.webhookQueryTest.updateError;
          return globalThis.webhookQueryTest.updateResponse;
        }
        async deleteWebhook(workspaceSlug, webhookId) {
          globalThis.webhookQueryTest.deleted.push({workspaceSlug, webhookId});
          if (globalThis.webhookQueryTest.deletePending) return globalThis.webhookQueryTest.deletePending.promise;
          if (globalThis.webhookQueryTest.deleteError) throw globalThis.webhookQueryTest.deleteError;
        }
        async regenerateSecretKey(workspaceSlug, webhookId) {
          globalThis.webhookQueryTest.regenerated.push({workspaceSlug, webhookId});
          if (globalThis.webhookQueryTest.regeneratePending) return globalThis.webhookQueryTest.regeneratePending.promise;
          return globalThis.webhookQueryTest.regenerateResponse;
        }
      }`,
      "@plane/types": "",
      "@plane/constants": "export const EModalPosition={TOP:'top'}; export const EModalWidth={XXL:'xxl'};",
      "@plane/i18n": "export const useTranslation=()=>({t:(key)=>key});",
      "@plane/propel/toast":
        "export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'}; export const setToast=(toast)=>globalThis.webhookQueryTest.toasts.push(toast);",
      "@plane/utils/file": "export const csvDownload=(data)=>globalThis.webhookQueryTest.downloads.push(data);",
      "@plane/utils/datetime": "export const renderFormattedPayloadDate=(value)=>value;",
      "@plane/ui": `const React=globalThis.webhookQueryReact; export const EModalPosition={TOP:'top'}; export const EModalWidth={XXL:'xxl'}; export const ModalCore=({children})=>children; export const AlertModalCore=props=>React.createElement('button',{id:'delete-webhook',disabled:props.isSubmitting,onClick:props.handleSubmit},'Delete webhook');`,
      "@/hooks/use-keypress": "export default ()=>{};",
    };
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

const query = await import("../../apps/web/core/queries/webhooks.ts");
const { skipToken } = await import("@tanstack/react-query");
function createClient(t) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false, gcTime: Infinity } },
  });
  t.after(() => client.clear());
  return client;
}

function waitForMutation(client, mutationKey) {
  const cache = client.getMutationCache();
  const isTargetMutation = (mutation) => JSON.stringify(mutation.options.mutationKey) === JSON.stringify(mutationKey);
  const settledMutation = () =>
    cache.getAll().find((mutation) => isTargetMutation(mutation) && mutation.state.status !== "pending");
  const settled = settledMutation();
  if (settled) return Promise.resolve(settled);

  return new Promise((resolve) => {
    const unsubscribe = cache.subscribe((event) => {
      if (event.type !== "updated" || !isTargetMutation(event.mutation)) return;
      if (event.mutation.state.status === "pending") return;
      unsubscribe();
      resolve(event.mutation);
    });
    const current = settledMutation();
    if (current) {
      unsubscribe();
      resolve(current);
    }
  });
}

const record = (id, url) => ({
  id,
  url,
  created_at: "2026-01-01",
  updated_at: "2026-01-02",
  is_active: true,
  project: true,
  cycle: true,
  module: true,
  issue: true,
  issue_comment: true,
});

await test("webhook queries are workspace-scoped, cancellable, permission-gated, and cache no secrets", async (t) => {
  const client = createClient(t);
  state.lists = [];
  state.details = [];
  state.listResponses = {
    alpha: [{ ...record("same-id", "https://alpha.example.test"), secret_key: "alpha-secret" }],
    beta: [{ ...record("same-id", "https://beta.example.test"), secret_key: "beta-secret" }],
  };
  state.detailResponses = {
    "alpha/same-id": { ...record("same-id", "https://alpha.example.test/detail"), secret_key: "detail-secret" },
    "beta/same-id": { ...record("same-id", "https://beta.example.test/detail"), secret_key: "other-secret" },
  };
  assert.equal(query.webhooksListOptions("alpha", false).queryFn, skipToken);
  assert.equal(query.webhookDetailsOptions("alpha", "same-id", false).queryFn, skipToken);
  assert.deepEqual(await client.fetchQuery(query.webhooksListOptions("alpha", true)), [
    record("same-id", "https://alpha.example.test"),
  ]);
  assert.deepEqual(await client.fetchQuery(query.webhooksListOptions("beta", true)), [
    record("same-id", "https://beta.example.test"),
  ]);
  assert.deepEqual(
    await client.fetchQuery(query.webhookDetailsOptions("alpha", "same-id", true)),
    record("same-id", "https://alpha.example.test/detail")
  );
  assert.deepEqual(
    await client.fetchQuery(query.webhookDetailsOptions("beta", "same-id", true)),
    record("same-id", "https://beta.example.test/detail")
  );
  assert.ok(state.lists.every(({ signal }) => signal instanceof AbortSignal));
  assert.ok(state.details.every(({ signal }) => signal instanceof AbortSignal));
  assert.equal(client.getQueryData(query.webhookKeys.list("alpha"))[0].secret_key, undefined);
  assert.equal(client.getQueryData(query.webhookKeys.detail("alpha", "same-id")).secret_key, undefined);
  assert.notDeepEqual(
    client.getQueryData(query.webhookKeys.list("alpha")),
    client.getQueryData(query.webhookKeys.list("beta"))
  );
});

await test("create exposes a one-time key locally but keeps it out of query and mutation results", async (t) => {
  const client = createClient(t);
  state.created = [];
  state.createResponse = { ...record("created", "https://created.example.test"), secret_key: "one-time" };
  client.setQueryData(query.webhookKeys.list("acme"), [record("existing", "https://existing.example.test")]);
  let exposedSecret;
  const result = await new MutationObserver(client, query.createWebhookMutationOptions(client, "acme")).mutate({
    data: { url: "https://created.example.test" },
    exposeSecret: (secret) => {
      exposedSecret = secret;
    },
  });
  assert.equal(exposedSecret, "one-time");
  assert.deepEqual(result, record("created", "https://created.example.test"));
  assert.equal(result.secret_key, undefined);
  assert.deepEqual(client.getQueryData(query.webhookKeys.list("acme")), [
    record("created", "https://created.example.test"),
    record("existing", "https://existing.example.test"),
  ]);
  assert.equal(client.getMutationCache().getAll().at(-1).state.data.secret_key, undefined);
});

await test("update writes canonical server data and failed delete preserves detail and list cache", async (t) => {
  const client = createClient(t);
  const original = record("hook", "https://draft.example.test");
  client.setQueryData(query.webhookKeys.list("acme"), [original]);
  client.setQueryData(query.webhookKeys.detail("acme", "hook"), original);
  state.updateResponse = { ...record("hook", "https://canonical.example.test"), secret_key: "must-not-cache" };
  const updated = await new MutationObserver(client, query.updateWebhookMutationOptions(client, "acme", "hook")).mutate(
    {
      url: "https://submitted.example.test",
    }
  );
  assert.deepEqual(updated, record("hook", "https://canonical.example.test"));
  assert.deepEqual(client.getQueryData(query.webhookKeys.list("acme")), [
    record("hook", "https://canonical.example.test"),
  ]);
  assert.deepEqual(
    client.getQueryData(query.webhookKeys.detail("acme", "hook")),
    record("hook", "https://canonical.example.test")
  );

  state.deleteError = new Error("offline");
  await assert.rejects(
    new MutationObserver(client, query.deleteWebhookMutationOptions(client, "acme", "hook")).mutate(),
    /offline/
  );
  assert.deepEqual(client.getQueryData(query.webhookKeys.list("acme")), [
    record("hook", "https://canonical.example.test"),
  ]);
  assert.deepEqual(
    client.getQueryData(query.webhookKeys.detail("acme", "hook")),
    record("hook", "https://canonical.example.test")
  );
  state.deleteError = null;
});

await test("late old-session mutation does not cancel a replacement workspace query", async (t) => {
  const client = createClient(t);
  const original = record("hook", "https://old.example.test");
  client.setQueryData(query.webhookKeys.list("acme"), [original]);
  const updatePending = Promise.withResolvers();
  state.updatePending = updatePending;
  const mutationPromise = new MutationObserver(
    client,
    query.updateWebhookMutationOptions(client, "acme", "hook")
  ).mutate({ url: "https://submitted.example.test" });
  await new Promise((resolve) => setImmediate(resolve));

  client.removeQueries({ queryKey: query.webhookKeys.list("acme"), exact: true });
  const listPending = Promise.withResolvers();
  state.listPending = listPending;
  const replacementList = client.fetchQuery(query.webhooksListOptions("acme", true));
  await new Promise((resolve) => setImmediate(resolve));
  const replacementRequest = state.lists.at(-1);
  updatePending.resolve(record("hook", "https://server.example.test"));
  await mutationPromise;

  assert.equal(replacementRequest.signal.aborted, false);
  listPending.resolve([record("fresh", "https://fresh.example.test")]);
  await replacementList;
  state.updatePending = null;
  state.listPending = null;
});

await test("update and regenerate mutations for one webhook run in a shared native scope", async (t) => {
  const client = createClient(t);
  const initial = record("hook", "https://initial.example.test");
  client.setQueryData(query.webhookKeys.list("acme"), [initial]);
  client.setQueryData(query.webhookKeys.detail("acme", "hook"), initial);
  state.updated = [];
  state.regenerated = [];
  const regeneratePending = Promise.withResolvers();
  state.regeneratePending = regeneratePending;
  state.updateResponse = record("hook", "https://final.example.test");
  const regeneratePromise = new MutationObserver(
    client,
    query.regenerateWebhookSecretMutationOptions(client, "acme", "hook")
  ).mutate({ exposeSecret() {} });
  await new Promise((resolve) => setImmediate(resolve));
  const updatePromise = new MutationObserver(client, query.updateWebhookMutationOptions(client, "acme", "hook")).mutate(
    { url: "https://final.example.test" }
  );
  await new Promise((resolve) => setImmediate(resolve));

  assert.equal(state.regenerated.length, 1);
  assert.equal(state.updated.length, 0);
  regeneratePending.resolve({ ...record("hook", "https://regenerated.example.test"), secret_key: "one-time" });
  await Promise.all([regeneratePromise, updatePromise]);

  assert.equal(state.updated.length, 1);
  assert.deepEqual(
    client.getQueryData(query.webhookKeys.detail("acme", "hook")),
    record("hook", "https://final.example.test")
  );
  state.regeneratePending = null;
});

await test("late create after modal unmount cannot reveal a secret while its Query remains cached", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/settings/webhooks/");
  const { environment, client } = fixture;
  const pending = Promise.withResolvers();
  state.createPending = pending;
  state.createPromise = Promise.resolve();
  state.downloads = [];
  client.setQueryData(query.webhookKeys.list("acme"), []);
  const { CreateWebhookModal } = await import("../../apps/web/core/components/web-hooks/create-webhook-modal.tsx");
  await fixture.render(React.createElement(CreateWebhookModal, { isOpen: true, currentWorkspace: null, onClose() {} }));
  await act(async () => environment.container.querySelector("button").click());
  await new Promise((resolve) => setImmediate(resolve));
  const createPromise = state.createPromise;
  await environment.render(null);
  pending.resolve({ ...record("late", "https://late.example.test"), secret_key: "late-secret" });
  await act(async () => createPromise);

  assert.equal(
    client.getQueryCache().find({ queryKey: query.webhookKeys.list("acme"), exact: true }) !== undefined,
    true
  );
  assert.deepEqual(state.downloads, []);
  assert.equal(client.getQueryData(query.webhookKeys.list("acme"))[0].secret_key, undefined);
  assert.equal(client.getMutationCache().getAll().at(-1).state.data.secret_key, undefined);
  state.createPending = null;
});

await test("create from a prior workspace flow stays invalid after an A-to-B-to-A switch", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/settings/webhooks/");
  const { environment, client, router } = fixture;
  client.setQueryData(query.webhookKeys.list("acme"), []);
  client.setQueryData(query.webhookKeys.list("beta"), []);
  const pending = Promise.withResolvers();
  state.createPending = pending;
  state.createPromise = Promise.resolve();
  state.downloads = [];
  const { CreateWebhookModal } = await import("../../apps/web/core/components/web-hooks/create-webhook-modal.tsx");
  await fixture.render(React.createElement(CreateWebhookModal, { isOpen: true, currentWorkspace: null, onClose() {} }));
  await act(async () => environment.container.querySelector("button").click());
  await new Promise((resolve) => setImmediate(resolve));
  const createPromise = state.createPromise;

  await act(async () => router.navigate({ href: "/beta/settings/webhooks/" }));
  await act(async () => router.navigate({ href: "/acme/settings/webhooks/" }));
  pending.resolve({ ...record("late-a", "https://late-a.example.test"), secret_key: "late-a-secret" });
  await act(async () => createPromise);

  assert.equal(environment.container.querySelector("output"), null);
  assert.deepEqual(state.downloads, []);
  state.createPending = null;
});

await test("late delete after modal unmount cleans captured cache without navigating", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/settings/webhooks/hook/");
  const { client, environment, router } = fixture;
  const original = record("hook", "https://hook.example.test");
  client.setQueryData(query.webhookKeys.list("acme"), [original]);
  client.setQueryData(query.webhookKeys.detail("acme", "hook"), original);
  const pending = Promise.withResolvers();
  state.deletePending = pending;
  state.toasts = [];
  const { DeleteWebhookModal } = await import("../../apps/web/core/components/web-hooks/delete-webhook-modal.tsx");
  await fixture.render(
    React.createElement(DeleteWebhookModal, {
      isOpen: true,
      workspaceSlug: "acme",
      webhookId: "hook",
      onClose() {},
    })
  );
  await act(async () => environment.container.querySelector("#delete-webhook").click());
  const deletionSettled = waitForMutation(client, [...query.webhookKeys.detail("acme", "hook"), "delete"]);
  const startPath = router.state.location.pathname;
  await environment.render(null);
  await act(async () => {
    pending.resolve();
    await deletionSettled;
  });
  await settleQuery();

  assert.deepEqual(client.getQueryData(query.webhookKeys.list("acme")), []);
  assert.equal(
    client.getQueryCache().find({ queryKey: query.webhookKeys.detail("acme", "hook"), exact: true }),
    undefined
  );
  assert.equal(router.state.location.pathname, startPath);
  assert.deepEqual(state.toasts, []);
  state.deletePending = null;
});
