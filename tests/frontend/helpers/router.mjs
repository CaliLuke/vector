import { React, act } from "./react.mjs";
import { registerHooks } from "node:module";
import { routes } from "../../../apps/web/routes.ts";

// jsdom exercises TanStack's browser entry points. Keep Node resolution for
// unrelated packages, which may be imported before a DOM is installed.
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.startsWith("@tanstack/")) {
      return nextResolve(specifier, { ...context, conditions: [...context.conditions, "browser"] });
    }
    return nextResolve(specifier, context);
  },
});
export const routerModuleUrl = import.meta.resolve("@tanstack/react-router");
export const routerApi = await import(routerModuleUrl);

const Content = React.createContext(null);
function TestContent() {
  return React.useContext(Content);
}

// Use the production URL/layout tree with real in-memory routing. Components
// under test keep their route context while services and other UI are isolated.
export function createRouterFixture(initialPath = "/workspace/", { modules = {} } = {}) {
  const { createMemoryHistory, createRootRoute, createRoute, createRouter, Outlet, RouterProvider } = routerApi;
  const root = createRootRoute({ component: Outlet, notFoundComponent: TestContent });
  function build(node, parent) {
    const options = {
      getParentRoute: () => parent,
      ...(node.type === "layout" ? { id: node.id } : { path: node.type === "index" ? "/" : node.path }),
      component: node.children ? Outlet : TestContent,
    };
    const route = modules[node.file]?.update(options) ?? createRoute(options);
    return node.children ? route.addChildren(node.children.map((child) => build(child, route))) : route;
  }
  const router = createRouter({
    routeTree: root.addChildren(routes.children.map((node) => build(node, root))),
    history: createMemoryHistory({ initialEntries: [initialPath] }),
    isServer: false,
    trailingSlash: "preserve",
    defaultPendingMinMs: 0,
  });
  function Provider({ children }) {
    return React.createElement(Content.Provider, { value: children }, React.createElement(RouterProvider, { router }));
  }
  return {
    router,
    Provider,
    async render(environment, element) {
      await act(async () => {
        await router.load();
      });
      await environment.render(React.createElement(Provider, null, element));
    },
  };
}
