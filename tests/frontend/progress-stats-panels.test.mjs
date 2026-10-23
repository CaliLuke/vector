import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const packageStubs = {
  "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
  "@plane/propel/icons": `export const StateGroupIcon = () => null;`,
  "@plane/ui": `export const Avatar = () => null;`,
  "@plane/utils": `export const cn = (...values) => values.filter(Boolean).join(" ");
    export const toFilterArray = value => Array.isArray(value) ? value : [];
    export const getFileURL = value => value;`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.endsWith("?url")) return createModuleStub('export default "asset";');
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

await test("cycle and module progress tabs display matching projections and metric counts", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { observable, runInAction } = webRequire("mobx");
  const [{ CycleProgressStats }, { ModuleProgressStats }] = await Promise.all([
    import("../../apps/web/core/components/cycles/analytics-sidebar/progress-stats.tsx"),
    import("../../apps/web/core/components/modules/analytics-sidebar/progress-stats.tsx"),
  ]);
  const filters = { assignees: { value: [] }, labels: { value: [] }, stateGroups: { value: [] } };
  const commonProps = {
    groupedIssues: { Started: 1 },
    handleFiltersUpdate: () => {},
    selectedFilters: filters,
    totalIssuesCount: 4,
  };
  const activePanelText = () => {
    const activeTab = environment.container.querySelector('[role="tab"][aria-selected="true"]');
    const activePanelId = activeTab?.getAttribute("aria-controls");
    return (
      [...environment.container.querySelectorAll('[role="tabpanel"]')].find((panel) => panel.id === activePanelId)
        ?.textContent ?? ""
    );
  };

  const cycleDistribution = observable({
    assignees: [
      {
        assignee_id: "cycle-user",
        display_name: "Cycle Assignee",
        avatar_url: null,
        completed_estimates: 3,
        total_estimates: 5,
      },
    ],
    labels: [
      {
        label_id: "cycle-label",
        label_name: "Cycle Label",
        color: "#f00",
        completed_estimates: 2,
        total_estimates: 5,
      },
    ],
  });
  await environment.render(
    React.createElement(CycleProgressStats, {
      ...commonProps,
      cycleId: "cycle-1",
      distribution: cycleDistribution,
      plotType: "burnup",
    })
  );
  assert.match(activePanelText(), /Cycle Assignee/, "selected assignee tab should show assignee data");
  assert.match(activePanelText(), /60%/);
  assert.doesNotMatch(activePanelText(), /Cycle Label/);

  await act(async () => {
    runInAction(() => {
      cycleDistribution.assignees[0].display_name = "Updated Cycle Assignee";
    });
  });
  assert.match(activePanelText(), /Updated Cycle Assignee/);

  await act(async () => environment.container.querySelectorAll('[role="tab"]')[0].click());
  assert.match(activePanelText(), /Started/);
  assert.match(activePanelText(), /25%/);
  await act(async () => environment.container.querySelectorAll('[role="tab"]')[2].click());
  assert.match(activePanelText(), /Cycle Label/);
  assert.match(activePanelText(), /40%/);

  await environment.render(null);
  await environment.render(
    React.createElement(ModuleProgressStats, {
      ...commonProps,
      moduleId: "module-1",
      distribution: {
        assignees: [
          {
            assignee_id: "module-user",
            display_name: "Module Assignee",
            avatar_url: null,
            completed_estimates: 3,
            total_estimates: 5,
          },
        ],
        labels: [
          {
            label_id: "module-label",
            label_name: "Module Label",
            color: "#00f",
            completed_estimates: 1,
            total_estimates: 5,
          },
        ],
      },
      plotType: "points",
    })
  );
  assert.match(activePanelText(), /Module Assignee/);
  assert.match(activePanelText(), /60%/);
  assert.doesNotMatch(activePanelText(), /Module Label/);

  await act(async () => environment.container.querySelectorAll('[role="tab"]')[0].click());
  assert.match(activePanelText(), /Started/);
  await act(async () => environment.container.querySelectorAll('[role="tab"]')[2].click());
  assert.match(activePanelText(), /Module Label/);
  assert.match(activePanelText(), /20%/);
});
