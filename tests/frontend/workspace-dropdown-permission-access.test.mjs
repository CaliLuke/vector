import assert from "node:assert/strict";
import test from "node:test";
import { React, createPermissionUiFixture, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";

globalThis.workspaceDropdownPermissionTest = { React };
const reactBinding = "const React=globalThis.workspaceDropdownPermissionTest.React;";
const stubs = {
  "@headlessui/react": `${reactBinding} export const Menu={Item:({children})=>React.createElement(React.Fragment,null,children)};`,
  "@plane/i18n": `export const useTranslation=()=>({t:(key)=>key==='settings'?'Settings':key==='project_settings.members.invite_members.title'?'Members':key});`,
  "@plane/propel/icons": `${reactBinding} export const CheckIcon=()=>React.createElement('span',null,'Selected');`,
  "@plane/utils/common": "export const cn=(...values)=>values.filter((value)=>typeof value==='string').join(' ');",
  "@plane/utils/file": "export const getFileURL=(url)=>url;",
  "@plane/utils/permission": "export const getUserRole=(role)=>String(role);",
};
registerPermissionUiStubs("/workspace/sidebar/dropdown-item.tsx", stubs);

const SidebarDropdownItem = (await import("../../apps/web/core/components/workspace/sidebar/dropdown-item.tsx"))
  .default;
const { EUserPermissions, EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");

const activeWorkspace = {
  id: "workspace-acme",
  name: "Acme",
  slug: "acme",
  role: EUserPermissions.ADMIN,
  logo_url: null,
  total_members: 4,
};

async function render(t, nativeRole, catalogueRole = EUserPermissions.ADMIN) {
  const workspace = { ...activeWorkspace, role: catalogueRole };
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/settings/projects/p1/members/",
    workspaceRole: nativeRole,
  });
  await fixture.render(
    React.createElement(SidebarDropdownItem, {
      workspace,
      activeWorkspace: workspace,
      handleItemClick() {},
      handleWorkspaceNavigation() {},
      handleClose() {},
    })
  );
  return fixture.environment.container;
}

await test("stale catalogue admin role does not expose Members link to a workspace member", async (t) => {
  const container = await render(t, EUserWorkspaceRoles.MEMBER);
  const settingsLinks = [...container.querySelectorAll("a")].filter((link) =>
    link.getAttribute("href")?.includes("/settings")
  );

  assert.equal(settingsLinks.length, 1);
  assert.match(settingsLinks[0].textContent, /Settings/);
  assert.doesNotMatch(settingsLinks[0].textContent, /Members/);
});

await test("native workspace admin role exposes both settings links despite stale catalogue member role", async (t) => {
  const container = await render(t, EUserWorkspaceRoles.ADMIN, EUserPermissions.MEMBER);
  const settingsLinks = [...container.querySelectorAll("a")].filter((link) =>
    link.getAttribute("href")?.includes("/settings")
  );

  assert.equal(settingsLinks.length, 2);
});
