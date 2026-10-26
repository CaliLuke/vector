/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PanelLeft } from "lucide-react";
// hooks
import { useSidebarActions, useSidebarPeek } from "@/providers/sidebar-state";
import { IconButton } from "@plane/propel/icon-button";

export function AppSidebarToggleButton() {
  // store hooks
  const sidebarPeek = useSidebarPeek();
  const { toggleSidebar, toggleSidebarPeek } = useSidebarActions();

  return (
    <IconButton
      size="base"
      variant="ghost"
      icon={PanelLeft}
      onClick={() => {
        if (sidebarPeek) toggleSidebarPeek(false);
        toggleSidebar();
      }}
    />
  );
}
