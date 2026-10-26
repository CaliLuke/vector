/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { queryOptions, mutationOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { TProjectPublishSettings } from "@plane/types";
import { captureProjectCatalogueScope, projectKeys } from "@/queries/projects";
import type { TProjectCatalogueScope } from "@/queries/projects";
import { ProjectPublishService } from "@/services/project/project-publish.service";

const service = new ProjectPublishService();
const projectPublishingKey = (scope: TProjectCatalogueScope) =>
  ["project-publishing", ...projectKeys.detail(scope)] as const;

export const projectPublishingOptions = (client: QueryClient, scope: TProjectCatalogueScope, enabled = true) => {
  const queryKey = projectPublishingKey(scope);
  return queryOptions({
    queryKey,
    enabled: Boolean(enabled && scope.workspaceSlug && scope.projectId),
    retry: false,
    queryFn: async ({ signal }) => {
      const access = captureProjectCatalogueScope(client, scope);
      const query = client.getQueryCache().find({ queryKey, exact: true });
      const isCurrent = () =>
        access.isVerified() && query && client.getQueryCache().find({ queryKey, exact: true }) === query;
      if (!scope.workspaceSlug || !scope.projectId || !isCurrent())
        throw new Error("Project access changed before loading publishing settings.");
      const settings = await service.fetchPublishSettings(scope.workspaceSlug, scope.projectId, signal);
      if (!isCurrent()) throw new Error("Project access changed while loading publishing settings.");
      return settings;
    },
  });
};
type PublishAction =
  | { type: "save"; id?: string; settings: Partial<TProjectPublishSettings> }
  | { type: "unpublish"; id: string };
export function projectPublishingMutationOptions(client: QueryClient, scope: TProjectCatalogueScope) {
  const key = projectPublishingKey(scope);
  const access = captureProjectCatalogueScope(client, scope);
  const capturedQuery = client.getQueryCache().find({ queryKey: key, exact: true });
  return mutationOptions({
    scope: {
      id: `project-publication/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}`,
    },
    retry: false,
    onMutate: async () => {
      if (
        !scope.workspaceSlug ||
        !scope.projectId ||
        !capturedQuery ||
        capturedQuery.state.status !== "success" ||
        !access.isVerified()
      )
        throw new Error("Project access changed before publishing settings could be updated.");
      await client.cancelQueries({ queryKey: key, exact: true });
      if (!access.isVerified() || client.getQueryCache().find({ queryKey: key, exact: true }) !== capturedQuery)
        throw new Error("Project access changed before publishing settings could be updated.");
      return capturedQuery;
    },
    mutationFn: async (action: PublishAction) => {
      if (!scope.workspaceSlug || !scope.projectId || !access.isVerified())
        throw new Error("Project access changed before publishing settings could be updated.");
      if (action.type === "unpublish") {
        await service.unpublishProject(scope.workspaceSlug, scope.projectId, action.id);
        return { settings: null, access };
      }
      const settings = action.id
        ? await service.updatePublishSettings(scope.workspaceSlug, scope.projectId, action.id, action.settings)
        : await service.publishProject(scope.workspaceSlug, scope.projectId, action.settings);
      return { settings, access };
    },
    onSuccess: async ({ settings, access: acceptedAccess }, _action, query) => {
      if (!capturedQuery || query !== capturedQuery || !acceptedAccess.isCurrent()) return;
      await client.cancelQueries({ queryKey: key, exact: true });
      if (client.getQueryCache().find({ queryKey: key, exact: true }) !== capturedQuery || !acceptedAccess.isCurrent())
        return;
      if (settings) client.setQueryData(key, settings);
      else client.setQueryData(key, (current) => current && { ...current, id: undefined, anchor: undefined });
    },
  });
}
