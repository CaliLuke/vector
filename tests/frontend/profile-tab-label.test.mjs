import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouterFixture } from "./helpers/router.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

globalThis.__profileTabLabelReact = React;
const react = "const React = globalThis.__profileTabLabelReact;";
const modules = {
  "@plane/i18n": 'export const useTranslation = () => ({t: key => key === "profile.tabs.created" ? "Créés" : key});',
  "@/components/core/app-header": "export const AppHeader = ({header}) => header;",
  "@/components/core/content-wrapper": "export const ContentWrapper = ({children}) => children;",
  "@/components/profile/sidebar": "export const ProfileSidebar = () => null;",
  "@/hooks/use-window-size": "export default () => [500, 800];",
  "@/services/user.service":
    "export class UserService { async currentUser() { throw new Error('unexpected identity request'); } async getUserProfileProjectsSegregation() { return []; } }",
  "@/services/workspace.service":
    "export class WorkspaceService { workspaceMemberMe() { throw new Error('unexpected membership request'); } getWorkspaceUserProjectsRole() { throw new Error('unexpected roles request'); } }",
  "./header": `${react} export const UserProfileHeader = ({type}) => React.createElement('button', null, type);`,
  "./mobile-header": "export const ProfileIssuesMobileHeader = () => null;",
  "./navbar": "export const ProfileNavbar = () => null;",
};
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      decodeURIComponent(context.parentURL ?? "").includes("/profile/[userId]/layout.tsx") &&
      Object.hasOwn(modules, specifier)
    ) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(modules[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

test("narrow profile navigation receives the translated active-tab label", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { Route } = await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx");
  const ProfileLayout = Route.options.component;
  const fixture = createRouterFixture("/default/profile/user/created/", {
    modules: { "(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx": Route },
  });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => queryClient.clear());
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  queryClient.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(queryClient, {
    userId: "alice",
    workspaceSlug: "default",
    workspaceRole: EUserWorkspaceRoles.ADMIN,
  });
  await fixture.render(
    environment,
    React.createElement(QueryClientProvider, { client: queryClient }, React.createElement(ProfileLayout))
  );

  assert.equal(environment.container.querySelector("button").textContent, "Créés");
});
