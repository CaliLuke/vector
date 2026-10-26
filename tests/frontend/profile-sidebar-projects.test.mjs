import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { createRouterFixture } from "./helpers/router.mjs";

const stubs = {
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/i18n": "export const useTranslation = () => ({t: key => key});",
  "@plane/propel/emoji-icon-picker": "export const Logo = () => null;",
  "@plane/propel/icon-button": "export const IconButton = () => null;",
  "@plane/propel/icons": "export const EditIcon = () => null; export const ChevronDownIcon = () => null;",
  "@plane/propel/tooltip": "export const Tooltip = ({children}) => children;",
  "@plane/ui": "export const Loader = Object.assign(() => null, {Item: () => null});",
  "@/components/common/cover-image": "export const CoverImage = () => null;",
  "@/providers/sidebar-state": `export const useProfileSidebarCollapsed = () => false;
    export const useSidebarActions = () => ({toggleProfileSidebar() {}});`,
  "@/hooks/use-command-palette": "export const useCommandPalette = () => ({toggleProfileSettingsModal() {}});",
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails = () => ({projects: [
    {id: 'first', name: 'First project', logo_props: {}},
    {id: 'second', name: 'Second project', logo_props: {}}
  ]});`,
  "@/hooks/use-current-user": "export const useCurrentUser = () => ({data: {id: 'reader'}});",
  "@/hooks/use-platform-os": "export const usePlatformOS = () => ({isMobile: false});",
  "./time": "export const ProfileSidebarTime = () => null;",
};
registerWebTestModules(stubs);

const statistics = (id) => ({ id, created_issues: 2, assigned_issues: 0, pending_issues: 0, completed_issues: 0 });

await test("profile project statistics use matching catalogue identities and omit unavailable projects", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ProfileSidebar } = await import("../../apps/web/core/components/profile/sidebar.tsx");
  const route = createRouterFixture("/workspace/profile/target/");
  await route.render(
    environment,
    React.createElement(ProfileSidebar, {
      userProjectsData: {
        user_data: { first_name: "Target", display_name: "target", date_joined: "2026-10-01T00:00:00Z" },
        project_data: [statistics("second"), statistics("unavailable"), statistics("first")],
      },
    })
  );

  const projectButtons = [...environment.container.querySelectorAll("button[aria-expanded]")];
  assert.deepEqual(
    projectButtons.map((button) => button.textContent),
    ["Second project", "First project"]
  );
});
