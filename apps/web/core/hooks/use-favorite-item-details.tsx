/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { IFavorite, TLogoProps } from "@plane/types";
// components
import { getPageName } from "@plane/utils/page";
import {
  FavoriteItemIcon,
  generateFavoriteItemLink,
} from "@/components/workspace/sidebar/favorites/favorite-items/common";
// helpers
// hooks
import { useCycleDetails } from "@/hooks/use-cycles";
import { useModuleDetails } from "@/hooks/use-modules";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useProjectView } from "@/hooks/use-project-views";
// plane web hooks
import { useProjectPage } from "@/hooks/use-project-pages";
import { useAdditionalFavoriteItemDetails } from "@/hooks/use-additional-favorite-item-details";

export const useFavoriteItemDetails = (workspaceSlug: string, favorite: IFavorite) => {
  const { entity_type: favoriteItemEntityType } = favorite;
  const detailInput = favoriteDetailInput(favorite);
  // store hooks
  const { projects } = useWorkspaceProjects(workspaceSlug);
  const { cycle: cycleDetail } = useCycleDetails(workspaceSlug, detailInput.projectId, detailInput.cycleId);
  const { module: moduleDetail } = useModuleDetails(workspaceSlug, detailInput.projectId, detailInput.moduleId);
  // additional details
  const { getAdditionalFavoriteItemDetails } = useAdditionalFavoriteItemDetails();
  // derived values
  const pageQuery = useProjectPage(workspaceSlug, detailInput.pageProjectId, detailInput.pageId, false);
  const pageDetail = pageQuery.data;
  const viewDetailsQuery = useProjectView(workspaceSlug, detailInput.viewProjectId, detailInput.viewId);
  const viewDetails = viewDetailsQuery.view;
  const currentProjectDetails = projects?.find((project) => project.id === favorite.project_id);

  const entities = new Map<string, FavoriteEntity | null | undefined>([
    ["project", currentProjectDetails],
    ["page", pageDetail],
    ["view", viewDetails],
    ["cycle", cycleDetail],
    ["module", moduleDetail],
  ]);
  const presentation = entities.has(favoriteItemEntityType)
    ? favoritePresentation(favorite, entities.get(favoriteItemEntityType))
    : getAdditionalFavoriteItemDetails(workspaceSlug, favorite);
  return { ...presentation, itemLink: generateFavoriteItemLink(workspaceSlug, favorite) };
};

function favoriteDetailInput(favorite: IFavorite) {
  const projectId = favorite.project_id ?? undefined;
  const itemId = favorite.entity_identifier ?? undefined;
  const type = favorite.entity_type;
  return {
    projectId: type === "cycle" || type === "module" ? projectId : undefined,
    cycleId: type === "cycle" ? itemId : undefined,
    moduleId: type === "module" ? itemId : undefined,
    pageProjectId: type === "page" ? projectId : undefined,
    pageId: type === "page" ? itemId : undefined,
    viewProjectId: type === "view" ? projectId : undefined,
    viewId: type === "view" ? itemId : undefined,
  };
}

type FavoriteEntity = { name?: string | null; logo_props?: TLogoProps | null };

function favoritePresentation(favorite: IFavorite, entity: FavoriteEntity | null | undefined) {
  const type = favorite.entity_type;
  const name = entity?.name ?? (favorite.entity_data.name || favorite.name);
  const logo = ["project", "page", "view"].includes(type)
    ? (entity?.logo_props ?? favorite.entity_data.logo_props)
    : undefined;
  return {
    itemTitle: type === "page" ? getPageName(name) : name,
    itemIcon: <FavoriteItemIcon type={type} logo={logo} />,
  };
}
