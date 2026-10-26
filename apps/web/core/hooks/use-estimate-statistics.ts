/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCycleDetails } from "@/hooks/use-cycles";
import { useModuleDetails } from "@/hooks/use-modules";

type ParentEstimateRead = {
  isError: boolean;
  isAccessDenied: boolean;
  isPending: boolean;
  query: { isFetching: boolean };
  estimates: {
    isError: boolean;
    isAccessDenied: boolean;
    isPending: boolean;
    retry: () => Promise<unknown>;
  };
  retry: () => Promise<unknown>;
};

function estimateStatisticsFailed(
  parentId: string | undefined,
  parent: ParentEstimateRead,
  estimates: ParentEstimateRead["estimates"]
) {
  return Boolean(
    parentId && (parent.isError || parent.isAccessDenied || estimates.isError || estimates.isAccessDenied)
  );
}

async function retryStatistics(estimates: ParentEstimateRead["estimates"], retryParent: () => Promise<unknown>) {
  if (estimates.isError || estimates.isAccessDenied) await estimates.retry();
  await retryParent();
}
/** Native detail query inputs include all live numeric systems, including inactive systems referenced by issues. */
export function useEstimateStatistics(
  domain: "cycle" | "module",
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  parentId: string | undefined,
  archived = false
) {
  const cycle = useCycleDetails(
    domain === "cycle" ? workspaceSlug : undefined,
    domain === "cycle" ? projectId : undefined,
    domain === "cycle" ? parentId : undefined,
    archived
  );
  const module = useModuleDetails(
    domain === "module" ? workspaceSlug : undefined,
    domain === "module" ? projectId : undefined,
    domain === "module" ? parentId : undefined,
    archived
  );
  const parent = domain === "cycle" ? cycle : module;
  const estimates = parent.estimates;
  const error = estimateStatisticsFailed(parentId, parent, estimates);
  return {
    ...estimates,
    statistics: {
      isError: error,
      isPending: !error && Boolean(parentId && (parent.isPending || parent.query.isFetching || estimates.isPending)),
      retry: () => retryStatistics(estimates, parent.retry),
    },
  };
}
