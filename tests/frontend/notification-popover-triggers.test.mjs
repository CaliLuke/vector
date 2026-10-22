import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import path from "node:path";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { act, React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const repositoryRoot = new URL("../../", import.meta.url);
const requireFromWeb = createRequire(new URL("../../apps/web/package.json", import.meta.url));
const reactModules = Object.fromEntries(
  ["react", "react/jsx-runtime", "react/jsx-dev-runtime", "react-dom/client"].map((name) => [
    name,
    pathToFileURL(requireFromWeb.resolve(name)).href,
  ])
);
const dataModules = {
  "@/hooks/store/notifications": `export const useWorkspaceNotifications = () => globalThis.__notificationPopoverTest.notifications;`,
  "@plane/constants": `export const FILTER_TYPE_OPTIONS = [{ value: "assigned", i18n_label: "assigned" }, { value: "created", i18n_label: "created" }];`,
  "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
  "@plane/propel/icons": `export const ArchiveIcon = () => null; export const CheckIcon = () => null;`,
  "@plane/utils": `export const cn = (...values) => values.filter(value => typeof value === "string").join(" ");`,
};
const sourceModules = {
  "@plane/ui": "packages/ui/src/popovers/popover-menu.tsx",
  "@plane/propel/icon-button": "packages/propel/dist/icon-button/index.js",
};

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "../utils" && context.parentURL?.includes("/packages/ui/src/popovers/")) {
      return {
        shortCircuit: true,
        url: pathToFileURL(path.join(new URL(repositoryRoot).pathname, "packages/ui/src/utils/classname.tsx")).href,
      };
    }
    if (Object.hasOwn(dataModules, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(dataModules[specifier])}` };
    }
    if (Object.hasOwn(sourceModules, specifier)) {
      return {
        shortCircuit: true,
        url: pathToFileURL(path.join(new URL(repositoryRoot).pathname, sourceModules[specifier])).href,
      };
    }
    if (Object.hasOwn(reactModules, specifier)) {
      return { shortCircuit: true, url: reactModules[specifier] };
    }
    return nextResolve(specifier, context);
  },
});

globalThis.__notificationPopoverTest = {
  React,
  notifications: {
    filters: { read: false, archived: false, snoozed: false, type: {} },
    updateFilters() {},
    updateBulkFilters() {},
  },
};

test("activity filter options expose keyboard-focusable toggle buttons", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ActivityFilter } =
    await import("../../apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx");

  function Harness() {
    const [selected, setSelected] = React.useState(false);
    return React.createElement(ActivityFilter, {
      selectedFilters: selected ? ["status"] : [],
      filterOptions: [
        {
          key: "status",
          labelTranslationKey: "filter.status",
          isSelected: selected,
          onClick: () => setSelected((value) => !value),
        },
      ],
    });
  }

  await environment.render(React.createElement(Harness));
  await act(async () => environment.container.querySelector("button").click());
  const option = document.querySelector('button[aria-pressed="false"]');
  assert.ok(option, "the filter option must be a native toggle button");
  option.focus();
  assert.equal(document.activeElement, option);
  await act(async () => {
    option.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
    // jsdom does not synthesize native keyboard activation.
    option.click();
  });
  assert.equal(option.getAttribute("aria-pressed"), "true");
});

for (const [name, modulePath, exportName, visibleOption] of [
  [
    "notification header menu",
    "../../apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx",
    "NotificationHeaderMenuOption",
    "notification.options.show_unread",
  ],
  [
    "notification filter menu",
    "../../apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx",
    "NotificationFilter",
    "assigned",
  ],
  [
    "issue activity filter",
    "../../apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx",
    "ActivityFilter",
    "filter.status",
  ],
]) {
  test(`${name} uses one trigger and opens/closes with focus restoration`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const errors = [];
    t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
    const componentModule = await import(modulePath);

    const props =
      exportName === "ActivityFilter"
        ? {
            selectedFilters: [],
            filterOptions: [{ key: "status", labelTranslationKey: "filter.status", isSelected: false, onClick() {} }],
          }
        : undefined;
    await environment.render(React.createElement(componentModule[exportName], props));

    const buttons = environment.container.querySelectorAll("button");
    assert.equal(buttons.length, 1, environment.container.innerHTML);
    assert.equal(buttons[0].querySelector("button"), null);
    assert.equal(buttons[0].getAttribute("aria-expanded"), "false");
    if (exportName === "NotificationFilter") {
      assert.equal(buttons[0].getAttribute("aria-label"), "notification.options.filters");
      assert.equal(buttons[0].getAttribute("title"), "notification.options.filters");
    }

    const trigger = buttons[0];
    trigger.focus();
    await act(async () => trigger.click());
    assert.equal(trigger.getAttribute("aria-expanded"), "true");
    assert.ok([...document.querySelectorAll("body *")].some((element) => element.textContent.trim() === visibleOption));

    await act(async () => {
      document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    });
    assert.equal(trigger.getAttribute("aria-expanded"), "false");
    assert.equal(document.activeElement, trigger);
    assert.deepEqual(errors, []);
  });
}
