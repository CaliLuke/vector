/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useLocation, useRouter } from "@tanstack/react-router";
import type { SyntheticEvent } from "react";
import React, { useRef } from "react";
import { observer } from "mobx-react";

import { Info, SquareUser } from "lucide-react";
// plane package imports
import {
  MODULE_STATUS,
  PROGRESS_STATE_GROUPS_DETAILS,
  EUserPermissions,
  EUserPermissionsLevel,
} from "@plane/constants";
import { WorkItemsIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { IModule } from "@plane/types";
import { Card, FavoriteStar, LinearProgressIndicator } from "@plane/ui";
import { getDate, renderFormattedPayloadDate, generateQueryParams } from "@plane/utils";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { ButtonAvatars } from "@/components/dropdowns/member/avatar";
import { ModuleQuickActions } from "@/components/modules";
import { ModuleStatusDropdown } from "@/components/modules/module-status-dropdown";
import { useModuleActions } from "@/components/modules/use-module-actions";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useModule } from "@/hooks/store/use-module";
import { useUserPermissions } from "@/hooks/store/user";

import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  moduleId: string;
};

export const ModuleCardItem = observer(function ModuleCardItem(props: Props) {
  const { moduleId } = props;
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
  const { allowPermissions } = useUserPermissions();
  const { getModuleById } = useModule();
  const { getUserDetails } = useMember();
  const { handleAddToFavorites, handleRemoveFromFavorites, handleModuleDetailsChange } = useModuleActions({
    moduleId,
    workspaceSlug,
    projectId,
  });
  // derived values
  const moduleDetails = getModuleById(moduleId);
  const isEditingAllowed = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT
  );
  const isDisabled = !isEditingAllowed || !!moduleDetails?.archived_at;
  const renderIcon = Boolean(moduleDetails?.start_date) || Boolean(moduleDetails?.target_date);

  const { isMobile } = usePlatformOS();
  const handleEventPropagation = (e: SyntheticEvent<HTMLDivElement>) => {
    e.stopPropagation();
    e.preventDefault();
  };

  const openModuleOverview = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();

    const query = generateQueryParams(searchParams, ["peekModule"]);
    if (searchParams.has("peekModule") && searchParams.get("peekModule") === moduleId) {
      void router.navigate({ href: `${pathname}?${query}` });
    } else {
      void router.navigate({ href: `${pathname}?${query && `${query}&`}peekModule=${moduleId}` });
    }
  };

  if (!moduleDetails) return null;

  const moduleTotalIssues =
    moduleDetails.backlog_issues +
    moduleDetails.unstarted_issues +
    moduleDetails.started_issues +
    moduleDetails.completed_issues +
    moduleDetails.cancelled_issues;

  const moduleCompletedIssues = moduleDetails.completed_issues;

  // const areYearsEqual = startDate.getFullYear() === endDate.getFullYear();

  const moduleStatus = MODULE_STATUS.find((status) => status.value === moduleDetails.status);

  const issueCount = moduleDetails
    ? !moduleTotalIssues || moduleTotalIssues === 0
      ? `0 work items`
      : moduleTotalIssues === moduleCompletedIssues
        ? `${moduleTotalIssues} Work item${moduleTotalIssues > 1 ? `s` : ``}`
        : `${moduleCompletedIssues}/${moduleTotalIssues} Work items`
    : `0 work items`;

  const moduleLeadDetails = moduleDetails.lead_id ? getUserDetails(moduleDetails.lead_id) : undefined;

  const progressIndicatorData = PROGRESS_STATE_GROUPS_DETAILS.map((group, index) => ({
    id: index,
    name: group.title,
    value: moduleTotalIssues > 0 ? (moduleDetails[group.key as keyof IModule] as number) : 0,
    color: group.color,
  }));

  return (
    <div className="relative" data-prevent-progress>
      <Link
        ref={parentRef}
        to="/$workspaceSlug/projects/$projectId/modules/$moduleId/"
        params={{ workspaceSlug: workspaceSlug, projectId: moduleDetails.project_id, moduleId: moduleDetails.id }}
      >
        <Card>
          <div>
            <div className="flex items-center justify-between gap-2">
              <Tooltip tooltipContent={moduleDetails.name} position="top" isMobile={isMobile}>
                <span className="truncate text-14 font-medium">{moduleDetails.name}</span>
              </Tooltip>
              <div className="flex items-center gap-2" onClick={handleEventPropagation}>
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
                  <ButtonAvatars showTooltip={false} userIds={moduleLeadDetails?.id} />
                </span>
              ) : (
                <Tooltip tooltipContent="No lead">
                  <SquareUser className="mx-1 h-4 w-4 text-tertiary" />
                </Tooltip>
              )}
            </div>
            <LinearProgressIndicator size="lg" data={progressIndicatorData} />
            <div className="flex items-center justify-between py-0.5" onClick={handleEventPropagation}>
              <DateRangeDropdown
                buttonContainerClassName={`h-6 w-full flex ${isDisabled ? "cursor-not-allowed" : "cursor-pointer"} items-center gap-1.5 text-tertiary border-[0.5px] border-strong rounded-sm text-11`}
                buttonVariant="transparent-with-text"
                className="h-7"
                value={{
                  from: getDate(moduleDetails.start_date),
                  to: getDate(moduleDetails.target_date),
                }}
                onSelect={(val) => {
                  handleModuleDetailsChange({
                    start_date: val?.from ? renderFormattedPayloadDate(val.from) : null,
                    target_date: val?.to ? renderFormattedPayloadDate(val.to) : null,
                  });
                }}
                placeholder={{
                  from: "Start date",
                  to: "End date",
                }}
                disabled={isDisabled}
                hideIcon={{ from: renderIcon ?? true, to: renderIcon }}
              />
            </div>
          </div>
        </Card>
      </Link>
      <div className="absolute right-4 bottom-[18px] flex items-center gap-1.5">
        {isEditingAllowed && (
          <FavoriteStar
            onClick={(e) => {
              if (moduleDetails.is_favorite) handleRemoveFromFavorites(e);
              else handleAddToFavorites(e);
            }}
            selected={!!moduleDetails.is_favorite}
          />
        )}
        {workspaceSlug && projectId && (
          <ModuleQuickActions
            parentRef={parentRef}
            moduleId={moduleId}
            projectId={projectId.toString()}
            workspaceSlug={workspaceSlug.toString()}
          />
        )}
      </div>
    </div>
  );
});
