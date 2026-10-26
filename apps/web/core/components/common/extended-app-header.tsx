/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import type { ReactNode } from "react";

// components
import { AppSidebarToggleButton } from "@/components/sidebar/sidebar-toggle-button";
// hooks
import { useSidebarCollapsed } from "@/providers/sidebar-state";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";

export function ExtendedAppHeader(props: { header: ReactNode }) {
  const { header } = props;
  // params
  const { projectId, workItem } = useParams({ strict: false });
  // preferences
  const { preferences: projectPreferences } = useProjectNavigationPreferences();
  // store hooks
  const sidebarCollapsed = useSidebarCollapsed();
  // derived values
  const shouldShowSidebarToggleButton = projectPreferences.navigationMode === "ACCORDION" || (!projectId && !workItem);

  return (
    <>
      {sidebarCollapsed && shouldShowSidebarToggleButton && <AppSidebarToggleButton />}
      <div className="w-full">{header}</div>
    </>
  );
}
