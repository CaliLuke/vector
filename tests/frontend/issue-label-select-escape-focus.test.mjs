import { setPositioningGeometry } from "./helpers/positioning.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { openSearchCombobox } from "./helpers/combobox.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const packageStubs = {
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/i18n": `export const useTranslation = () => ({ t: key => ({ search: "Search", loading: "Loading", labels: "Labels" })[key] ?? key });`,
  "@plane/propel/icons":
    "export const CheckIcon = () => null; export const SearchIcon = () => null; export const LabelPropertyIcon = () => null; export const PlusIcon = () => null; export const ChevronDownIcon = () => null;",
  "@/components/ui/labels-list": "export const IssueLabelsList = () => null;",
  "@/hooks/use-platform-os": "export const usePlatformOS = () => ({ isMobile: false });",
  "@/services/user.service":
    "export class UserService { currentUser() { throw new Error('unexpected identity request'); } } export default new UserService();",
  "@/services/workspace.service":
    "export class WorkspaceService { workspaceMemberMe() { throw new Error('unexpected membership request'); } getWorkspaceUserProjectsRole() { throw new Error('unexpected roles request'); } }",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/ui")
      return nextResolve(new URL("../../packages/ui/src/dropdowns/combo-box.tsx", import.meta.url).href, context);
    if (specifier === "@plane/propel/button") {
      return nextResolve(pathToFileURL(path.join(repositoryRoot, "packages/propel/src/button/index.ts")).href, context);
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    if (specifier === "@headlessui/react" || specifier === "react-popper") {
      const webRoot = path.join(repositoryRoot, "apps/web");
      return nextResolve(specifier, { ...context, parentURL: pathToFileURL(path.join(webRoot, "index.js")).href });
    }
    return nextResolve(specifier, context);
  },
});

function positionedLabelTrigger(environment) {
  const trigger = environment.container.querySelector("button");
  setPositioningGeometry(trigger);
  return trigger;
}

function assertPositionedLabelListbox() {
  const menu = document.querySelector('[role="listbox"]');
  assert.ok(menu, "the label options are open");
  assert.equal(menu.getAttribute("data-popper-placement"), "bottom-start");
  assert.equal(menu.style.transform, "translate(400px, 132px)");
}

await test("ticket detail label picker returns focus to its real trigger on Escape", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { IssueLabelSelect } =
    await import("../../apps/web/core/components/issues/issue-detail/label/select/label-select.tsx");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "workspace",
    projectRoles: { project: 20 },
  });
  const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => []);
  t.after(() => client.clear());
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
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
    )
  );
  const trigger = positionedLabelTrigger(environment);
  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assertPositionedLabelListbox();
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

await test("Escape closes the label popup and returns focus to its trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { WorkItemLabelSelectBase } = await import("../../apps/web/core/components/issues/select/base.tsx");
  await environment.render(
    React.createElement(WorkItemLabelSelectBase, {
      getLabelById: () => null,
      labelIds: [],
      onChange() {},
      value: [],
    })
  );

  const trigger = positionedLabelTrigger(environment);
  const search = await openSearchCombobox(trigger);
  assertPositionedLabelListbox();

  await React.act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });

  assert.equal(document.querySelector('input[placeholder="Search"]'), null);
  const activeElementAfterEscape = document.activeElement;
  assert.equal(activeElementAfterEscape, trigger);
  await openSearchCombobox(trigger);
  assertPositionedLabelListbox();
});

await test("label creation stays keyboard accessible and empty search keeps creation disabled", async (t) => {
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
  await clickWithPointer(environment.container.querySelector("button"));
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
  assert.equal(created.length, 1);
  assert.equal(created[0].name, "Review label");
  assert.ok(created[0].color);
  assert.deepEqual(selected, [["new-label"]]);
  assert.equal(document.querySelector('[role="listbox"] button').disabled, true);
});

await test("layout label creation preserves its draft after failure and permits a retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { LabelDropdown } =
    await import("../../apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, { userId: "alice", workspaceSlug: "workspace", projectRoles: { project: 20 } });
  t.after(() => client.clear());
  let fail = true;
  let selected;
  const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => []);
  t.mock.method(IssueLabelService.prototype, "createIssueLabel", async (_workspace, _project, label) => {
    if (fail) throw new Error("Offline");
    return { id: "created", project_id: "project", ...label };
  });
  const router = createRouterFixture("/workspace/");
  await router.render(
    environment,
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(LabelDropdown, {
        projectId: "project",
        value: [],
        onChange: (ids) => {
          selected = ids;
        },
        label: "Labels",
      })
    )
  );
  const trigger = positionedLabelTrigger(environment);
  await clickWithPointer(trigger);
  const search = document.querySelector('[role="combobox"]');
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assertPositionedLabelListbox();
  await React.act(() => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(search, "New label");
    search.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await React.act(async () => document.querySelector('[role="listbox"] button').click());
  assert.ok(document.querySelector('[role="alert"]'));
  assertPositionedLabelListbox();
  assert.equal(search.value, "New label");
  assert.equal(selected, undefined);
  fail = false;
  await React.act(async () => document.querySelector('[role="listbox"] button').click());
  assert.deepEqual(selected, ["created"]);
  assert.equal(search.value, "");
  assert.equal(document.querySelector('[role="alert"]'), null);
  assertPositionedLabelListbox();
});

async function expectCreationRetry(environment, create) {
  const trigger = environment.container.querySelector("button");
  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  const search = document.querySelector('input[role="combobox"]');
  assert.ok(search);
  await React.act(() => search.focus());
  await React.act(() => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(search, "  Retry label  ");
    search.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await React.act(async () => document.querySelector('[role="listbox"] button').click());
  assert.equal(search.readOnly, true, "pending creation locks the submitted draft");
  assert.equal(create.names.length, 1);
  assert.equal(create.names[0], "Retry label");
  assert.deepEqual(create.selected, []);

  await React.act(async () => create.reject(new Error("Offline")));
  assert.ok(document.querySelector('[role="alert"]'), "creation failure is visible in the popup");
  assert.equal(search.value, "  Retry label  ", "failed submission keeps the exact editable draft");
  assert.equal(search.readOnly, false, "rejection releases the form for retry");
  assert.deepEqual(create.selected, [], "a label name must never be emitted as a selected label ID");

  await React.act(async () => document.querySelector('[role="listbox"] button').click());
  assert.equal(create.names.length, 2);
  assert.deepEqual(create.selected, [["created-label"]]);
  assert.equal(search.value, "");
  assert.equal(document.querySelector('[role="alert"]'), null);
}

function creationFixture() {
  let reject;
  const pending = new Promise((_resolve, rejectRequest) => {
    reject = rejectRequest;
  });
  const fixture = {
    names: [],
    selected: [],
    reject: (error) => reject(error),
    create: async ({ name }) => {
      fixture.names.push(name);
      if (fixture.names.length === 1) await pending;
      return { id: "created-label", name };
    },
  };
  return fixture;
}

await test("new-work-item label creation keeps rejected drafts and retries without closing its search", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { WorkItemLabelSelectBase } = await import("../../apps/web/core/components/issues/select/base.tsx");
  const create = creationFixture();
  await environment.render(
    React.createElement(WorkItemLabelSelectBase, {
      getLabelById: () => null,
      labelIds: [],
      value: [],
      createLabelEnabled: true,
      createLabel: create.create,
      onChange: (ids) => create.selected.push(ids),
    })
  );
  await expectCreationRetry(environment, create);
});

await test("detail label creation releases a rejected submission and retains the label name for retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { IssueLabelSelect } =
    await import("../../apps/web/core/components/issues/issue-detail/label/select/label-select.tsx");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, { userId: "alice", workspaceSlug: "workspace", projectRoles: { project: 20 } });
  const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => []);
  const create = creationFixture();
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(IssueLabelSelect, {
        workspaceSlug: "workspace",
        projectId: "project",
        issueId: "issue",
        values: [],
        onAddLabel: create.create,
        onSelect: (ids) => create.selected.push(ids),
      })
    )
  );
  await expectCreationRetry(environment, create);
});

await test("a label creation from an earlier picker scope cannot select into a replacement scope", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { WorkItemLabelSelectBase } = await import("../../apps/web/core/components/issues/select/base.tsx");
  let accept;
  const pending = new Promise((resolve) => {
    accept = resolve;
  });
  const selected = [];
  const props = {
    getLabelById: () => null,
    labelIds: [],
    value: [],
    createLabelEnabled: true,
    createLabel: async () => pending,
    onChange: (ids) => selected.push(ids),
  };
  const render = (creationScope) =>
    environment.render(React.createElement(WorkItemLabelSelectBase, { ...props, creationScope }));
  await render("A");
  const search = await openSearchCombobox(environment.container.querySelector("button"));
  await React.act(() => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(search, "Old scope label");
    search.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await React.act(async () => document.querySelector('[role="listbox"] button').click());
  assert.equal(search.readOnly, true);
  await render("B");
  await render("A");
  await React.act(async () => accept({ id: "old-scope-label", name: "Old scope label" }));
  assert.deepEqual(selected, []);
  const replacement = await openSearchCombobox(environment.container.querySelector("button"));
  assert.equal(replacement.value, "");
  assert.equal(replacement.readOnly, false);
});
