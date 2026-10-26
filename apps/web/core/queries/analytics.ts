/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import { EUserPermissions } from "@plane/constants";
import { EUserWorkspaceRoles } from "@plane/types";
import sortBy from "lodash-es/sortBy.js";
import type {
  IAnalyticsResponse,
  IWorkspaceMemberMe,
  IUserProjectsRole,
  TAnalyticsFilterParams,
  TAnalyticsGraphsBase,
  TAnalyticsTabsBase,
  TProjectAnalyticsCount,
  TProjectAnalyticsCountParams,
  TUserPermissions,
  WorkItemInsightColumns,
} from "@plane/types";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import { captureIdentityScope, getIdentitySessionGeneration, isIdentityScopeVerified } from "@/queries/identity";
import { permissionKeys } from "@/queries/permissions";
import { AnalyticsService } from "@/services/analytics.service";
import { ProjectService } from "@/services/project/project.service";

const analyticsService = new AnalyticsService();
const projectService = new ProjectService();

export function getProjectRolesRevision(roles: IUserProjectsRole | undefined) {
  if (!roles) return undefined;
  return JSON.stringify(sortBy(Object.entries(roles), ([projectId]) => projectId));
}

export type AnalyticsRequestScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  workspaceRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectRolesRevision: string | undefined;
  sessionGeneration: number;
  projectId: string | undefined;
  projectRole: TUserPermissions | undefined;
  isActive: boolean;
};

export const analyticsKeys = {
  scope: (scope: AnalyticsRequestScope) =>
    [
      "workspace-analytics",
      scope.accountId,
      scope.workspaceSlug,
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectRolesRevision,
      scope.sessionGeneration,
      scope.projectId,
      scope.projectRole,
    ] as const,
  insights: (
    scope: AnalyticsRequestScope,
    analyticsType: TAnalyticsTabsBase,
    params: TAnalyticsFilterParams,
    isPeekView: boolean
  ) => [...analyticsKeys.scope(scope), "insights", analyticsType, params, isPeekView] as const,
  chart: (
    scope: AnalyticsRequestScope,
    chartType: TAnalyticsGraphsBase,
    params: Record<string, unknown>,
    isPeekView: boolean
  ) => [...analyticsKeys.scope(scope), "chart", chartType, params, isPeekView] as const,
  stats: (scope: AnalyticsRequestScope, params: TAnalyticsFilterParams, isPeekView: boolean) =>
    [...analyticsKeys.scope(scope), "stats", params, isPeekView] as const,
  projectCounts: (scope: AnalyticsRequestScope, params: TProjectAnalyticsCountParams) =>
    [...analyticsKeys.scope(scope), "project-counts", params] as const,
};

type TReadPermission = "workspace-member" | "workspace-active-member" | "project-member" | "project-chart";

function hasPermission(scope: AnalyticsRequestScope, permission: TReadPermission) {
  if (
    !scope.isActive ||
    !scope.accountId ||
    !scope.workspaceSlug ||
    !scope.membershipRevision ||
    !scope.projectRolesRevision
  )
    return false;
  if (permission === "workspace-active-member") return Boolean(scope.workspaceRole);
  if (permission === "workspace-member")
    return scope.workspaceRole === EUserWorkspaceRoles.ADMIN || scope.workspaceRole === EUserWorkspaceRoles.MEMBER;
  if (!scope.projectId) return false;
  if (permission === "project-member")
    return scope.projectRole === EUserPermissions.ADMIN || scope.projectRole === EUserPermissions.MEMBER;
  return (
    scope.projectRole === EUserPermissions.ADMIN ||
    scope.projectRole === EUserPermissions.MEMBER ||
    scope.projectRole === EUserPermissions.GUEST
  );
}

function hasCurrentIdentity(
  queryClient: QueryClient,
  identity: ReturnType<typeof captureIdentityScope>,
  scope: AnalyticsRequestScope
) {
  return Boolean(
    identity &&
    isIdentityScopeVerified(queryClient, identity) &&
    identity.generation === scope.sessionGeneration &&
    getIdentitySessionGeneration(queryClient) === scope.sessionGeneration
  );
}

function hasCurrentMembership(
  queryClient: QueryClient,
  membershipKey: readonly unknown[],
  membershipQuery: Query | undefined,
  scope: AnalyticsRequestScope
) {
  const membership = queryClient.getQueryData<IWorkspaceMemberMe>(membershipKey);
  return Boolean(
    membershipQuery &&
    queryClient.getQueryCache().find({ queryKey: membershipKey, exact: true }) === membershipQuery &&
    membershipQuery.state.status === "success" &&
    membership?.is_active === true &&
    membership.role === scope.workspaceRole &&
    membership.updated_at === scope.membershipRevision
  );
}

function hasCurrentProjectRoles(
  queryClient: QueryClient,
  rolesKey: readonly unknown[],
  rolesQuery: Query | undefined,
  scope: AnalyticsRequestScope
) {
  const roleMap = queryClient.getQueryData<IUserProjectsRole>(rolesKey);
  if (
    !rolesQuery ||
    queryClient.getQueryCache().find({ queryKey: rolesKey, exact: true }) !== rolesQuery ||
    rolesQuery.state.status !== "success" ||
    getProjectRolesRevision(roleMap) !== scope.projectRolesRevision
  )
    return false;
  return (
    !scope.projectId ||
    effectiveProjectRole(scope.workspaceRole ?? null, roleMap, scope.projectId) === scope.projectRole
  );
}

function createScopedRead<T>(
  scope: AnalyticsRequestScope,
  queryKey: readonly unknown[],
  permission: TReadPermission,
  load: (signal: AbortSignal) => Promise<T>
) {
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    retry: false,
    queryFn: hasPermission(scope, permission)
      ? async ({ client: queryClient, signal }) => {
          const identity = captureIdentityScope(queryClient, scope.accountId);
          const query = queryClient.getQueryCache().find({ queryKey, exact: true });
          const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
          const membershipQuery = queryClient.getQueryCache().find({ queryKey: membershipKey, exact: true });
          const rolesKey = permissionKeys.workspaceProjectRoles(
            scope.accountId,
            scope.workspaceSlug,
            scope.workspaceRole,
            scope.membershipRevision
          );
          const rolesQuery = queryClient.getQueryCache().find({ queryKey: rolesKey, exact: true });
          const isCurrent = () => {
            return Boolean(
              hasCurrentIdentity(queryClient, identity, scope) &&
              query &&
              queryClient.getQueryCache().find({ queryKey, exact: true }) === query &&
              hasCurrentMembership(queryClient, membershipKey, membershipQuery, scope) &&
              hasCurrentProjectRoles(queryClient, rolesKey, rolesQuery, scope)
            );
          };
          if (!isCurrent()) throw new Error("Analytics access changed before the request started.");
          const data = await load(signal);
          if (!isCurrent()) throw new Error("Analytics access changed while the request was loading.");
          return data;
        }
      : skipToken,
  });
}

function projectIdForPeek(scope: AnalyticsRequestScope, isPeekView: boolean) {
  return isPeekView ? scope.projectId : undefined;
}

export function analyticsInsightsOptions(
  scope: AnalyticsRequestScope,
  analyticsType: TAnalyticsTabsBase,
  params: TAnalyticsFilterParams,
  isPeekView: boolean
) {
  const projectId = projectIdForPeek(scope, isPeekView);
  const scopedParams = params;
  const serviceScope = { ...scope, projectId };
  const queryKey = analyticsKeys.insights(serviceScope, analyticsType, scopedParams, isPeekView);
  return createScopedRead<IAnalyticsResponse>(
    serviceScope,
    queryKey,
    isPeekView ? "project-member" : "workspace-member",
    (signal) =>
      analyticsService.getAdvanceAnalytics(scope.workspaceSlug!, analyticsType, scopedParams, isPeekView, signal)
  );
}

export function analyticsChartOptions<T>(
  scope: AnalyticsRequestScope,
  chartType: TAnalyticsGraphsBase,
  params: TAnalyticsFilterParams & Record<string, unknown>,
  isPeekView: boolean
) {
  const projectId = projectIdForPeek(scope, isPeekView);
  const serviceScope = { ...scope, projectId };
  const queryKey = analyticsKeys.chart(serviceScope, chartType, params, isPeekView);
  return createScopedRead<T>(serviceScope, queryKey, isPeekView ? "project-chart" : "workspace-member", (signal) =>
    analyticsService.getAdvanceAnalyticsCharts<T>(scope.workspaceSlug!, chartType, params, isPeekView, signal)
  );
}

export function analyticsStatsOptions(
  scope: AnalyticsRequestScope,
  params: TAnalyticsFilterParams,
  isPeekView: boolean
) {
  const projectId = projectIdForPeek(scope, isPeekView);
  const scopedParams = params;
  const serviceScope = { ...scope, projectId };
  const queryKey = analyticsKeys.stats(serviceScope, scopedParams, isPeekView);
  return createScopedRead<WorkItemInsightColumns[]>(
    serviceScope,
    queryKey,
    isPeekView ? "project-member" : "workspace-member",
    (signal) =>
      analyticsService.getAdvanceAnalyticsStats(scope.workspaceSlug!, "work-items", scopedParams, isPeekView, signal)
  );
}

export function projectAnalyticsCountsOptions(scope: AnalyticsRequestScope, params: TProjectAnalyticsCountParams) {
  const queryKey = analyticsKeys.projectCounts(scope, params);
  return createScopedRead<TProjectAnalyticsCount[]>(scope, queryKey, "workspace-active-member", (signal) =>
    projectService.getProjectAnalyticsCount(scope.workspaceSlug!, params, signal)
  );
}
