import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.React = React;
globalThis.moduleActionCalls = [];
globalThis.moduleActionToasts = [];
globalThis.favoriteMenuChanges = [];
globalThis.favoriteMenuStored = false;

const packageStubs = {
  "@plane/hooks": `export const useLocalStorage = () => ({ storedValue: globalThis.favoriteMenuStored, setValue: value => globalThis.favoriteMenuChanges.push(value) });`,
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: 'success', ERROR: 'error' }; export const setToast = value => globalThis.moduleActionToasts.push(value); export const setPromiseToast = (promise, options) => globalThis.moduleActionToasts.push({ promise, options });`,

  "@/hooks/use-favorites": `export const useFavorites = workspaceSlug => ({ favorites: [], favoritesByEntity: new Map(), toggleFavorite: (...args) => { globalThis.moduleActionCalls.push(['toggle', workspaceSlug, ...args]); return globalThis.addModulePromise; } });`,
  "@/hooks/use-member-rosters": `export const useWorkspaceMemberRoster=()=>({membersById:new Map()});`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

async function actionFixture(t) {
  const view = await createPermissionUiFixture(t, {
    path: "/acme/projects/project-1/modules",
    workspaceRole: 15,
    projectRoles: { "project-1": 20 },
  });
  const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
  t.mock.method(ModuleService.prototype, "getModules", async () => [
    { id: "module-1", project_id: "project-1", workspace_id: "w1", name: "Original" },
  ]);
  t.mock.method(ModuleService.prototype, "patchModule", async (...args) => {
    globalThis.moduleActionCalls.push(["update", ...args]);
    await globalThis.updateModulePromise;
    return { id: "module-1", project_id: "project-1", workspace_id: "w1", ...args[3] };
  });
  return view;
}
async function renderActions(view, params = {}) {
  const workspaceSlug = Object.hasOwn(params, "workspaceSlug") ? params.workspaceSlug : "acme";
  const projectId = Object.hasOwn(params, "projectId") ? params.projectId : "project-1";
  const { useModuleActions } = await import("../../apps/web/core/components/modules/use-module-actions.ts");
  function Actions() {
    const { handleAddToFavorites, handleRemoveFromFavorites, handleModuleDetailsChange } = useModuleActions({
      moduleId: "module-1",
      workspaceSlug,
      projectId,
    });
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("button", { type: "button", onClick: handleAddToFavorites }, "Add favorite"),
      React.createElement("button", { type: "button", onClick: handleRemoveFromFavorites }, "Remove favorite"),
      React.createElement(
        "button",
        {
          type: "button",
          onClick: () => handleModuleDetailsChange({ name: "Updated module" }),
        },
        "Update module"
      )
    );
  }
  await view.render(React.createElement(Actions));
}

await test("module actions preserve favorite and update outcomes", async (t) => {
  globalThis.moduleActionCalls = [];
  globalThis.moduleActionToasts = [];
  globalThis.favoriteMenuChanges = [];
  globalThis.favoriteMenuStored = false;
  const view = await actionFixture(t);
  const environment = view.environment;
  await renderActions(view);

  const addRequest = deferred();
  globalThis.addModulePromise = addRequest.promise;
  const addFavorite = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Add favorite"
  );
  await React.act(async () => addFavorite.click());
  assert.deepEqual(globalThis.moduleActionCalls, [["toggle", "acme", "module", "module-1", "project-1"]]);
  assert.deepEqual(globalThis.favoriteMenuChanges, [], "the favorites menu stays closed while the request is pending");
  const addPromise = globalThis.moduleActionToasts[0].promise;
  addRequest.resolve();
  await React.act(async () => await addPromise);
  assert.deepEqual(globalThis.favoriteMenuChanges, [true], "successful addition opens the favorites menu");

  const updateSuccess = deferred();
  globalThis.updateModulePromise = updateSuccess.promise;
  const updateButton = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Update module"
  );
  await React.act(async () => {
    updateButton.click();
    updateSuccess.resolve();
    await updateSuccess.promise;
  });
  assert.deepEqual(globalThis.moduleActionCalls.at(-1), [
    "update",
    "acme",
    "project-1",
    "module-1",
    { name: "Updated module" },
  ]);
  await settleQuery();
  assert.deepEqual(globalThis.moduleActionToasts.at(-1), {
    type: "success",
    title: "Success!",
    message: "Module updated successfully.",
  });

  const detailFailure = deferred();
  globalThis.updateModulePromise = detailFailure.promise;
  await React.act(async () => {
    updateButton.click();
    detailFailure.reject({ detail: "Permission denied" });
    await detailFailure.promise.catch(() => undefined);
  });
  await settleQuery();
  assert.equal(globalThis.moduleActionToasts.at(-1).message, "Permission denied");

  const fallbackFailure = deferred();
  globalThis.updateModulePromise = fallbackFailure.promise;
  await React.act(async () => {
    updateButton.click();
    fallbackFailure.reject({ detail: { message: "not a string" } });
    await fallbackFailure.promise.catch(() => undefined);
  });
  await settleQuery();
  assert.equal(globalThis.moduleActionToasts.at(-1).message, "Module could not be updated. Please try again.");
});

await test("module actions do not send requests when route identifiers are absent", async (t) => {
  globalThis.moduleActionCalls = [];
  globalThis.moduleActionToasts = [];
  globalThis.favoriteMenuChanges = [];
  const view = await actionFixture(t);
  const environment = view.environment;
  await renderActions(view, { workspaceSlug: undefined, projectId: undefined });

  await React.act(async () => {
    for (const button of environment.container.querySelectorAll("button")) button.click();
  });
  assert.deepEqual(globalThis.moduleActionCalls, []);
  assert.deepEqual(globalThis.moduleActionToasts, []);
  assert.deepEqual(globalThis.favoriteMenuChanges, []);
});
