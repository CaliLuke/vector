import { createRouterFixture } from "./helpers/router.mjs";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import test from "node:test";
import { pathToFileURL, fileURLToPath } from "node:url";
import path from "node:path";
import { registerHooks } from "node:module";
import assert from "node:assert/strict";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const webRoot = path.join(repositoryRoot, "apps/web");
const projectStateStub = `
  const state = globalThis.projectsListTestState ??= { loader: "init-loader", joinedProjectIds: ["project-1"] };
  export const useProject = () => ({
    loader: state.loader,
    joinedProjectIds: state.joinedProjectIds,
    getPartialProjectById: id => ({ id }),
    updateProjectView: async () => {},
  });
`;

const moduleStubs = {
  "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element": "export const autoScrollForElements = () => () => {};",
  "@plane/constants": `export const EUserPermissions = { ADMIN: "ADMIN", MEMBER: "MEMBER" };
    export const EUserPermissionsLevel = { WORKSPACE: "WORKSPACE" };
    export const PROJECT_TRACKER_ELEMENTS = { SIDEBAR_CREATE_PROJECT_TOOLTIP: "create-tooltip", SIDEBAR_CREATE_PROJECT_BUTTON: "create-button" };`,
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
  "@plane/utils": `export const copyUrlToClipboard = async () => {};
    export const cn = (...values) => values.flatMap(value => typeof value === "string" ? [value] : value && typeof value === "object" ? Object.entries(value).filter(([, enabled]) => enabled).map(([key]) => key) : []).join(" ");
    export const orderJoinedProjects = () => [];`,
  "mobx-react": "export const observer = component => component;",

  "@/components/project/create-project-modal": "export const CreateProjectModal = () => null;",
  "@/components/sidebar/sidebar-navigation": `const React = globalThis.projectsListReact;
    export const SidebarNavItem = ({ children }) => React.createElement("div", null, children);`,
  "@/hooks/store/use-app-theme": `export const useAppTheme = () => ({ isExtendedProjectSidebarOpened: false, toggleExtendedProjectSidebar() {} });`,
  "@/hooks/store/use-command-palette": `export const useCommandPalette = () => ({ toggleCreateProjectModal() {} });`,
  "@/hooks/store/use-project": projectStateStub,
  "@/hooks/store/user": `export const useUserPermissions = () => ({ allowPermissions: () => false });`,
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

function TestHarness() {
  const [, setRevision] = React.useState(0);
  React.useEffect(() => {
    const update = () => setRevision((revision) => revision + 1);
    window.addEventListener("projects-list-state-change", update);
    return () => window.removeEventListener("projects-list-state-change", update);
  }, []);
  return React.createElement(SidebarProjectsList);
}

let SidebarProjectsList;

void test("projects sidebar transitions through initial loading and closed/open disclosure without ref errors", async (t) => {
  globalThis.projectsListReact = React;
  globalThis.projectsListTestState = { loader: "init-loader", joinedProjectIds: ["project-1"] };
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/workspace/");
  const warnings = [];
  const originalError = console.error;
  const originalWarn = console.warn;
  const recordWarning = (...args) => {
    if (warnings.length < 50) warnings.push(args.join(" "));
  };
  console.error = recordWarning;
  console.warn = recordWarning;
  t.after(async () => {
    await environment.unmount();
    console.error = originalError;
    console.warn = originalWarn;
    delete globalThis.projectsListTestState;
    delete globalThis.projectsListReact;
  });

  ({ SidebarProjectsList } = await import("../../apps/web/core/components/workspace/sidebar/projects-list.tsx"));
  await routerFixture.render(environment, React.createElement(TestHarness));

  const loader = environment.container.querySelector('[data-testid="project-loader"]');
  const projectRow = environment.container.querySelector('[data-project-id="project-1"]');
  assert.ok(loader, "the initial request should render project-list skeleton rows");
  assert.ok(projectRow, "the expanded disclosure should preserve the loaded project row");

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

  globalThis.projectsListTestState.loader = "loaded";
  await React.act(() => window.dispatchEvent(new Event("projects-list-state-change")));
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
