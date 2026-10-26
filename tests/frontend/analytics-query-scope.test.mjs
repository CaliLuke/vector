import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { QueryClient, skipToken } from "@tanstack/react-query";

registerWebModuleHooks();

const { AnalyticsService } = await import("../../apps/web/core/services/analytics.service.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { analyticsChartOptions, analyticsInsightsOptions, analyticsStatsOptions } =
  await import("../../apps/web/core/queries/analytics.ts");
const { advanceIdentitySession, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");

const ACCOUNT = "analytics-account";
const WORKSPACE = "analytics-workspace";
const REVISION = "membership-revision";
const MEMBER = 15;
const GUEST = 5;

function scope({ role = MEMBER, projectId, projectRole, projectRoles: passedProjectRoles, generation = 0 } = {}) {
  const projectRoles = passedProjectRoles ?? (projectId ? { [projectId]: projectRole } : {});
  return {
    accountId: ACCOUNT,
    workspaceSlug: WORKSPACE,
    workspaceRole: role,
    membershipRevision: REVISION,
    projectRolesRevision: JSON.stringify(
      Object.entries(projectRoles).toSorted(([left], [right]) => left.localeCompare(right))
    ),
    sessionGeneration: generation,
    projectId,
    projectRole,
    isActive: true,
  };
}

function primeScope(client, currentScope) {
  const membership = {
    is_active: true,
    role: currentScope.workspaceRole,
    updated_at: currentScope.membershipRevision,
  };
  client.setQueryData(identityKeys.currentUser(), { id: currentScope.accountId });
  client.setQueryData(permissionKeys.workspaceMembership(ACCOUNT, WORKSPACE), membership);
  const projectRoles = Object.fromEntries(JSON.parse(currentScope.projectRolesRevision ?? "[]"));
  client.setQueryData(
    permissionKeys.workspaceProjectRoles(
      ACCOUNT,
      WORKSPACE,
      currentScope.workspaceRole,
      currentScope.membershipRevision
    ),
    projectRoles
  );
}

function attachQuery(client, options) {
  client.getQueryCache().build(client, options);
  return options.queryFn;
}

void test("analytics request keys and inputs preserve account, membership, filters, and abort signal", async (t) => {
  const client = new QueryClient();
  t.after(() => client.clear());
  const currentScope = scope({ projectId: "project-member", projectRole: MEMBER });
  primeScope(client, currentScope);
  const calls = [];
  t.mock.method(AnalyticsService.prototype, "getAdvanceAnalytics", async (...args) => {
    calls.push(args);
    return { completed: 3 };
  });

  const params = { project_ids: "project-member", cycle_id: "cycle-1", epic: true };
  const options = analyticsInsightsOptions(currentScope, "work-items", params, true);
  const queryFn = attachQuery(client, options);
  const signal = new AbortController().signal;
  const data = await queryFn({ client, signal });
  assert.deepEqual(data, { completed: 3 });
  assert.deepEqual(calls[0], [WORKSPACE, "work-items", params, true, signal]);
  assert.deepEqual(options.queryKey.slice(0, 9), [
    "workspace-analytics",
    ACCOUNT,
    WORKSPACE,
    MEMBER,
    REVISION,
    JSON.stringify([["project-member", MEMBER]]),
    0,
    "project-member",
    MEMBER,
  ]);
  assert.equal(options.queryKey.at(-2), params);
});

void test("project guest can read chart endpoints while member-only insight and stats remain disabled", async (t) => {
  const client = new QueryClient();
  t.after(() => client.clear());
  const guestScope = scope({ role: GUEST, projectId: "guest-project", projectRole: GUEST });
  primeScope(client, guestScope);
  const calls = [];
  t.mock.method(AnalyticsService.prototype, "getAdvanceAnalyticsCharts", async (...args) => {
    calls.push(args);
    return [];
  });

  const chartParams = { project_ids: "guest-project", epic: true, x_axis: "priority" };
  const chartOptions = analyticsChartOptions(guestScope, "custom-work-items", chartParams, true);
  const chartQueryFn = attachQuery(client, chartOptions);
  assert.equal(typeof chartQueryFn, "function");
  await chartQueryFn({ client, signal: new AbortController().signal });
  assert.equal(calls.length, 1);
  assert.equal(calls[0][3], true);
  assert.deepEqual(calls[0][2], chartParams);
  assert.equal(
    analyticsInsightsOptions(guestScope, "work-items", { project_ids: "guest-project" }, true).queryFn,
    skipToken
  );
  assert.equal(analyticsStatsOptions(guestScope, { project_ids: "guest-project" }, true).queryFn, skipToken);
});

void test("in-flight analytics response is rejected after the account session changes", async (t) => {
  const client = new QueryClient();
  t.after(() => client.clear());
  const currentScope = scope();
  primeScope(client, currentScope);
  let resolveRequest;
  t.mock.method(
    ProjectService.prototype,
    "getProjectAnalyticsCount",
    () =>
      new Promise((resolve) => {
        resolveRequest = resolve;
      })
  );

  const options = (await import("../../apps/web/core/queries/analytics.ts")).projectAnalyticsCountsOptions(
    currentScope,
    { fields: "total_work_items" }
  );
  const queryFn = attachQuery(client, options);
  const request = queryFn({ client, signal: new AbortController().signal });
  advanceIdentitySession(client);
  resolveRequest([{ id: "project", total_work_items: 2 }]);
  await assert.rejects(request, /Analytics access changed while the request was loading/);
});

void test("workspace aggregates are keyed and guarded by the canonical project membership snapshot", async (t) => {
  const client = new QueryClient();
  t.after(() => client.clear());
  const currentScope = scope({ projectRoles: { "project-a": MEMBER, "project-b": GUEST } });
  primeScope(client, currentScope);
  let resolveRequest;
  t.mock.method(
    AnalyticsService.prototype,
    "getAdvanceAnalytics",
    () =>
      new Promise((resolve) => {
        resolveRequest = resolve;
      })
  );

  const options = analyticsInsightsOptions(currentScope, "work-items", {}, false);
  const queryFn = attachQuery(client, options);
  const request = queryFn({ client, signal: new AbortController().signal });
  client.setQueryData(permissionKeys.workspaceProjectRoles(ACCOUNT, WORKSPACE, MEMBER, REVISION), {
    "project-a": GUEST,
    "project-b": GUEST,
  });
  resolveRequest({ count: 4 });
  await assert.rejects(request, /Analytics access changed while the request was loading/);

  const changedScope = scope({ projectRoles: { "project-a": GUEST, "project-b": GUEST } });
  const changedOptions = analyticsInsightsOptions(changedScope, "work-items", {}, false);
  assert.notDeepEqual(changedOptions.queryKey, options.queryKey);
});
