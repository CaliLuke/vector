import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const effects = { languageRequests: [], clearedThemes: 0, appliedThemes: [] };
globalThis.profileEffects = effects;
const stubs = {
  "@/hooks/use-current-user": `export const useCurrentUser = () => globalThis.profileEffects.useCurrentUser();`,
  "@plane/i18n": `export const FALLBACK_LANGUAGE = "en"; export const SUPPORTED_LANGUAGES = ["en", "fr", "de"].map(value => ({value})); export const setLanguage = language => globalThis.profileEffects.setLanguage(language);`,
  "@plane/utils/theme":
    "export const applyCustomTheme = (...args) => globalThis.profileEffects.appliedThemes.push(args);\nexport const clearCustomTheme = () => { globalThis.profileEffects.clearedThemes++; };",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

const flush = () => act(async () => Promise.resolve());

await test("ProfileEffects clears old custom theme and serializes language changes through logout", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const { currentUserOptions, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const ProfileEffects = (await import("../../apps/web/core/lib/wrappers/profile-effects.tsx")).default;

  effects.languageRequests = [];
  effects.clearedThemes = 0;
  effects.appliedThemes = [];
  effects.autoResolveLanguages = false;
  effects.setLanguage = (language) =>
    new Promise((resolve) => {
      effects.languageRequests.push({ language, resolve });
      if (effects.autoResolveLanguages) resolve();
    });
  effects.useCurrentUser = () => useQuery(currentUserOptions());

  const userA = { id: "user-a" };
  const userB = { id: "user-b" };
  const profileA = {
    user: userA.id,
    language: "fr",
    theme: { theme: "custom", primary: "#123456", background: "#101010", darkPalette: true },
  };
  const profileB = { user: userB.id, language: "de", theme: { theme: "dark" } };
  client.setQueryData(currentUserOptions().queryKey, userA);
  client.setQueryData(identityKeys.profile(userA.id), profileA);

  const router = createRouterFixture("/default/");
  t.after(async () => {
    effects.autoResolveLanguages = true;
    await environment.render(null);
    effects.languageRequests.forEach(({ resolve }) => resolve());
    await flush();
    await environment.unmount();
    await client.cancelQueries();
    client.clear();
  });

  const renderApp = () =>
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(ProfileEffects, null, React.createElement("div", null, "app"))
    );
  await router.render(environment, renderApp());
  await flush();
  assert.deepEqual(
    effects.languageRequests.map(({ language }) => language),
    ["fr"]
  );
  assert.equal(effects.appliedThemes.length, 1);

  client.setQueryData(identityKeys.profile(userB.id), profileB);
  await act(async () => client.setQueryData(currentUserOptions().queryKey, userB));
  await environment.render(React.createElement(router.Provider, null, renderApp()));
  await flush();
  assert.equal(effects.clearedThemes, 1);
  assert.deepEqual(
    effects.languageRequests.map(({ language }) => language),
    ["fr"]
  );

  await act(async () => effects.languageRequests[0].resolve());
  assert.deepEqual(
    effects.languageRequests.map(({ language }) => language),
    ["fr", "de"]
  );

  await act(async () => client.setQueryData(currentUserOptions().queryKey, null));
  await environment.render(React.createElement(router.Provider, null, renderApp()));
  await flush();
  await act(async () => effects.languageRequests[1].resolve());
  assert.deepEqual(
    effects.languageRequests.map(({ language }) => language),
    ["fr", "de", "en"]
  );
  await act(async () => effects.languageRequests[2].resolve());
  assert.equal(effects.clearedThemes, 2);
});
