import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, MutationObserver, useQuery, useQueryClient } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { AppStateProvider } = await import("../../apps/web/core/providers/app-state.tsx");
const { resetIdentitySession, endSessionOptions } = await import("../../apps/web/core/queries/identity-session.ts");
const { currentUserOptions, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
const { IntakePreferencesScope, useIntakePreferences } =
  await import("../../apps/web/core/providers/intake-preferences.tsx");

await test("application intake preferences retain each project selection and reset with the account session", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.mock.method(userService, "currentUser", async () => ({ id: "alice" }));
  let client;
  function IntakeControls() {
    const currentClient = useQueryClient();
    React.useEffect(() => {
      client = currentClient;
    }, [currentClient]);
    const preferences = useIntakePreferences();
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("output", null, preferences.sorting.sort_by),
      React.createElement("button", { onClick: () => preferences.setSorting("sort_by", "asc") }, "Oldest first")
    );
  }
  const renderProject = (projectId) =>
    environment.render(
      React.createElement(
        AppStateProvider,
        null,
        React.createElement(
          IntakePreferencesScope,
          { workspaceSlug: "workspace", projectId },
          React.createElement(IntakeControls)
        )
      )
    );
  const sorting = () => environment.container.querySelector("output").textContent;
  t.after(async () => {
    await environment.unmount();
    client?.clear();
  });
  await renderProject("first");
  assert.equal(sorting(), "desc");
  await act(async () => environment.container.querySelector("button").click());
  assert.equal(sorting(), "asc");
  await renderProject("second");
  assert.equal(sorting(), "desc");
  await renderProject("first");
  assert.equal(sorting(), "asc");
  await act(async () => resetIdentitySession(client));
  assert.equal(sorting(), "desc");
});

await test("independent application providers do not share private query data", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.mock.method(userService, "currentUser", async () => null);
  const clients = new Map();
  function Probe({ name }) {
    const client = useQueryClient();
    React.useEffect(() => {
      clients.set(name, client);
    }, [name, client]);
    return null;
  }
  t.after(async () => {
    await environment.unmount();
    for (const client of clients.values()) client.clear();
  });
  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(AppStateProvider, null, React.createElement(Probe, { name: "first" })),
      React.createElement(AppStateProvider, null, React.createElement(Probe, { name: "second" }))
    )
  );
  assert.notEqual(clients.get("first"), clients.get("second"));
  clients.get("first").setQueryData(["private-tokens"], [{ id: "private-token" }]);
  assert.equal(clients.get("second").getQueryData(["private-tokens"]), undefined);
});

await test("a session reset discards private form drafts even when the same account signs in again", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.mock.method(userService, "currentUser", async () => ({ id: "alice" }));
  const { useForm } = webRequire("@tanstack/react-form");
  let client;
  function PrivateDraft() {
    const currentClient = useQueryClient();
    React.useEffect(() => {
      client = currentClient;
    }, [currentClient]);
    const form = useForm({ defaultValues: { note: "" } });
    return React.createElement(form.Field, { name: "note" }, (field) =>
      React.createElement("input", {
        value: field.state.value,
        onChange: (event) => field.handleChange(event.target.value),
      })
    );
  }
  t.after(async () => {
    await environment.unmount();
    client?.clear();
  });
  await environment.render(React.createElement(AppStateProvider, null, React.createElement(PrivateDraft)));
  await act(async () => {
    const input = environment.container.querySelector("input");
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, "Private unsaved note");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  assert.equal(environment.container.querySelector("input").value, "Private unsaved note");
  await act(async () => {
    resetIdentitySession(client);
    client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  });
  assert.equal(environment.container.querySelector("input").value, "");
});

await test("sign-out clears cached private data and prevents a late request from repopulating the cache", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient();
  t.mock.method(userService, "currentUser", async () => null);
  t.after(async () => {
    client.clear();
    await environment.unmount();
  });
  const { authService } = await import("../../apps/web/core/services/auth.service.ts");
  t.mock.method(authService, "requestCSRFToken", async () => ({ csrf_token: "test-csrf" }));
  t.mock.method(authService, "post", async () => ({ status: 200 }));
  client.setQueryData(currentUserOptions().queryKey, { id: "user" });
  client.setQueryData(["private-tokens"], [{ id: "private-token" }]);
  let resolveRequest;
  const request = client.fetchQuery({
    queryKey: ["late-private-data"],
    queryFn: () =>
      new Promise((resolve) => {
        resolveRequest = resolve;
      }),
  });
  const cancellation = assert.rejects(request);
  await new MutationObserver(
    client,
    endSessionOptions(client, "user", () => resetIdentitySession(client), "sign-out")
  ).mutate();
  resolveRequest({ id: "old-session" });
  await cancellation;
  assert.equal(client.getQueryCache().getAll().length, 0);
  assert.equal(client.getQueryData(currentUserOptions().queryKey), undefined);
  assert.equal(localStorage.getItem("theme"), "system");
});

await test("remounting a native query reuses its fresh result instead of reinitializing a retired store", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.mock.method(userService, "currentUser", async () => null);
  const fetch = t.mock.fn(async () => ({ name: "Fresh record" }));
  let client;
  function Reader() {
    const currentClient = useQueryClient();
    React.useEffect(() => {
      client = currentClient;
    }, [currentClient]);
    const query = useQuery({ queryKey: ["fresh-native-record"], queryFn: fetch });
    return React.createElement("span", null, query.data?.name);
  }
  t.after(async () => {
    await environment.unmount();
    client?.clear();
  });
  await environment.render(React.createElement(AppStateProvider, null, React.createElement(Reader)));
  await act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
  assert.equal(environment.container.textContent, "Fresh record");
  await environment.render(React.createElement(AppStateProvider, null));
  await environment.render(React.createElement(AppStateProvider, null, React.createElement(Reader)));
  assert.equal(environment.container.textContent, "Fresh record");
  assert.equal(fetch.mock.callCount(), 1);
});
