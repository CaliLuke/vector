import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const requireFromWeb = createRequire(new URL("../../apps/web/package.json", import.meta.url));
const React = requireFromWeb("react");
const { act } = React;
const reactModules = Object.fromEntries(
  ["react", "react/jsx-runtime", "react/jsx-dev-runtime", "react-dom/client"].map((name) => [
    name,
    pathToFileURL(requireFromWeb.resolve(name)).href,
  ])
);
const notifications = {
  filters: { read: true, archived: false, snoozed: true },
  calls: [],
  updateFilters: (...args) => notifications.calls.push(args),
  updateBulkFilters: (value) => notifications.calls.push(value),
};
globalThis.__notificationMenuTest = { React, notifications };

// Keep the actual menu and item components. The surrounding popover is opened
// unconditionally here so the test exercises their render callback and actions
// without depending on positioning, translation, or the application singleton.
const substitutes = {
  "@/hooks/store/notifications":
    "export const useWorkspaceNotifications = () => globalThis.__notificationMenuTest.notifications;",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/ui": `export function PopoverMenu({ data, render }) {
    return globalThis.__notificationMenuTest.React.createElement('div', null, data.map(render));
  }`,
  "@plane/propel/icons": "export const ArchiveIcon = () => null; export const CheckIcon = () => null;",
  "@plane/propel/icon-button": "export const IconButton = () => null;",
  "@plane/utils": "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');",
};

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (Object.hasOwn(substitutes, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(substitutes[specifier])}` };
    }
    if (Object.hasOwn(reactModules, specifier)) {
      return { shortCircuit: true, url: reactModules[specifier] };
    }
    return nextResolve(specifier, context);
  },
});

const { createReactDomTestEnvironment } = await import("./helpers/react-dom.mjs");

test("notification filter items render without key warnings and expose their active state", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { NotificationHeaderMenuOption } =
    await import("../../apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx");

  await environment.render(React.createElement(NotificationHeaderMenuOption));

  const buttons = [...environment.container.querySelectorAll("button")];
  assert.deepEqual(
    buttons.map((button) => [button.textContent.trim(), button.getAttribute("aria-pressed")]),
    [
      ["notification.options.show_unread", "true"],
      ["notification.options.show_archived", "false"],
      ["notification.options.show_snoozed", "true"],
    ]
  );
  assert.deepEqual(errors, []);

  notifications.calls.length = 0;
  await act(async () => {
    for (const button of buttons) button.click();
  });
  assert.deepEqual(notifications.calls, [
    ["read", false],
    { archived: true, snoozed: false },
    { snoozed: false, archived: false },
  ]);
});
