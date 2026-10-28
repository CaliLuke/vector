/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { IIssueDisplayFilterOptions, IEstimatePoint } from "@plane/types";
import { useProjectEstimates, useWorkspaceEstimates } from "@/hooks/use-estimates";
import type { TEstimatePointProjection } from "@/queries/estimates";
import { captureEstimateScope, estimateKeys } from "@/queries/estimates";

/** Immutable native records cross the list boundary; stores never own this catalogue. */
export function useIssueEstimateProjection(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  isWorkspaceLevel: boolean
) {
  const client = useQueryClient();
  const project = useProjectEstimates(
    isWorkspaceLevel ? undefined : workspaceSlug,
    isWorkspaceLevel ? undefined : projectId
  );
  const workspace = useWorkspaceEstimates(isWorkspaceLevel ? workspaceSlug : undefined);
  const source = isWorkspaceLevel ? workspace : project;
  const rows = useMemo(() => {
    if (isWorkspaceLevel) return workspace.estimates;
    if (project.estimates === undefined) return undefined;
    return project.activeEstimate ? [project.activeEstimate] : [];
  }, [isWorkspaceLevel, workspace.estimates, project.estimates, project.activeEstimate]);
  const captured = captureEstimateScope(client, source.scope);
  const selectionIsCurrent = () => isWorkspaceLevel || project.isSelectionCurrent();
  const pointsByProject = useMemo(() => {
    if (!rows) return undefined;
    const result: Record<string, Record<string, IEstimatePoint>> = {};
    for (const estimate of rows) {
      if (!estimate.project) continue;
      const points: Record<string, IEstimatePoint> = {};
      for (const point of estimate.points ?? []) {
        if (point.id && point.project === estimate.project && point.estimate === estimate.id) points[point.id] = point;
      }
      result[estimate.project] = points;
    }
    return result;
  }, [rows]);
  const projection: TEstimatePointProjection = {
    workspaceSlug: workspaceSlug ?? "",
    scopeKey: JSON.stringify(estimateKeys.list(source.scope)),
    pointsByProject,
    isReadCurrent: captured.isVerified,
    isCurrent: () =>
      captured.isVerified() &&
      selectionIsCurrent() &&
      captured.query?.state.status === "success" &&
      !captured.query.state.isInvalidated &&
      source.isCurrent(),
  };
  return {
    ...source,
    isPending: source.isPending || Boolean(captured.query?.state.isInvalidated && source.isFetching),
    projection,
  };
}

/** Point-key changes rerender loaded records; only catalogue availability gates sorting. */
export function useIssueEstimateOrder(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  workspaceLevel: boolean,
  orderBy: IIssueDisplayFilterOptions["order_by"]
) {
  const estimates = useIssueEstimateProjection(workspaceSlug, projectId, workspaceLevel);
  const required = orderBy === "estimate_point__key" || orderBy === "-estimate_point__key";
  return {
    projection: estimates.projection,
    isPending: required && estimates.isPending,
    isError: required && (estimates.isError || estimates.isAccessDenied),
    retry: async () => {
      if (required) await estimates.retry();
    },
  };
}
