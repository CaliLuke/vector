import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const providerFixture = new URL("./helpers/app-provider-lazy-components.mjs", import.meta.url).href;
const aliases = new Set([
  "@/lib/b-progress/AppProgressBar",
  "@/lib/wrappers/profile-effects",
  "@/lib/wrappers/instance-wrapper",
  "@/providers/app-state",
  "next-themes",
  "@plane/constants",
  "@plane/i18n",
  "@plane/propel/toast",
  "@plane/utils/theme-legacy",
]);
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (aliases.has(specifier)) return { shortCircuit: true, url: providerFixture };
    return nextResolve(specifier, context);
  },
});

const { AppProvider } = await import("../../apps/web/app/provider.tsx");

test("app provider catches lazy wrapper suspension during a synchronous refresh", async () => {
  const environment = await createReactDomTestEnvironment();
  const errors = [];
  const originalConsoleError = console.error;
  console.error = (...args) => errors.push(args.join(" "));

  try {
    await environment.render(
      React.createElement(AppProvider, null, React.createElement("div", { "data-testid": "app-content" }, "Ready"))
    );

    assert.equal(environment.container.querySelector("[data-testid='app-content']")?.textContent, "Ready");
    assert.equal(
      errors.some((message) => message.includes("A component suspended while responding to synchronous input")),
      false
    );
  } finally {
    console.error = originalConsoleError;
    await environment.unmount();
  }
});
