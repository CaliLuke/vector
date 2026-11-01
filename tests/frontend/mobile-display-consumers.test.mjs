import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.mobileDisplayReact = React;
const parentSelection = `const R=globalThis.mobileDisplayReact;export const ParentIssueDisplaySelection=({project})=>R.createElement('button',{type:'button',onClick:()=>globalThis.mobileDisplayState.changes.push({project:project.id})},'Apply grouping');`;
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/ui": `export {CustomMenu} from ${JSON.stringify(new URL("../../packages/ui/src/dropdowns/custom-menu.tsx", import.meta.url).href)};`,
  "@plane/ui/dropdowns": `export {FiltersDropdown} from ${JSON.stringify(new URL("../../packages/ui/src/dropdowns/filters.tsx", import.meta.url).href)};`,
  "@/components/issues/issue-layouts/filters": `
    export {FiltersDropdown} from ${JSON.stringify(new URL("../../apps/web/core/components/issues/issue-layouts/filters/header/helpers/dropdown.tsx", import.meta.url).href)};
    export {MobileLayoutSelection} from ${JSON.stringify(new URL("../../apps/web/core/components/issues/issue-layouts/filters/header/mobile-layout-selection.tsx", import.meta.url).href)};
    const R=globalThis.mobileDisplayReact;
    export const DisplayFiltersSelection=({handleDisplayFiltersUpdate,cycleViewDisabled,moduleViewDisabled})=>R.createElement('button',{type:'button',onClick:()=>handleDisplayFiltersUpdate({group_by:'priority'}),'data-cycle-disabled':String(cycleViewDisabled),'data-module-disabled':String(moduleViewDisabled)},'Apply grouping');`,
  "./display-filters/parent-issue-display-selection": parentSelection,
  "@/hooks/use-route-project": "export const useRouteProject=()=>({project:globalThis.mobileDisplayState.project});",
  "@/hooks/use-projects": "export const useRouteProject=()=>({project:globalThis.mobileDisplayState.project});",
  "@/hooks/use-project-issue-display-preferences": `export const useProjectIssueDisplayPreferences=()=>({issueFilters:{displayFilters:{layout:'list'},displayProperties:{}},activeLayout:'list',handleLayoutChange:layout=>globalThis.mobileDisplayState.layouts.push(layout),handleDisplayFilters:patch=>globalThis.mobileDisplayState.changes.push(patch),handleDisplayProperties:()=>{}});`,
  "@/components/analytics/work-items/modal": `const R=globalThis.mobileDisplayReact;export const WorkItemsModal=({isOpen,onClose,workspaceSlug,projectDetails})=>isOpen?R.createElement('section',{'aria-label':'Analytics','data-workspace':workspaceSlug,'data-project':projectDetails.id},R.createElement('button',{type:'button',onClick:onClose},'Close analytics')):null;`,
});

const findButton = (text) => Array.from(document.querySelectorAll("button")).find((node) => node.textContent === text);
async function verifyConsumer(parent, env, kind) {
  await parent.test(`${kind} mobile display popup preserves edits, Escape focus and scoped analytics`, async () => {
    await env.render(null);
    const state = (globalThis.mobileDisplayState = {
      project: { id: "p1", cycle_view: true, module_view: false },
      changes: [],
      layouts: [],
    });
    let render;
    if (kind === "project") {
      const { ProjectIssuesMobileHeader } =
        await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(list)/mobile-header.tsx");
      const router = createRouterFixture("/acme/projects/p1/issues/");
      render = () => router.render(env, React.createElement(ProjectIssuesMobileHeader));
    } else {
      const { ParentIssueMobileHeader } =
        await import("../../apps/web/core/components/issues/issue-layouts/filters/header/parent-issue-mobile-header.tsx");
      render = () =>
        env.render(
          React.createElement(ParentIssueMobileHeader, {
            workspaceSlug: "acme",
            projectDetails: state.project,
            canUpdateLayout: true,
            setDisplay: async (patch) => state.layouts.push(patch),
          })
        );
    }
    await render();
    const trigger = findButton("common.display");
    assert.ok(trigger);
    assert.equal(trigger.type, "button");
    trigger.focus();
    await act(async () => trigger.click());
    await waitForReactCondition(() => Boolean(findButton("Apply grouping")), "The native display popup should open");
    const apply = findButton("Apply grouping");
    if (kind === "project") {
      assert.equal(apply.dataset.cycleDisabled, "false");
      assert.equal(apply.dataset.moduleDisabled, "true");
    }
    await act(async () => apply.click());
    assert.deepEqual(state.changes, kind === "project" ? [{ group_by: "priority" }] : [{ project: "p1" }]);
    apply.focus();
    await act(async () =>
      apply.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
    );
    await waitForReactCondition(() => !findButton("Apply grouping"), "Escape should close display options");
    assert.equal(document.activeElement, trigger);
    await act(async () => findButton("common.analytics").click());
    const analytics = document.querySelector('[aria-label="Analytics"]');
    assert.equal(analytics.dataset.workspace, "acme");
    assert.equal(analytics.dataset.project, "p1");
    await act(async () => findButton("Close analytics").click());
    assert.equal(Boolean(document.querySelector('[aria-label="Analytics"]')), false);
  });
}
await test("mobile display controls retain the native popup lifecycle", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await verifyConsumer(t, env, "project");
  await verifyConsumer(t, env, "parent");
});
