import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { settleQuery } from "./helpers/query-router.mjs";

registerWebModuleHooks();
const stubs = {
  "@/components/common/logo-spinner": `export const LogoSpinner = () => "Loading instance";`,
  "@/components/instance": `export const MaintenanceView = () => "Instance unavailable"; export const InstanceNotReady = () => "Instance setup required";`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { InstanceService } = webRequire("@plane/services");
const { instanceInfoOptions } = await import("../../apps/web/core/queries/instance.ts");
const { default: InstanceWrapper } = await import("../../apps/web/core/lib/wrappers/instance-wrapper.tsx");
const { useFileSize } = await import("../../apps/web/core/hooks/use-file-size.ts");

function FileLimit() {
  return React.createElement("output", null, useFileSize().maxFileSize);
}
function InstanceName() {
  const { data } = useQuery(instanceInfoOptions);
  return React.createElement("p", null, data?.instance.name);
}
async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnMount: "always" } } });
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  return {
    client,
    env,
    render: () =>
      env.render(
        React.createElement(
          QueryClientProvider,
          { client },
          React.createElement(InstanceWrapper, null, React.createElement(FileLimit), React.createElement(InstanceName))
        )
      ),
  };
}

await test("instance consumers share query data and react to configuration changes without a store", async (t) => {
  const { client, env, render } = await fixture(t);
  const pending = Promise.withResolvers();
  const request = t.mock.method(InstanceService.prototype, "info", () => pending.promise);
  await render();
  assert.match(env.container.textContent, /Loading instance/);
  await act(async () =>
    pending.resolve({ instance: { name: "Vector", is_setup_done: true }, config: { file_size_limit: 42 } })
  );
  await settleQuery();
  assert.equal(env.container.querySelector("output").textContent, "42");
  assert.equal(env.container.querySelector("p").textContent, "Vector");
  assert.equal(
    request.mock.callCount(),
    1,
    "mounting more consumers reuses fresh data despite the legacy client default"
  );
  await act(async () =>
    client.setQueryData(instanceInfoOptions.queryKey, {
      instance: { name: "Updated Vector", is_setup_done: true },
      config: { file_size_limit: 84 },
    })
  );
  await settleQuery();
  assert.equal(env.container.querySelector("output").textContent, "84");
  assert.equal(env.container.querySelector("p").textContent, "Updated Vector");
});

await test("failed instance loading blocks children and recovers through query invalidation", async (t) => {
  const { client, env, render } = await fixture(t);
  t.mock.method(InstanceService.prototype, "info", async () => {
    throw new Error("offline");
  });
  await render();
  await settleQuery();
  assert.match(env.container.textContent, /Instance unavailable/);
  assert.equal(env.container.querySelector("output"), null);
  t.mock.method(InstanceService.prototype, "info", async () => ({ instance: { is_setup_done: false }, config: {} }));
  await act(async () => client.invalidateQueries({ queryKey: instanceInfoOptions.queryKey }));
  await settleQuery();
  assert.match(env.container.textContent, /Instance setup required/);
  assert.equal(env.container.querySelector("output"), null);
  t.mock.method(InstanceService.prototype, "info", async () => ({
    instance: { is_setup_done: true },
    config: { file_size_limit: 16 },
  }));
  await act(async () => client.invalidateQueries({ queryKey: instanceInfoOptions.queryKey }));
  await settleQuery();
  assert.equal(env.container.querySelector("output").textContent, "16");
});

await test("clearing the query cache cancels instance loading and rejects late configuration data", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(() => client.clear());
  const pending = Promise.withResolvers();
  let signal;
  t.mock.method(InstanceService.prototype, "info", (requestSignal) => {
    signal = requestSignal;
    return pending.promise;
  });
  const loading = client.fetchQuery(instanceInfoOptions);
  const rejected = assert.rejects(loading);
  client.clear();
  assert.equal(signal.aborted, true);
  t.mock.method(InstanceService.prototype, "info", async () => ({
    instance: { name: "Current instance" },
    config: { file_size_limit: 100 },
  }));
  await client.fetchQuery(instanceInfoOptions);
  pending.resolve({ instance: { name: "Obsolete instance" }, config: { file_size_limit: 1 } });
  await rejected;
  assert.equal(client.getQueryData(instanceInfoOptions.queryKey).instance.name, "Current instance");
  assert.equal(client.getQueryData(instanceInfoOptions.queryKey).config.file_size_limit, 100);
});
