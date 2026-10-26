import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

globalThis.__appliedNotificationFiltersTest = {
  filters: { type: { assigned: true, created: false, subscribed: false } },
  calls: [],
};

const headerUrl = new URL("../../packages/ui/src/header/header.tsx", import.meta.url).href;
const tagUrl = new URL("../../packages/ui/src/tag/tag.tsx", import.meta.url).href;
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    const substitutes = {
      "@/components/workspace-notifications/provider": `export const useNotifications = () => ({
        filters: globalThis.__appliedNotificationFiltersTest.filters,
        updateFilters: (...args) => globalThis.__appliedNotificationFiltersTest.calls.push(args),
      });`,
      "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
      "@plane/ui": `export { Header, EHeaderVariant } from ${JSON.stringify(headerUrl)}; export { Tag } from ${JSON.stringify(tagUrl)};`,
    };
    if (Object.hasOwn(substitutes, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(substitutes[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

test("applied notification filters render without key warnings and can be cleared", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount();
    delete globalThis.__appliedNotificationFiltersTest;
  });

  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { AppliedFilters } =
    await import("../../apps/web/core/components/workspace-notifications/sidebar/filters/applied-filter.tsx");

  await environment.render(React.createElement(AppliedFilters, { workspaceSlug: "workspace" }));

  assert.match(environment.container.textContent, /notification\.filter\.assigned/);
  assert.doesNotMatch(environment.container.textContent, /notification\.filter\.(created|subscribed)/);
  assert.equal(
    errors.some((message) => message.includes('unique "key" prop')),
    false,
    errors.join("\n")
  );

  await act(() => environment.container.querySelector("button").click());
  assert.deepEqual(globalThis.__appliedNotificationFiltersTest.calls, [
    ["type", { assigned: false, created: false, subscribed: false }],
  ]);
});
