import assert from "node:assert/strict";
import test from "node:test";
import { React, createPermissionUiFixture, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";

globalThis.modulesGanttPermissionTest = { React };
const reactBinding = "const React=globalThis.modulesGanttPermissionTest.React;";
const stubs = {
  "@/components/gantt-chart": `${reactBinding}
    export const GanttChartRoot = (props) => React.createElement('div', {
      'data-can-resize': String(props.enableBlockLeftResize),
      'data-can-move': String(props.enableBlockMove),
      'data-can-reorder': String(props.enableReorder),
      'data-can-add': String(props.enableAddBlock),
    });
    export const ModuleGanttSidebar = () => null;`,
  "@/components/gantt-chart/contexts": `${reactBinding} export const TimeLineTypeContext=React.createContext(null);`,
  "@/components/modules": "export const ModuleGanttBlock=()=>null;",
  "@/hooks/use-favorites": "export const useFavorites=()=>({favorites:[]});",
};
registerPermissionUiStubs("/modules/gantt-chart/modules-list-layout.tsx", stubs);

const { ModulesListGanttChartView } =
  await import("../../apps/web/core/components/modules/gantt-chart/modules-list-layout.tsx");
const { CycleModulePreferencesProvider } = await import("../../apps/web/core/providers/cycle-module-preferences.tsx");
const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
const { EUserPermissions, EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");

async function render(t, { workspaceRole, projectRole }) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/modules/",
    workspaceRole,
    projectRoles: { p1: projectRole },
  });
  t.mock.method(ModuleService.prototype, "getModules", async () => [
    { id: "module-1", project_id: "p1", workspace_id: "w1", name: "Native module", sort_order: 1 },
  ]);
  const preferenceKey = JSON.stringify(["alice", "acme", "p1"]);
  window.localStorage.setItem(
    `vector:module-catalogue-preferences:${preferenceKey}`,
    JSON.stringify({ display: { layout: "gantt", order_by: "sort_order" } })
  );
  await fixture.render(
    React.createElement(CycleModulePreferencesProvider, null, React.createElement(ModulesListGanttChartView))
  );
  return fixture.environment.container.querySelector("div[data-can-add]");
}

await test("native module Gantt enables permitted project member edits", async (t) => {
  const chart = await render(t, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRole: EUserPermissions.MEMBER,
  });

  assert.equal(chart?.getAttribute("data-can-resize"), "true");
  assert.equal(chart?.getAttribute("data-can-move"), "true");
  assert.equal(chart?.getAttribute("data-can-reorder"), "true");
  assert.equal(chart?.getAttribute("data-can-add"), "true");
});

await test("native module Gantt forbids project guest edits", async (t) => {
  const chart = await render(t, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRole: EUserPermissions.GUEST,
  });

  assert.equal(chart?.getAttribute("data-can-resize"), "false");
  assert.equal(chart?.getAttribute("data-can-move"), "false");
  assert.equal(chart?.getAttribute("data-can-reorder"), "false");
  assert.equal(chart?.getAttribute("data-can-add"), "false");
});
