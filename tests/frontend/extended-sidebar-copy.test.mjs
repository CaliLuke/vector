import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const fixture = {
  toasts: [],
  copy: async () => {},
  renderItem: ({ handleCopyText }) => React.createElement("button", { onClick: handleCopyText }, "Copy project link"),
};
globalThis.extendedSidebarCopyFixture = fixture;
const stubs = {
  "@plane/i18n": "export const useTranslation = () => ({t: key => key});",
  "@plane/propel/empty-state": "export const EmptyStateCompact = () => null;",
  "@plane/propel/icons": "export const PlusIcon = () => null; export const SearchIcon = () => null;",
  "@plane/propel/toast": `export const TOAST_TYPE = {SUCCESS: "success", ERROR: "error"};
    export const setToast = toast => globalThis.extendedSidebarCopyFixture.toasts.push(toast);`,
  "@plane/propel/tooltip": "export const Tooltip = ({children}) => children;",
  "@plane/utils/string": "export const copyUrlToClipboard = path => globalThis.extendedSidebarCopyFixture.copy(path);",
  "@/components/project/create-project-modal": "export const CreateProjectModal = () => null;",
  "@/components/workspace/sidebar/projects-list-item":
    "export const SidebarProjectsListItem = props => globalThis.extendedSidebarCopyFixture.renderItem(props);",
  "@/providers/sidebar-state": `export const useExtendedProjectSidebarOpen = () => true;
    export const useSidebarActions = () => ({toggleExtendedProjectSidebar() {}});`,
  "@/hooks/use-projects":
    "export const useWorkspaceProjects = () => ({projects: [{id: 'project', name: 'Project', identifier: 'P'}]});",
  "@/hooks/use-joined-project-ids": "export const useJoinedProjectIds = () => ['project'];",
  "@/hooks/use-workspace-project-reordering": "export const useWorkspaceProjectReordering = () => () => {};",
  "@/hooks/use-permissions": "export const useWorkspacePermissions = () => ({can: () => false});",
  "./extended-sidebar-wrapper": "export const ExtendedSidebarWrapper = ({children}) => children;",
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { ExtendedProjectSidebar } =
  await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx");

await test("extended sidebar reports clipboard success only after acceptance and handles denial", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount();
    delete globalThis.extendedSidebarCopyFixture;
  });
  let finish;
  const paths = [];
  fixture.copy = (path) => {
    paths.push(path);
    return new Promise((resolve) => {
      finish = resolve;
    });
  };
  await createRouterFixture("/acme/projects/").render(environment, React.createElement(ExtendedProjectSidebar));
  await act(async () => environment.container.querySelector("button").click());
  assert.deepEqual(paths, ["acme/projects/project/issues"]);
  assert.deepEqual(fixture.toasts, []);
  await act(async () => finish());
  assert.deepEqual(
    fixture.toasts.map((toast) => toast.type),
    ["success"]
  );
  fixture.copy = async () => {
    throw new Error("Clipboard denied");
  };
  await act(async () => environment.container.querySelector("button").click());
  assert.deepEqual(
    fixture.toasts.map((toast) => toast.type),
    ["success", "error"]
  );
});
