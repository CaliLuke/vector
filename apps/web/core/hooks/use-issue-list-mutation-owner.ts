/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import type { EIssuesStoreType, TIssueServiceType } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType as IssuesStoreType } from "@plane/types";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useOptionalParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { useOptionalProjectViewIssuePreferences } from "@/providers/project-view-issue-preferences";
import { useIssueFilterPreferences } from "@/hooks/use-issue-filter-preferences";
import type { IssueListMutationOwner } from "@/queries/issue-list-mutations";
import { currentIssueListMutationOwner } from "@/queries/issue-list-mutations";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { getIdentitySessionGeneration } from "@/queries/identity";
import type {
  TArchivedIssueRequestInput,
  TProfileIssueRequestInput,
  TProjectIssueRequestInput,
} from "@/helpers/issue-list/request";
import type { TProjectViewIssueRequestInput } from "@/helpers/issue-list/project-view-request";

type IssueRequestInput =
  | TProjectIssueRequestInput
  | TProjectViewIssueRequestInput
  | TProfileIssueRequestInput
  | TArchivedIssueRequestInput;

export function useIssueListMutationOwner(
  storeType: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>,
  suppliedInput?: IssueRequestInput
) {
  const client = useQueryClient();
  const {
    workspaceSlug: routeWorkspace,
    projectId: routeProject,
    userId: routeProfileUserId,
    cycleId,
    moduleId,
    viewId,
  } = useParams({
    strict: false,
  });
  const accountId = useCurrentUserId();
  useIdentityGeneration(client);
  const projectPermissions = useWorkspaceProjectRoles(routeWorkspace);
  const parent = useOptionalParentIssuePreferences();
  const projectView = useOptionalProjectViewIssuePreferences();
  const preferences = useIssueFilterPreferences(storeType);
  const routeScope = useCommittedScope(
    JSON.stringify([storeType, routeWorkspace, routeProject, routeProfileUserId, cycleId, moduleId, viewId])
  );

  return useCallback(
    async (targetWorkspace: string, targetProject: string): Promise<IssueListMutationOwner | undefined> => {
      if (routeWorkspace !== targetWorkspace) return undefined;
      const resolved = resolveMutationScope({
        client,
        storeType,
        targetWorkspace,
        targetProject,
        routeProject,
        routeProfileUserId,
        cycleId,
        moduleId,
        viewId,
        accountId,
        projectPermissions,
        parent,
        projectView,
        requestInput: preferences.requestInput,
        suppliedInput,
      });
      const serviceType: TIssueServiceType =
        storeType === IssuesStoreType.EPIC ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES;
      return currentIssueListMutationOwner(
        client,
        targetWorkspace,
        targetProject,
        serviceType,
        () => routeScope() && routeWorkspace === targetWorkspace && resolved.isScopeCurrent(),
        resolved.parentStatistics
      );
    },
    [
      client,
      cycleId,
      moduleId,
      parent,
      projectView,
      routeProject,
      routeScope,
      routeWorkspace,
      storeType,
      suppliedInput,
      accountId,
      routeProfileUserId,
      viewId,
      projectPermissions,
      preferences.requestInput,
    ]
  );
}

type MutationScope = Pick<IssueListMutationOwner, "isScopeCurrent" | "parentStatistics">;

type MutationScopeInput = {
  client: QueryClient;
  storeType: Exclude<EIssuesStoreType, EIssuesStoreType.WORKSPACE_DRAFT>;
  targetWorkspace: string;
  targetProject: string;
  routeProject: string | undefined;
  routeProfileUserId: string | undefined;
  cycleId: string | undefined;
  moduleId: string | undefined;
  viewId: string | undefined;
  accountId: string | undefined;
  projectPermissions: ReturnType<typeof useWorkspaceProjectRoles>;
  parent: ReturnType<typeof useOptionalParentIssuePreferences>;
  projectView: ReturnType<typeof useOptionalProjectViewIssuePreferences>;
  requestInput: ReturnType<typeof useIssueFilterPreferences>["requestInput"];
  suppliedInput?: IssueRequestInput;
};

function resolveMutationScope(input: MutationScopeInput): MutationScope {
  switch (input.storeType) {
    case IssuesStoreType.CYCLE:
      return resolveParentMutationScope(input, "cycle");
    case IssuesStoreType.MODULE:
      return resolveParentMutationScope(input, "module");
    case IssuesStoreType.PROJECT_VIEW:
      return resolveProjectViewMutationScope(input);
    case IssuesStoreType.PROFILE:
      return resolveProfileMutationScope(input);
    case IssuesStoreType.ARCHIVED:
      return resolveArchivedMutationScope(input);
    default:
      return { isScopeCurrent: resolveProjectMutationScope(input) };
  }
}

function resolveParentMutationScope(input: MutationScopeInput, domain: "cycle" | "module"): MutationScope {
  const expectedId = domain === "cycle" ? input.cycleId : input.moduleId;
  const preference = input.parent?.requestInput;
  if (
    input.routeProject !== input.targetProject ||
    !expectedId ||
    !preference ||
    preference.kind !== domain ||
    preference.workspaceSlug !== input.targetWorkspace ||
    preference.projectId !== input.targetProject ||
    preference.viewId !== expectedId ||
    !preference.isScopeCurrent()
  )
    throw new Error("Parent issue preferences are changing. Retry the operation.");
  return { isScopeCurrent: preference.isScopeCurrent, parentStatistics: preference.parentStatistics };
}

function resolveProjectViewMutationScope(input: MutationScopeInput): MutationScope {
  const preference =
    input.suppliedInput && "viewId" in input.suppliedInput ? input.suppliedInput : input.projectView?.requestInput;
  if (
    !preference ||
    preference.workspaceSlug !== input.targetWorkspace ||
    preference.projectId !== input.targetProject ||
    preference.viewId !== input.viewId ||
    !preference.isScopeCurrent()
  )
    throw new Error("The project view issue list is changing. Retry the operation.");
  return { isScopeCurrent: preference.isScopeCurrent };
}

function resolveProfileMutationScope(input: MutationScopeInput): MutationScope {
  const preference = profileInput(input);
  if (
    !preference ||
    preference.workspaceSlug !== input.targetWorkspace ||
    preference.userId !== input.routeProfileUserId ||
    !preference.isCurrent()
  )
    throw new Error("Profile issue preferences are changing. Retry the operation.");
  return { isScopeCurrent: preference.isCurrent };
}

function resolveArchivedMutationScope(input: MutationScopeInput): MutationScope {
  const preference = archivedInput(input);
  if (
    !preference ||
    preference.workspaceSlug !== input.targetWorkspace ||
    preference.projectId !== input.targetProject ||
    !preference.isCurrent()
  )
    throw new Error("Archived issue preferences are changing. Retry the operation.");
  return { isScopeCurrent: preference.isCurrent };
}

function profileInput(input: MutationScopeInput): TProfileIssueRequestInput | undefined {
  if (input.suppliedInput && "kind" in input.suppliedInput && input.suppliedInput.kind === "profile")
    return input.suppliedInput;
  return input.requestInput && "kind" in input.requestInput && input.requestInput.kind === "profile"
    ? input.requestInput
    : undefined;
}

function archivedInput(input: MutationScopeInput): TArchivedIssueRequestInput | undefined {
  if (input.suppliedInput && "kind" in input.suppliedInput && input.suppliedInput.kind === "archived")
    return input.suppliedInput;
  return input.requestInput && "kind" in input.requestInput && input.requestInput.kind === "archived"
    ? input.requestInput
    : undefined;
}

function resolveProjectMutationScope(input: MutationScopeInput): () => boolean {
  const { projectPermissions, accountId, client, targetWorkspace, targetProject } = input;
  const membershipQuery = projectPermissions.membershipQuery;
  const rolesQuery = projectPermissions.rolesQuery;
  const membership = membershipQuery.data;
  const roles = rolesQuery.data;
  const rawProjectRole = roles?.[targetProject];
  const projectRole = membership?.is_active
    ? effectiveProjectRole(projectPermissions.workspaceRole ?? null, roles, targetProject)
    : undefined;
  if (
    !accountId ||
    !membershipQuery.isSuccess ||
    !membership?.is_active ||
    !projectPermissions.workspaceRole ||
    !membership.updated_at ||
    !rolesQuery.isSuccess ||
    rawProjectRole === undefined ||
    projectRole === undefined
  )
    throw new Error("Project issue access is not ready. Retry the operation.");

  const access = captureProjectAccessScope(client, {
    accountId,
    workspaceSlug: targetWorkspace,
    projectId: targetProject,
    membershipRole: projectPermissions.workspaceRole,
    membershipRevision: membership.updated_at,
    projectMembershipRole: rawProjectRole,
    projectRole,
    sessionGeneration: getIdentitySessionGeneration(client),
  });
  if (!access.isVerified()) throw new Error("Project issue access is not ready. Retry the operation.");
  return access.isCurrent;
}
