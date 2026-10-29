import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { openSearchCombobox } from "./helpers/combobox.mjs";
import { setPositioningGeometry } from "./helpers/positioning.mjs";
const { routerApi } = await import("./helpers/router.mjs");
const { useParams } = routerApi;
globalThis.estimateUiReact = React;
globalThis.estimateUiToasts = [];
const h = `const React=globalThis.estimateUiReact;`;
const ui =
  h +
  `
export const Avatar=()=>null;
export const EModalPosition={TOP:'top'}, EModalWidth={XXL:'xxl'};
export const ModalCore=({isOpen,children})=>isOpen?React.createElement('div',{role:'dialog'},children):null;
export const Input=({hasError,...props})=>React.createElement('input',props);
export const Sortable=({data,render})=>React.createElement(React.Fragment,null,...data.map(render));
export const Loader=Object.assign(({children})=>React.createElement('div',{'data-testid':'estimate-loader'},children),{Item:()=>null});
export const ToggleSwitch=({value,onChange,disabled})=>React.createElement('button',{role:'switch','aria-checked':value,onClick:onChange,disabled},'Enable estimates');`;
const stubs = {
  "@plane/ui": ui,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/button":
    h + `export const Button=({children,variant,size,loading,...props})=>React.createElement('button',props,children);`,
  "@plane/propel/empty-state":
    h +
    `export const EmptyStateCompact=({actions})=>React.createElement('div',null,'No estimates',...(actions??[]).map(action=>React.createElement('button',{key:action.label,onClick:action.onClick},action.label)));`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=value=>globalThis.estimateUiToasts.push(value);`,
};
const layoutStubs = {
  "@/hooks/use-workspace-issue-properties": "export const useWorkspaceIssueProperties=()=>{};",
  "@/components/issues/issue-layouts/quick-action-dropdowns": "export const AllIssueQuickActions=()=>null;",
  "@/components/ui/loader/layouts/spreadsheet-layout-loader":
    h + "export const SpreadsheetLayoutLoader=()=>React.createElement('output',null,'Loading spreadsheet');",
  "../spreadsheet-view":
    h +
    "export const SpreadsheetView=({issueIds})=>React.createElement('div',{'data-testid':'spreadsheet-order'},issueIds.join(','));",
  "../../issue-layout-HOC": "export const IssueLayoutHOC=({children})=>children;",
  "@/hooks/use-intersection-observer": "export const useIntersectionObserver=()=>{};",
  "@/components/dropdowns/state/dropdown": "export const StateDropdown=()=>null;",
  "@/components/core/sidebar/single-progress-stats": "export const SingleProgressStats=()=>null;",
  "@/components/empty-state/simple-empty-state-root": "export const SimpleEmptyState=()=>null;",
  "@/components/issues/issue-detail/issue-identifier": "export const IssueIdentifier=()=>null;",
};
registerHooks({
  resolve(specifier, context, next) {
    const parent = context.parentURL ?? "";
    const isLayout =
      parent.endsWith("/spreadsheet/roots/workspace-root.tsx") || parent.endsWith("/active-cycle/cycle-stats.tsx");
    if (isLayout && specifier.endsWith("?url"))
      return { shortCircuit: true, url: 'data:text/javascript,export default "asset";' };

    if (isLayout && layoutStubs[specifier])
      return { shortCircuit: true, url: "data:text/javascript," + encodeURIComponent(layoutStubs[specifier]) };
    if (isLayout && (specifier === "@plane/ui" || specifier === "@plane/i18n"))
      return { shortCircuit: true, url: "data:text/javascript," + encodeURIComponent(stubs[specifier]) };

    if (parent.includes("/components/estimates/") && stubs[specifier])
      return { shortCircuit: true, url: "data:text/javascript," + encodeURIComponent(stubs[specifier]) };
    return next(specifier, context);
  },
});
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const { IssueDetailUiStateProvider } = await import("../../apps/web/core/providers/issue-detail-ui-state.tsx");
const { EstimateService } = await import("../../apps/web/core/services/estimate.service.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { UserService } = await import("../../apps/web/core/services/user.service.ts");
const { projectKeys } = await import("../../apps/web/core/queries/projects.ts");
const { cycleDetailOptions } = await import("../../apps/web/core/queries/cycle-modules.ts");
const { estimateStatisticsRevision } = await import("../../apps/web/core/queries/parent-statistics.ts");
const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
const { CycleModulePreferencesProvider } = await import("../../apps/web/core/providers/cycle-module-preferences.tsx");
const { ParentIssuePreferencesProvider } = await import("../../apps/web/core/providers/parent-issue-preferences.tsx");
const { seedParentListPreferences } = await import("./helpers/parent-catalogue-fixture.mjs");
const { estimateKeys } = await import("../../apps/web/core/queries/estimates.ts");
const initialScope = {
  kind: "project",
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  membershipRole: 15,
  membershipRevision: "2026-10-24T00:00:00Z",
  projectId: "p1",
  projectMembershipRole: 20,
  projectRole: 20,
};
function nativeProjectScope(projectId = "p1") {
  return {
    ...initialScope,
    workspaceRole: 15,
    projectId,
    projectRoles: [
      ["p1", 20],
      ["p2", 20],
    ],
  };
}
const parentScope = { ...initialScope, workspaceRole: 15 };
function cycleOptions(rows) {
  return cycleDetailOptions(parentScope, "c1", false, estimateStatisticsRevision(rows?.[0]?.id ?? null, "p1", rows));
}
function withParentPreferences(element) {
  return React.createElement(
    CycleModulePreferencesProvider,
    null,
    React.createElement(
      ParentIssuePreferencesProvider,
      { domain: "cycle", workspaceSlug: "acme", projectId: "p1", parentId: "c1" },
      element
    )
  );
}
function row(id, project = "p1", value = "3") {
  return {
    id,
    project,
    workspace: "w1",
    name: id,
    type: "points",
    last_used: true,
    points: [{ id: id + "-point", estimate: id, project, workspace: "w1", key: 1, value }],
  };
}
function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
function button(container, text) {
  return [...container.querySelectorAll("button")].find(
    (node) => node.textContent === text || node.textContent.startsWith(text)
  );
}
async function click(node) {
  assert.ok(node, "the requested real control exists");
  await act(async () => node.click());
  await settleQuery();
}

async function fixture(t, rows) {
  globalThis.estimateUiToasts.length = 0;
  const view = await createPermissionUiFixture(t, {
    path: "/acme/settings/projects/p1/estimates",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20 },
  });
  // Load browser UI after the DOM exists so its subscriptions use client effects.
  const { EstimateRoot } = await import("../../apps/web/core/components/estimates/root.tsx");
  const { DeleteEstimateModal } = await import("../../apps/web/core/components/estimates/delete/modal.tsx");
  function Settings() {
    const { projectId } = useParams({ strict: false });
    return React.createElement(EstimateRoot, { workspaceSlug: "acme", projectId, isAdmin: true });
  }
  view.client.setDefaultOptions({ queries: { retry: false, gcTime: Infinity, staleTime: Infinity } });
  const projectRows = [
    { id: "p1", workspace: "w1", estimate: rows?.[0]?.id ?? null, identifier: "ALP", name: "Alpha" },
    { id: "p2", workspace: "w1", estimate: "Beta", identifier: "BET", name: "Beta" },
  ];
  view.client.setQueryData(projectKeys.detail(nativeProjectScope()), projectRows[0]);
  view.client.setQueryData(projectKeys.detail(nativeProjectScope("p2")), projectRows[1]);
  view.client.setQueryData(projectKeys.workspace(nativeProjectScope(), "partial"), projectRows);
  view.client.setQueryData(projectKeys.workspace(nativeProjectScope(), "full"), projectRows);
  t.mock.method(ProjectService.prototype, "getProject", async (_workspace, id) =>
    projectRows.find((project) => project.id === id)
  );
  t.mock.method(UserService.prototype, "currentUser", async () => ({ id: "alice" }));
  if (rows !== undefined) view.client.setQueryData(estimateKeys.list(initialScope), rows);
  const render = (element) => view.render(React.createElement(IssueDetailUiStateProvider, null, element));
  return { ...view, render, Settings, DeleteEstimateModal };
}

await test("real estimate settings recover a failed catalogue through Retry rather than an endless loader", async (t) => {
  let offline = true;
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => {
    if (offline) throw new Error("offline");
    return [];
  });
  const view = await fixture(t);
  await view.render(React.createElement(view.Settings));
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.equal(view.environment.container.querySelector('[data-testid="estimate-loader"]'), null);
  await click(button(view.environment.container, "Retry"));
  assert.ok(view.environment.container.querySelector('[role="alert"]'), "a failed Retry stays visibly recoverable");
  offline = false;
  await click(button(view.environment.container, "Retry"));
  await settleQuery();
  assert.match(view.environment.container.textContent, /No estimates/);
  assert.equal(view.environment.container.querySelector('[role="alert"]'), null);
});

await test("accepted archived estimate deletion preserves a different active project pointer and needs no second PATCH", async (t) => {
  const active = row("Active"),
    archived = { ...row("Archived"), last_used: false };
  t.mock.method(EstimateService.prototype, "deleteEstimate", async () => {});
  let patches = 0;
  t.mock.method(ProjectService.prototype, "updateProject", async () => {
    patches++;
    throw new Error("unexpected second write");
  });
  const view = await fixture(t, [active, archived]);
  let closes = 0;
  await view.render(
    React.createElement(view.DeleteEstimateModal, {
      workspaceSlug: "acme",
      projectId: "p1",
      estimateId: "Archived",
      isOpen: true,
      handleClose: () => closes++,
    })
  );
  await click(button(view.environment.container, "Delete Estimate"));
  assert.equal(closes, 1);
  assert.equal(patches, 0);
  assert.equal(view.client.getQueryData(projectKeys.detail(nativeProjectScope())).estimate, "Active");
  assert.deepEqual(
    view.client.getQueryData(estimateKeys.list(initialScope)).map((item) => item.id),
    ["Active"]
  );
});

await test("pending delete from real settings cannot disable or close a replacement project dialog", async (t) => {
  const pending = deferred(),
    started = deferred();
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async (_workspace, project) => [
    row(project === "p1" ? "Alpha" : "Beta", project),
  ]);
  t.mock.method(EstimateService.prototype, "deleteEstimate", async (_workspace, project, id) => {
    started.resolve({ project, id });
    await pending.promise;
  });
  const view = await fixture(t, [row("Alpha")]);
  await view.render(React.createElement(view.Settings));
  await click(view.environment.container.querySelector("button[data-ph-element]"));
  await click(button(view.environment.container, "Delete Estimate"));
  assert.equal((await started.promise).id, "Alpha");
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p2/estimates" }));
  await settleQuery();
  await click(view.environment.container.querySelector("button[data-ph-element]"));
  assert.match(view.environment.container.querySelector('[role="dialog"]').textContent, /Beta/);
  assert.equal(button(view.environment.container, "Delete Estimate").disabled, false);
  await act(async () => pending.resolve());
  await settleQuery();
  assert.match(view.environment.container.querySelector('[role="dialog"]').textContent, /Beta/);
  assert.equal(button(view.environment.container, "Delete Estimate").disabled, false);
  assert.equal(view.client.getQueryData(projectKeys.detail(nativeProjectScope("p2"))).estimate, "Beta");
  assert.deepEqual(globalThis.estimateUiToasts, []);
});

async function assertReplacedEstimateDraft(t, rejected) {
  const pending = deferred(),
    started = deferred();
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => []);
  t.mock.method(EstimateService.prototype, "createEstimate", async (_workspace, project, data) => {
    started.resolve(project);
    await pending.promise;
    return {
      id: "New",
      project,
      workspace: "w1",
      name: "New",
      type: "points",
      last_used: true,
      points: data.estimate_points.map((point, index) => ({
        ...point,
        id: "new-" + index,
        project,
        workspace: "w1",
        estimate: "New",
      })),
    };
  });
  const view = await fixture(t, []);
  await view.render(React.createElement(React.StrictMode, null, React.createElement(view.Settings)));
  await click(button(view.environment.container, "settings_empty_state.estimates.cta_primary"));
  await click(button(view.environment.container, "project_settings.estimates.create.custom"));
  await changeFormControl(view.environment.container.querySelector('[aria-label="Estimate value 1"]'), "11");
  await submitForm(view.environment.container);
  assert.equal(await started.promise, "p1");
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p2/estimates" }));
  await settleQuery();
  await click(button(view.environment.container, "settings_empty_state.estimates.cta_primary"));
  await click(button(view.environment.container, "project_settings.estimates.create.custom"));
  const draft = view.environment.container.querySelector('[aria-label="Estimate value 1"]');
  await changeFormControl(draft, "22");
  await act(async () => {
    if (rejected) pending.reject(new Error("late failure"));
    else pending.resolve();
  });
  await settleQuery();
  assert.equal(view.environment.container.querySelector('[aria-label="Estimate value 1"]').value, "22");
  assert.equal(view.environment.container.querySelector('[aria-label="Estimate value 1"]').disabled, false);
  assert.deepEqual(globalThis.estimateUiToasts, []);
}
await test("old accepted native estimate form cannot close or toast into a replacement draft", (t) =>
  assertReplacedEstimateDraft(t, false));
await test("old rejected native estimate form cannot close or toast into a replacement draft", (t) =>
  assertReplacedEstimateDraft(t, true));

await test("live estimate dropdown follows canonical values and hides cached choices after read error", async (t) => {
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => {
    throw new Error("offline");
  });
  const view = await fixture(t, [row("Effort")]);
  const { EstimateDropdown } = await import("../../apps/web/core/components/dropdowns/estimate.tsx");
  await view.render(
    React.createElement(EstimateDropdown, {
      projectId: "p1",
      value: "Effort-point",
      buttonVariant: "transparent-with-text",
      onChange() {},
    })
  );
  assert.equal(view.environment.container.textContent, "3");
  await act(async () => view.client.setQueryData(estimateKeys.list(initialScope), [row("Effort", "p1", "13")]));
  await settleQuery();
  assert.equal(view.environment.container.textContent, "13");
  const trigger = view.environment.container.querySelector("button");
  setPositioningGeometry(trigger);
  await openSearchCombobox(trigger);
  assert.match(document.querySelector('[role="listbox"]').textContent, /13/);
  await act(async () => {
    await view.client.refetchQueries({ queryKey: estimateKeys.list(initialScope), exact: true });
  });
  await settleQuery();
  assert.ok(document.querySelector('[role="alert"]'));
  assert.doesNotMatch(document.body.textContent, /13/);
});

await test("numeric point changes refresh actual native parent statistics, keys only reorder, and failed refresh can retry", async (t) => {
  const { useEstimateStatistics } = await import("../../apps/web/core/hooks/use-estimate-statistics.ts");
  const { useCycleDetails } = await import("../../apps/web/core/hooks/use-cycles.ts");
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  const view = await fixture(t, [row("Effort")]);
  view.client.setQueryData(cycleOptions([row("Effort")]).queryKey, {
    id: "c1",
    project_id: "p1",
    total_estimate_points: 3,
  });
  const reads = [];
  t.mock.method(CycleService.prototype, "retrieve", () => {
    const read = deferred();
    reads.push(read);
    return read.promise;
  });
  let listReads = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    listReads++;
    throw new Error("unexpected issue list refetch");
  });
  function Probe() {
    const result = useEstimateStatistics("cycle", "acme", "p1", "c1");
    const { cycle } = useCycleDetails("acme", "p1", "c1");
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(
        "output",
        null,
        result.statistics.isError ? "error" : result.statistics.isPending ? "pending" : "ready",
        ":",
        cycle?.total_estimate_points ?? "none"
      ),
      React.createElement("button", { onClick: () => void result.statistics.retry() }, "Retry statistics")
    );
  }
  await view.render(React.createElement(Probe));
  assert.equal(view.environment.container.querySelector("output").textContent, "ready:3");
  const keyed = row("Effort");
  keyed.points[0].key = 2;
  await act(async () => view.client.setQueryData(estimateKeys.list(initialScope), [keyed]));
  await settleQuery();
  assert.equal(reads.length, 0);
  await act(async () => view.client.setQueryData(estimateKeys.list(initialScope), [row("Effort", "p1", "8")]));
  await settleQuery();
  assert.equal(reads.length, 1);
  assert.equal(view.environment.container.querySelector("output").textContent, "pending:none");
  await act(async () => reads[0].reject(new Error("statistics offline")));
  await settleQuery();
  assert.equal(view.environment.container.querySelector("output").textContent, "error:none");
  await click(button(view.environment.container, "Retry statistics"));
  assert.equal(reads.length, 2);
  await act(async () => reads[1].resolve({ id: "c1", project_id: "p1", total_estimate_points: 8 }));
  await settleQuery();
  assert.equal(view.environment.container.querySelector("output").textContent, "ready:8");
  const nextRows = [row("Effort", "p1", "13")];
  await act(async () => view.client.setQueryData(estimateKeys.list(initialScope), nextRows));
  await settleQuery();
  const next = cycleOptions(nextRows);
  let replacement;
  await act(async () => {
    await view.client.cancelQueries({ queryKey: next.queryKey, exact: true });
    replacement = view.client.fetchQuery({ ...next, staleTime: 0 });
  });
  assert.equal(reads.length, 4);
  await act(async () => {
    reads[3].resolve({ id: "c1", project_id: "p1", total_estimate_points: 13 });
    await replacement;
  });
  await act(async () => reads[2].resolve({ id: "c1", project_id: "p1", total_estimate_points: 8 }));
  await settleQuery();
  assert.equal(view.environment.container.querySelector("output").textContent, "ready:13");
  await act(async () => view.client.setQueryData(estimateKeys.list(initialScope), [row("Effort", "p1", "21")]));
  await settleQuery();
  assert.equal(reads.length, 5);
  await act(async () =>
    view.client
      .getQueryCache()
      .find({ queryKey: permissionKeys.workspaceMembership("alice", "acme"), exact: true })
      .setState({ status: "error", error: { response: { status: 403 } } })
  );
  await act(async () => reads[4].resolve({ id: "c1", project_id: "p1", total_estimate_points: 999 }));
  await settleQuery();
  assert.notEqual(
    view.client.getQueryData(cycleOptions([row("Effort", "p1", "21")]).queryKey)?.total_estimate_points,
    999
  );
  assert.match(view.environment.container.querySelector("output").textContent, /^error:/);
  assert.equal(listReads, 0);
});

async function prepareNativeListRendering(t) {
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => ({
    is_active: true,
    role: 15,
    updated_at: initialScope.membershipRevision,
  }));
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => []);
  t.mock.method(IssueLabelService.prototype, "getWorkspaceIssueLabels", async () => []);
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => []);
  let reads = 0;
  t.mock.method(WorkspaceService.prototype, "getViewIssues", async () => {
    reads++;
    throw new Error("Unexpected issue-list refetch");
  });
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    reads++;
    throw new Error("Unexpected issue-list refetch");
  });
  const issuesById = {
    high: {
      id: "high",
      name: "High estimate",
      project_id: "p1",
      estimate_point: "Effort-high",
      priority: "high",
      created_at: "2026-10-24",
    },
    low: {
      id: "low",
      name: "Low estimate",
      project_id: "p1",
      estimate_point: "Effort-low",
      priority: "high",
      created_at: "2026-10-24",
    },
  };
  return { readCount: () => reads, issuesById };
}
function orderedCatalogue() {
  return {
    ...row("Effort"),
    points: [
      { ...row("Effort").points[0], id: "Effort-high", key: 2 },
      { ...row("Effort").points[0], id: "Effort-low", key: 1 },
    ],
  };
}
await test("actual workspace spreadsheet hides unavailable estimate order, retries metadata and reorders without issue IO", async (t) => {
  const view = await fixture(t, [orderedCatalogue()]);
  const { readCount, issuesById } = await prepareNativeListRendering(t);
  const { workspaceViewKeys } = await import("../../apps/web/core/queries/global-views.ts");
  const { WorkspaceIssuePreferencesProvider } =
    await import("../../apps/web/core/providers/workspace-issue-preferences.tsx");
  const { WorkspaceSpreadsheetRoot } =
    await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/roots/workspace-root.tsx");
  const { ALL_ISSUES } = await import("../../packages/constants/dist/index.mjs");
  let offline = true;
  t.mock.method(EstimateService.prototype, "fetchWorkspaceEstimates", async () => {
    if (offline) throw new Error("catalogue offline");
    return [orderedCatalogue()];
  });
  await act(async () => view.router.navigate({ to: "/acme/workspace-views/view-a" }));
  view.client.setQueryData(
    workspaceViewKeys.list({
      accountId: "alice",
      workspaceSlug: "acme",
      workspaceRole: 15,
      membershipRevision: initialScope.membershipRevision,
      sessionGeneration: 0,
    }),
    [
      {
        id: "view-a",
        rich_filters: {},
        display_filters: { layout: "spreadsheet", order_by: "estimate_point__key" },
        display_properties: {},
      },
    ]
  );
  await view.render(
    React.createElement(
      WorkspaceIssuePreferencesProvider,
      null,
      React.createElement(WorkspaceSpreadsheetRoot, {
        workspaceSlug: "acme",
        globalViewId: "view-a",
        isDefaultView: false,
        readState: {
          issuesById,
          groupedIssueIds: { [ALL_ISSUES]: ["high", "low"] },
          isPending: false,
          getPaginationData: () => undefined,
          fetchNextIssues: async () => {},
          getIssueLoader: () => undefined,
          getGroupIssueCount: () => 2,
        },
        routeFilters: {},
        globalViewsLoading: false,
      })
    )
  );
  await settleQuery();
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.equal(view.environment.container.querySelector('[data-testid="spreadsheet-order"]'), null);
  offline = false;
  await click(button(view.environment.container, "Retry"));
  await settleQuery();
  assert.equal(view.environment.container.querySelector('[data-testid="spreadsheet-order"]').textContent, "low,high");
  const key = estimateKeys.list({
    ...initialScope,
    kind: "workspace",
    projectId: undefined,
    projectRole: undefined,
    projectMembershipRole: undefined,
  });
  const next = structuredClone(orderedCatalogue());
  for (const point of next.points) point.key = 3 - point.key;
  await act(async () => view.client.setQueryData(key, [next]));
  await settleQuery();
  assert.equal(view.environment.container.querySelector('[data-testid="spreadsheet-order"]').textContent, "high,low");
  offline = true;
  await act(async () => view.client.refetchQueries({ queryKey: key, exact: true }));
  await settleQuery();
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.equal(view.environment.container.querySelector('[data-testid="spreadsheet-order"]'), null);
  assert.equal(readCount(), 0);
});
await test("actual active-cycle issue list exposes estimate metadata failure and recovers its loaded order without issue IO", async (t) => {
  const view = await fixture(t, [orderedCatalogue()]);
  const { readCount, issuesById } = await prepareNativeListRendering(t);
  const { ActiveCycleStats } = await import("../../apps/web/core/components/cycles/active-cycle/cycle-stats.tsx");
  await act(async () => view.router.navigate({ to: "/acme/projects/p1/cycles/c1" }));
  const chartKey = JSON.stringify([JSON.stringify(["alice", "acme", "p1"]), "cycle", "c1"]);
  window.localStorage.setItem(
    "vector:parent-chart-preferences:" + chartKey,
    JSON.stringify({ statisticsTab: "Priority-Issues" })
  );
  await seedParentListPreferences(t, view.client, "cycle", "c1", {
    richFilters: {},
    displayFilters: { order_by: "estimate_point__key" },
    displayProperties: {},
  });
  let priorityReads = 0;
  t.mock.method(CycleService.prototype, "getCycleIssues", async () => {
    priorityReads++;
    return {
      results: [issuesById.high, issuesById.low],
      total_count: 2,
      next_page_results: false,
    };
  });
  let offline = true;
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => {
    if (offline) throw new Error("catalogue offline");
    return [orderedCatalogue()];
  });
  await view.render(
    withParentPreferences(
      React.createElement(ActiveCycleStats, {
        workspaceSlug: "acme",
        projectId: "p1",
        cycleId: "c1",
        cycle: null,
        handleFiltersUpdate() {},
      })
    )
  );
  await act(async () => view.client.refetchQueries({ queryKey: estimateKeys.list(initialScope), exact: true }));
  await settleQuery();
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.doesNotMatch(view.environment.container.textContent, /High estimate|Low estimate/);
  offline = false;
  await click(button(view.environment.container, "Retry"));
  await settleQuery();
  assert.match(view.environment.container.textContent, /Low estimate.*High estimate/);
  assert.equal(readCount(), 0);
  assert.equal(priorityReads, 1, "metadata Retry does not refetch the native priority issue page");
});

await test("actual cycle mutation projection preserves accepted lineage through transient identity error and rejects auth denial", async (t) => {
  const view = await fixture(t, [orderedCatalogue()]);
  const { useIssueListMutationOwner } = await import("../../apps/web/core/hooks/use-issue-list-mutation-owner.ts");
  const { captureIssueListMutationOwner } = await import("../../apps/web/core/queries/issue-list-mutations.ts");
  const { ProjectStateService } = await import("../../apps/web/core/services/project/project-state.service.ts");
  t.mock.method(ProjectStateService.prototype, "getStates", async () => []);
  const { EIssuesStoreType } = await import("../../packages/types/dist/index.mjs");
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  await act(async () => view.router.navigate({ to: "/acme/projects/p1/cycles/c1" }));
  await seedParentListPreferences(t, view.client, "cycle", "c1", {
    richFilters: {},
    displayFilters: { order_by: "estimate_point__key" },
    displayProperties: {},
  });
  let input;
  function Resolve({ onResolve }) {
    const resolve = useIssueListMutationOwner(EIssuesStoreType.CYCLE);
    return React.createElement(
      "button",
      {
        onClick: async () => {
          onResolve(await resolve("acme", "p1"));
        },
      },
      "Resolve"
    );
  }
  await view.render(
    withParentPreferences(
      React.createElement(Resolve, {
        onResolve: (value) => {
          input = captureIssueListMutationOwner(view.client, value);
        },
      })
    )
  );
  await click(button(view.environment.container, "Resolve"));
  assert.equal(input.isCurrent(), true);
  const query = view.client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  await act(async () =>
    query.setState({
      status: "error",
      error: Object.assign(new Error("temporary verification failure"), { response: { status: 500 } }),
    })
  );
  await settleQuery();
  assert.equal(input.isCurrent(), true, "accepted list projection keeps the verified session lineage");
  assert.equal(input.access.isVerified(), false, "new reads and writes require verified identity");
  await act(async () =>
    query.setState({ status: "error", error: Object.assign(new Error("forbidden"), { response: { status: 403 } }) })
  );
  await settleQuery();
  assert.equal(input.isCurrent(), false, "authorization denial is never accepted as transient lineage");
});

await test("accepted native create cannot close or toast after its permission query is replaced", async (t) => {
  const view = await fixture(t, []);
  const { CreateEstimateModal } = await import("../../apps/web/core/components/estimates/create/modal.tsx");
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  const pending = deferred(),
    started = deferred();
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => []);
  t.mock.method(EstimateService.prototype, "createEstimate", async (_workspace, project, data) => {
    started.resolve();
    await pending.promise;
    return {
      ...row("Created", project),
      points: data.estimate_points.map((point, index) => ({
        ...row("Created", project).points[0],
        ...point,
        id: "created-" + index,
      })),
    };
  });
  let closed = 0;
  await view.render(
    React.createElement(CreateEstimateModal, {
      workspaceSlug: "acme",
      projectId: "p1",
      isOpen: true,
      handleClose: () => closed++,
    })
  );
  await click(button(view.environment.container, "project_settings.estimates.create.custom"));
  await submitForm(view.environment.container);
  await started.promise;
  const key = permissionKeys.workspaceProjectRoles("alice", "acme", 15, initialScope.membershipRevision);
  await act(async () => {
    const query = view.client.getQueryCache().find({ queryKey: key, exact: true });
    view.client.getQueryCache().remove(query);
    view.client.setQueryData(key, { p1: 20, p2: 20 });
  });
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(closed, 0);
  assert.deepEqual(globalThis.estimateUiToasts, []);
  assert.equal(view.environment.container.querySelector('[aria-label="Estimate value 1"]').value, "1");
});
