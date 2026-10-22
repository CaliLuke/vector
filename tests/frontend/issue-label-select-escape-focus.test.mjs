import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const packageStubs = {
  "@plane/constants":
    "export const getRandomLabelColor = () => '#000000'; export const EUserPermissionsLevel = { PROJECT: 'PROJECT' };",
  "@plane/types": "export const EUserProjectRoles = { ADMIN: 20 };",
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/i18n": `export const useTranslation = () => ({ t: key => ({ search: "Search", loading: "Loading", labels: "Labels" })[key] ?? key });`,
  "@plane/propel/icons":
    "export const CheckIcon = () => null; export const SearchIcon = () => null; export const LabelPropertyIcon = () => null; export const PlusIcon = () => null;",
  "@plane/utils":
    "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' '); export const getTabIndex = () => ({ baseTabIndex: 0 });",
  "@/components/ui/labels-list": "export const IssueLabelsList = () => null;",
  "@/hooks/use-platform-os": "export const usePlatformOS = () => ({ isMobile: false });",
  "@/hooks/store/use-label":
    "export const useLabel = () => ({ getProjectLabels: () => [{ id: 'existing', name: 'Existing label', color: '#000000' }], fetchProjectLabels: async () => {} });",
  "@/hooks/store/user": "export const useUserPermissions = () => ({ allowPermissions: () => false });",
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/propel/button") {
      return nextResolve(pathToFileURL(path.join(repositoryRoot, "packages/propel/src/button/index.ts")).href, context);
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    if (specifier === "@headlessui/react") {
      const webRoot = path.join(repositoryRoot, "apps/web");
      return nextResolve(specifier, { ...context, parentURL: pathToFileURL(path.join(webRoot, "index.js")).href });
    }
    return nextResolve(specifier, context);
  },
});

test("ticket detail label picker returns focus to its real trigger on Escape", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { IssueLabelSelect } =
    await import("../../apps/web/core/components/issues/issue-detail/label/select/label-select.tsx");
  await environment.render(
    React.createElement(IssueLabelSelect, {
      workspaceSlug: "workspace",
      projectId: "project",
      issueId: "issue",
      values: [],
      onSelect() {},
      onAddLabel: async () => {
        throw new Error("creation is disabled");
      },
    })
  );
  const trigger = environment.container.querySelector("button");
  await React.act(async () => trigger.click());
  const search = document.querySelector('[role="combobox"]');
  assert.ok(search);
  await React.act(() => search.focus());
  assert.equal(document.activeElement, search);
  await React.act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  assert.equal(document.querySelector('[role="combobox"]'), null);
  const focusReturned = document.activeElement === trigger;
  assert.equal(focusReturned, true, "Escape should return focus to the ticket label trigger");
});

test("Escape closes the label popup and returns focus to its trigger", async () => {
  const environment = await createReactDomTestEnvironment();
  const { WorkItemLabelSelectBase } = await import("../../apps/web/core/components/issues/select/base.tsx");
  await environment.render(
    React.createElement(WorkItemLabelSelectBase, {
      getLabelById: () => null,
      labelIds: [],
      onChange() {},
      value: [],
    })
  );

  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  await React.act(() => trigger.click());
  const search = document.querySelector('input[placeholder="Search"]');
  assert.ok(search);
  assert.equal(document.activeElement, search);

  await React.act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });

  assert.equal(document.querySelector('input[placeholder="Search"]'), null);
  const activeElementAfterEscape = document.activeElement;
  await environment.unmount();
  assert.equal(activeElementAfterEscape, trigger);
});

test("label creation stays keyboard accessible and empty search keeps creation disabled", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { WorkItemLabelSelectBase } = await import("../../apps/web/core/components/issues/select/base.tsx");
  const created = [];
  const selected = [];
  await environment.render(
    React.createElement(WorkItemLabelSelectBase, {
      createLabelEnabled: true,
      createLabel: async (label) => {
        created.push(label);
        return { ...label, id: "new-label" };
      },
      getLabelById: () => null,
      labelIds: [],
      onChange: (ids) => selected.push(ids),
      value: [],
    })
  );
  await React.act(() => environment.container.querySelector("button").click());
  const search = document.querySelector('input[placeholder="Search"]');
  const action = document.querySelector('[role="listbox"] button');
  assert.ok(action);
  assert.equal(action.disabled, true);
  await React.act(() => action.click());
  assert.equal(created.length, 0);

  await React.act(() => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(search, "Review label");
    search.dispatchEvent(new Event("input", { bubbles: true }));
  });
  assert.equal(action.disabled, false);
  await React.act(() => action.focus());
  await React.act(() => {
    for (const key of ["Enter", " "]) {
      const event = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
      action.dispatchEvent(event);
      assert.equal(event.defaultPrevented, false, `${key} must retain native button activation`);
    }
  });
  assert.ok(document.querySelector('input[placeholder="Search"]'), "activation must not toggle the dropdown");
  // JSDOM does not synthesize the native click from a keyboard activation.
  await React.act(async () => action.click());
  assert.deepEqual(created, [{ name: "Review label", color: "#000000" }]);
  assert.deepEqual(selected, [["new-label"]]);
  assert.equal(document.querySelector('[role="listbox"] button').disabled, true);
});
