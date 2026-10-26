/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSidebar } from "@/providers/sidebar";
import { useEffect, useRef } from "react";
// plane helpers
import { useOutsideClickDetector } from "@plane/hooks";
// hooks
// components
import { AdminSidebarDropdown } from "./sidebar-dropdown";
import { AdminSidebarHelpSection } from "./sidebar-help-section";
import { AdminSidebarMenu } from "./sidebar-menu";

export const AdminSidebar = function AdminSidebar() {
  const { isSidebarCollapsed, toggleSidebar } = useSidebar();

  const ref = useRef<HTMLDivElement>(null);

  useOutsideClickDetector(ref, () => {
    if (!isSidebarCollapsed) {
      if (window.innerWidth < 768) {
        toggleSidebar(!isSidebarCollapsed);
      }
    }
  });

  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth <= 768) {
        toggleSidebar(true);
      }
    };
    handleResize();
    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, [toggleSidebar]);

  return (
    <div
      className={`fixed inset-y-0 z-20 flex h-full flex-shrink-0 flex-grow-0 flex-col border-r border-subtle bg-surface-1 duration-300 md:relative ${isSidebarCollapsed ? "-ml-[290px]" : ""} sm:${isSidebarCollapsed ? "-ml-[290px]" : ""} md:ml-0 ${isSidebarCollapsed ? "w-[70px]" : "w-[290px]"} lg:ml-0 ${isSidebarCollapsed ? "w-[70px]" : "w-[290px]"} `}
    >
      <div ref={ref} className="flex h-full w-full flex-1 flex-col">
        <AdminSidebarDropdown />
        <AdminSidebarMenu />
        <AdminSidebarHelpSection />
      </div>
    </div>
  );
};
