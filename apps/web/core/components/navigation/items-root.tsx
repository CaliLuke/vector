/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { AppSidebarItem } from "@/components/sidebar/sidebar-item";
import { useAppRailItems } from "./use-app-rail-items";

type Props = {
  showLabel?: boolean;
};

export function AppSidebarItemsRoot({ showLabel = true }: Props) {
  const dockItems = useAppRailItems();
  return (
    <>
      {dockItems
        .filter((item) => item.shouldRender)
        .map((item) => (
          <AppSidebarItem key={item.label} item={{ ...item, showLabel }} variant="link" />
        ))}
    </>
  );
}
