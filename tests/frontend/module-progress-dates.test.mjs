import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "./helpers/module-hooks.mjs";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const state = { moduleDetails: undefined };
globalThis.moduleProgressDates = state;
globalThis.moduleProgressDatesReact = React;
const react = "const React = globalThis.moduleProgressDatesReact;";
const stubs = {
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:'error'};export const setToast=value=>{(globalThis.moduleProgressDates.toasts??=[]).push(value);};`,
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
    const Selection=React.createContext(null);
    const CustomSelect = ({children,onChange,value}) => React.createElement(Selection.Provider,{value:{onChange,value}},children);
    CustomSelect.Option = ({value}) => {const select=React.useContext(Selection);return React.createElement('button',{type:'button','aria-pressed':value===select.value,onClick:()=>select.onChange(value)},value);};
    export { CustomSelect };
    export const Spinner = () => null;`,
  "@/components/core/sidebar/progress-chart": `${react}
    export default function ProgressChart({totalIssues,plotTitle}) {
      return React.createElement('div', { 'data-testid': 'module-progress-chart' },plotTitle+':'+totalIssues);
    }`,
  "@/components/core/sidebar/progress-stats/progress-stats": `${react}
    export const ProgressStats = ({handleFiltersUpdate,isEditable,totalIssuesCount}) => React.createElement('button', {type:'button',disabled:!isEditable,'data-testid':'module-progress-stats',onClick:()=>handleFiltersUpdate({property:'state_group',operator:'in',value:['completed']})},'Completed:'+totalIssuesCount);`,
  "@/hooks/use-estimate-statistics":
    "export const useEstimateStatistics = () => ({ isEnabled: globalThis.moduleProgressDates.estimates ?? false, activeEstimate: globalThis.moduleProgressDates.estimates ? {type:'points'} : undefined, statistics: { isError:false, isPending:false, retry:async()=>{} } });",
  "@/hooks/use-modules": `export const useModuleDetails=()=>({module:globalThis.moduleProgressDates.moduleDetails});`,
  "@/providers/cycle-module-preferences": `export const useParentChartPreferences=()=>({measure:globalThis.moduleProgressDates.chart?.measure??"issues",update:patch=>{globalThis.moduleProgressDates.chart=patch;}});`,
  "@/providers/parent-issue-preferences": `export const useParentIssuePreferences=()=>({
    filters:{richFilters:globalThis.moduleProgressDates.rich??{}}, setRich:async value=>{globalThis.moduleProgressDates.saved=value;if(globalThis.moduleProgressDates.fail)throw new Error("offline");}, isSaveError:false, retrySave:async()=>{}
  });`,
};

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

const { ModuleAnalyticsProgress } =
  await import("../../apps/web/core/components/modules/analytics-sidebar/issue-progress.tsx");

async function renderModuleProgress(moduleDetails, t) {
  state.moduleDetails = moduleDetails;
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/workspace/");
  t.after(() => environment.unmount());
  await routerFixture.render(
    environment,
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

// Register eagerly; node:test owns completion and reports failures before suite cleanup.
void test("modules without either date bound show add-date guidance", async (t) => {
  const environment = await renderModuleProgress(moduleWithDates(null, null), t);

  assert.match(environment.container.textContent, /Add date/);
  assert.doesNotMatch(environment.container.textContent, /Invalid date/);
  assert.equal(environment.container.querySelector("svg"), null);
});

void test("a valid partial module schedule asks for the missing bound instead of reporting an invalid date", async (t) => {
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

void test("malformed and reversed module date ranges retain the invalid-date message", async (t) => {
  const malformed = await renderModuleProgress(moduleWithDates("not-a-date", "2030-01-01"), t);
  assert.match(malformed.container.textContent, /Invalid date/);
  assert.ok(malformed.container.querySelector("svg"));

  await malformed.unmount();
  const reversed = await renderModuleProgress(moduleWithDates("2030-01-02", "2030-01-01"), t);
  assert.match(reversed.container.textContent, /Invalid date/);
});

void test("a rollover date is invalid and cannot render a progress chart", async (t) => {
  const moduleDetails = moduleWithDates("2020-02-31", "2099-12-31");
  moduleDetails.distribution = { completion_chart: { results: [] } };
  const environment = await renderModuleProgress(moduleDetails, t);

  assert.match(environment.container.textContent, /Invalid date/);
  assert.equal(environment.container.querySelector('[data-testid="module-progress-chart"]'), null);
});

void test("a complete valid module date range keeps the progress chart", async (t) => {
  const moduleDetails = moduleWithDates("2020-01-01", "2099-12-31");
  moduleDetails.distribution = { completion_chart: { results: [] } };
  const environment = await renderModuleProgress(moduleDetails, t);

  assert.match(environment.container.textContent, /Progress/);
  assert.equal(environment.container.querySelector('[data-testid="module-progress-chart"]') !== null, true);
});

await test("module points selection drives the matching chart totals and reports failed sidebar edits", async (t) => {
  state.estimates = true;
  state.chart = { measure: "points" };
  state.fail = true;
  state.toasts = [];
  state.saved = undefined;
  t.after(() => {
    state.estimates = false;
    state.chart = undefined;
    state.fail = false;
    state.toasts = [];
  });
  const module = {
    ...moduleWithDates("2020-01-01", "2099-12-31"),
    total_issues: 4,
    completed_issues: 1,
    total_estimate_points: 10,
    completed_estimate_points: 5,
    distribution: { completion_chart: { results: [] } },
    estimate_distribution: { completion_chart: { results: [] } },
  };
  const environment = await renderModuleProgress(module, t);
  assert.match(environment.container.textContent, /50%/);
  assert.equal(environment.container.querySelector('[data-testid="module-progress-chart"]').textContent, "points:10");
  const filter = environment.container.querySelector('[data-testid="module-progress-stats"]');
  assert.equal(filter.textContent, "Completed:10");
  await act(async () => filter.click());
  assert.deepEqual(state.saved, { state_group__in: "completed" });
  assert.equal(state.toasts.length, 1);
  assert.equal(state.toasts[0].message, "common.something_went_wrong");
  const issues = [...environment.container.querySelectorAll("button")].find((item) => item.textContent === "burndown");
  await act(async () => issues.click());
  assert.deepEqual(state.chart, { measure: "issues" });
});
