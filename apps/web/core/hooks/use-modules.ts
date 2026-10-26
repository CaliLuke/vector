/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useMemo, useCallback } from "react";
import { useProjectEstimates } from "./use-estimates";
import { captureParentStatistics, estimateStatisticsRevision } from "@/queries/parent-statistics";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { useParentCatalogueScope } from "./use-parent-catalogue-scope";
import {
  projectModuleOptions,
  workspaceModuleOptions,
  moduleDetailOptions,
  moduleMutationOptions,
} from "@/queries/cycle-modules";

export function useProjectModules(workspaceSlug: string | undefined, projectId: string | undefined, archived = false) {
  const client = useQueryClient(),
    access = useParentCatalogueScope(workspaceSlug, projectId);
  const query = useQuery(projectModuleOptions(access.projectScope, archived));
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
    query = useQuery(workspaceModuleOptions(access.workspaceScope));
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
  const query = useQuery(options);
  const captureStatistics = useCallback(
    () =>
      moduleId ? captureParentStatistics(client, "module", access.projectScope, moduleId, options.queryKey) : undefined,
    [client, access.projectScope, moduleId, options.queryKey]
  );
  return {
    scope: access.projectScope,
    queryKey: options.queryKey,
    query,
    captureStatistics,
    estimates,
    module: access.isReady && query.isSuccess ? query.data : undefined,
    isPending: Boolean(moduleId) && !access.isAccessDenied && (access.isPending || query.isPending),
    isError: Boolean(moduleId) && (access.isError || query.isError),
    isAccessDenied: Boolean(moduleId) && access.isAccessDenied,
    retry: async () => {
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
    },
  };
}
