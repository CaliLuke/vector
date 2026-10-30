import { createRouterFixture } from "./helpers/router.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import test from "node:test";
import { pathToFileURL, fileURLToPath } from "node:url";
import path from "node:path";
import { registerHooks } from "node:module";
import assert from "node:assert/strict";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const webRoot = path.join(repositoryRoot, "apps/web");

const moduleStubs = {
  "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element": "export const autoScrollForElements = () => () => {};",
  "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
  "@plane/propel/icons": `export const PlusIcon = () => null;
    export const ChevronRightIcon = () => null;
    export const Ellipsis = () => null;`,
  "@plane/propel/icon-button": `const React = globalThis.projectsListReact;
    export const IconButton = ({ icon: Icon, iconClassName, variant, size, ...props }) =>
      React.createElement("button", { type: "button", ...props }, Icon ? React.createElement(Icon) : null);`,
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: "success", ERROR: "error" };
    export const setToast = () => {};`,
  "@plane/propel/tooltip": `const React = globalThis.projectsListReact;
    export const Tooltip = ({ children }) => React.createElement(React.Fragment, null, children);`,
  "@plane/ui": `const React = globalThis.projectsListReact;
    const Item = () => React.createElement("div", { className: "project-loader-item" });
    export const Loader = Object.assign(({ children, ...props }) => React.createElement("div", { ...props, "data-testid": "project-loader" }, children), { Item });`,
  "@/components/project/create-project-modal": "export const CreateProjectModal = () => null;",
  "@/components/sidebar/sidebar-navigation": `const React = globalThis.projectsListReact;
    export const SidebarNavItem = ({ children }) => React.createElement("div", null, children);`,
  "@/hooks/use-command-palette": `export const useCommandPalette = () => ({ toggleCreateProjectModal() {} });`,
  "@/services/user.service":
    "export class UserService { currentUser() { throw new Error('unexpected identity request'); } } export default new UserService();",
  "@/services/workspace.service":
    "export class WorkspaceService { userWorkspaces() { throw new Error('unexpected workspace request'); } workspaceMemberMe() { throw new Error('unexpected membership request'); } getWorkspaceUserProjectsRole() { throw new Error('unexpected roles request'); } }",
  "@/hooks/use-navigation-preferences": `export const useProjectNavigationPreferences = () => ({ preferences: { showLimitedProjects: false, limitedProjectsCount: 20 } });`,
  "./projects-list-item": `const React = globalThis.projectsListReact;
    export const SidebarProjectsListItem = ({ projectId }) => React.createElement("button", { type: "button", "data-project-id": projectId }, "Project " + projectId);`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@headlessui/react") {
      return nextResolve(specifier, {
        ...context,
        parentURL: pathToFileURL(path.join(webRoot, "index.js")).href,
      });
    }
    if (Object.hasOwn(moduleStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(moduleStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

async function flushAnimationFrames() {
  await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
}

function waitForState(container, predicate, message) {
  return new Promise((resolve, reject) => {
    if (predicate()) return resolve();
    const observer = new MutationObserver(() => {
      if (!predicate()) return;
      observer.disconnect();
      clearTimeout(timeout);
      resolve();
    });
    observer.observe(container, { attributes: true, childList: true, subtree: true });
    const timeout = setTimeout(() => {
      observer.disconnect();
      reject(new Error(message));
    }, 1500);
  });
}

let SidebarProjectsList;

void test("projects sidebar transitions through initial loading and closed/open disclosure without ref errors", async (t) => {
  globalThis.projectsListReact = React;
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/workspace/");
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity, staleTime: Infinity },
      mutations: { gcTime: Infinity },
    },
  });
  const warnings = [];
  const originalError = console.error;
  const originalWarn = console.warn;
  const recordWarning = (...args) => {
    if (warnings.length < 50) warnings.push(args.join(" "));
  };
  console.error = recordWarning;
  console.warn = recordWarning;
  t.after(async () => {
    try {
      await environment.unmount(async () => {
        await client.cancelQueries();
        client.clear();
      });
    } finally {
      console.error = originalError;
      console.warn = originalWarn;
      delete globalThis.projectsListReact;
    }
  });

  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "workspace",
    workspaceRole: EUserWorkspaceRoles.GUEST,
    projectRoles: { "project-1": EUserWorkspaceRoles.MEMBER },
  });
  const { workspaceKeys } = await import("../../apps/web/core/queries/workspaces.ts");
  client.setQueryData(workspaceKeys.list("alice"), [{ id: "workspace-id", slug: "workspace" }]);

  const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
  const projects = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "getProjectsLite", () => projects.promise);

  ({ SidebarProjectsList } = await import("../../apps/web/core/components/workspace/sidebar/projects-list.tsx"));
  const { SidebarStateProvider } = await import("../../apps/web/core/providers/sidebar-state.tsx");
  await routerFixture.render(
    environment,
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(SidebarStateProvider, null, React.createElement(SidebarProjectsList))
    )
  );

  const loader = environment.container.querySelector('[data-testid="project-loader"]');
  const projectRow = environment.container.querySelector('[data-project-id="project-1"]');
  assert.ok(loader, "the initial request should render project-list skeleton rows");
  assert.equal(projectRow, null, "an initial catalogue load must not invent project rows");

  const closeButton = [
    ...environment.container.querySelectorAll('button[aria-label="aria_labels.projects_sidebar.close_projects_menu"]'),
  ].find((button) => !button.hasAttribute("aria-expanded"));
  assert.ok(closeButton, "the expanded disclosure button should be available");
  closeButton.focus();
  await React.act(() => closeButton.click());
  await React.act(flushAnimationFrames);
  await waitForState(
    environment.container,
    () =>
      environment.container.querySelector('[data-project-id="project-1"]') === null &&
      environment.container.querySelector('[data-testid="project-loader"]') === null,
    "closing the list should finish its leave transition"
  );
  const closedButton = [
    ...environment.container.querySelectorAll('button[aria-label="aria_labels.projects_sidebar.open_projects_menu"]'),
  ].find((button) => !button.hasAttribute("aria-expanded"));
  assert.ok(closedButton, "closing should update the disclosure button label");
  assert.ok(document.activeElement === closedButton, "closing should keep keyboard focus on the disclosure button");
  assert.ok(
    environment.container.querySelector('[data-testid="project-loader"]') === null,
    "loading skeleton should be unmounted"
  );
  assert.ok(
    environment.container.querySelector('[data-project-id="project-1"]') === null,
    "closed project row should be unmounted"
  );

  await React.act(async () => {
    projects.resolve([
      { id: "project-1", workspace: "workspace-id", name: "Project", identifier: "PRJ", sort_order: 1 },
    ]);
    await projects.promise;
  });
  const openButton = closedButton;
  assert.ok(openButton, "the closed disclosure button should be available");
  openButton.focus();
  await React.act(() => openButton.click());
  await React.act(flushAnimationFrames);
  await waitForState(
    environment.container,
    () => environment.container.querySelector('[data-project-id="project-1"]') !== null,
    "opening the list should finish its enter transition"
  );
  const reopenedButton = [
    ...environment.container.querySelectorAll('button[aria-label="aria_labels.projects_sidebar.close_projects_menu"]'),
  ].find((button) => !button.hasAttribute("aria-expanded"));
  assert.ok(reopenedButton, "opening should update the disclosure button label");
  assert.ok(document.activeElement === reopenedButton, "opening should keep keyboard focus on the disclosure button");
  assert.ok(environment.container.querySelector('[data-project-id="project-1"]'));
  assert.ok(
    environment.container.querySelector('[data-testid="project-loader"]') === null,
    "loading skeleton should be unmounted"
  );

  const transitionWarnings = warnings.filter((warning) => /Transition|Fragment|ref/i.test(warning));
  assert.deepEqual(transitionWarnings, []);
});
