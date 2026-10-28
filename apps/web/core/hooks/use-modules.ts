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
  projectModuleOptions,
  workspaceModuleOptions,
  moduleDetailOptions,
  moduleMutationOptions,
} from "@/queries/cycle-modules";

export function useProjectModules(workspaceSlug: string | undefined, projectId: string | undefined, archived = false) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug, projectId);
  const query = useRecordCollection(projectModuleOptions(access.projectScope, archived));
  const mutation = useMutation(moduleMutationOptions(client, access.projectScope));
  return {
    scope: access.projectScope,
    query,
    modules: access.isReady && query.isSuccess ? query.data : undefined,
    mutation,
    isPending: !access.isAccessDenied && (access.isPending || query.isPending),
    isError: access.isError || query.isError,
    isAccessDenied: access.isAccessDenied,
    retry: async () => {
      if (await access.retry())
        await client.fetchQuery({
          ...projectModuleOptions(currentParentCatalogueScope(client, workspaceSlug, projectId).project, archived),
          staleTime: 0,
        });
    },
  };
}
export function useWorkspaceModules(workspaceSlug: string | undefined) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug),
    query = useRecordCollection(workspaceModuleOptions(access.workspaceScope));
  return {
    scope: access.workspaceScope,
    query,
    modules: access.isReady && query.isSuccess ? query.data : undefined,
    isPending: !access.isAccessDenied && (access.isPending || query.isPending),
    isError: access.isError || query.isError,
    isAccessDenied: access.isAccessDenied,
    retry: async () => {
      if (await access.retry())
        await client.fetchQuery({
          ...workspaceModuleOptions(currentParentCatalogueScope(client, workspaceSlug, undefined).workspace),
          staleTime: 0,
        });
    },
  };
}
export function useModuleDetails(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  moduleId: string | undefined,
  archived = false
) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug, projectId);
  const estimates = useProjectEstimates(workspaceSlug, projectId);
  const revision = estimateStatisticsRevision(estimates.selectedId, projectId, estimates.estimates);
  const options = useMemo(
    () => moduleDetailOptions(access.projectScope, moduleId, archived, revision),
    [access.projectScope, moduleId, archived, revision]
  );
  const query = useQuery({ ...options, enabled: estimates.estimates !== undefined });
  const captureStatistics = useCallback(
    () =>
      moduleId ? captureParentStatistics(client, "module", access.projectScope, moduleId, options.queryKey) : undefined,
    [client, access.projectScope, moduleId, options.queryKey]
  );
  const isAccessDenied = Boolean(moduleId) && (access.isAccessDenied || estimates.isAccessDenied);
  const isError = Boolean(moduleId) && (access.isError || estimates.isError || query.isError);
  return {
    scope: access.projectScope,
    queryKey: options.queryKey,
    query,
    captureStatistics,
    estimates,
    module: access.isReady && estimates.estimates !== undefined && query.isSuccess ? query.data : undefined,
    isPending:
      Boolean(moduleId) && !isAccessDenied && !isError && (access.isPending || estimates.isPending || query.isPending),
    isError,
    isAccessDenied,
    retry: () =>
      retryParentDetail(estimates, async () => {
        if (await access.retry())
          await client.fetchQuery({
            ...moduleDetailOptions(
              currentParentCatalogueScope(client, workspaceSlug, projectId).project,
              moduleId,
              archived,
              revision
            ),
            staleTime: 0,
          });
      }),
  };
}
