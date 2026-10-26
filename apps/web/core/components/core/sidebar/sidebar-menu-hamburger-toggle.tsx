/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PanelRight } from "lucide-react";
import { useSidebarActions } from "@/providers/sidebar-state";

export function SidebarHamburgerToggle() {
  // store hooks
  const { toggleSidebar } = useSidebarActions();

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    toggleSidebar();
  };

  return (
    <button
      type="button"
      className="group grid size-7 flex-shrink-0 place-items-center rounded-sm bg-surface-2 transition-all hover:bg-layer-1"
      onClick={handleClick}
    >
      <PanelRight className="size-3.5 text-secondary transition-all group-hover:text-primary" />
    </button>
  );
}
