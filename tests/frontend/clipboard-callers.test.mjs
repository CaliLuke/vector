import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
const state = { React, toasts: [], copy: async () => {} };
globalThis.clipboardCallerFixture = state;
const react = "const React=globalThis.clipboardCallerFixture.React;";
const empty = "()=>null";
const menu = `${react} export const CustomMenu=({children})=>children;CustomMenu.MenuItem=({children,onClick,disabled})=>React.createElement('button',{onClick,disabled},children);export const ContextMenu=()=>null;export const LinkItemBlock=({menuItems})=>React.createElement('button',{onClick:menuItems.find(x=>x.key==='copy-link').action},'Copy');`;
const nullModals =
  "export const ArchiveModuleModal=()=>null,CreateUpdateModuleModal=()=>null,DeleteModuleModal=()=>null;";
const stubs = {
  "@/components/views/publish": "export const useViewPublish=()=>({});",
  "@/components/workspace/confirm-workspace-member-remove": "export const ConfirmWorkspaceMemberRemove=()=>null;",
  "@/hooks/use-member-roster-mutations":
    "export const useDeleteWorkspaceInvitation=()=>({});export const useUpdateWorkspaceInvitation=()=>({});",
  "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element": "export const autoScrollForElements=()=>()=>{};",
  "@headlessui/react":
    "export const Disclosure=({children})=>children;Disclosure.Button=()=>null;Disclosure.Panel=({children})=>children;export const Transition=({children})=>children;",
  "@/components/project/create-project-modal": "export const CreateProjectModal=()=>null;",
  "@/components/sidebar/sidebar-navigation": "export const SidebarNavItem=({children})=>children;",
  "@/providers/sidebar-state":
    "export const useSidebarActions=()=>({});export const useExtendedProjectSidebarOpen=()=>false;",
  "@/hooks/use-command-palette": "export const useCommandPalette=()=>({});",
  "@/hooks/use-projects": 'export const useWorkspaceProjects=()=>({projects:[{id:"project"}]});',
  "@/hooks/use-workspace-project-reordering": "export const useWorkspaceProjectReordering=()=>()=>{};",
  "@/hooks/use-navigation-preferences":
    "export const useProjectNavigationPreferences=()=>({preferences:{showLimitedProjects:false,limitedProjectsCount:10}});",
  "@/hooks/use-joined-project-ids": 'export const useJoinedProjectIds=()=>["project"];',
  "./projects-list.utils": "export const findSidebarScrollViewport=()=>null;",
  "./projects-list-item":
    react +
    'export const SidebarProjectsListItem=({handleCopyText})=>React.createElement("button",{onClick:handleCopyText},"Copy");',

  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/icons": `const Icon=${empty};export const CopyIcon=Icon,LinkIcon=Icon,NewTabIcon=Icon,EditIcon=Icon,TrashIcon=Icon,ChevronDownIcon=Icon,PlusIcon=Icon,ChevronRightIcon=Icon;`,
  "@plane/propel/button":
    react + `export const Button=({children,onClick})=>React.createElement('button',{onClick},children);`,
  "@plane/propel/icon-button": "export const IconButton=()=>null;",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@plane/propel/toast":
    'export const TOAST_TYPE={SUCCESS:"success",INFO:"info",ERROR:"error"};export const setToast=t=>globalThis.clipboardCallerFixture.toasts.push(t);',
  "@plane/ui":
    menu +
    "export const CustomSelect=()=>null;CustomSelect.Option=()=>null;export const Loader=()=>null;Loader.Item=()=>null;",
  "lucide-react": "export const MoreHorizontal=()=>null,Ellipsis=()=>null;",
  "@plane/utils/string":
    "export const copyTextToClipboard=value=>globalThis.clipboardCallerFixture.copy(value);export const copyUrlToClipboard=value=>globalThis.clipboardCallerFixture.copy(value);",
  "@plane/utils/datetime":
    'export const renderFormattedDate=()=>"date",renderFormattedTime=()=>"time",calculateTimeAgo=()=>"now";',
  "@plane/utils/get-icon-for-link": "export const getIconForLink=()=>()=>null;",
  "@tanstack/react-router":
    'export const useRouter=()=>({navigate(){}});export const useParams=()=>({workspaceSlug:"acme"});export const useLocation=({select})=>select({searchStr:"",pathname:"/acme/projects"});',
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/hooks/use-permissions":
    "export const useProjectPermissions=()=>({can:()=>true});export const useWorkspacePermissions=()=>({workspaceRole:20,can:()=>true});",
  "@/hooks/use-current-user": 'export const useCurrentUser=()=>({data:{id:"actor"}});',
  "@/hooks/use-member-rosters": "export const useWorkspaceMemberRoster=()=>({memberById:()=>undefined});",
  "@/hooks/use-cycles": 'export const useProjectCycles=()=>({mutation:{},scope:"project"});',
  "@/hooks/use-modules": 'export const useProjectModules=()=>({mutation:{},scope:"project"});',
  "@/components/common/quick-actions-helper": `const menu=({handleCopyLink})=>({items:[{key:'copy',title:'Copy',action:handleCopyLink}]});export const useViewMenuItems=menu,useCycleMenuItems=menu,useModuleMenuItems=menu;`,
  "@/components/common/quick-action-menu":
    react +
    `export const renderQuickActionMenuItems=items=>items.map(item=>React.createElement('button',{key:item.key,onClick:item.action},item.title));`,
  "@/components/modules": nullModals,
  "./archived-cycles/modal": "export const ArchiveCycleModal=()=>null;",
  "./delete-modal": "export const CycleDeleteModal=()=>null;",
  "./modal":
    "export const CycleCreateUpdateModal=()=>null;export const CreateUpdateWorkspaceViewModal=()=>null;export const CreateUpdateProjectViewModal=()=>null;",
  "./delete-view-modal": "export const DeleteGlobalViewModal=()=>null;export const DeleteProjectViewModal=()=>null;",
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { DefaultWorkspaceViewQuickActions } =
  await import("../../apps/web/core/components/workspace/views/default-view-quick-action.tsx");
const { WorkspaceViewQuickActions } = await import("../../apps/web/core/components/workspace/views/quick-action.tsx");
const { GeneratedTokenDetails } =
  await import("../../apps/web/core/components/api-token/modal/generated-token-details.tsx");
const { ProjectLinkDetail } = await import("../../apps/web/core/components/home/widgets/links/link-detail.tsx");
const { ModulesLinksListItem } = await import("../../apps/web/core/components/modules/links/list-item.tsx");
const { CycleQuickActions } = await import("../../apps/web/core/components/cycles/quick-actions.tsx");
const { ModuleQuickActions } = await import("../../apps/web/core/components/modules/quick-actions.tsx");
const { ViewQuickActions } = await import("../../apps/web/core/components/views/quick-actions.tsx");
const { WorkspaceInvitationsListItem } =
  await import("../../apps/web/core/components/workspace/settings/invitations-list-item.tsx");
const { SidebarProjectsList } = await import("../../apps/web/core/components/workspace/sidebar/projects-list.tsx");
const scope = { workspaceSlug: "acme", projectId: "project", parentRef: { current: null } };
const cases = [
  {
    name: "project view",
    Component: ViewQuickActions,
    props: { ...scope, view: { id: "view", owned_by: "actor" } },
    value: "acme/projects/project/views/view",
    label: "Copy",
  },
  {
    name: "invitation",
    Component: WorkspaceInvitationsListItem,
    props: { invitation: { id: "invite", email: "test@example.com", role: 15, invite_link: "/invite/test" } },
    value: "http://localhost/invite/test",
    label: "common.actions.copy_link",
  },
  {
    name: "project sidebar",
    Component: SidebarProjectsList,
    props: {},
    value: "acme/projects/project/issues",
    label: "Copy",
  },
  {
    name: "default workspace view",
    Component: DefaultWorkspaceViewQuickActions,
    props: { workspaceSlug: "acme", view: { key: "assigned", i18n_label: "Assigned" } },
    value: "acme/workspace-views/assigned",
    label: "copy_link",
  },
  {
    name: "workspace view",
    Component: WorkspaceViewQuickActions,
    props: { workspaceSlug: "acme", view: { id: "view", owned_by: "actor" } },
    value: "acme/workspace-views/view",
    label: "Copy",
  },
  {
    name: "API token",
    Component: GeneratedTokenDetails,
    props: { tokenDetails: { token: "generated-test-token" }, handleClose() {} },
    value: "generated-test-token",
    label: "generated-test-token",
  },
  {
    name: "home link",
    Component: ProjectLinkDetail,
    props: { linkDetail: { id: "link", url: "https://example.com/" }, onEdit() {}, linkOperations: {} },
    value: "https://example.com/",
    label: "Copy",
  },
  {
    name: "module link",
    Component: ModulesLinksListItem,
    props: {
      workspaceSlug: "acme",
      link: { id: "link", url: "https://example.com/", title: "Link" },
      isEditingAllowed: false,
      handleEditLink() {},
      handleDeleteLink() {},
    },
    value: "https://example.com/",
    label: null,
  },
  {
    name: "cycle menu",
    Component: CycleQuickActions,
    props: { ...scope, cycleId: "cycle", cycle: { id: "cycle" } },
    value: "acme/projects/project/cycles/cycle",
    label: "Copy",
  },
  {
    name: "module menu",
    Component: ModuleQuickActions,
    props: { ...scope, moduleId: "module", module: { id: "module" } },
    value: "acme/projects/project/modules/module",
    label: "Copy",
  },
];
for (const { name, Component, props, value, label } of cases)
  void test(`${name} waits for acceptance and handles copy denial followed by retry`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    state.toasts.length = 0;
    const writes = [];
    let pending = Promise.withResolvers();
    state.copy = (text) => {
      writes.push(text);
      return pending.promise;
    };
    await env.render(React.createElement(Component, props));
    const button = label
      ? [...env.container.querySelectorAll("button")].find((b) => b.textContent.includes(label))
      : env.container.querySelector("button");
    assert.ok(button, name + " copy control");
    await act(async () => button.click());
    assert.deepEqual(writes, [value]);
    assert.deepEqual(state.toasts, []);
    await act(async () => pending.reject(new Error("clipboard denied")));
    assert.equal(state.toasts.length, 1);
    assert.equal(state.toasts[0].type, "error");
    pending = Promise.withResolvers();
    await act(async () => button.click());
    assert.equal(state.toasts.length, 1);
    await act(async () => pending.resolve());
    assert.equal(state.toasts.length, 2);
    assert.notEqual(state.toasts[1].type, "error");
    assert.deepEqual(writes, [value, value]);
  });
for (const { name, Component, props, label } of cases.filter((c) =>
  ["cycle menu", "module menu", "default workspace view", "workspace view", "project view"].includes(c.name)
))
  for (const outcome of ["resolve", "reject"])
    void test(`${name} ignores ${outcome} after scope changes`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      state.toasts.length = 0;
      const pending = Promise.withResolvers();
      state.copy = () => pending.promise;
      await env.render(React.createElement(Component, props));
      await act(async () =>
        [...env.container.querySelectorAll("button")].find((b) => b.textContent.includes(label)).click()
      );
      await env.render(
        React.createElement(Component, { ...props, cycleId: "other", moduleId: "other", workspaceSlug: "other" })
      );
      await act(async () => pending[outcome](new Error("denied")));
      assert.deepEqual(state.toasts, []);
    });
