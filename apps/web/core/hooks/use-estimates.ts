/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import sortBy from "lodash-es/sortBy.js";
import { useCallback, useMemo } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useWorkspacePermissions, useProjectPermissions } from "@/hooks/use-permissions";
import { useProjectDetail } from "@/hooks/use-projects";
import { captureProjectCatalogueScope, projectKeys } from "@/queries/projects";
import type { TPartialProject, TProject } from "@plane/types";
import {
  activeProjectEstimate,
  estimateListOptions,
  estimateMutationOptions,
  orderedEstimatePoints,
  retryEstimateList,
} from "@/queries/estimates";
import type { TEstimateScope } from "@/queries/estimates";

type ReadStatus = { isPending: boolean; isError: boolean };
function estimateReadState(denied: boolean, ...queries: ReadStatus[]) {
  const isError = queries.some((query) => query.isError);
  return { isError, isPending: !denied && !isError && queries.some((query) => query.isPending) };
}
function estimateScopeDenied(scope: TEstimateScope, active: boolean | undefined, ...queries: ReadStatus[]) {
  const resolved = queries.every((query) => !query.isPending && !query.isError);
  const projectReadable =
    scope.kind !== "project" || Boolean(scope.projectId && scope.projectMembershipRole !== undefined);
  return resolved && !(scope.accountId && scope.workspaceSlug && active && projectReadable);
}

function projectEstimateAccessDenied(
  projectId: string | null | undefined,
  scopeDenied: boolean,
  projectAccessDenied: boolean
) {
  return scopeDenied || projectAccessDenied || !projectId;
}

async function retryProjectEstimateReads(
  projectReadFailed: boolean,
  retryProjectRead: () => Promise<unknown>,
  retryEstimates: () => Promise<{ error?: unknown }>
) {
  if (projectReadFailed) {
    await retryProjectRead();
    return;
  }
  await retryEstimates();
}

function useEstimateScope(kind: TEstimateScope["kind"], workspaceSlug: string | undefined, projectId?: string) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const sessionGeneration = useIdentityGeneration(client);
  const projectAccess = useProjectPermissions(kind === "project" && projectId ? workspaceSlug : undefined, projectId);
  const workspaceAccess = useWorkspacePermissions(kind === "workspace" ? workspaceSlug : undefined);
  const permissions = kind === "project" ? projectAccess : workspaceAccess;
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const projectMembershipRole =
    projectId && projectAccess.rolesQuery.isSuccess ? projectAccess.rolesQuery.data?.[projectId] : undefined;
  const scope = useMemo<TEstimateScope>(
    () => ({
      kind,
      accountId: account.data?.id,
      sessionGeneration,
      workspaceSlug,
      membershipRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectId,
      projectMembershipRole,
      projectRole: projectAccess.projectRole,
    }),
    [
      kind,
      account.data?.id,
      sessionGeneration,
      workspaceSlug,
      permissions.workspaceRole,
      membership?.updated_at,
      projectId,
      projectMembershipRole,
      projectAccess.projectRole,
    ]
  );
  const isAccessDenied = estimateScopeDenied(scope, membership?.is_active, account, permissions);
  return { scope, account, permissions, isAccessDenied };
}

export function useEstimateEditorKey(workspaceSlug: string | undefined, projectId: string | undefined) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const generation = useIdentityGeneration(client);
  return JSON.stringify([account.data?.id, generation, workspaceSlug, projectId]);
}

function useEstimateRetry(scope: TEstimateScope) {
  const client = useQueryClient();
  const key = JSON.stringify([
    scope.kind,
    scope.accountId,
    scope.sessionGeneration,
    scope.workspaceSlug,
    scope.projectId,
  ]);
  const isCurrent = useCommittedScope(key);
  return async () => {
    try {
      await retryEstimateList(client, scope, isCurrent);
      return { error: undefined };
    } catch (error) {
      // The corresponding native query retains the error for the visible Retry state.
      return { error };
    }
  };
}

export function useWorkspaceEstimates(workspaceSlug: string | undefined) {
  const access = useEstimateScope("workspace", workspaceSlug);
  const query = useRecordCollection(estimateListOptions(access.scope));
  const retry = useEstimateRetry(access.scope);
  const { isError, isPending } = estimateReadState(access.isAccessDenied, access.account, access.permissions, query);
  return {
    ...query,
    retry,
    scope: access.scope,
    estimates: !isError && !access.isAccessDenied && query.isSuccess ? query.data : undefined,
    isError,
    isPending,
    isAccessDenied: access.isAccessDenied,
  };
}

export function useProjectEstimates(workspaceSlug: string | undefined, projectId: string | null | undefined) {
  const client = useQueryClient();
  const access = useEstimateScope("project", workspaceSlug, projectId ?? undefined);
  const query = useRecordCollection(estimateListOptions(access.scope));
  const retryEstimates = useEstimateRetry(access.scope);
  const projectQuery = useProjectDetail(workspaceSlug, projectId ?? undefined);
  const project = projectQuery.project;
  const selectedId = project?.estimate ?? null;
  const isAccessDenied = projectEstimateAccessDenied(projectId, access.isAccessDenied, projectQuery.accessDenied);
  const { isError, isPending } = estimateReadState(
    isAccessDenied,
    access.account,
    access.permissions,
    query,
    ...(projectId ? [projectQuery] : [])
  );
  const projectKey = projectKeys.detail(projectQuery.scope);
  const projectQueryOwner = client.getQueryCache().find({ queryKey: projectKey, exact: true });
  const projectQueryData = projectQueryOwner?.state.data;
  const projectAccessOwner = useMemo(
    () => captureProjectCatalogueScope(client, projectQuery.scope),
    [client, projectQuery.scope]
  );
  const isSelectionCurrent = useCallback(() => {
    return Boolean(
      projectId &&
      project &&
      projectQueryOwner &&
      projectQueryOwner.state.status === "success" &&
      projectQueryOwner.state.data === projectQueryData &&
      projectQueryData === project &&
      client.getQueryCache().find({ queryKey: projectKey, exact: true }) === projectQueryOwner &&
      projectAccessOwner.isCurrent() &&
      project.estimate === selectedId &&
      project.id === projectId
    );
  }, [client, project, projectAccessOwner, projectId, projectKey, projectQueryData, projectQueryOwner, selectedId]);
  const projectReadFailed = projectQuery.isError || projectQuery.accessError;
  const retry = useCallback(
    () => retryProjectEstimateReads(projectReadFailed, projectQuery.retryAccess, retryEstimates),
    [projectQuery.retryAccess, projectReadFailed, retryEstimates]
  );
  const estimates = !isError && !isAccessDenied && query.isSuccess ? query.data : undefined;
  const activeEstimate = activeProjectEstimate(estimates, projectId ?? undefined, project?.estimate);
  const points = useMemo(() => orderedEstimatePoints(activeEstimate), [activeEstimate]);
  const archivedEstimates = useMemo(
    () =>
      estimates &&
      sortBy(
        estimates.filter((estimate) => !estimate.last_used),
        (estimate) => -Number(new Date(estimate.created_at ?? 0))
      ),
    [estimates]
  );
  return {
    ...query,
    retry,
    scope: access.scope,
    project,
    projectScope: projectQuery.scope,
    estimates,
    activeEstimate,
    points,
    archivedEstimates,
    selectedId,
    isSelectionCurrent,
    hasSelectedEstimate: Boolean(selectedId),
    isEnabled: estimates !== undefined && Boolean(project?.estimate),
    isError,
    isPending,
    isAccessDenied,
  };
}

export function useEstimateMutations(workspaceSlug: string | undefined, projectId: string | undefined) {
  const client = useQueryClient();
  const estimates = useProjectEstimates(workspaceSlug, projectId);
  const options = estimateMutationOptions(client, estimates.scope);
  const mutation = useMutation({
    ...options,
    onMutate: async (write) => {
      if (!projectId || !estimates.project || !estimates.isSelectionCurrent())
        throw new Error("The selected project estimate is not available.");
      const queryKey = projectKeys.detail(estimates.projectScope);
      const query = client.getQueryCache().find({ queryKey, exact: true });
      const access = captureProjectCatalogueScope(client, estimates.projectScope);
      if (!query || query.state.status !== "success" || !access.isVerified())
        throw new Error("Project access is not ready to update the selected estimate.");
      const current = client.getQueryData<TProject>(queryKey);
      if (
        !current ||
        current.id !== projectId ||
        (write.type === "select" && (current.estimate ?? null) !== write.expectedId)
      )
        throw new Error("The selected project estimate changed before the write.");
      const listQueries = (["partial", "full"] as const).flatMap((kind) => {
        const key = projectKeys.workspace(estimates.projectScope, kind);
        const listQuery = client.getQueryCache().find({ queryKey: key, exact: true });
        return listQuery ? [{ key, query: listQuery }] : [];
      });
      await Promise.all(listQueries.map((list) => client.cancelQueries({ queryKey: list.key, exact: true })));
      if (
        listQueries.some(
          (list) =>
            !access.isVerified() || client.getQueryCache().find({ queryKey: list.key, exact: true }) !== list.query
        )
      )
        throw new Error("Project access changed before the estimate write.");
      return { queryKey, query, access, listQueries, currentEstimate: current.estimate ?? null };
    },
    onSuccess: async (result, write, captured, context) => {
      if (projectId && captured?.access.isCurrent()) {
        await Promise.all([
          client.cancelQueries({ queryKey: captured.queryKey, exact: true }),
          ...captured.listQueries.map((list) => client.cancelQueries({ queryKey: list.key, exact: true })),
        ]);
        if (
          captured.access.isCurrent() &&
          client.getQueryCache().find({ queryKey: captured.queryKey, exact: true }) === captured.query
        ) {
          client.setQueryData<TProject>(captured.queryKey, (current) => {
            if (!current || current.id !== projectId || (current.estimate ?? null) !== captured.currentEstimate)
              return current;
            if (write.type === "delete" && current.estimate === write.estimateId) return { ...current, estimate: null };
            if (write.type === "select" && result.project?.id === projectId)
              return { ...current, estimate: result.project.estimate ?? null };
            return current;
          });
        }
        for (const list of captured.listQueries) {
          if (
            !captured.access.isCurrent() ||
            client.getQueryCache().find({ queryKey: list.key, exact: true }) !== list.query
          )
            continue;
          client.setQueryData<TProject[] | TPartialProject[]>(list.key, (rows) =>
            rows?.map((project) => {
              if (
                !("estimate" in project) ||
                project.id !== projectId ||
                (project.estimate ?? null) !== captured.currentEstimate
              )
                return project;
              const nextEstimate =
                write.type === "delete" && project.estimate === write.estimateId
                  ? null
                  : write.type === "select" && result.project?.id === projectId
                    ? (result.project.estimate ?? null)
                    : project.estimate;
              return { ...project, estimate: nextEstimate };
            })
          );
        }
      }
      await options.onSuccess?.(result, write, captured, context);
    },
  });
  return { ...estimates, mutation };
}
