/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSidebar } from "@/providers/sidebar";
import { useLocation } from "react-router";
import { Menu, Settings } from "lucide-react";
// icons
import { Breadcrumbs } from "@plane/ui";
// components
import { BreadcrumbLink } from "../breadcrumb-link";
// hooks
// local imports
import { CORE_HEADER_SEGMENT_LABELS } from "./core";
import { EXTENDED_HEADER_SEGMENT_LABELS } from "./extended";

export const HamburgerToggle = function HamburgerToggle() {
  const { isSidebarCollapsed, toggleSidebar } = useSidebar();
  return (
    <button
      type="button"
      aria-label="Toggle sidebar"
      className="group flex size-7 cursor-pointer items-center justify-center rounded-sm bg-layer-1 transition-all hover:bg-layer-1-hover md:hidden"
      onClick={() => toggleSidebar(!isSidebarCollapsed)}
    >
      <Menu size={14} className="text-secondary transition-all group-hover:text-primary" />
    </button>
  );
};

const HEADER_SEGMENT_LABELS = {
  ...CORE_HEADER_SEGMENT_LABELS,
  ...EXTENDED_HEADER_SEGMENT_LABELS,
};

// Function to dynamically generate breadcrumb items based on pathname
const generateBreadcrumbItems = (pathname: string) => {
  const pathSegments = pathname.split("/").filter(Boolean);

  let currentUrl = "";
  const breadcrumbItems = pathSegments.map((segment) => {
    currentUrl += "/" + segment;
    return {
      title: HEADER_SEGMENT_LABELS[segment] ?? segment.toUpperCase(),
      href: currentUrl,
    };
  });
  return breadcrumbItems;
};

export const AdminHeader = function AdminHeader() {
  const { pathname: pathName } = useLocation();

  const breadcrumbItems = generateBreadcrumbItems(pathName || "");

  return (
    <div className="relative z-10 flex h-header w-full flex-shrink-0 flex-row items-center justify-between gap-x-2 gap-y-4 border-b border-subtle bg-surface-1 p-4">
      <div className="flex w-full flex-grow items-center gap-2 overflow-ellipsis whitespace-nowrap">
        <HamburgerToggle />
        <div>
          <Breadcrumbs>
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  href="/general/"
                  label="Settings"
                  icon={<Settings className="h-4 w-4 text-tertiary" />}
                />
              }
            />
            {breadcrumbItems.map(
              (item) =>
                item.title && (
                  <Breadcrumbs.Item
                    key={item.title}
                    component={<BreadcrumbLink href={item.href} label={item.title} />}
                  />
                )
            )}
          </Breadcrumbs>
        </div>
      </div>
    </div>
  );
};
