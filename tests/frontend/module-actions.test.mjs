import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.React = React;
globalThis.moduleActionCalls = [];
globalThis.moduleActionToasts = [];
globalThis.favoriteMenuChanges = [];
globalThis.favoriteMenuStored = false;

const packageStubs = {
  "@plane/constants":
    "export const IS_FAVORITE_MENU_OPEN = 'favorite-menu-open'; export const TOAST_TYPE = { SUCCESS: 'success', ERROR: 'error' };",
  "@plane/hooks": `export const useLocalStorage = () => ({ storedValue: globalThis.favoriteMenuStored, setValue: value => globalThis.favoriteMenuChanges.push(value) });`,
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: 'success', ERROR: 'error' }; export const setToast = value => globalThis.moduleActionToasts.push(value); export const setPromiseToast = (promise, options) => globalThis.moduleActionToasts.push({ promise, options });`,
  "@plane/types": "",
  "@/hooks/store/use-module": `export const useModule = () => ({ addModuleToFavorites: (...args) => { globalThis.moduleActionCalls.push(['add', ...args]); return globalThis.addModulePromise; }, removeModuleFromFavorites: (...args) => { globalThis.moduleActionCalls.push(['remove', ...args]); return Promise.resolve(); }, updateModuleDetails: (...args) => { globalThis.moduleActionCalls.push(['update', ...args]); return globalThis.updateModulePromise; } });`,
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

async function renderActions(environment, params = {}) {
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
  await environment.render(React.createElement(Actions));
}

await test("module actions preserve favorite and update outcomes", async (t) => {
  globalThis.moduleActionCalls = [];
  globalThis.moduleActionToasts = [];
  globalThis.favoriteMenuChanges = [];
  globalThis.favoriteMenuStored = false;
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await renderActions(environment);

  const addRequest = deferred();
  globalThis.addModulePromise = addRequest.promise;
  const addFavorite = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Add favorite"
  );
  await React.act(async () => addFavorite.click());
  assert.deepEqual(globalThis.moduleActionCalls, [["add", "acme", "project-1", "module-1"]]);
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
  assert.equal(globalThis.moduleActionToasts.at(-1).message, "Permission denied");

  const fallbackFailure = deferred();
  globalThis.updateModulePromise = fallbackFailure.promise;
  await React.act(async () => {
    updateButton.click();
    fallbackFailure.reject({ detail: { message: "not a string" } });
    await fallbackFailure.promise.catch(() => undefined);
  });
  assert.equal(globalThis.moduleActionToasts.at(-1).message, "Module could not be updated. Please try again.");
});

await test("module actions do not call stores when route identifiers are absent", async (t) => {
  globalThis.moduleActionCalls = [];
  globalThis.moduleActionToasts = [];
  globalThis.favoriteMenuChanges = [];
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await renderActions(environment, { workspaceSlug: undefined, projectId: undefined });

  await React.act(async () => {
    for (const button of environment.container.querySelectorAll("button")) button.click();
  });
  assert.deepEqual(globalThis.moduleActionCalls, []);
  assert.deepEqual(globalThis.moduleActionToasts, []);
  assert.deepEqual(globalThis.favoriteMenuChanges, []);
});
