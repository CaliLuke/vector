import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

globalThis.__profileTabLabelReact = React;
const react = "const React = globalThis.__profileTabLabelReact;";
const modules = {
  "next/navigation": 'export const usePathname = () => "/default/profile/user/created/";',
  "react-router": "export const Outlet = () => null;",
  swr: "export default () => ({data: undefined});",
  "@plane/i18n": 'export const useTranslation = () => ({t: key => key === "profile.tabs.created" ? "Créés" : key});',
  "@/components/core/app-header": "export const AppHeader = ({header}) => header;",
  "@/components/core/content-wrapper": "export const ContentWrapper = ({children}) => children;",
  "@/components/profile/sidebar": "export const ProfileSidebar = () => null;",
  "@/hooks/store/user": "export const useUserPermissions = () => ({allowPermissions: () => true});",
  "@/hooks/use-window-size": "export default () => [500, 800];",
  "@/services/user.service": "export class UserService {}",
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
  const { default: ProfileLayout } =
    await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx");
  await environment.render(
    React.createElement(ProfileLayout, { params: { workspaceSlug: "default", userId: "user" } })
  );
  assert.equal(environment.container.querySelector("button").textContent, "Créés");
});
