/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { PlaneNewIcon } from "@plane/propel/icons";
import type { AppSidebarItemData } from "@/components/sidebar/sidebar-item";
import { useWorkspacePaths } from "@/hooks/use-workspace-paths";

export function useAppRailItems(): (AppSidebarItemData & { shouldRender: boolean })[] {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { isProjectsPath, isNotificationsPath } = useWorkspacePaths();
  return [
    {
      label: "Projects",
      icon: <PlaneNewIcon className="size-5" />,
      href: `/${workspaceSlug}/`,
      isActive: isProjectsPath && !isNotificationsPath,
      shouldRender: true,
    },
  ];
}
