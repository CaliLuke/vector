/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useMemo, useCallback } from "react";
import { useProjectEstimates } from "./use-estimates";
import { captureParentStatistics, estimateStatisticsRevision } from "@/queries/parent-statistics";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { retryParentDetail, useParentCatalogueScope } from "./use-parent-catalogue-scope";
import {
  workspaceCycleUniverseOptions,
  projectCycleOptions,
  workspaceCycleOptions,
  cycleDetailOptions,
  cycleMutationOptions,
} from "@/queries/cycle-modules";

export function useProjectCycles(workspaceSlug: string | undefined, projectId: string | undefined, archived = false) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug, projectId);
  const query = useRecordCollection(projectCycleOptions(access.projectScope, archived));
  const mutation = useMutation(cycleMutationOptions(client, access.projectScope));
  return {
    scope: access.projectScope,
    query,
    cycles: access.isReady && query.isSuccess ? query.data : undefined,
    mutation,
    isPending: !access.isAccessDenied && (access.isPending || query.isPending),
    isError: access.isError || query.isError,
    isAccessDenied: access.isAccessDenied,
    retry: async () => {
      if (await access.retry())
        await client.fetchQuery({
          ...projectCycleOptions(currentParentCatalogueScope(client, workspaceSlug, projectId).project, archived),
          staleTime: 0,
        });
    },
  };
}
export function useWorkspaceCycles(workspaceSlug: string | undefined) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug),
    query = useRecordCollection(workspaceCycleOptions(access.workspaceScope));
  return {
    scope: access.workspaceScope,
    query,
    cycles: access.isReady && query.isSuccess ? query.data : undefined,
    isPending: !access.isAccessDenied && (access.isPending || query.isPending),
    isError: access.isError || query.isError,
    isAccessDenied: access.isAccessDenied,
    retry: async () => {
      if (await access.retry())
        await client.fetchQuery({
          ...workspaceCycleOptions(currentParentCatalogueScope(client, workspaceSlug, undefined).workspace),
          staleTime: 0,
        });
    },
  };
}
export function useCycleDetails(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  cycleId: string | undefined,
  archived = false
) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug, projectId);
  const estimates = useProjectEstimates(workspaceSlug, projectId);
  const revision = estimateStatisticsRevision(estimates.selectedId, projectId, estimates.estimates);
  const options = useMemo(
    () => cycleDetailOptions(access.projectScope, cycleId, archived, revision),
    [access.projectScope, cycleId, archived, revision]
  );
  const query = useQuery({ ...options, enabled: estimates.estimates !== undefined });
  const captureStatistics = useCallback(
    () =>
      cycleId ? captureParentStatistics(client, "cycle", access.projectScope, cycleId, options.queryKey) : undefined,
    [client, access.projectScope, cycleId, options.queryKey]
  );
  const isAccessDenied = Boolean(cycleId) && (access.isAccessDenied || estimates.isAccessDenied);
  const isError = Boolean(cycleId) && (access.isError || estimates.isError || query.isError);
  return {
    scope: access.projectScope,
    query,
    captureStatistics,
    estimates,
    cycle: access.isReady && estimates.estimates !== undefined && query.isSuccess ? query.data : undefined,
    isPending:
      Boolean(cycleId) && !isAccessDenied && !isError && (access.isPending || estimates.isPending || query.isPending),
    isError,
    isAccessDenied,
    retry: () =>
      retryParentDetail(estimates, async () => {
        if (await access.retry())
          await client.fetchQuery({
            ...cycleDetailOptions(
              currentParentCatalogueScope(client, workspaceSlug, projectId).project,
              cycleId,
              archived,
              revision
            ),
            staleTime: 0,
          });
      }),
  };
}

export function useWorkspaceCycleUniverse(workspaceSlug: string | undefined) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug),
    query = useRecordCollection(workspaceCycleUniverseOptions(access.workspaceScope));
  return {
    cycles: access.isReady && query.isSuccess ? query.data : undefined,
    isPending: access.isPending || query.isPending,
    isError: access.isError || query.isError,
    isAccessDenied: access.isAccessDenied,
    retry: async () => {
      if (await access.retry())
        await client.fetchQuery({
          ...workspaceCycleUniverseOptions(currentParentCatalogueScope(client, workspaceSlug, undefined).workspace),
          staleTime: 0,
        });
    },
  };
}
