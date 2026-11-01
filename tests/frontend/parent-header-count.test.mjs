import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.parentHeaderReact = React;
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/ui": `export {Header} from ${JSON.stringify(new URL("../../packages/ui/src/header/header.tsx", import.meta.url).href)};
    export {CustomMenu} from ${JSON.stringify(new URL("../../packages/ui/src/dropdowns/custom-menu.tsx", import.meta.url).href)};
    const R=globalThis.parentHeaderReact;const Part=({children,component})=>component??children;
    export const Breadcrumbs=Object.assign(Part,{Item:Part,Icon:Part});
    export const BreadcrumbNavigationSearchDropdown=({title})=>R.createElement('span',null,title);`,
  "@/components/analytics/work-items/modal": "export const WorkItemsModal=()=>null;",
  "@/components/breadcrumbs/common": "export const ProjectSectionBreadcrumbs=({children})=>children;",
  "@/components/common/switcher-label": "export const SwitcherLabel=()=>null;",
  "@/components/cycles/quick-actions": "export const CycleQuickActions=()=>null;",
  "@/components/modules": "export const ModuleQuickActions=()=>null;",
  "@/components/work-item-filters/filters-toggle": "export const WorkItemFiltersToggle=()=>null;",
  "@/components/issues/issue-layouts/filters": `export const FiltersDropdown=()=>null;export const ParentIssueResponsiveLayoutSelection=()=>null;
    export {persistParentIssueLayout} from ${JSON.stringify(new URL("../../apps/web/core/components/issues/issue-layouts/filters/header/parent-issue-layout-selection.tsx", import.meta.url).href)};`,
  "@/components/issues/issue-layouts/filters/header/display-filters/parent-issue-display-selection":
    "export const ParentIssueDisplaySelection=()=>null;",
  "@/hooks/use-command-palette": "export const useCommandPalette=()=>({toggleCreateIssueModal:()=>{}});",
  "@/hooks/use-cycles":
    "export const useProjectCycles=()=>({cycles:[]});export const useCycleDetails=()=>({cycle:{id:'parent',name:'Cycle',status:'ongoing'}});",
  "@/hooks/use-modules":
    "export const useProjectModules=()=>({modules:[]});export const useModuleDetails=()=>({module:{id:'parent',name:'Module'}});",
  "@/providers/parent-issue-preferences":
    "export const useParentIssuePreferences=()=>({filters:{displayFilters:{layout:'list'}},setDisplay:async()=>{}});",
  "@/providers/cycle-module-preferences":
    "export const useParentChartPreferences=()=>({sidebarCollapsed:false,update:()=>{}});",
  "@/hooks/use-projects": "export const useRouteProject=()=>({project:{id:'p1'},isPending:false});",
  "@/hooks/use-permissions": "export const useProjectPermissions=()=>({can:()=>false});",
  "@/hooks/use-issue-list-count": "export const useIssueListCount=()=>({count:globalThis.parentHeaderCount});",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
});

async function verifyCount(parent, env, kind) {
  await parent.test(
    `${kind} header hides empty counts and keeps singular and plural native tooltip descriptions`,
    async () => {
      await env.render(null);
      globalThis.parentHeaderCount = 0;
      const Component =
        kind === "cycle"
          ? (
              await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx")
            ).CycleIssuesHeader
          : (
              await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx")
            ).ModuleIssuesHeader;
      const router = createRouterFixture(`/acme/projects/p1/${kind}s/parent/`);
      const render = () => router.render(env, React.createElement(Component));
      await render();
      const count = (value) =>
        Array.from(env.container.querySelectorAll("span")).find((node) => node.textContent === String(value));
      assert.equal(Boolean(count(0)), false);
      globalThis.parentHeaderCount = 1;
      await render();
      assert.ok(count(1));
      await act(async () => count(1).dispatchEvent(new FocusEvent("focusin", { bubbles: true })));
      await waitForReactCondition(
        () => document.body.textContent.includes(`There are 1 work item in this ${kind}`),
        "The native tooltip should describe one work item"
      );
      globalThis.parentHeaderCount = 2;
      await render();
      assert.ok(count(2));
      assert.match(document.body.textContent, new RegExp(`There are 2 work items in this ${kind}`));
      globalThis.parentHeaderCount = undefined;
      await render();
      assert.equal(Boolean(count(2)), false);
    }
  );
}
await test("parent headers retain their count descriptions", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await verifyCount(t, env, "cycle");
  await verifyCount(t, env, "module");
});
