/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { QueryClient } from "@tanstack/react-query";
import sortBy from "lodash-es/sortBy.js";
import type { IUserProjectsRole, IWorkspaceMemberMe } from "@plane/types";
import { captureProjectAccessScope } from "./project-access-scope";
import { getIdentitySessionGeneration, getCurrentUserId } from "./identity";
import { effectiveProjectRole, isWorkspaceRole } from "@/helpers/permission-policy";
import { permissionKeys } from "./permissions";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "./catalogue-scope";
import { workspaceCatalogueScopeReady } from "./catalogue-scope";

export type ParentCatalogueScope = ProjectCatalogueScope | WorkspaceCatalogueScope;
export function parentScopeKey(scope: ParentCatalogueScope) {
  return [
    scope.accountId,
    scope.sessionGeneration,
    scope.workspaceSlug,
    scope.workspaceRole,
    scope.membershipRevision,
    "projectId" in scope ? [scope.projectId, scope.projectRole, scope.projectMembershipRole] : scope.projectRoles,
  ] as const;
}
export function parentScopeReady(scope: ParentCatalogueScope) {
  if (!("projectId" in scope)) return workspaceCatalogueScopeReady(scope);
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    scope.membershipRevision &&
    scope.workspaceRole !== undefined &&
    scope.projectMembershipRole !== undefined &&
    scope.projectRole !== undefined &&
    [5, 15, 20].includes(scope.projectRole)
  );
}
/** Capture permissions and query instances independently of the consumer's render lifetime. */
export function captureParentAccess(client: QueryClient, scope: ParentCatalogueScope) {
  const project = "projectId" in scope ? scope : undefined;
  const access = captureProjectAccessScope(client, {
    accountId: scope.accountId,
    sessionGeneration: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    membershipRole: scope.workspaceRole,
    membershipRevision: scope.membershipRevision,
    projectId: project?.projectId,
    projectRole: project?.projectRole,
    projectMembershipRole: project?.projectMembershipRole,
    rolesRevision: "projectRoles" in scope && scope.projectRoles ? JSON.stringify(scope.projectRoles) : undefined,
  });
  return {
    isVerified: () => parentScopeReady(scope) && access.isVerified(),
    isCurrent: () => parentScopeReady(scope) && access.isCurrent(),
  };
}

export function currentParentCatalogueScope(
  client: QueryClient,
  workspaceSlug: string | undefined,
  projectId: string | undefined
) {
  const accountId = getCurrentUserId(client);
  const membership = client.getQueryData<IWorkspaceMemberMe>(
    permissionKeys.workspaceMembership(accountId, workspaceSlug)
  );
  const workspaceRole = isWorkspaceRole(membership?.role) ? membership.role : undefined;
  const roles = client.getQueryData<IUserProjectsRole>(
    permissionKeys.workspaceProjectRoles(accountId, workspaceSlug, workspaceRole, membership?.updated_at)
  );
  const base = {
    accountId,
    sessionGeneration: getIdentitySessionGeneration(client),
    workspaceSlug,
    workspaceRole,
    membershipRevision: membership?.updated_at,
  };
  const project: ProjectCatalogueScope = {
    ...base,
    projectId,
    projectRole: effectiveProjectRole(workspaceRole ?? null, roles, projectId) ?? undefined,
    projectMembershipRole: roles?.[projectId ?? ""],
  };
  const workspace: WorkspaceCatalogueScope = {
    ...base,
    projectRoles: roles ? sortBy(Object.entries(roles), ([id]) => id) : undefined,
  };
  return { project, workspace };
}
