/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useRouter } from "@tanstack/react-router";
import type { MouseEvent } from "react";
import { useRef } from "react";

import { CheckIcon } from "@plane/propel/icons";
// plane imports
import type { TCycleGroups, ICycle } from "@plane/types";
import { CircularProgressIndicator } from "@plane/ui";
// components
import { calculateCycleProgress } from "@plane/utils/cycle";
import { ListItem } from "@/components/core/list";
// hooks

import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { CycleQuickActions } from "../quick-actions";
import { CycleListItemAction } from "./cycle-list-item-action";
import { getCycleOverviewHref } from "./cycle-overview-navigation";

type TCyclesListItem = {
  cycle: ICycle;
  handleEditCycle?: () => void;
  handleDeleteCycle?: () => void;
  handleAddToFavorites?: () => void;
  handleRemoveFromFavorites?: () => void;
  workspaceSlug: string;
  projectId: string;
  className?: string;
};

export const CyclesListItem = function CyclesListItem(props: TCyclesListItem) {
  const { cycle: cycleDetails, workspaceSlug, projectId, className = "" } = props;
  // refs
  const parentRef = useRef(null);
  // router
  const router = useRouter();
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const pathname = useLocation({ select: (location) => location.pathname });
  // hooks
  const { isMobile } = usePlatformOS();
  // store hooks
  const cycleId = cycleDetails.id;

  // derived values

  if (!cycleDetails) return null;

  // computed
  // TODO: change this logic once backend fix the response
  const cycleStatus = cycleDetails.status ? (cycleDetails.status.toLocaleLowerCase() as TCycleGroups) : "draft";
  const isActive = cycleStatus === "current";

  // handlers
  const openCycleOverview = (e: MouseEvent<HTMLButtonElement | HTMLAnchorElement>) => {
    e.preventDefault();
    e.stopPropagation();

    void router.navigate({ href: getCycleOverviewHref(pathname, searchParams, cycleId) });
  };

  // handlers
  const handleArchivedCycleClick = (e: MouseEvent<HTMLAnchorElement>) => {
    openCycleOverview(e);
  };

  const handleItemClick = cycleDetails.archived_at ? handleArchivedCycleClick : undefined;

  const progress = calculateCycleProgress(cycleDetails);

  return (
    <ListItem
      title={cycleDetails?.name ?? ""}
      itemLink={`/${workspaceSlug}/projects/${projectId}/cycles/${cycleDetails.id}`}
      onItemClick={handleItemClick}
      className={className}
      prependTitleElement={
        <CircularProgressIndicator size={30} percentage={progress} strokeWidth={3}>
          {progress === 100 ? (
            <CheckIcon className="h-3 w-3 stroke-2" />
          ) : (
            <span className="text-9 text-primary">{`${progress}%`}</span>
          )}
        </CircularProgressIndicator>
      }
      actionableItems={
        <CycleListItemAction
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          cycleId={cycleId}
          cycleDetails={cycleDetails}
          parentRef={parentRef}
          isActive={isActive}
        />
      }
      quickActionElement={
        <div className="block md:hidden">
          <CycleQuickActions
            cycle={cycleDetails}
            parentRef={parentRef}
            cycleId={cycleId}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
        </div>
      }
      isMobile={isMobile}
      parentRef={parentRef}
      isSidebarOpen={searchParams.has("peekCycle")}
    />
  );
};
