/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";
import React from "react";

import { Tooltip } from "@plane/propel/tooltip";
import { useSidebarActions } from "@/providers/sidebar-state";
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  href: string;
  title: string;
  icon: React.ReactNode;
};

export function FavoriteItemTitle(props: Props) {
  const { href, title, icon } = props;
  // store hooks
  const { toggleSidebar } = useSidebarActions();
  const { isMobile } = usePlatformOS();

  const handleOnClick = () => {
    if (isMobile) toggleSidebar();
  };

  return (
    <Tooltip tooltipContent={title} isMobile={isMobile} position="right" className="ml-8">
      <Link to={href} className="flex w-full items-center gap-1.5 truncate" draggable onClick={handleOnClick}>
        <span className="flex size-5 items-center justify-center">{icon}</span>
        <span className="flex-1 truncate text-13 leading-5 font-medium">{title}</span>
      </Link>
    </Tooltip>
  );
}
