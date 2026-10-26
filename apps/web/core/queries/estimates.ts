/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import sortBy from "lodash-es/sortBy.js";
import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IEstimate, IEstimateFormData, IEstimatePoint, IUserProjectsRole, TUserPermissions } from "@plane/types";
import { EEstimateSystem, EUserWorkspaceRoles } from "@plane/types";
import {
  captureIdentityScope,
  currentUserOptions,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
} from "@/queries/identity";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { permissionKeys, workspaceMembershipOptions, workspaceProjectRolesOptions } from "@/queries/permissions";
import { ProjectService } from "@/services/project/project.service";
import { effectiveProjectRole, isWorkspaceRole } from "@/helpers/permission-policy";
import estimateService from "@/services/estimate.service";

export type TEstimateScope = {
  kind: "project" | "workspace";
  accountId: string | undefined;
  sessionGeneration: number;
  workspaceSlug: string | undefined;
  membershipRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectId?: string;
  projectMembershipRole?: TUserPermissions;
  projectRole?: TUserPermissions;
};

export type TEstimatePointProjection = {
  workspaceSlug: string;
  scopeKey: string;
  pointsByProject: Readonly<Record<string, Readonly<Record<string, IEstimatePoint>>>> | undefined;
  isCurrent: () => boolean;
  isReadCurrent: () => boolean;
};

export const estimateKeys = {
  workspacePrefix: (scope: Pick<TEstimateScope, "accountId" | "sessionGeneration" | "workspaceSlug">) =>
    ["estimates", scope.accountId, scope.sessionGeneration, scope.workspaceSlug] as const,
  list: (scope: TEstimateScope) =>
    [
      ...estimateKeys.workspacePrefix(scope),
      scope.membershipRole,
      scope.membershipRevision,
      scope.kind,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
    ] as const,
};

function normalizedEstimate(estimate: IEstimate): IEstimate {
  const raw = estimate.type?.toLowerCase();
  const type =
    raw === EEstimateSystem.POINTS || raw === EEstimateSystem.TIME || raw === EEstimateSystem.CATEGORIES
      ? raw
      : undefined;
  return { ...estimate, type };
}

export function orderedEstimatePoints(estimate: IEstimate | undefined): readonly IEstimatePoint[] | undefined {
  if (!estimate?.points) return undefined;
  return sortBy(
    estimate.points.filter((point) => point.estimate === estimate.id),
    (point) => point.key ?? 0
  );
}

export function activeProjectEstimate(
  estimates: readonly IEstimate[] | undefined,
  projectId: string | undefined,
  selectedId?: string | null
) {
  if (!projectId) return undefined;
  return estimates?.find(
    (estimate) => estimate.project === projectId && (selectedId ? estimate.id === selectedId : estimate.last_used)
  );
}

function scopeReady(scope: TEstimateScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.membershipRole !== undefined &&
    scope.membershipRevision &&
    (scope.kind === "workspace" || Boolean(scope.projectId && scope.projectMembershipRole !== undefined))
  );
}

/** Capture actual permission objects and query ownership at the operation boundary. */
export function captureEstimateScope(client: QueryClient, scope: TEstimateScope) {
  const identity = captureIdentityScope(client, scope.accountId);
  const queryKey = estimateKeys.list(scope);
  const query = client.getQueryCache().find({ queryKey, exact: true });
  const access = captureProjectAccessScope(client, {
    ...scope,
    projectId: scope.projectId,
    projectRole: scope.projectRole,
  });
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  const isLineageCurrent = () =>
    Boolean(
      identity?.generation === scope.sessionGeneration &&
      isIdentityScopeCurrent(client, identity) &&
      query &&
      client.getQueryCache().find({ queryKey, exact: true }) === query
    );
  const isCurrent = () =>
    scopeReady(scope) &&
    isLineageCurrent() &&
    access.isCurrent() &&
    (scope.kind === "workspace" ||
      client.getQueryData<IUserProjectsRole>(rolesKey)?.[scope.projectId ?? ""] === scope.projectMembershipRole);
  const membershipQuery = client
    .getQueryCache()
    .find({ queryKey: permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug), exact: true });
  const rolesQuery =
    scope.kind === "project" ? client.getQueryCache().find({ queryKey: rolesKey, exact: true }) : undefined;
  return {
    isLineageCurrent,
    isCurrent,
    isVerified: () => isCurrent() && access.isVerified(),
    query,
    sources: [identity?.query, query, membershipQuery, rolesQuery] as const,
  };
}

export function estimateListOptions(scope: TEstimateScope) {
  return queryOptions({
    queryKey: estimateKeys.list(scope),
    queryFn: scopeReady(scope)
      ? async ({ client, signal }) => {
          const captured = captureEstimateScope(client, scope);
          if (!captured.isVerified()) throw new Error("Estimate access changed.");
          const estimates =
            scope.kind === "project"
              ? await estimateService.fetchProjectEstimates(scope.workspaceSlug!, scope.projectId!, signal)
              : await estimateService.fetchWorkspaceEstimates(scope.workspaceSlug!, signal);
          if (!captured.isVerified()) throw new Error("Estimate access changed.");
          return estimates.map(normalizedEstimate);
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

const projectService = new ProjectService();

export type TEstimateWrite =
  | { type: "create"; data: IEstimateFormData }
  | { type: "delete"; estimateId: string }
  | { type: "select"; estimateId: string | null; expectedId: string | null };

/** Callbacks reconcile the separately-owned project pointer, never estimate records. */
export function estimateMutationOptions(client: QueryClient, scope: TEstimateScope) {
  return mutationOptions({
    mutationKey: [...estimateKeys.workspacePrefix(scope), scope.projectId, "write"],
    scope: { id: `estimates/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}` },
    retry: false,
    mutationFn: async (write: TEstimateWrite) => {
      const captured = captureEstimateScope(client, scope);
      if (!scope.projectId || ![15, 20].includes(scope.projectRole ?? 0) || !captured.isVerified())
        throw new Error("Project member access is required to change estimates.");
      if (write.type === "select") {
        const project = await projectService.updateProject(scope.workspaceSlug!, scope.projectId, {
          estimate: write.estimateId,
        });
        if (project.id !== scope.projectId || (project.estimate ?? null) !== write.estimateId)
          throw new Error("The server returned a different project estimate selection.");
        return { write, estimate: undefined, project, captured };
      }
      if (write.type === "create") {
        const estimate = await estimateService.createEstimate(scope.workspaceSlug!, scope.projectId, write.data);
        if (
          typeof estimate.id !== "string" ||
          !estimate.id ||
          estimate.project !== scope.projectId ||
          estimate.points?.some(
            (point) => !point.id || point.project !== scope.projectId || point.estimate !== estimate.id
          )
        )
          throw new Error("The server returned an estimate outside the requested project.");
        return { write, estimate: normalizedEstimate(estimate), project: undefined, captured };
      }
      await estimateService.deleteEstimate(scope.workspaceSlug!, scope.projectId, write.estimateId);
      return { write, estimate: undefined, project: undefined, captured };
    },
    onSuccess: async ({ write, estimate, captured }) => {
      const queryKey = estimateKeys.list(scope);
      if (!captured.isCurrent()) {
        if (captured.isLineageCurrent()) await client.invalidateQueries({ queryKey, exact: true, refetchType: "none" });
        return;
      }
      await client.cancelQueries({ queryKey, exact: true });
      if (!captured.isCurrent()) return;
      const rows = client.getQueryData<IEstimate[]>(queryKey);
      if (rows && write.type !== "select")
        client.setQueryData(
          queryKey,
          write.type === "delete" ? rows.filter((row) => row.id !== write.estimateId) : [...rows, estimate!]
        );
      // Cache maintenance must not turn an accepted durable write into failure.
      const prefix = estimateKeys.workspacePrefix(scope);
      await client.invalidateQueries(
        {
          predicate: (query) =>
            prefix.every((value, index) => query.queryKey[index] === value) &&
            (query.queryKey[6] === "workspace" || (rows === undefined && query === captured.query)),
        },
        { throwOnError: false }
      );
    },
  });
}

/** Recover prerequisites, then build options for their new permission identity. */
export async function retryEstimateList(client: QueryClient, scope: TEstimateScope, isMountedScope: () => boolean) {
  const account = await client.fetchQuery({ ...currentUserOptions(), staleTime: 0 });
  if (!account?.id || !scope.workspaceSlug || !canRecoverEstimateAccount(scope, account.id, isMountedScope)) return;
  const membership = await client.fetchQuery({
    ...workspaceMembershipOptions(account.id, scope.workspaceSlug),
    staleTime: 0,
  });
  if (
    !isMountedScope() ||
    !membership.is_active ||
    !isWorkspaceRole(membership.role) ||
    membership.updated_at === undefined
  )
    return;
  const next: TEstimateScope = {
    ...scope,
    accountId: account.id,
    sessionGeneration: getIdentitySessionGeneration(client),
    membershipRole: membership.role,
    membershipRevision: membership.updated_at,
  };
  if (scope.kind === "project") {
    const roles = await client.fetchQuery({
      ...workspaceProjectRolesOptions(account.id, scope.workspaceSlug, membership.role, membership.updated_at),
      staleTime: 0,
    });
    next.projectMembershipRole = roles?.[scope.projectId ?? ""];
    next.projectRole = effectiveProjectRole(membership.role, roles, scope.projectId) ?? undefined;
  }
  if (!isMountedScope() || !scopeReady(next)) return;
  await client.fetchQuery({ ...estimateListOptions(next), staleTime: 0 });
}

function canRecoverEstimateAccount(scope: TEstimateScope, accountId: string, isMountedScope: () => boolean) {
  return isMountedScope() && (!scope.accountId || scope.accountId === accountId);
}
