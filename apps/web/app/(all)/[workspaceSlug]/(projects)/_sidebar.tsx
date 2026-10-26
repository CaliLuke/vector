/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation } from "@tanstack/react-router";
import { useState } from "react";
// plane imports

import { SIDEBAR_WIDTH } from "@plane/constants";
import { useLocalStorage } from "@plane/hooks";
// components
import { ResizableSidebar } from "@/components/sidebar/resizable-sidebar";
// hooks
import {
  useSidebarActions,
  useSidebarCollapsed,
  useSidebarDropdownOpen,
  useSidebarPeek,
  useExtendedSidebarOpen,
} from "@/providers/sidebar-state";
// local imports
import { ExtendedAppSidebar } from "./extended-sidebar";
import { AppSidebar } from "./sidebar";

export function ProjectAppSidebar() {
  // store hooks
  const sidebarCollapsed = useSidebarCollapsed();
  const sidebarPeek = useSidebarPeek();
  const isExtendedSidebarOpened = useExtendedSidebarOpen();
  const isAnySidebarDropdownOpen = useSidebarDropdownOpen();
  const { toggleSidebar, toggleSidebarPeek } = useSidebarActions();
  const { storedValue, setValue } = useLocalStorage("sidebarWidth", SIDEBAR_WIDTH);
  // states
  const [sidebarWidth, setSidebarWidth] = useState<number>(storedValue ?? SIDEBAR_WIDTH);
  // routes
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug/_projects" });
  const pathname = useLocation({ select: (location) => location.pathname });
  // derived values
  const isAnyExtendedSidebarOpen = isExtendedSidebarOpened;

  const isNotificationsPath = pathname.includes(`/${workspaceSlug}/notifications`);

  // handlers
  const handleWidthChange = (width: number) => setValue(width);

  if (isNotificationsPath) return null;

  return (
    <>
      <ResizableSidebar
        showPeek={sidebarPeek}
        defaultWidth={storedValue ?? 250}
        width={sidebarWidth}
        setWidth={setSidebarWidth}
        defaultCollapsed={sidebarCollapsed}
        peekDuration={1500}
        onWidthChange={handleWidthChange}
        onCollapsedChange={toggleSidebar}
        isCollapsed={sidebarCollapsed}
        toggleCollapsed={toggleSidebar}
        togglePeek={toggleSidebarPeek}
        extendedSidebar={
          <>
            <ExtendedAppSidebar />
          </>
        }
        isAnyExtendedSidebarExpanded={isAnyExtendedSidebarOpen}
        isAnySidebarDropdownOpen={isAnySidebarDropdownOpen}
      >
        <AppSidebar />
      </ResizableSidebar>
    </>
  );
}
