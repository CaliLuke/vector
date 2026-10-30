/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useLocation, useRouter } from "@tanstack/react-router";
import React, { useRef } from "react";

import { Info, SquareUser } from "lucide-react";
// plane package imports
import { MODULE_STATUS, PROGRESS_STATE_GROUPS_DETAILS } from "@plane/constants";
import { WorkItemsIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { IModule } from "@plane/types";
import { Card, LinearProgressIndicator } from "@plane/ui";
// components
import { ButtonAvatars } from "@/components/dropdowns/member/avatar";
import { ModuleQuickActions } from "@/components/modules";
import { ModuleDateRangeField } from "@/components/modules/module-date-range-field";
import { ModuleStatusDropdown } from "@/components/modules/module-status-dropdown";
import { useModuleActions } from "@/components/modules/use-module-actions";
import { AuthorizedFavoriteStar } from "@/components/common/authorized-favorite-star";
// hooks

import { usePlatformOS } from "@/hooks/use-platform-os";
import { getModulePeekHref } from "./module-peek-navigation";

type Props = {
  module: IModule;
};

function getModuleIssueCount(total: number, completed: number) {
  if (total === 0) return "0 work items";
  if (total === completed) return `${total} Work item${total > 1 ? "s" : ""}`;
  return `${completed}/${total} Work items`;
}

function getModuleProgressIndicatorData(module: IModule, total: number) {
  return PROGRESS_STATE_GROUPS_DETAILS.map((group, index) => ({
    id: index,
    name: group.title,
    value: total > 0 ? (module[group.key as keyof IModule] as number) : 0,
    color: group.color,
  }));
}

function preventCardNavigationFromControl(event: React.MouseEvent<HTMLAnchorElement>) {
  if (event.target instanceof Element && event.target.closest("[data-prevent-link-navigation]")) {
    event.preventDefault();
  }
}

export const ModuleCardItem = function ModuleCardItem(props: Props) {
  const { module: moduleDetails } = props;
  const moduleId = moduleDetails.id;
  // refs
  const parentRef = useRef(null);
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId",
  });
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const pathname = useLocation({ select: (location) => location.pathname });
  // store hooks
  const {
    handleAddToFavorites,
    handleRemoveFromFavorites,
    handleModuleDetailsChange,
    isFavorite,
    isFavoritesReady,
    isEditingAllowed,
    getUserDetails,
  } = useModuleActions({
    moduleId,
    workspaceSlug,
    projectId,
  });
  // derived values
  const isDisabled = !isEditingAllowed || !!moduleDetails?.archived_at;

  const { isMobile } = usePlatformOS();
  const openModuleOverview = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();

    void router.navigate({ href: getModulePeekHref(pathname, searchParams.toString(), moduleId) });
  };

  const moduleTotalIssues =
    moduleDetails.backlog_issues +
    moduleDetails.unstarted_issues +
    moduleDetails.started_issues +
    moduleDetails.completed_issues +
    moduleDetails.cancelled_issues;

  const moduleCompletedIssues = moduleDetails.completed_issues;

  // const areYearsEqual = startDate.getFullYear() === endDate.getFullYear();

  const moduleStatus = MODULE_STATUS.find((status) => status.value === moduleDetails.status);

  const issueCount = getModuleIssueCount(moduleTotalIssues, moduleCompletedIssues);

  const moduleLeadDetails = moduleDetails.lead_id ? getUserDetails(moduleDetails.lead_id) : undefined;

  const progressIndicatorData = getModuleProgressIndicatorData(moduleDetails, moduleTotalIssues);

  return (
    <div className="relative" data-prevent-progress>
      <Link
        ref={parentRef}
        onClickCapture={preventCardNavigationFromControl}
        to="/$workspaceSlug/projects/$projectId/modules/$moduleId/"
        params={{ workspaceSlug, projectId: moduleDetails.project_id, moduleId: moduleDetails.id }}
      >
        <Card>
          <div>
            <div className="flex items-center justify-between gap-2">
              <Tooltip tooltipContent={moduleDetails.name} position="top" isMobile={isMobile}>
                <span className="truncate text-14 font-medium">{moduleDetails.name}</span>
              </Tooltip>
              <div className="flex items-center gap-2" data-prevent-link-navigation>
                {moduleStatus && (
                  <ModuleStatusDropdown
                    isDisabled={isDisabled}
                    moduleDetails={moduleDetails}
                    handleModuleDetailsChange={handleModuleDetailsChange}
                  />
                )}
                <button onClick={openModuleOverview}>
                  <Info className="h-4 w-4 text-placeholder" />
                </button>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1.5 text-secondary">
                <WorkItemsIcon className="h-4 w-4 text-tertiary" />
                <span className="text-11 text-tertiary">{issueCount ?? "0 Work item"}</span>
              </div>
              {moduleLeadDetails ? (
                <span className="cursor-default">
                  <ButtonAvatars showTooltip={false} userIds={moduleLeadDetails?.id} workspaceSlug={workspaceSlug} />
                </span>
              ) : (
                <Tooltip tooltipContent="No lead">
                  <SquareUser className="mx-1 h-4 w-4 text-tertiary" />
                </Tooltip>
              )}
            </div>
            <LinearProgressIndicator size="lg" data={progressIndicatorData} />
            <div className="flex items-center justify-between py-0.5" data-prevent-link-navigation>
              <ModuleDateRangeField
                moduleDetails={moduleDetails}
                isDisabled={isDisabled}
                handleModuleDetailsChange={handleModuleDetailsChange}
                placeholder={{ from: "Start date", to: "End date" }}
              />
            </div>
          </div>
        </Card>
      </Link>
      <div className="absolute right-4 bottom-[18px] flex items-center gap-1.5">
        <AuthorizedFavoriteStar
          canFavorite={isEditingAllowed}
          isReady={isFavoritesReady}
          onAdd={handleAddToFavorites}
          onRemove={handleRemoveFromFavorites}
          selected={isFavorite}
        />
        {workspaceSlug && projectId && (
          <ModuleQuickActions
            module={moduleDetails}
            parentRef={parentRef}
            moduleId={moduleId}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
        )}
      </div>
    </div>
  );
};
