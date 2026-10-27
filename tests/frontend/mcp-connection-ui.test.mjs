import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const reactQueryModuleUrl = pathToFileURL(webRequire.resolve("@tanstack/react-query")).href;
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@tanstack/react-query") return { shortCircuit: true, url: reactQueryModuleUrl };
    return nextResolve(specifier, context);
  },
});
const { QueryClient, QueryClientProvider } = webRequire("@tanstack/react-query");
const { MCPConnection } =
  await import("../../apps/web/core/components/settings/profile/content/pages/mcp-connection.tsx");
const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
const { mcpLinkOptions } = await import("../../apps/web/core/queries/mcp-link.ts");
const { mcpLinkService } = await import("../../apps/web/core/services/mcp-link.service.ts");
const { authService } = await import("../../apps/web/core/services/auth.service.ts");

const disconnected = { issuer: "https://issuer.example", connected: false, subject: null, pending: null };

async function mountConnection(t) {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(MCPConnection)));
  await act(async () => {
    await client.fetchQuery(mcpLinkOptions("alice"));
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  return { ...environment, client };
}

function button(environment, label) {
  const result = [...environment.container.querySelectorAll("button")].find((item) => item.textContent === label);
  assert.ok(result, `Expected button ${label}`);
  assert.equal(result.disabled, false);
  return result;
}

await test("a verified provider identity requires an explicit confirmation before linking", async (t) => {
  let serverStatus = {
    ...disconnected,
    pending: { id: "verified-flow", subject: "provider-alice", expires_at: "2099-01-01T00:00:00Z" },
  };
  t.mock.method(mcpLinkService, "status", async () => serverStatus);
  const confirmations = [];
  t.mock.method(mcpLinkService, "confirm", async (id, guard) => {
    guard();
    confirmations.push(id);
    serverStatus = { ...disconnected, connected: true, subject: "provider-alice" };
  });
  const environment = await mountConnection(t);
  assert.match(environment.container.textContent, /Connect identity provider-alice/);
  assert.match(environment.container.textContent, /https:\/\/issuer.example/);
  assert.deepEqual(confirmations, []);

  await act(async () => {
    button(environment, "Connect account").click();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.deepEqual(confirmations, ["verified-flow"]);
  assert.match(environment.container.textContent, /Connected as provider-alice/);
  button(environment, "Disconnect");
});

await test("leaving connection settings during CSRF retrieval prevents starting a provider flow", async (t) => {
  t.mock.method(mcpLinkService, "status", async () => disconnected);
  const csrf = Promise.withResolvers();
  const started = Promise.withResolvers();
  t.mock.method(authService, "requestCSRFToken", () => {
    started.resolve();
    return csrf.promise;
  });
  let writes = 0;
  t.mock.method(mcpLinkService, "post", async () => {
    writes++;
    return { data: { authorization_url: "https://issuer.example/authorize" } };
  });
  const environment = await mountConnection(t);
  await act(async () => {
    button(environment, "Continue to identity provider").click();
    await started.promise;
  });
  await environment.render(null);
  await act(async () => {
    csrf.resolve({ csrf_token: "fixture-csrf" });
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(writes, 0);
});
