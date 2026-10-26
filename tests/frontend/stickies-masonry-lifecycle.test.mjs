import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { registerHooks } from "node:module";
import { EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

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
  "next-themes": 'export const useTheme = () => ({ resolvedTheme: "light" });',
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/propel/icons": "export const PlusIcon = () => null;",
  "@/components/empty-state/detailed-empty-state-root": "export const DetailedEmptyState = () => null;",
  "@/components/empty-state/simple-empty-state-root": "export const SimpleEmptyState = () => null;",
  "@/components/home/widgets/empty-states/stickies": "export const StickiesEmptyState = () => null;",
  "@/services/user.service":
    "export class UserService { currentUser() { throw new Error('unexpected identity request'); } } export default new UserService();",
  "@/services/workspace.service":
    "export class WorkspaceService { workspaceMemberMe() { throw new Error('unexpected membership request'); } getWorkspaceUserProjectsRole() { throw new Error('unexpected roles request'); } }",
  "@/components/stickies/provider": "export const useStickies = () => globalThis.masonryLifecycleForTest.store;",
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
    stickyIds: ids,
    list: { isPending: loader === "init-loader", isError: false },
    searchQuery: "",
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
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  const cancelledEnvironment = await createReactDomTestEnvironment();
  const cancelledClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  cancelledClient.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(cancelledClient, {
    userId: "alice",
    workspaceSlug: "workspace",
    workspaceRole: EUserWorkspaceRoles.ADMIN,
  });
  t.after(() => cancelledEnvironment.unmount());
  await createRouterFixture("/workspace/stickies/").render(
    cancelledEnvironment,
    React.createElement(
      QueryClientProvider,
      { client: cancelledClient },
      React.createElement(StickiesList, props(["pending"]))
    )
  );
  await lifecycle.importReady;
  await cancelledEnvironment.unmount();
  await cancelledClient.cancelQueries();
  cancelledClient.clear();
  lifecycle.resolveImport();
  await settleEffects();
  assert.equal(lifecycle.instances.length, 0);

  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "workspace",
    workspaceRole: EUserWorkspaceRoles.ADMIN,
  });
  const routerFixture = createRouterFixture("/workspace/stickies/");
  t.after(async () => {
    await environment.unmount();
    await client.cancelQueries();
    client.clear();
  });

  await routerFixture.render(
    environment,
    React.createElement(QueryClientProvider, { client }, React.createElement(StickiesList, props(["one", "two"])))
  );
  await settleEffects();
  const [masonry] = lifecycle.instances;
  assert.ok(masonry);
  assert.equal(masonry.options.itemSelector, ".sticky-masonry-item");
  assert.equal(environment.container.querySelectorAll(".sticky-masonry-item").length, 2);

  const firstItem = environment.container.querySelector(".sticky-masonry-item");
  const layoutCallsBeforeResize = masonry.calls.layout;
  await environment.resize(firstItem);
  assert.equal(masonry.calls.layout, layoutCallsBeforeResize + 1);

  const callsBeforeItemChange = { layout: masonry.calls.layout, reloadItems: masonry.calls.reloadItems };
  await routerFixture.render(
    environment,
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(StickiesList, props(["one", "two", "three"], 700))
    )
  );
  await settleEffects();
  assert.equal(environment.container.querySelectorAll(".sticky-masonry-item").length, 3);
  assert.ok(masonry.calls.reloadItems > callsBeforeItemChange.reloadItems);
  assert.ok(masonry.calls.layout > callsBeforeItemChange.layout);

  const layoutCallsBeforeRemoval = masonry.calls.layout;
  await routerFixture.render(
    environment,
    React.createElement(QueryClientProvider, { client }, React.createElement(StickiesList, props([], 700)))
  );
  assert.equal(masonry.calls.destroy, 1);
  await environment.resize(firstItem);
  assert.equal(masonry.calls.layout, layoutCallsBeforeRemoval);
});
