/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useLocation } from "@tanstack/react-router";
import type { ReactNode } from "react";

// plane imports
import type { IWorkspaceSidebarNavigationItem } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { joinUrlPath } from "@plane/utils/string";
// components
import { SidebarNavItem } from "@/components/sidebar/sidebar-navigation";
// hooks
import { useSidebarActions, useExtendedSidebarOpen } from "@/providers/sidebar-state";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceNavigationPreferences } from "@/hooks/use-navigation-preferences";
// plane web imports
import { getSidebarNavigationItemIcon } from "@/components/workspace/sidebar/helper";

type Props = {
  item: IWorkspaceSidebarNavigationItem;
  additionalRender?: (itemKey: string, workspaceSlug: string) => ReactNode;
  additionalStaticItems?: string[];
};

export function SidebarItemBase({ item, additionalRender, additionalStaticItems }: Props) {
  const { t } = useTranslation();
  const pathname = useLocation({ select: (location) => location.pathname });
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  const { isWorkspaceItemPinned } = useWorkspaceNavigationPreferences();
  const { data } = useCurrentUser();

  const isExtendedSidebarOpened = useExtendedSidebarOpen();
  const { toggleSidebar, toggleExtendedSidebar } = useSidebarActions();

  const handleLinkClick = () => {
    if (window.innerWidth < 768) toggleSidebar();
    if (isExtendedSidebarOpened) toggleExtendedSidebar(false);
  };

  const staticItems = [
    "home",
    "pi_chat",
    "projects",
    "your_work",
    "stickies",
    "drafts",
    ...(additionalStaticItems || []),
  ];
  const slug = workspaceSlug || "";

  if (!canWorkspace(item.access)) return null;

  const isPinned = isWorkspaceItemPinned(item.key);
  if (!isPinned && !staticItems.includes(item.key)) return null;

  const itemHref =
    item.key === "your_work" && data?.id ? joinUrlPath(slug, item.href, data?.id) : joinUrlPath(slug, item.href);
  const icon = getSidebarNavigationItemIcon(item.key);

  return (
    <Link to={itemHref} onClick={handleLinkClick}>
      <SidebarNavItem isActive={item.highlight(pathname, itemHref)}>
        <div className="flex items-center gap-1.5 py-[1px]">
          {icon}
          <p className="text-13 leading-5 font-medium">{t(item.labelTranslationKey)}</p>
        </div>
        {additionalRender?.(item.key, slug)}
      </SidebarNavItem>
    </Link>
  );
}
