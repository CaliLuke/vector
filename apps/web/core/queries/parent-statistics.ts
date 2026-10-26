/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { QueryClient, QueryKey } from "@tanstack/react-query";
import sortBy from "lodash-es/sortBy.js";
import { EEstimateSystem } from "@plane/types";
import type { IEstimate, ICycle, IModule } from "@plane/types";
import type { DistributionUpdates } from "@plane/utils/distribution-update";
import { updateDistribution } from "@plane/utils/distribution-update";
import { isAuthorizationError } from "./authorization-errors";
import { captureParentAccess } from "./cycle-module-scope";
import type { ProjectCatalogueScope } from "./catalogue-scope";
import { parentCatalogueKeys } from "./cycle-modules";
import type { ParentDomain } from "./cycle-modules";

export function estimateStatisticsRevision(
  selectedId: string | null,
  projectId: string | undefined,
  estimates: readonly IEstimate[] | undefined
) {
  return JSON.stringify([
    selectedId,
    sortBy(
      estimates?.filter((estimate) => estimate.project === projectId && estimate.type === EEstimateSystem.POINTS),
      "id"
    ).map((estimate) => [
      estimate.id,
      sortBy(estimate.points, "id")
        .filter(
          (point) =>
            point.project === projectId && point.estimate === estimate.id && point.workspace === estimate.workspace
        )
        .map((point) => [point.id, point.value]),
    ]),
  ]);
}
/** Explicit native effects cross the issue-list boundary; no parent records enter MobX. */
type ParentStatisticsOwner = {
  workspaceSlug: string;
  projectId: string;
  isCurrent: () => boolean;
  refreshAssignments?: (domain: ParentDomain, parentIds: readonly string[]) => Promise<void>;
};
export type ParentStatisticsEffects = ParentStatisticsOwner &
  (
    | { domain: "assignments" }
    | { domain: "project"; parentId: string; refresh: () => Promise<void> }
    | {
        domain: ParentDomain;
        parentId: string;
        refresh: () => Promise<void>;
        applyDistribution: (updates: DistributionUpdates) => (() => void) | undefined;
      }
  );
/** Capture only native query ownership; accepted effects never discover a replacement scope. */
function captureParentAssignmentRefresh(client: QueryClient, scope: ProjectCatalogueScope) {
  const access = captureParentAccess(client, scope);
  const queries = new Map(
    (["cycle", "module"] as const).map((domain) => [
      domain,
      new Set(
        client
          .getQueryCache()
          .findAll({ queryKey: parentCatalogueKeys.scope(domain, scope) })
          .concat(
            client.getQueryCache().findAll({ queryKey: [...parentCatalogueKeys.owner(domain, scope), "workspace"] })
          )
      ),
    ])
  );
  const isCurrent = access.isCurrent;
  return {
    isCurrent,
    refreshAssignments: async (domain: ParentDomain, parentIds: readonly string[]) => {
      if (!isCurrent() || parentIds.length === 0) return;
      const captured = queries.get(domain);
      const affected = (query: import("@tanstack/react-query").Query) =>
        Boolean(
          captured?.has(query) &&
          (query.queryKey.at(-2) === "list" || parentIds.some((id) => query.queryKey.includes(id)))
        );
      await client.cancelQueries({ predicate: affected });
      if (isCurrent())
        await client.invalidateQueries({ predicate: affected, refetchType: access.isVerified() ? "active" : "none" });
    },
  };
}
export function captureParentStatistics(
  client: QueryClient,
  domain: ParentDomain,
  scope: ProjectCatalogueScope,
  parentId: string,
  queryKey: QueryKey
): ParentStatisticsEffects | undefined {
  if (!scope.workspaceSlug || !scope.projectId) return undefined;
  const access = captureParentAccess(client, scope),
    query = client.getQueryCache().find({ queryKey, exact: true }),
    original = client.getQueryData<ICycle | IModule | null>(queryKey);
  const related = new Set(
    client
      .getQueryCache()
      .findAll({ queryKey: parentCatalogueKeys.scope(domain, scope) })
      .filter((row) => row.queryKey.at(-2) === "list" || row.queryKey.includes(parentId))
  );
  let expected: ICycle | IModule | null | undefined = original;
  let expectedVersion = query?.state.dataUpdateCount;
  const isCurrent = () =>
    access.isCurrent() &&
    query !== undefined &&
    !isAuthorizationError(query.state.error) &&
    client.getQueryCache().find({ queryKey, exact: true }) === query;
  const assignments = captureParentAssignmentRefresh(client, scope);
  return {
    domain,
    workspaceSlug: scope.workspaceSlug,
    projectId: scope.projectId,
    parentId,
    isCurrent,
    refreshAssignments: async (assignmentDomain, ids) => {
      if (isCurrent()) await assignments.refreshAssignments(assignmentDomain, ids);
    },
    applyDistribution: (updates) => {
      if (
        expected == null ||
        !isCurrent() ||
        query?.state.status !== "success" ||
        query.state.isInvalidated ||
        query.state.data !== expected ||
        query.state.dataUpdateCount !== expectedVersion
      )
        return undefined;
      const before = expected;
      expected = client.setQueryData<ICycle | IModule | null>(queryKey, (current) => {
        if (!current) return current;
        const next = structuredClone(current);
        updateDistribution(next, updates);
        return next;
      });
      expectedVersion = query.state.dataUpdateCount;
      const applied = expected,
        version = expectedVersion;
      return () => {
        if (
          !isCurrent() ||
          query.state.status !== "success" ||
          query.state.isInvalidated ||
          query.state.data !== applied ||
          query.state.dataUpdateCount !== version
        )
          return;
        expected = client.setQueryData<ICycle | IModule | null>(queryKey, before);
        expectedVersion = query.state.dataUpdateCount;
      };
    },
    refresh: async () => {
      if (!isCurrent()) return;
      await client.cancelQueries({ predicate: (current) => related.has(current) });
      if (isCurrent())
        await client.invalidateQueries({
          predicate: (current) => related.has(current),
          refetchType: access.isVerified() ? "active" : "none",
        });
    },
  };
}
