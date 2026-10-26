import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { notifyManager } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import { registerAdminFormHooks, formToasts } from "./helpers/admin-forms.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { waitForMutation } from "./helpers/wait-for-mutation.mjs";

registerAdminFormHooks();
notifyManager.setScheduler(queueMicrotask);
const stubs = {
  "@/components/common/page-wrapper": `export const PageWrapper=({children})=>children;`,
  "@/components/authentication/authentication-method-card": `import React from "react";
    export const AuthenticationMethodCard=({name,config})=>React.createElement("section",{"aria-label":name},config);`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error",SUCCESS:"success"};
    export const setToast=value=>globalThis.adminFormToasts.push(value); export const setPromiseToast=()=>{};`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const requireAdmin = createRequire(new URL("../../apps/admin/package.json", import.meta.url));
const { MemoryRouter } = requireAdmin("react-router");
const { InstanceService } = requireAdmin("@plane/services");
const { instanceConfigurationsOptions } = await import("../../apps/admin/queries/instance.ts");
const { default: AuthenticationPage } = await import("../../apps/admin/app/(all)/(dashboard)/authentication/page.tsx");

async function mountAuthentication(t, configurations) {
  const fixture = await createQueryDomFixture(t);
  fixture.client.setQueryData(instanceConfigurationsOptions.queryKey, configurations);
  await fixture.render(React.createElement(MemoryRouter, null, React.createElement(AuthenticationPage)));
  return fixture;
}

await test("authentication controls preserve the last enabled method using the current configuration", async (t) => {
  const write = t.mock.method(InstanceService.prototype, "updateConfigurations", async () => {
    throw new Error("The last authentication method must not be disabled.");
  });
  const { env } = await mountAuthentication(t, [
    { key: "ENABLE_EMAIL_PASSWORD", value: "1" },
    { key: "ENABLE_MAGIC_LINK_LOGIN", value: "0" },
  ]);
  const password = env.container.querySelector('section[aria-label="Passwords"] [role="switch"]');
  assert.equal(password.getAttribute("aria-checked"), "true");
  await act(async () => password.click());
  assert.equal(write.mock.callCount(), 0);
  assert.equal(password.getAttribute("aria-checked"), "true");
  assert.equal(formToasts.at(-1).type, "error");
});

await test("authentication toggles stay disabled during native mutation and recover after rejection", async (t) => {
  const pending = Promise.withResolvers();
  t.mock.method(console, "error", () => {});
  t.mock.method(InstanceService.prototype, "updateConfigurations", () => pending.promise);
  const { env, client } = await mountAuthentication(t, [
    { key: "ENABLE_EMAIL_PASSWORD", value: "1" },
    { key: "ENABLE_MAGIC_LINK_LOGIN", value: "1" },
  ]);
  const password = env.container.querySelector('section[aria-label="Passwords"] [role="switch"]');
  await act(async () => password.click());
  assert.equal(password.disabled, true);
  assert.equal(password.getAttribute("aria-checked"), "true");
  await act(async () => {
    pending.reject(new Error("Offline"));
    await waitForMutation(client, (mutation) => mutation.state.status === "error");
  });
  assert.equal(password.disabled, false);
  assert.equal(password.getAttribute("aria-checked"), "true");
});
