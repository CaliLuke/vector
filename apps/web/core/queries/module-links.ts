/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { mutationOptions } from "@tanstack/react-query";
import type { Query, QueryClient, QueryKey } from "@tanstack/react-query";
import type { ILinkDetails, IModule, ModuleLink } from "@plane/types";
import type { ProjectCatalogueScope } from "./catalogue-scope";
import { captureParentAccess } from "./cycle-module-scope";
import { parentCatalogueKeys } from "./cycle-modules";
import { isAuthorizationError } from "./authorization-errors";
import { ModuleService } from "@/services/module.service";
const service = new ModuleService();
export type ModuleLinkWrite =
  | { type: "create"; data: ModuleLink }
  | { type: "patch"; id: string; data: ModuleLink }
  | { type: "delete"; id: string };

function moduleLinkScopeIsReady(
  client: QueryClient,
  scope: ProjectCatalogueScope,
  moduleId: string,
  detailKey: QueryKey
) {
  const source = client.getQueryCache().find({ queryKey: detailKey, exact: true });
  const sourceIsCurrent = () =>
    source !== undefined && client.getQueryCache().find({ queryKey: detailKey, exact: true }) === source;
  const access = captureParentAccess(client, scope);
  const isReady = Boolean(
    scope.workspaceSlug &&
    scope.projectId &&
    access.isVerified() &&
    [15, 20].includes(scope.projectRole ?? 0) &&
    sourceIsCurrent() &&
    source?.state.status === "success" &&
    client.getQueryData<IModule | null>(detailKey)?.id === moduleId
  );
  return { access, source, sourceIsCurrent, isReady };
}

async function sendModuleLinkWrite(scope: ProjectCatalogueScope, moduleId: string, input: ModuleLinkWrite) {
  const workspaceSlug = scope.workspaceSlug;
  const projectId = scope.projectId;
  if (!workspaceSlug || !projectId) throw new Error("Reload the module before editing its links.");
  if (input.type === "create")
    return service.createModuleLink(workspaceSlug, projectId, moduleId, { metadata: {}, ...input.data });
  if (input.type === "patch")
    return service.updateModuleLink(workspaceSlug, projectId, moduleId, input.id, { metadata: {}, ...input.data });
  await service.deleteModuleLink(workspaceSlug, projectId, moduleId, input.id);
  return undefined;
}

function applyModuleLinkToRecord(
  module: IModule,
  moduleId: string,
  input: ModuleLinkWrite,
  record: ILinkDetails | undefined
) {
  if (module.id !== moduleId) return module;
  const editedLinkId = input.type === "create" ? record?.id : input.id;
  return {
    ...module,
    link_module: [
      ...(module.link_module ?? []).filter((link) => link.id !== editedLinkId),
      ...(record ? [record] : []),
    ],
  };
}

async function reconcileModuleLinkWrite(
  client: QueryClient,
  moduleId: string,
  input: ModuleLinkWrite,
  record: ILinkDetails | undefined,
  affected: ReadonlySet<Query>,
  isCurrent: () => boolean
) {
  if (!isCurrent()) return;
  await client.cancelQueries({ predicate: (query) => affected.has(query) });
  if (!isCurrent()) return;
  for (const query of affected) {
    if (client.getQueryCache().get(query.queryHash) !== query) continue;
    if (query.queryKey.at(-3) === "detail")
      client.setQueryData<IModule | null>(query.queryKey, (row) =>
        row ? applyModuleLinkToRecord(row, moduleId, input, record) : row
      );
    else if (query.queryKey.at(-2) === "list")
      client.setQueryData<IModule[]>(query.queryKey, (rows) =>
        rows?.map((module) => applyModuleLinkToRecord(module, moduleId, input, record))
      );
  }
}

export function moduleLinkMutationOptions(
  client: QueryClient,
  scope: ProjectCatalogueScope,
  moduleId: string,
  detailKey: QueryKey
) {
  return mutationOptions({
    mutationKey: [...detailKey, "links"],
    scope: { id: JSON.stringify([...parentCatalogueKeys.owner("module", scope), scope.projectId, "write"]) },
    retry: false,
    mutationFn: async (input: ModuleLinkWrite) => {
      const readiness = moduleLinkScopeIsReady(client, scope, moduleId, detailKey);
      if (!readiness.isReady) throw new Error("Reload the module before editing its links.");
      const affected = new Set(
        client.getQueryCache().findAll({ queryKey: parentCatalogueKeys.scope("module", scope) })
      );
      const record = await sendModuleLinkWrite(scope, moduleId, input);
      if (record && (!record.id || (input.type === "patch" && record.id !== input.id)))
        throw new Error("The module link response did not match the edited link.");
      const isCurrent = () =>
        readiness.access.isCurrent() &&
        readiness.sourceIsCurrent() &&
        !isAuthorizationError(readiness.source?.state.error);
      await reconcileModuleLinkWrite(client, moduleId, input, record, affected, isCurrent);
      return { record, captured: { isCurrent } };
    },
  });
}
