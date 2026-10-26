/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useQuery, useQueryClient } from "@tanstack/react-query";
import sortBy from "lodash-es/sortBy.js";
import type { IIssueDisplayFilterOptions } from "@plane/types";
import type { ICycle, IModule } from "@plane/types";
import { useParentCatalogueScope } from "./use-parent-catalogue-scope";
import {
  projectCycleUniverseOptions,
  projectModuleUniverseOptions,
  workspaceCycleUniverseOptions,
  workspaceModuleUniverseOptions,
} from "@/queries/cycle-modules";
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { captureParentCatalogueProjection } from "@/queries/parent-catalogue-projection";
import type { ParentCatalogueProjection } from "@/queries/parent-catalogue-projection";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "@/queries/catalogue-scope";

type CatalogueRequirements = {
  groupCycle: boolean;
  groupModule: boolean;
  cycleRequired: boolean;
  moduleRequired: boolean;
};
type ParentCatalogueAccess = ReturnType<typeof useParentCatalogueScope>;

function useParentQueries(workspaceLevel: boolean, access: ParentCatalogueAccess, required: CatalogueRequirements) {
  const cycle = useQuery({
    ...(workspaceLevel
      ? workspaceCycleUniverseOptions(access.workspaceScope)
      : projectCycleUniverseOptions(access.projectScope)),
    enabled: required.cycleRequired,
  });
  const module = useQuery({
    ...(workspaceLevel
      ? workspaceModuleUniverseOptions(access.workspaceScope)
      : projectModuleUniverseOptions(access.projectScope)),
    enabled: required.moduleRequired,
  });
  return { cycle, module };
}

function parentCatalogueReadStatus(
  required: CatalogueRequirements,
  access: ParentCatalogueAccess,
  cycle: ReturnType<typeof useParentQueries>["cycle"],
  module: ReturnType<typeof useParentQueries>["module"],
  cycles: ParentCatalogueProjection<ICycle>,
  modules: ParentCatalogueProjection<IModule>
) {
  const anyRequired = required.cycleRequired || required.moduleRequired;
  const failed =
    (anyRequired && (access.isError || access.isAccessDenied)) ||
    (required.cycleRequired && cycle.isError) ||
    (required.moduleRequired && module.isError);
  const pending =
    (anyRequired && access.isPending) ||
    (required.cycleRequired && (cycle.isPending || !cycles.isCurrent())) ||
    (required.moduleRequired && (module.isPending || !modules.isCurrent()));
  return { failed, pending };
}

function catalogueRequirements(filters: IIssueDisplayFilterOptions | undefined): CatalogueRequirements {
  const groupCycle = filters?.group_by === "cycle" || filters?.sub_group_by === "cycle";
  const groupModule = filters?.group_by === "module" || filters?.sub_group_by === "module";
  return {
    groupCycle,
    groupModule,
    cycleRequired: groupCycle || isCycleNameOrder(filters?.order_by),
    moduleRequired: groupModule || isModuleNameOrder(filters?.order_by),
  };
}

function isCycleNameOrder(order: string | undefined) {
  return order === "issue_cycle__cycle__name" || order === "-issue_cycle__cycle__name";
}

function isModuleNameOrder(order: string | undefined) {
  return order === "issue_module__module__name" || order === "-issue_module__module__name";
}

function cycleOptions(workspaceLevel: boolean, workspace: WorkspaceCatalogueScope, project: ProjectCatalogueScope) {
  return workspaceLevel ? workspaceCycleUniverseOptions(workspace) : projectCycleUniverseOptions(project);
}

function moduleOptions(workspaceLevel: boolean, workspace: WorkspaceCatalogueScope, project: ProjectCatalogueScope) {
  return workspaceLevel ? workspaceModuleUniverseOptions(workspace) : projectModuleUniverseOptions(project);
}

function structuralRevision(
  required: CatalogueRequirements,
  cycles: ParentCatalogueProjection<ICycle>,
  modules: ParentCatalogueProjection<IModule>
) {
  if (!required.groupCycle && !required.groupModule) return undefined;
  if ((required.groupCycle && !cycles.isCurrent()) || (required.groupModule && !modules.isCurrent())) return undefined;
  return JSON.stringify([
    required.groupCycle ? sortBy(cycles.rows?.map((row) => row.id) ?? [], (id) => id) : null,
    required.groupModule ? sortBy(modules.rows?.map((row) => row.id) ?? [], (id) => id) : null,
  ]);
}

async function retryCatalogues(
  client: ReturnType<typeof useQueryClient>,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  workspaceLevel: boolean,
  required: CatalogueRequirements,
  retryAccess: () => Promise<boolean>
) {
  if (!(await retryAccess())) return;
  const current = currentParentCatalogueScope(client, workspaceSlug, projectId);
  const requests = [
    ...(required.cycleRequired
      ? [client.fetchQuery({ ...cycleOptions(workspaceLevel, current.workspace, current.project), staleTime: 0 })]
      : []),
    ...(required.moduleRequired
      ? [client.fetchQuery({ ...moduleOptions(workspaceLevel, current.workspace, current.project), staleTime: 0 })]
      : []),
  ];
  await Promise.all(requests);
}

export function useIssueParentCatalogues(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  workspaceLevel: boolean,
  filters: IIssueDisplayFilterOptions | undefined
) {
  const client = useQueryClient();
  const required = catalogueRequirements(filters);
  const access = useParentCatalogueScope(
    required.cycleRequired || required.moduleRequired ? workspaceSlug : undefined,
    workspaceLevel ? undefined : projectId
  );
  const scope = workspaceLevel ? access.workspaceScope : access.projectScope;
  const { cycle, module } = useParentQueries(workspaceLevel, access, required);
  const cycles = captureParentCatalogueProjection(
    client,
    "cycle",
    scope,
    access.isReady && cycle.isSuccess ? cycle.data : undefined,
    "universe"
  );
  const modules = captureParentCatalogueProjection(
    client,
    "module",
    scope,
    access.isReady && module.isSuccess ? module.data : undefined,
    "universe"
  );
  const { failed, pending } = parentCatalogueReadStatus(required, access, cycle, module, cycles, modules);
  return {
    required: required.cycleRequired || required.moduleRequired,
    parents: { cycles, modules },
    isError: failed,
    isPending: !failed && pending,
    structuralRevision: structuralRevision(required, cycles, modules),
    grouped: required.groupCycle || required.groupModule,
    retry: () => retryCatalogues(client, workspaceSlug, projectId, workspaceLevel, required, access.retry),
  };
}
