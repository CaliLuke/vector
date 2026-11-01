import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.projectTabsReact = React;
const tabs = `const R=globalThis.projectTabsReact;export const TabNavigationRoot=({workspaceSlug,projectId})=>R.createElement('nav',{'aria-label':'Project tabs','data-workspace':workspaceSlug,'data-project':projectId});`;
registerWebTestModules({
  "@plane/ui": `export {Header} from ${JSON.stringify(new URL("../../packages/ui/src/header/header.tsx", import.meta.url).href)};
    export {Row} from ${JSON.stringify(new URL("../../packages/ui/src/row/row.tsx", import.meta.url).href)};`,
  "@/components/navigation": tabs,
  "@/components/navigation/tab-navigation-root": tabs,
  "@/components/sidebar/sidebar-toggle-button": `export const AppSidebarToggleButton=()=>globalThis.projectTabsReact.createElement('button',{type:'button'},'Expand sidebar');`,
  "@/providers/sidebar-state": "export const useSidebarCollapsed=()=>globalThis.projectTabsFixture.collapsed;",
  "@/hooks/use-navigation-preferences":
    "export const useProjectNavigationPreferences=()=>({preferences:{navigationMode:globalThis.projectTabsFixture.mode}});",
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails=()=>({projects:[{id:'locator-project',identifier:'PRJ'}]});",
  "@/hooks/use-work-item-detail":
    "export const useWorkItemByIdentifier=(...args)=>{globalThis.projectTabsFixture.lookup=args;return {issue:globalThis.projectTabsFixture.issue};};",
  "@/components/core/app-header": "export const AppHeader=({header})=>header;",
  "./work-item-header":
    "export const WorkItemDetailsHeader=()=>globalThis.projectTabsReact.createElement('h1',null,'Item details');",
  "@/layouts/auth-layout/project-wrapper": `const R=globalThis.projectTabsReact;export const ProjectAuthWrapper=({workspaceSlug,projectId,children})=>R.createElement('section',{'data-auth-workspace':workspaceSlug,'data-auth-project':projectId},children);`,
});

async function verifyConsumer(kind) {
  await test(`${kind} tab header follows navigation mode, sidebar visibility and its own project scope`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const state = (globalThis.projectTabsFixture = {
      mode: "TABBED",
      collapsed: true,
      issue: { project_id: "item-project" },
    });
    let Component;
    if (kind === "project")
      Component = (
        await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx")
      ).Route.options.component;
    else
      Component = (await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/header.tsx"))
        .ProjectWorkItemDetailsHeader;
    const router = createRouterFixture(kind === "project" ? "/acme/projects/p1/issues/" : "/acme/browse/PRJ-42");
    const render = () => router.render(env, React.createElement(Component));
    await render();
    const nav = () => env.container.querySelector('nav[aria-label="Project tabs"]');
    assert.equal(nav().dataset.workspace, "acme");
    assert.equal(nav().dataset.project, kind === "project" ? "p1" : "item-project");
    assert.ok(env.container.querySelector("button"));
    if (kind === "project") assert.equal(env.container.querySelector("[data-auth-project]").dataset.authProject, "p1");
    else assert.deepEqual(state.lookup, ["acme", "PRJ", "42", "locator-project"]);
    state.collapsed = false;
    await render();
    assert.equal(env.container.querySelector("button"), null);
    state.mode = "SIDEBAR";
    await render();
    assert.equal(nav(), null);
    if (kind === "project") assert.ok(env.container.querySelector("[data-auth-project]"));
    else assert.equal(env.container.querySelector("h1").textContent, "Item details");
    state.mode = "TABBED";
    state.issue = undefined;
    await render();
    assert.equal(nav().dataset.project, kind === "project" ? "p1" : "");
  });
}
await verifyConsumer("project");
await verifyConsumer("browse");
