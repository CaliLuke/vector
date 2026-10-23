import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { createReactDomTestEnvironment, React, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const reactModuleUrl = pathToFileURL(webRequire.resolve("react")).href;
const lifecycle = { calls: [], instances: [] };
let signalImportReady;
lifecycle.importReady = new Promise((resolve) => {
  signalImportReady = resolve;
});
lifecycle.signalImportReady = () => signalImportReady();
const modules = {
  "masonry-layout": `await new Promise(resolve => {
    globalThis.masonryLifecycleForTest.resolveImport = resolve;
    globalThis.masonryLifecycleForTest.signalImportReady();
  });
  export default class Masonry {
    constructor(container, options) {
      this.container = container;
      this.options = options;
      this.calls = { layout: 0, reloadItems: 0, destroy: 0 };
      globalThis.masonryLifecycleForTest.instances.push(this);
      globalThis.masonryLifecycleForTest.calls.push(["construct", options.itemSelector]);
    }
    layout() { this.calls.layout++; }
    reloadItems() { this.calls.reloadItems++; }
    destroy() { this.calls.destroy++; }
  }`,
  "mobx-react": "export const observer = component => component;",

  "next-themes": 'export const useTheme = () => ({ resolvedTheme: "light" });',
  "@plane/constants": "export const EUserPermissionsLevel = { WORKSPACE: 'workspace' };",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/propel/icons": "export const PlusIcon = () => null;",
  "@plane/types": "export const EUserWorkspaceRoles = { ADMIN: 'admin', MEMBER: 'member', GUEST: 'guest' };",
  "@/components/empty-state/detailed-empty-state-root": "export const DetailedEmptyState = () => null;",
  "@/components/empty-state/simple-empty-state-root": "export const SimpleEmptyState = () => null;",
  "@/components/home/widgets/empty-states/stickies": "export const StickiesEmptyState = () => null;",
  "@/hooks/store/user": "export const useUserPermissions = () => ({ allowPermissions: () => true });",
  "@/hooks/use-stickies": "export const useSticky = () => globalThis.masonryLifecycleForTest.store;",
  "../sticky/use-operations": "export const useStickyOperations = () => ({ stickyOperations: {} });",
  "./stickies-loader": "export const StickiesLoader = () => null;",
  "./sticky-dnd-wrapper": `import React from ${JSON.stringify(reactModuleUrl)};
    export const StickyDNDWrapper = ({ stickyId, workspaceSlug, itemWidth, handleDrop, isLastChild, isInFirstRow, isInLastRow, handleLayout, ...props }) => React.createElement("article", { ...props, "data-sticky-id": stickyId, style: { width: itemWidth } });`,
  "./sticky.helpers": "export const getInstructionFromPayload = () => null;",
};

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.startsWith("@/app/assets/")) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent('export default "asset";')}` };
    }
    if (Object.hasOwn(modules, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(modules[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

globalThis.masonryLifecycleForTest = lifecycle;
const { StickiesList } = await import("../../apps/web/core/components/stickies/layout/stickies-list.tsx");

function props(ids, containerWidth = 900, loader = "loaded") {
  lifecycle.store = {
    getWorkspaceStickyIds: () => ids,
    toggleShowNewSticky() {},
    searchQuery: "",
    loader,
  };
  return { workspaceSlug: "workspace", columnCount: 4, containerWidth };
}

async function settleEffects() {
  await React.act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

test("sticky masonry cancels pending imports and reloads changed cards until removed", async (t) => {
  lifecycle.calls.length = 0;
  lifecycle.instances.length = 0;
  const cancelledEnvironment = await createReactDomTestEnvironment();
  t.after(() => cancelledEnvironment.unmount());
  await createRouterFixture("/workspace/stickies/").render(
    cancelledEnvironment,
    React.createElement(StickiesList, props(["pending"]))
  );
  await lifecycle.importReady;
  await cancelledEnvironment.unmount();
  lifecycle.resolveImport();
  await settleEffects();
  assert.equal(lifecycle.instances.length, 0);

  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/workspace/stickies/");
  t.after(() => environment.unmount());

  await routerFixture.render(environment, React.createElement(StickiesList, props(["one", "two"])));
  await settleEffects();
  const [masonry] = lifecycle.instances;
  assert.ok(masonry);
  assert.equal(masonry.options.itemSelector, ".sticky-masonry-item");
  assert.equal(environment.container.querySelectorAll(".sticky-masonry-item").length, 2);

  const firstItem = environment.container.querySelector(".sticky-masonry-item");
  await environment.resize(firstItem);
  assert.equal(masonry.calls.layout, 1);

  await routerFixture.render(environment, React.createElement(StickiesList, props(["one", "two", "three"], 700)));
  assert.equal(environment.container.querySelectorAll(".sticky-masonry-item").length, 3);
  assert.ok(masonry.calls.reloadItems >= 1);
  assert.ok(masonry.calls.layout >= 2);

  const layoutCallsBeforeRemoval = masonry.calls.layout;
  await routerFixture.render(environment, React.createElement(StickiesList, props([], 700)));
  assert.equal(masonry.calls.destroy, 1);
  await environment.resize(firstItem);
  assert.equal(masonry.calls.layout, layoutCallsBeforeRemoval);
});
