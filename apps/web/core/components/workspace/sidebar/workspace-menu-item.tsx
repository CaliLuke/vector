/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useLocation } from "@tanstack/react-router";

// plane imports
import { useTranslation } from "@plane/i18n";
import type { EUserWorkspaceRoles } from "@plane/types";
import { cn } from "@plane/utils/common";
// components
import { SidebarNavItem } from "@/components/sidebar/sidebar-navigation";
// hooks
import { useSidebarActions } from "@/providers/sidebar-state";
import { useWorkspacePermissions } from "@/hooks/use-permissions";

export type SidebarWorkspaceMenuItemProps = {
  item: {
    labelTranslationKey: string;
    key: string;
    href: string;
    Icon: any;
    access: EUserWorkspaceRoles[];
  };
};

export function SidebarWorkspaceMenuItem(props: SidebarWorkspaceMenuItemProps) {
  const { item } = props;

  const { t } = useTranslation();
  // nextjs hooks
  const pathname = useLocation({ select: (location) => location.pathname });
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug.toString());
  // store hooks
  const { toggleSidebar } = useSidebarActions();

  const handleLinkClick = () => {
    if (window.innerWidth < 768) {
      toggleSidebar();
    }
  };

  if (!canWorkspace(item.access as any)) {
    return null;
  }

  const isActive = item.href === pathname;

  return (
    <Link to={item.href} onClick={() => handleLinkClick()}>
      <SidebarNavItem isActive={isActive}>
        <div className="flex items-center gap-1.5 py-[1px]">
          <item.Icon
            className={cn("size-4", {
              "rotate-180": item.key === "active_cycles",
            })}
          />
          <p className="text-13 leading-5 font-medium">{t(item.labelTranslationKey)}</p>
        </div>
      </SidebarNavItem>
    </Link>
  );
}
