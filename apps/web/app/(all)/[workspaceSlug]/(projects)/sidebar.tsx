/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCurrentWorkspace } from "@/hooks/use-workspaces";
// plane helpers
import { EUserPermissions } from "@plane/constants";
// components
import { SidebarWrapper } from "@/components/sidebar/sidebar-wrapper";
import { SidebarFavoritesMenu } from "@/components/workspace/sidebar/favorites/favorites-menu";
import { SidebarProjectsList } from "@/components/workspace/sidebar/projects-list";
import { SidebarQuickActions } from "@/components/workspace/sidebar/quick-actions";
import { SidebarMenuItems } from "@/components/workspace/sidebar/sidebar-menu-items";
// hooks
import { useFavorites } from "@/hooks/use-favorites";
import { useWorkspacePermissions } from "@/hooks/use-permissions";

export function AppSidebar() {
  // store hooks
  const { data: catalogueWorkspace } = useCurrentWorkspace();
  const { can: canWorkspace } = useWorkspacePermissions(catalogueWorkspace?.slug);
  const { favorites, isError } = useFavorites(catalogueWorkspace?.slug);

  // derived values
  const canPerformWorkspaceMemberActions = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const hasFavorites = favorites !== undefined && favorites.length > 0;

  return (
    <SidebarWrapper title="Projects" quickActions={<SidebarQuickActions />}>
      <SidebarMenuItems />
      {/* Favorites Menu */}
      {canPerformWorkspaceMemberActions && (hasFavorites || isError) && <SidebarFavoritesMenu />}
      {/* Projects List */}
      <SidebarProjectsList />
    </SidebarWrapper>
  );
}
