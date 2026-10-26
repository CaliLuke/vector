import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { registerHooks } from "node:module";

registerWebModuleHooks();
globalThis.archiveRouteTest = { React };
const react = "const React=globalThis.archiveRouteTest.React;";
const stubs = {
  "lucide-react": "export const Info=()=>null;export const AlertCircle=()=>null;",
  "@plane/propel/icons":
    "export const CheckIcon=()=>null;export const SearchIcon=()=>null;export const CycleIcon=()=>null;export const TransferIcon=()=>null;export const CloseIcon=()=>null;",
  "@plane/propel/toast": "export const TOAST_TYPE={SUCCESS:'success'};export const setToast=()=>{};",
  "@plane/ui": `${react} export const CircularProgressIndicator=({children})=>React.createElement('span',null,children);export const EModalPosition={TOP:'top'};export const EModalWidth={XXL:'xxl'};export const ModalForm=()=>null;export const FormErrors=()=>null;`,
  "@plane/utils/router": "export const generateQueryParams=()=>'';",
  "@/components/core/list": `${react} export const ListItem=(props)=>React.createElement('a',{href:props.itemLink,'data-testid':'module-list-item'},props.title,props.actionableItems,props.quickActionElement);`,
  "@/components/modules":
    "export const ModuleListItemAction=()=>null;export const ModuleQuickActions=()=>null;export const ArchivedModuleLayoutRoot=()=>null;export const ArchivedModulesHeader=()=>null;",
  "@/components/core/page-title": "export const PageHead=()=>null;",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/hooks/use-projects":
    "export const useRouteProject=()=>({});export const useWorkspaceProjectDetails=()=>({});export const useWorkspaceProjects=()=>({});",
  "@/hooks/use-cycles": "export const useProjectCycles=()=>({cycles:[]});",
  "@/queries/cycle-transfers": "export const cycleTransferOptions=()=>({});",
  "@/queries/cycle-module-selectors": "export const cycleIsCompleted=()=>false;",
  "@/providers/parent-issue-preferences": "export const useOptionalParentIssuePreferences=()=>undefined;",
  "@/hooks/use-committed-scope": "export const useCommittedScope=()=>()=>true;",
  "@/components/common/catalogue-read-failure": "export const CatalogueReadFailure=()=>null;",
  "@/hooks/store/use-issues": "export const useIssues=()=>({issues:{fetchIssuesWithExistingPagination:async()=>{}}});",
};
registerHooks({
  resolve(specifier, context, next) {
    if (
      context.parentURL?.endsWith("module-list-item.tsx") ||
      context.parentURL?.endsWith("transfer-issues-modal.tsx") ||
      context.parentURL?.endsWith("archives/modules/page.tsx")
    )
      return resolvePackageStub(specifier, stubs, context, next);
    return next(specifier, context);
  },
});

const routeFile = "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/modules/page.tsx";
const { Route } = await import(`../../apps/web/app/${routeFile}`);
const { ModuleListItem } = await import("../../apps/web/core/components/modules/module-list-item.tsx");
const { TransferIssuesModal } = await import("../../apps/web/core/components/cycles/transfer-issues-modal.tsx");
const module = {
  id: "module-1",
  project_id: "p1",
  workspace_id: "w1",
  name: "Archived module",
  total_issues: 0,
  completed_issues: 0,
  cancelled_issues: 0,
  archived_at: "2026-01-01T00:00:00Z",
};
function ArchiveRouteFixture() {
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(ModuleListItem, { module }),
    React.createElement(TransferIssuesModal, {
      isOpen: false,
      handleClose() {},
      cycleId: "cycle-1",
      workspaceSlug: "acme",
      projectId: "p1",
    })
  );
}

await test("archived module and cycle list actions use route context available on archive routes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const router = createRouterFixture("/acme/projects/p1/archives/modules/", {
    modules: { [routeFile]: { update: (options) => Route.update({ ...options, component: ArchiveRouteFixture }) } },
  });
  t.after(async () => {
    await environment.render(null);
    await environment.unmount();
  });
  await act(async () => router.router.load());
  await environment.render(React.createElement(router.Provider));
  assert.equal(
    environment.container.querySelector('[data-testid="module-list-item"]')?.getAttribute("href"),
    "/acme/projects/p1/modules/module-1"
  );
});
