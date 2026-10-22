import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const state = { moduleDetails: undefined };
globalThis.__moduleProgressDates = state;
globalThis.__moduleProgressDatesReact = React;
const react = "const React = globalThis.__moduleProgressDatesReact;";
const stubs = {
  "next/navigation": "export const useSearchParams = () => new URLSearchParams();",
  "@plane/i18n": `export const useTranslation = () => ({t: key => ({
    'project_cycles.add_date': 'Add date',
    'project_module.empty_state.sidebar.in_active': "This module isn't active yet.",
    'project_module.empty_state.sidebar.invalid_date': 'Invalid date. Please enter valid date.',
    'common.progress': 'Progress',
  })[key] ?? key});`,
  "@plane/propel/icons": `${react}
    export const ChevronUpIcon = props => React.createElement('span', props);
    export const ChevronDownIcon = props => React.createElement('span', props);`,
  "@plane/ui": `${react}
    const CustomSelect = () => null;
    CustomSelect.Option = () => null;
    export { CustomSelect };
    export const Spinner = () => null;`,
  "@/components/core/sidebar/progress-chart": `${react}
    export default function ProgressChart() {
      return React.createElement('div', { 'data-testid': 'module-progress-chart' });
    }`,
  "@/components/modules": `${react}
    export const ModuleProgressStats = () => React.createElement('div', { 'data-testid': 'module-progress-stats' });`,
  "@/hooks/store/estimates":
    "export const useProjectEstimates = () => ({ areEstimateEnabledByProjectId: () => false });",
  "@/hooks/store/use-module": `export const useModule = () => ({
    getPlotTypeByModuleId: () => 'burndown',
    setPlotType: () => {},
    getModuleById: () => globalThis.__moduleProgressDates.moduleDetails,
    fetchModuleDetails: async () => {},
    fetchArchivedModuleDetails: async () => {},
  });`,
  "@/hooks/store/work-item-filters/use-work-item-filters": `export const useWorkItemFilters = () => ({
    getFilter: () => undefined,
    updateFilterValueFromSidebar: () => {},
  });`,
};

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (Object.hasOwn(stubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const { ModuleAnalyticsProgress } =
  await import("../../apps/web/core/components/modules/analytics-sidebar/issue-progress.tsx");

async function renderModuleProgress(moduleDetails, t) {
  state.moduleDetails = moduleDetails;
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(
    React.createElement(ModuleAnalyticsProgress, {
      workspaceSlug: "default",
      projectId: "project-id",
      moduleId: "module-id",
    })
  );
  return environment;
}

function moduleWithDates(startDate, targetDate) {
  return {
    id: "module-id",
    status: "completed",
    start_date: startDate,
    target_date: targetDate,
    completed_issues: 1,
    total_issues: 1,
  };
}

test("modules without either date bound show add-date guidance", async (t) => {
  const environment = await renderModuleProgress(moduleWithDates(null, null), t);

  assert.match(environment.container.textContent, /Add date/);
  assert.doesNotMatch(environment.container.textContent, /Invalid date/);
  assert.equal(environment.container.querySelector("svg"), null);
});

test("a valid partial module schedule asks for the missing bound instead of reporting an invalid date", async (t) => {
  const withStartDate = await renderModuleProgress(moduleWithDates("2020-01-01", null), t);
  assert.match(withStartDate.container.textContent, /Add date/);
  assert.doesNotMatch(withStartDate.container.textContent, /Invalid date/);
  assert.equal(withStartDate.container.querySelector("svg"), null);

  await withStartDate.unmount();
  const withTargetDate = await renderModuleProgress(moduleWithDates(null, "2030-01-01"), t);
  assert.match(withTargetDate.container.textContent, /Add date/);
  assert.doesNotMatch(withTargetDate.container.textContent, /Invalid date/);
  assert.equal(withTargetDate.container.querySelector("svg"), null);
});

test("malformed and reversed module date ranges retain the invalid-date message", async (t) => {
  const malformed = await renderModuleProgress(moduleWithDates("not-a-date", "2030-01-01"), t);
  assert.match(malformed.container.textContent, /Invalid date/);
  assert.ok(malformed.container.querySelector("svg"));

  await malformed.unmount();
  const reversed = await renderModuleProgress(moduleWithDates("2030-01-02", "2030-01-01"), t);
  assert.match(reversed.container.textContent, /Invalid date/);
});

test("a rollover date is invalid and cannot render a progress chart", async (t) => {
  const moduleDetails = moduleWithDates("2020-02-31", "2099-12-31");
  moduleDetails.distribution = { completion_chart: { results: [] } };
  const environment = await renderModuleProgress(moduleDetails, t);

  assert.match(environment.container.textContent, /Invalid date/);
  assert.equal(environment.container.querySelector('[data-testid="module-progress-chart"]'), null);
});

test("a complete valid module date range keeps the progress chart", async (t) => {
  const moduleDetails = moduleWithDates("2020-01-01", "2099-12-31");
  moduleDetails.distribution = { completion_chart: { results: [] } };
  const environment = await renderModuleProgress(moduleDetails, t);

  assert.match(environment.container.textContent, /Progress/);
  assert.equal(environment.container.querySelector('[data-testid="module-progress-chart"]') !== null, true);
});
