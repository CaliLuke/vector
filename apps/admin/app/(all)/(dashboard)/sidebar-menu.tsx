/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSidebar } from "@/providers/sidebar";
import { Link, useLocation, matchPath } from "react-router";
// plane internal packages
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
// hooks
import { useSidebarMenu } from "@/hooks/use-sidebar-menu";

export const AdminSidebarMenu = function AdminSidebarMenu() {
  // router
  const { pathname: pathName } = useLocation();
  const { isSidebarCollapsed, toggleSidebar } = useSidebar();
  // derived values
  const sidebarMenu = useSidebarMenu();

  const handleItemClick = () => {
    if (window.innerWidth < 768) {
      toggleSidebar(!isSidebarCollapsed);
    }
  };

  return (
    <div className="vertical-scrollbar flex scrollbar-sm h-full w-full flex-col gap-2.5 overflow-y-scroll px-4 py-4">
      {sidebarMenu.map((item) => {
        const isActive = matchPath({ path: item.href, end: false }, pathName) !== null;
        return (
          <Link key={item.href} to={item.href} onClick={handleItemClick} aria-current={isActive ? "page" : undefined}>
            <div>
              <Tooltip tooltipContent={item.name} position="right" className="ml-2" disabled={!isSidebarCollapsed}>
                <div
                  className={cn(
                    "group flex w-full items-center gap-3 rounded-md px-3 py-2 transition-colors outline-none",
                    {
                      "!bg-layer-transparent-active text-primary": isActive,
                      "text-secondary hover:bg-layer-transparent-hover active:bg-layer-transparent-active": !isActive,
                    },
                    isSidebarCollapsed ? "justify-center" : "w-[260px]"
                  )}
                >
                  {<item.Icon className="h-4 w-4 flex-shrink-0" />}
                  {!isSidebarCollapsed && (
                    <div className="w-full">
                      <div className={cn(`text-body-xs-medium transition-colors`)}>{item.name}</div>
                      <div className={cn(`text-caption-sm-regular transition-colors`)}>{item.description}</div>
                    </div>
                  )}
                </div>
              </Tooltip>
            </div>
          </Link>
        );
      })}
    </div>
  );
};
