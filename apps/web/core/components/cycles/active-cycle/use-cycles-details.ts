/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { useCallback, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { buildWorkItemFilterExpressionFromConditions } from "@plane/shared-state";
import type { TWorkItemFilterCondition } from "@plane/shared-state";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import {
  cycleProgressOptions,
  cycleIssueAnalyticsOptions,
  cycleEstimateAnalyticsOptions,
} from "@/queries/cycle-modules";
import { estimateStatisticsRevision } from "@/queries/parent-statistics";
import { parentIssuePreferenceOptions, patchParentIssuePreferenceOptions } from "@/queries/parent-issue-preferences";

type CycleReadState = {
  isArchived: boolean;
  detailError: boolean;
  detailDenied: boolean;
  detailPending: boolean;
  hasBase: boolean;
  progressError: boolean;
  progressPending: boolean;
  needDistribution: boolean;
  distributionError: boolean;
  distributionPending: boolean;
  needEstimates: boolean;
  estimateError: boolean;
  estimatePending: boolean;
  patchError: boolean;
};

function hasCycleReadError(state: CycleReadState) {
  return (
    state.detailError ||
    state.detailDenied ||
    (!state.isArchived && state.progressError) ||
    (state.needDistribution && state.distributionError) ||
    (state.needEstimates && state.estimateError) ||
    state.patchError
  );
}

function isCycleReadPending(state: CycleReadState) {
  return (
    state.detailPending ||
    (!state.isArchived && state.hasBase && state.progressPending) ||
    (state.needDistribution && state.distributionPending) ||
    (state.needEstimates && state.estimatePending)
  );
}

export default function useCyclesDetails({
  workspaceSlug,
  projectId,
  cycleId,
  isArchived = false,
}: {
  workspaceSlug: string;
  projectId: string;
  cycleId: string | null | undefined;
  isArchived?: boolean;
}) {
  const client = useQueryClient(),
    router = useRouter(),
    detail = useCycleDetails(workspaceSlug, projectId, cycleId ?? undefined, isArchived);
  const scope = detail.scope,
    base = detail.cycle;
  const revision = estimateStatisticsRevision(detail.estimates.selectedId, projectId, detail.estimates.estimates);
  const progress = useQuery(cycleProgressOptions(scope, isArchived ? undefined : base?.id, revision));
  const needDistribution = Boolean(!isArchived && base && !base.distribution),
    needEstimates = Boolean(!isArchived && base && !base.estimate_distribution);
  const distribution = useQuery({
    ...cycleIssueAnalyticsOptions(scope, base?.id, revision),
    enabled: needDistribution,
  });
  const estimateDistribution = useQuery({
    ...cycleEstimateAnalyticsOptions(scope, base?.id, revision),
    enabled: needEstimates,
  });
  const patch = useMutation(patchParentIssuePreferenceOptions(client, "cycle", scope, cycleId ?? undefined));
  const { isError: patchIsError, variables: patchVariables, mutateAsync: mutatePatch } = patch;
  const mounted = useCommittedScope(JSON.stringify([scope, cycleId]));
  const retryDetail = detail.retry;
  const cycle = useMemo(
    () =>
      base
        ? {
            ...base,
            ...(progress.isSuccess ? progress.data : {}),
            ...(needDistribution && distribution.isSuccess ? { distribution: distribution.data ?? undefined } : {}),
            ...(needEstimates && estimateDistribution.isSuccess
              ? { estimate_distribution: estimateDistribution.data ?? undefined }
              : {}),
          }
        : null,
    [
      base,
      progress.isSuccess,
      progress.data,
      needDistribution,
      distribution.isSuccess,
      distribution.data,
      needEstimates,
      estimateDistribution.isSuccess,
      estimateDistribution.data,
    ]
  );
  const finish = useCallback(
    (result: { captured: { isCurrent: () => boolean } }) => {
      if (mounted() && result.captured.isCurrent())
        void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}` });
    },
    [cycleId, mounted, projectId, router, workspaceSlug]
  );
  const handleFiltersUpdate = async (conditions: TWorkItemFilterCondition[]) => {
    if (!cycleId || !mounted()) return;
    const expression = buildWorkItemFilterExpressionFromConditions({ conditions });
    if (!expression) return;
    await client.ensureQueryData(parentIssuePreferenceOptions("cycle", scope, cycleId));
    if (!mounted()) return;
    const result = await patch.mutateAsync({ rich_filters: expression });
    finish(result);
  };
  const retry = useCallback(async () => {
    if (patchIsError && patchVariables) {
      finish(await mutatePatch(patchVariables));
      return;
    }
    await retryDetail();
    if (!mounted() || isArchived) return;
    const fresh = currentParentCatalogueScope(client, workspaceSlug, projectId).project;
    await Promise.all([
      client.fetchQuery({ ...cycleProgressOptions(fresh, cycleId ?? undefined, revision), staleTime: 0 }),
      ...(needDistribution
        ? [client.fetchQuery({ ...cycleIssueAnalyticsOptions(fresh, cycleId ?? undefined, revision), staleTime: 0 })]
        : []),
      ...(needEstimates
        ? [client.fetchQuery({ ...cycleEstimateAnalyticsOptions(fresh, cycleId ?? undefined, revision), staleTime: 0 })]
        : []),
    ]);
  }, [
    client,
    cycleId,
    finish,
    isArchived,
    mounted,
    needDistribution,
    needEstimates,
    mutatePatch,
    patchIsError,
    patchVariables,
    projectId,
    revision,
    retryDetail,
    workspaceSlug,
  ]);
  const readState: CycleReadState = {
    isArchived,
    detailError: detail.isError,
    detailDenied: detail.isAccessDenied,
    detailPending: detail.isPending,
    hasBase: Boolean(base),
    progressError: progress.isError,
    progressPending: progress.isPending,
    needDistribution,
    distributionError: distribution.isError,
    distributionPending: distribution.isPending,
    needEstimates,
    estimateError: estimateDistribution.isError,
    estimatePending: estimateDistribution.isPending,
    patchError: patch.isError,
  };
  return {
    cycle,
    cycleId,
    router,
    handleFiltersUpdate,
    isError: hasCycleReadError(readState),
    isPending: isCycleReadPending(readState),
    retry,
  };
}
