/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ANALYTICS_DURATION_FILTER_OPTIONS, EUserPermissions } from "@plane/constants";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IWorkspaceMemberMe, TAnalyticsFilterParams, TUserPermissions } from "@plane/types";
import { effectiveProjectRole, isWorkspaceRole } from "@/helpers/permission-policy";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { permissionKeys } from "@/queries/permissions";
import { getProjectRolesRevision } from "@/queries/analytics";
import type { AnalyticsRequestScope } from "@/queries/analytics";

type DurationType = (typeof ANALYTICS_DURATION_FILTER_OPTIONS)[number]["value"];
const DEFAULT_DURATION: DurationType = "last_30_days";

export type AnalyticsScope = AnalyticsRequestScope & {
  projectId: string | undefined;
  projectRole: TUserPermissions | undefined;
  isActive: boolean;
  isReady: boolean;
  isError: boolean;
  hasWorkspaceAnalyticsAccess: boolean;
  hasProjectAnalyticsAccess: boolean;
  hasProjectChartAccess: boolean;
  retryAccess: () => Promise<void>;
};

type AnalyticsState = {
  selectedProjects: string[];
  selectedCycle: string;
  selectedModule: string;
  selectedDurationLabel: string | null;
  requestFilters: TAnalyticsFilterParams;
  isPeekView: boolean;
  isEpic: boolean;
  scope: AnalyticsScope;
  updateSelectedProjects: (projects: string[]) => void;
  updateSelectedCycle: (cycle: string) => void;
  updateSelectedModule: (module: string) => void;
};

const AnalyticsStateContext = createContext<AnalyticsState | undefined>(undefined);

type Props = {
  children: React.ReactNode;
  workspaceSlug: string | undefined;
  projectId?: string;
  cycleId?: string;
  moduleId?: string;
  isPeekView?: boolean;
  isEpic?: boolean;
  isActive?: boolean;
  instanceKey?: string;
};

function getAnalyticsScopeStatus({
  workspaceSlug,
  currentUserLoaded,
  currentUserError,
  membershipLoaded,
  membershipError,
  membershipIsReady,
  isActive,
  projectRolesLoaded,
  projectRolesError,
  projectRolesRevision,
}: {
  workspaceSlug: string | undefined;
  currentUserLoaded: boolean;
  currentUserError: boolean;
  membershipLoaded: boolean;
  membershipError: boolean;
  membershipIsReady: boolean;
  isActive: boolean;
  projectRolesLoaded: boolean;
  projectRolesError: boolean;
  projectRolesRevision: string | undefined;
}) {
  const membershipResolved = currentUserLoaded && (!workspaceSlug || membershipLoaded);
  const requiresProjectRoles = isActive && membershipIsReady;
  return {
    isReady: Boolean(membershipResolved && (!requiresProjectRoles || (projectRolesLoaded && projectRolesRevision))),
    isError: currentUserError || membershipError || (requiresProjectRoles && projectRolesError),
    requiresProjectRoles,
  };
}

function getAnalyticsAccessFlags(
  isReady: boolean,
  membershipIsReady: boolean,
  workspaceRole: EUserWorkspaceRoles | undefined,
  projectRole: TUserPermissions | undefined
) {
  const hasWorkspaceRole = workspaceRole === EUserWorkspaceRoles.ADMIN || workspaceRole === EUserWorkspaceRoles.MEMBER;
  const hasProjectRole = projectRole === EUserPermissions.ADMIN || projectRole === EUserPermissions.MEMBER;
  const hasProjectChartRole = hasProjectRole || projectRole === EUserPermissions.GUEST;
  return {
    hasWorkspaceAnalyticsAccess: membershipIsReady && isReady && hasWorkspaceRole,
    hasProjectAnalyticsAccess: isReady && hasProjectRole,
    hasProjectChartAccess: isReady && hasProjectChartRole,
  };
}

function hasActiveWorkspaceMembership(
  accountId: string | undefined,
  workspaceSlug: string | undefined,
  membershipLoaded: boolean,
  membership: IWorkspaceMemberMe | undefined
) {
  return Boolean(
    accountId &&
    workspaceSlug &&
    membershipLoaded &&
    membership?.is_active === true &&
    isWorkspaceRole(membership.role) &&
    membership.updated_at
  );
}

function getProjectRoleForScope(
  projectId: string | undefined,
  workspaceRole: EUserWorkspaceRoles | undefined,
  projectRoles: Record<string, TUserPermissions> | undefined
) {
  if (!projectId || !projectRoles) return undefined;
  return effectiveProjectRole(workspaceRole ?? null, projectRoles, projectId);
}

function createAnalyticsScopeKey({
  accountId,
  sessionGeneration,
  workspaceSlug,
  membership,
  projectRolesRevision,
  projectId,
  projectRole,
  isActive,
  instanceKey,
}: {
  accountId: string | undefined;
  sessionGeneration: number;
  workspaceSlug: string | undefined;
  membership: IWorkspaceMemberMe | undefined;
  projectRolesRevision: string | undefined;
  projectId: string | undefined;
  projectRole: TUserPermissions | undefined;
  isActive: boolean;
  instanceKey: string;
}) {
  return [
    accountId ?? "pending-account",
    sessionGeneration,
    workspaceSlug ?? "pending-workspace",
    membership?.role ?? "pending-role",
    membership?.updated_at ?? "pending-membership",
    projectRolesRevision ?? "pending-project-roles",
    projectId ?? "",
    projectRole ?? "pending-project-role",
    isActive,
    instanceKey,
  ].join(":");
}

async function retryAnalyticsPermissionQueries(
  currentUserQuery: { isError: boolean; refetch: () => Promise<unknown> },
  membershipQuery: { isError: boolean; refetch: () => Promise<unknown> },
  projectRolesQuery: { isError: boolean; refetch: () => Promise<unknown> },
  requiresProjectRoles: boolean
) {
  if (currentUserQuery.isError) await currentUserQuery.refetch();
  if (membershipQuery.isError) await membershipQuery.refetch();
  if (requiresProjectRoles && projectRolesQuery.isError) await projectRolesQuery.refetch();
}

export function AnalyticsStateProvider({
  children,
  workspaceSlug,
  projectId,
  cycleId,
  moduleId,
  isPeekView = false,
  isEpic = false,
  isActive = true,
  instanceKey = "dashboard",
}: Props) {
  const client = useQueryClient();
  const currentUserQuery = useCurrentUser();
  const accountId = currentUserQuery.data?.id;
  const sessionGeneration = useIdentityGeneration(client);
  const permissions = useWorkspacePermissions(workspaceSlug);
  const membershipQuery = permissions.membershipQuery;
  const membership = membershipQuery.isSuccess ? membershipQuery.data : undefined;
  const projectPermissions = useWorkspaceProjectRoles(isActive ? workspaceSlug : undefined);
  const projectRoles = projectPermissions.roles;
  const projectRole = getProjectRoleForScope(projectId, permissions.workspaceRole, projectRoles);
  const projectRolesQueryKey = permissionKeys.workspaceProjectRoles(
    accountId,
    workspaceSlug,
    permissions.workspaceRole,
    membership?.updated_at
  );
  const projectRolesQuery = client.getQueryCache().find({ queryKey: projectRolesQueryKey, exact: true });
  const projectRolesRevision = getProjectRolesRevision(projectRoles);
  const membershipIsReady = hasActiveWorkspaceMembership(
    accountId,
    workspaceSlug,
    membershipQuery.isSuccess,
    membership
  );
  const scopeStatus = getAnalyticsScopeStatus({
    workspaceSlug,
    currentUserLoaded: currentUserQuery.isSuccess,
    currentUserError: currentUserQuery.isError,
    membershipLoaded: membershipQuery.isSuccess,
    membershipError: membershipQuery.isError,
    membershipIsReady,
    isActive,
    projectRolesLoaded: projectPermissions.rolesQuery.isSuccess && projectRolesQuery?.state.status === "success",
    projectRolesError: projectPermissions.rolesQuery.isError,
    projectRolesRevision,
  });
  const { isReady, isError } = scopeStatus;
  const accessFlags = useMemo(
    () => getAnalyticsAccessFlags(isReady, membershipIsReady, permissions.workspaceRole, projectRole),
    [isReady, membershipIsReady, permissions.workspaceRole, projectRole]
  );
  const scope = useMemo<AnalyticsScope>(
    () => ({
      accountId,
      workspaceSlug,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRolesRevision,
      sessionGeneration,
      projectId,
      projectRole,
      isActive,
      isReady,
      isError,
      ...accessFlags,
      retryAccess: () =>
        retryAnalyticsPermissionQueries(
          currentUserQuery,
          membershipQuery,
          projectPermissions.rolesQuery,
          scopeStatus.requiresProjectRoles
        ),
    }),
    [
      accountId,
      currentUserQuery,
      isActive,
      isError,
      isReady,
      membership,
      membershipQuery,
      permissions.workspaceRole,
      projectRolesRevision,
      accessFlags,
      projectId,
      projectPermissions.rolesQuery,
      projectRole,
      sessionGeneration,
      scopeStatus.requiresProjectRoles,
      workspaceSlug,
    ]
  );
  const scopeKey = createAnalyticsScopeKey({
    accountId,
    sessionGeneration,
    workspaceSlug,
    membership,
    projectRolesRevision,
    projectId,
    projectRole,
    isActive,
    instanceKey,
  });

  return (
    <AnalyticsStateRoot
      key={scopeKey}
      scope={scope}
      projectId={projectId}
      cycleId={cycleId}
      moduleId={moduleId}
      isPeekView={isPeekView}
      isEpic={isEpic}
    >
      {children}
    </AnalyticsStateRoot>
  );
}

function AnalyticsStateRoot({
  children,
  scope,
  projectId,
  cycleId,
  moduleId,
  isPeekView,
  isEpic,
}: Pick<Props, "children" | "projectId" | "cycleId" | "moduleId" | "isPeekView" | "isEpic"> & {
  scope: AnalyticsScope;
}) {
  const [selectedProjects, setSelectedProjects] = useState<string[]>(() => (projectId ? [projectId] : []));
  const [selectedCycle, setSelectedCycle] = useState(cycleId ?? "");
  const [selectedModule, setSelectedModule] = useState(moduleId ?? "");
  const updateSelectedProjects = useCallback((projects: string[]) => setSelectedProjects(projects), []);
  const updateSelectedCycle = useCallback((cycle: string) => setSelectedCycle(cycle), []);
  const updateSelectedModule = useCallback((module: string) => setSelectedModule(module), []);
  const selectedDurationLabel =
    ANALYTICS_DURATION_FILTER_OPTIONS.find((item) => item.value === DEFAULT_DURATION)?.name ?? null;
  const requestFilters = useMemo<TAnalyticsFilterParams>(
    () => ({
      ...(selectedProjects.length > 0 ? { project_ids: selectedProjects.join(",") } : {}),
      ...(selectedCycle ? { cycle_id: selectedCycle } : {}),
      ...(selectedModule ? { module_id: selectedModule } : {}),
      ...(isEpic ? { epic: true } : {}),
    }),
    [isEpic, selectedCycle, selectedModule, selectedProjects]
  );
  const value = useMemo<AnalyticsState>(
    () => ({
      selectedProjects,
      selectedCycle,
      selectedModule,
      selectedDurationLabel,
      requestFilters,
      isPeekView: isPeekView ?? false,
      isEpic: isPeekView ? (isEpic ?? false) : false,
      scope,
      updateSelectedProjects,
      updateSelectedCycle,
      updateSelectedModule,
    }),
    [
      isEpic,
      isPeekView,
      scope,
      selectedCycle,
      selectedDurationLabel,
      selectedModule,
      selectedProjects,
      requestFilters,
      updateSelectedCycle,
      updateSelectedModule,
      updateSelectedProjects,
    ]
  );

  if (scope.isActive && scope.isError)
    return (
      <div className="flex h-full items-center justify-center gap-2" role="alert">
        <span>Analytics permissions could not be loaded.</span>
        <button className="underline" onClick={() => void scope.retryAccess()} type="button">
          Retry
        </button>
      </div>
    );
  if (scope.isActive && !scope.isReady)
    return <output className="flex h-full items-center justify-center">Loading analytics access...</output>;
  if (scope.isActive && isPeekView && !projectId)
    return <output className="flex h-full items-center justify-center">Loading project analytics...</output>;

  return <AnalyticsStateContext.Provider value={value}>{children}</AnalyticsStateContext.Provider>;
}

export function useAnalyticsState() {
  const context = useContext(AnalyticsStateContext);
  if (!context) throw new Error("useAnalyticsState must be used within AnalyticsStateProvider");
  return context;
}
