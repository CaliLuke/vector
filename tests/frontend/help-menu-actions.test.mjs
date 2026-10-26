import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
globalThis.helpMenuReact = React;
const state = { shortcutsOpen: false };
globalThis.helpMenuState = state;
const stubs = {
  "@plane/i18n": `export const useTranslation = () => ({t:key=>key});`,
  "@/hooks/use-power-k": `export const usePowerK = () => ({toggleShortcutsListModal:open=>globalThis.helpMenuState.shortcutsOpen=open});`,
  "@/components/global": `export const ProductUpdatesModal = ({isOpen}) => isOpen ? globalThis.helpMenuReact.createElement("div", {role:"dialog"}, "Product updates") : null;`,
  "@/components/global/version-number": `export const PlaneVersionNumber = () => null;`,
  "@/components/sidebar/sidebar-item": `export const AppSidebarItem = () => "Help";`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { HelpMenuRoot } = await import("../../apps/web/core/components/workspace/sidebar/help-section/root.tsx");

await test("help menu activates updates and shortcuts from each single action button", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(HelpMenuRoot));
  const trigger = env.container.querySelector('button[aria-label="help"]');
  await act(async () => trigger.click());
  assert.equal(document.querySelector("button button"), null);
  const shortcuts = [...document.querySelectorAll("button")].find(
    (button) => button.textContent === "keyboard_shortcuts"
  );
  await act(async () => shortcuts.click());
  assert.equal(state.shortcutsOpen, true);
  await act(async () => trigger.click());
  const updates = [...document.querySelectorAll("button")].find((button) => button.textContent === "whats_new");
  await act(async () => updates.click());
  assert.equal(document.querySelector('[role="dialog"]').textContent, "Product updates");
});
