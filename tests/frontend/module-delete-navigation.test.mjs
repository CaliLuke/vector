import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createRouterFixture, routerApi } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { React, act, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));
const deleteModalUrl = pathToFileURL(`${repositoryRoot}/apps/web/core/components/modules/delete-module-modal.tsx`).href;
const alertModalUrl = pathToFileURL(`${repositoryRoot}/packages/ui/src/modals/alert-modal.tsx`).href;
const contextMenuUrl = pathToFileURL(`${repositoryRoot}/packages/ui/src/dropdowns/context-menu/index.ts`).href;
const customMenuUrl = pathToFileURL(`${repositoryRoot}/packages/ui/src/dropdowns/custom-menu.tsx`).href;
const mobx = webRequire("mobx");
const state = { React, moduleRecord: undefined, resolveDelete: undefined, runInAction: mobx.runInAction };
globalThis.__moduleDeleteNavigationTest = state;
registerWebModuleHooks();

const stubs = {
  "@plane/i18n": `export const useTranslation = () => ({ t: (value) => value });`,
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: "success", ERROR: "error" }; export const setToast = () => {};`,
  "@plane/ui": `export { AlertModalCore } from ${JSON.stringify(alertModalUrl)};
    export { ContextMenu } from ${JSON.stringify(contextMenuUrl)};
    export { CustomMenu } from ${JSON.stringify(customMenuUrl)};`,
  "@/components/modules": `export { DeleteModuleModal } from ${JSON.stringify(deleteModalUrl)};
    export const ArchiveModuleModal = () => null;
    export const CreateUpdateModuleModal = () => null;`,
  "@/components/common/quick-actions-helper": `export const useModuleMenuItems = ({ handleDelete }) => [
    { key: "delete-module", title: "Delete module", action: handleDelete },
  ];`,
  "@/hooks/store/use-module": `export const useModule = () => ({
    getModuleById: (id) => globalThis.__moduleDeleteNavigationTest.moduleRecord.value?.id === id
      ? globalThis.__moduleDeleteNavigationTest.moduleRecord.value : undefined,
    deleteModule: (...args) => {
      const state = globalThis.__moduleDeleteNavigationTest;
      state.deleteArgs = args;
      state.runInAction(() => {
        state.moduleRecord.value = undefined;
      });
      return new Promise((resolve) => { state.resolveDelete = resolve; });
    },
    restoreModule: async () => {},
  });`,
  "@/hooks/store/user": `export const useUserPermissions = () => ({ allowPermissions: () => true });`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

const { useLocation } = routerApi;

function LocationOutput() {
  const { pathname } = useLocation();
  return React.createElement("span", { "data-testid": "pathname" }, pathname);
}

test("module quick actions redirect after successful deletion unmounts the modal", async () => {
  const environment = await createReactDomTestEnvironment();
  const originalResizeObserver = globalThis.ResizeObserver;
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
  state.moduleRecord = mobx.observable({ value: { id: "module-id", project_id: "project-id", name: "Module" } });
  state.resolveDelete = undefined;
  state.deleteArgs = undefined;
  let ModuleQuickActions;

  function Harness() {
    const parentRef = React.useRef(null);
    return React.createElement(
      "div",
      { ref: parentRef },
      React.createElement(ModuleQuickActions, {
        parentRef,
        moduleId: "module-id",
        projectId: "project-id",
        workspaceSlug: "default",
      })
    );
  }

  try {
    ({ ModuleQuickActions } = await import("../../apps/web/core/components/modules/quick-actions.tsx"));
    const fixture = createRouterFixture("/default/projects/project-id/modules/module-id");
    await fixture.render(
      environment,
      React.createElement(React.Fragment, null, React.createElement(LocationOutput), React.createElement(Harness))
    );

    const menuTrigger = environment.container.querySelector("[data-main-menu] button");
    assert.ok(menuTrigger, "the real quick actions menu trigger should render");
    await act(async () => menuTrigger.click());
    const deleteMenuItem = [...document.querySelectorAll("button")].find((button) =>
      button.textContent.includes("Delete module")
    );
    assert.ok(deleteMenuItem, "the real custom menu should expose the delete action");
    await act(async () => deleteMenuItem.click());

    const confirmDelete = [...document.querySelectorAll("button")].find(
      (button) => button.textContent.trim() === "Delete"
    );
    assert.ok(confirmDelete, "the real alert modal should render its delete confirmation");
    await act(async () => confirmDelete.click());
    assert.deepEqual(state.deleteArgs, ["default", "project-id", "module-id"]);
    assert.ok(
      environment.container.querySelector("[data-main-menu]"),
      "the stable quick actions owner should remain mounted"
    );
    assert.equal(
      [...document.querySelectorAll("button")].some((button) => button.textContent.trim() === "Delete"),
      false,
      "module removal should unmount the in-flight modal before the delete promise resolves"
    );

    await act(async () => {
      state.resolveDelete();
      await new Promise((resolve) => setTimeout(resolve, 10));
    });

    assert.equal(
      environment.container.querySelector('[data-testid="pathname"]')?.textContent,
      "/default/projects/project-id/modules"
    );
  } finally {
    await environment.unmount();
    if (originalResizeObserver) globalThis.ResizeObserver = originalResizeObserver;
    else delete globalThis.ResizeObserver;
  }
});
