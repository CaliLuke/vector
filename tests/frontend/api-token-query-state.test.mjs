import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { MutationObserver, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const state = {
  list: [],
  listError: null,
  listResponse: null,
  listPending: null,
  create: [],
  deleted: [],
  createdToken: null,
  createPending: null,
  createPromise: Promise.resolve(),
  createError: null,
  deleteError: null,
  deletePending: null,
  toasts: [],
  downloads: [],
  closes: 0,
};
globalThis.apiTokenQueryTest = state;
globalThis.apiTokenQueryReact = React;

function resolveCreateDialogModule(specifier) {
  if (specifier === "./form")
    return createModuleStub(`const React = globalThis.apiTokenQueryReact;
      export const CreateApiTokenForm = ({ onSubmit }) => React.createElement("button",
        { onClick: () => { globalThis.apiTokenQueryTest.createPromise = onSubmit({ label: "New token" }); } },
        "Create token");`);
  if (specifier === "./generated-token-details")
    return createModuleStub(`const React = globalThis.apiTokenQueryReact;
      export const GeneratedTokenDetails = ({ tokenDetails, handleClose }) => React.createElement("div", null,
        React.createElement("output", null, tokenDetails.token),
        React.createElement("button", { onClick: handleClose }, "Close token"));`);
  return undefined;
}

function resolveSettingsPageModule(specifier) {
  if (specifier === "@plane/propel/button")
    return createModuleStub(`const React = globalThis.apiTokenQueryReact;
      export const Button = ({ children, ...props }) => React.createElement("button", props, children);`);
  if (specifier === "@plane/propel/empty-state")
    return createModuleStub(`const React = globalThis.apiTokenQueryReact;
      export const EmptyStateCompact = () => React.createElement("p", null, "No tokens");`);
  if (specifier === "@plane/i18n")
    return createModuleStub(`export const useTranslation = () => ({ t: (key) => key });`);
  if (specifier.endsWith("/token-list-item"))
    return createModuleStub(`const React = globalThis.apiTokenQueryReact;
      export const ApiTokenListItem = ({ token }) => React.createElement("p", null, token.label);`);
  if (specifier.endsWith("/create-token-modal"))
    return createModuleStub(`export const CreateApiTokenModal = () => null;`);
  if (specifier.endsWith("/heading"))
    return createModuleStub(`const React = globalThis.apiTokenQueryReact;
      export const ProfileSettingsHeading = ({ children }) => React.createElement("h1", null, children);`);
  if (specifier.includes("/loader/settings/api-token"))
    return createModuleStub(`const React = globalThis.apiTokenQueryReact;
      export const APITokenSettingsLoader = () => React.createElement("div", { role: "progressbar" });`);
  return undefined;
}

registerHooks({
  resolve(specifier, context, nextResolve) {
    const parentURL = context.parentURL ?? "";
    const scopedStub = parentURL.endsWith("/create-token-modal.tsx")
      ? resolveCreateDialogModule(specifier)
      : parentURL.endsWith("/pages/api-tokens.tsx")
        ? resolveSettingsPageModule(specifier)
        : undefined;
    if (scopedStub) return scopedStub;
    if (specifier === "@plane/ui")
      return createModuleStub(`export const EModalPosition = { TOP: "top" };
        export const EModalWidth = { XXL: "xxl" };
        const React = globalThis.apiTokenQueryReact;
        export const ModalCore = ({ children }) => children;
        export const AlertModalCore = ({ handleSubmit, handleClose }) => React.createElement("div", null,
          React.createElement("button", { onClick: handleClose }, "Cancel"),
          React.createElement("button", { onClick: handleSubmit }, "Confirm delete"));`);
    if (specifier === "@plane/i18n")
      return createModuleStub(`export const useTranslation = () => ({ t: (key) => key });`);
    if (specifier === "@plane/utils/datetime")
      return createModuleStub(`export const renderFormattedDate = () => "date";`);
    if (specifier === "@plane/utils/file")
      return createModuleStub(
        `export const csvDownload = (data) => globalThis.apiTokenQueryTest.downloads.push(data);`
      );
    if (specifier === "@plane/propel/toast")
      return createModuleStub(`export const TOAST_TYPE = { ERROR: "error", SUCCESS: "success" };
        export const setToast = (toast) => globalThis.apiTokenQueryTest.toasts.push(toast);`);
    if (specifier === "@plane/services")
      return resolvePackageStub(
        specifier,
        {
          "@plane/services": `export class APITokenService {
            async list(signal) {
              globalThis.apiTokenQueryTest.list.push(signal);
              if (globalThis.apiTokenQueryTest.listError) throw globalThis.apiTokenQueryTest.listError;
              if (globalThis.apiTokenQueryTest.listPending) return globalThis.apiTokenQueryTest.listPending.promise;
              return globalThis.apiTokenQueryTest.listResponse ?? [{ id: "existing", label: "Existing" }];
            }
            async create(data) {
              globalThis.apiTokenQueryTest.create.push(data);
              if (globalThis.apiTokenQueryTest.createError) throw globalThis.apiTokenQueryTest.createError;
              if (globalThis.apiTokenQueryTest.createPending) return globalThis.apiTokenQueryTest.createPending;
              return globalThis.apiTokenQueryTest.createdToken;
            }
            async destroy(tokenId) {
              globalThis.apiTokenQueryTest.deleted.push(tokenId);
              if (globalThis.apiTokenQueryTest.deleteError) throw globalThis.apiTokenQueryTest.deleteError;
              if (globalThis.apiTokenQueryTest.deletePending) return globalThis.apiTokenQueryTest.deletePending;
              return { id: tokenId };
            }
          }`,
        },
        context,
        nextResolve
      );
    return nextResolve(specifier, context);
  },
});

const { apiTokensOptions, createApiTokenOptions, deleteApiTokenOptions } =
  await import("../../apps/web/core/queries/api-tokens.ts");

function clientFor(t) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: 3, gcTime: Infinity } },
  });
  t.after(() => client.clear());
  return client;
}

await test("API token list query forwards its cancellation signal to the service", async (t) => {
  const client = clientFor(t);
  state.list = [];

  const tokens = await client.fetchQuery(apiTokensOptions);

  assert.deepEqual(tokens, [{ id: "existing", label: "Existing" }]);
  assert.ok(state.list[0] instanceof AbortSignal);
});

await test("creating a token updates the list without caching or retrying its one-time secret", async (t) => {
  const client = clientFor(t);
  state.create = [];
  state.createError = null;
  state.createdToken = { id: "created", label: "Created", token: "one-time-secret" };
  client.setQueryData(apiTokensOptions.queryKey, [{ id: "existing", label: "Existing" }]);

  const token = await new MutationObserver(client, createApiTokenOptions(client)).mutate({ label: "Created" });

  assert.equal(token.token, "one-time-secret");
  assert.deepEqual(state.create, [{ label: "Created" }]);
  assert.deepEqual(client.getQueryData(apiTokensOptions.queryKey), [
    { id: "created", label: "Created" },
    { id: "existing", label: "Existing" },
  ]);
});

await test("failed token creation is attempted once and leaves the list available for retry", async (t) => {
  const client = clientFor(t);
  state.create = [];
  state.createError = new Error("offline");
  client.setQueryData(apiTokensOptions.queryKey, [{ id: "existing", label: "Existing" }]);

  await assert.rejects(
    new MutationObserver(client, createApiTokenOptions(client)).mutate({ label: "Created" }),
    /offline/
  );

  assert.equal(state.create.length, 1);
  assert.deepEqual(client.getQueryData(apiTokensOptions.queryKey), [{ id: "existing", label: "Existing" }]);
  state.createError = null;
});

await test("deleting a token updates the list and a failed deletion preserves it", async (t) => {
  const client = clientFor(t);
  state.deleted = [];
  state.deleteError = null;
  client.setQueryData(apiTokensOptions.queryKey, [{ id: "remove" }, { id: "keep" }]);

  await new MutationObserver(client, deleteApiTokenOptions(client)).mutate("remove");
  assert.deepEqual(state.deleted, ["remove"]);
  assert.deepEqual(client.getQueryData(apiTokensOptions.queryKey), [{ id: "keep" }]);

  state.deleteError = new Error("offline");
  await assert.rejects(new MutationObserver(client, deleteApiTokenOptions(client)).mutate("keep"), /offline/);
  assert.deepEqual(client.getQueryData(apiTokensOptions.queryKey), [{ id: "keep" }]);
  state.deleteError = null;
});

await test("a create response from a cleared session cannot recreate the private token list", async (t) => {
  const client = clientFor(t);
  state.list = [];
  client.setQueryData(apiTokensOptions.queryKey, [{ id: "old-session" }]);
  const pending = Promise.withResolvers();
  state.createdToken = pending.promise;
  const create = new MutationObserver(client, createApiTokenOptions(client)).mutate({ label: "Created" });
  await new Promise((resolve) => setImmediate(resolve));

  client.clear();
  state.listPending = Promise.withResolvers();
  const nextSessionFetch = client.fetchQuery(apiTokensOptions).catch((error) => error);
  await new Promise((resolve) => setImmediate(resolve));
  const newSessionQuery = client.getQueryCache().find({ queryKey: apiTokensOptions.queryKey, exact: true });
  const newSessionSignal = state.list.at(-1);
  pending.resolve({ id: "new-token", token: "one-time-secret" });
  await create;

  assert.equal(client.getQueryCache().find({ queryKey: apiTokensOptions.queryKey, exact: true }), newSessionQuery);
  assert.equal(newSessionSignal.aborted, false);
  assert.equal(newSessionQuery.state.fetchStatus, "fetching");
  state.listPending.resolve([{ id: "new-session", label: "New session" }]);
  await nextSessionFetch;
  assert.deepEqual(client.getQueryData(apiTokensOptions.queryKey), [{ id: "new-session", label: "New session" }]);
  state.listPending = null;
  state.createdToken = null;
});

await test("a late create after closing the originating modal cannot reveal or download its secret", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = clientFor(t);
  const pending = Promise.withResolvers();
  state.createPending = pending.promise;
  state.createPromise = Promise.resolve();
  state.createError = null;
  state.downloads = [];
  state.closes = 0;
  client.setQueryData(apiTokensOptions.queryKey, []);
  const { CreateApiTokenModal } = await import("../../apps/web/core/components/api-token/modal/create-token-modal.tsx");
  const render = (isOpen) =>
    environment.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(CreateApiTokenModal, {
          isOpen,
          onClose: () => {
            state.closes++;
          },
        })
      )
    );
  t.after(async () => {
    state.createPending = null;
    await environment.unmount();
  });

  await render(true);
  await act(async () => environment.container.querySelector("button").click());
  await new Promise((resolve) => setImmediate(resolve));
  await render(false);
  await render(true);
  await act(async () => {
    pending.resolve({ id: "late", label: "Late", token: "one-time-secret" });
    await state.createPromise;
  });

  assert.equal(environment.container.querySelector("output"), null);
  assert.equal(state.downloads.length, 0);
  assert.equal(state.closes, 0);
  assert.deepEqual(client.getQueryData(apiTokensOptions.queryKey), [{ id: "late", label: "Late" }]);
});

await test("API token settings shows query errors, retries, and retains stale tokens", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = clientFor(t);
  state.listError = new Error("offline");
  state.listResponse = [];
  client.setQueryData(apiTokensOptions.queryKey, [{ id: "existing", label: "Existing" }]);
  const { APITokensProfileSettings } =
    await import("../../apps/web/core/components/settings/profile/content/pages/api-tokens.tsx");
  t.after(async () => {
    state.listError = null;
    await environment.unmount();
  });

  await environment.render(
    React.createElement(QueryClientProvider, { client }, React.createElement(APITokensProfileSettings))
  );
  await act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
  assert.equal(environment.container.querySelector("p")?.textContent, "Existing");
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Could not refresh/);

  state.listError = null;
  await act(async () => {
    [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Try again").click();
    await new Promise((resolve) => setTimeout(resolve, 50));
  });
  assert.equal(state.list.at(-1) instanceof AbortSignal, true);
  assert.equal(environment.container.querySelector("p")?.textContent, "No tokens");
});

await test("a delete completion after session reset cannot toast or close the replacement scope", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = clientFor(t);
  const pending = Promise.withResolvers();
  state.deletePending = pending.promise;
  state.toasts = [];
  state.closes = 0;
  client.setQueryData(apiTokensOptions.queryKey, [{ id: "private-token", label: "Private" }]);
  const { DeleteApiTokenModal } = await import("../../apps/web/core/components/api-token/delete-token-modal.tsx");
  t.after(async () => {
    state.deletePending = null;
    await environment.unmount();
  });

  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(DeleteApiTokenModal, {
        isOpen: true,
        tokenId: "private-token",
        onClose: () => state.closes++,
      })
    )
  );
  await act(async () => environment.container.querySelector("button:last-child").click());
  await new Promise((resolve) => setImmediate(resolve));
  client.clear();
  await act(async () => {
    pending.resolve({ id: "private-token" });
    await new Promise((resolve) => setTimeout(resolve, 100));
  });

  assert.equal(state.toasts.length, 0);
  assert.equal(state.closes, 0);
  assert.equal(client.getQueryCache().getAll().length, 0);
});
