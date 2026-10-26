/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation, useRouter } from "@tanstack/react-router";
import type { MouseEvent } from "react";
import React, { useState } from "react";

import { Eye, ArrowRight, CalendarDays } from "lucide-react";
// plane imports
import { EUserPermissions, IS_FAVORITE_MENU_OPEN } from "@plane/constants";
import { useLocalStorage } from "@plane/hooks";
import { useTranslation } from "@plane/i18n";
import { TransferIcon, WorkItemsIcon, MembersPropertyIcon } from "@plane/propel/icons";
import { setPromiseToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import type { ICycle, TCycleGroups, IUserLite } from "@plane/types";
import { Avatar, AvatarGroup } from "@plane/ui";
import { getDate } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { AuthorizedFavoriteStar } from "@/components/common/authorized-favorite-star";
import { ButtonAvatars } from "@/components/dropdowns/member/avatar";
import { MergedDateDisplay } from "@/components/dropdowns/merged-date";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useFavorites } from "@/hooks/use-favorites";

import { usePlatformOS } from "@/hooks/use-platform-os";
import { useTimeZoneConverter } from "@/hooks/use-timezone-converter";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
// local imports
import { CycleQuickActions } from "../quick-actions";
import { TransferIssuesModal } from "../transfer-issues-modal";
import { getCycleOverviewHref } from "./cycle-overview-navigation";

type Props = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
  cycleDetails: ICycle;
  parentRef: React.RefObject<HTMLDivElement | null>;
  isActive?: boolean;
};

type CycleTimezone = ReturnType<typeof useTimeZoneConverter>;
type Translate = ReturnType<typeof useTranslation>["t"];

function CycleDateSummary({
  cycle,
  isActive,
  projectUTCOffset,
  timezone,
  t,
}: {
  cycle: ICycle;
  isActive: boolean;
  projectUTCOffset: string | undefined;
  timezone: CycleTimezone;
  t: Translate;
}) {
  const [today] = useState(() => new Date());
  if (isActive)
    return (
      <div className="flex gap-2">
        <Tooltip
          tooltipContent={
            <span className="flex gap-1">
              {timezone.renderFormattedDateInUserTimezone(cycle.start_date ?? "")}
              <ArrowRight className="my-auto h-3 w-3 flex-shrink-0" />
              {timezone.renderFormattedDateInUserTimezone(cycle.end_date ?? "")}
            </span>
          }
          disabled={!timezone.isProjectTimeZoneDifferent()}
          tooltipHeading={t("project_cycles.in_your_timezone")}
        >
          <div className="flex items-center gap-1 text-11 font-medium text-tertiary">
            <CalendarDays className="my-auto h-3 w-3 flex-shrink-0" />
            <MergedDateDisplay startDate={cycle.start_date} endDate={cycle.end_date} />
          </div>
        </Tooltip>
        {projectUTCOffset && (
          <span className="cursor-default rounded-md bg-layer-1 px-2 py-1 text-11 text-tertiary">
            {projectUTCOffset}
          </span>
        )}
      </div>
    );
  if (!cycle.start_date) return null;
  return (
    <DateRangeDropdown
      buttonVariant="transparent-with-text"
      buttonContainerClassName="h-6 w-full cursor-auto flex items-center gap-1.5 text-tertiary rounded-sm text-11 [&>div]:hover:bg-transparent"
      buttonClassName="p-0"
      minDate={today}
      value={{ from: getDate(cycle.start_date), to: getDate(cycle.end_date) }}
      placeholder={{ from: t("project_cycles.start_date"), to: t("project_cycles.end_date") }}
      showTooltip={timezone.isProjectTimeZoneDifferent()}
      customTooltipHeading={t("project_cycles.in_your_timezone")}
      customTooltipContent={
        <span className="flex gap-1">
          {timezone.renderFormattedDateInUserTimezone(cycle.start_date ?? "")}
          <ArrowRight className="my-auto h-3 w-3 flex-shrink-0" />
          {timezone.renderFormattedDateInUserTimezone(cycle.end_date ?? "")}
        </span>
      }
      mergeDates
      required={cycle.status !== "draft"}
      disabled
      hideIcon={{ from: false, to: false }}
    />
  );
}

function CycleMembersSummary({
  cycle,
  isActive,
  workspaceSlug,
  isMobile,
  createdBy,
  assignees,
}: {
  cycle: ICycle;
  isActive: boolean;
  workspaceSlug: string;
  isMobile: boolean;
  createdBy: IUserLite | undefined;
  assignees: IUserLite[];
}) {
  return (
    <>
      {createdBy && <ButtonAvatars showTooltip={false} userIds={createdBy.id} workspaceSlug={workspaceSlug} />}
      {!isActive && (
        <Tooltip tooltipContent={`${cycle.assignee_ids?.length ?? 0} Members`} isMobile={isMobile}>
          <div className="flex w-min cursor-default items-center justify-center">
            {assignees.length > 0 ? (
              <AvatarGroup showTooltip={false}>
                {assignees.map((member) => (
                  <Avatar key={member.id} name={member.display_name} src={getFileURL(member.avatar_url ?? "")} />
                ))}
              </AvatarGroup>
            ) : (
              <MembersPropertyIcon className="h-4 w-4 text-tertiary" />
            )}
          </div>
        </Tooltip>
      )}
    </>
  );
}

export const CycleListItemAction = function CycleListItemAction(props: Props) {
  const { workspaceSlug, projectId, cycleId, cycleDetails, parentRef, isActive = false } = props;
  // router
  //states
  const [transferIssuesModal, setTransferIssuesModal] = useState(false);
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const timezone = useTimeZoneConverter(projectId);
  // router
  const router = useRouter();
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const pathname = useLocation({ select: (location) => location.pathname });
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { favorites, favoritesByEntity, toggleFavorite } = useFavorites(workspaceSlug);
  const isCycleFavorite = favoritesByEntity?.has(`cycle:${cycleId}`) ?? false;

  // local storage
  const { setValue: toggleFavoriteMenu, storedValue: isFavoriteMenuOpen } = useLocalStorage<boolean>(
    IS_FAVORITE_MENU_OPEN,
    false
  );

  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const getUserDetails = (userId: string) => membersById?.get(userId)?.member;

  // derived values
  const cycleStatus = cycleDetails.status ? (cycleDetails.status.toLocaleLowerCase() as TCycleGroups) : "draft";

  const showIssueCount = cycleStatus === "draft" || cycleStatus === "upcoming";

  const transferableIssuesCount = cycleDetails
    ? cycleDetails.total_issues - (cycleDetails.cancelled_issues + cycleDetails.completed_issues)
    : 0;

  const showTransferIssues = transferableIssuesCount > 0 && cycleStatus === "completed";

  const projectUTCOffset = timezone.getProjectUTCOffset();

  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  // handlers
  const handleAddToFavorites = (e: MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    if (!workspaceSlug || !projectId) return;

    const addToFavoritePromise = toggleFavorite("cycle", cycleId, projectId.toString()).then(
      // oxlint-disable-next-line promise/always-return
      () => {
        if (!isFavoriteMenuOpen) toggleFavoriteMenu(true);
      }
    );

    setPromiseToast(addToFavoritePromise, {
      loading: t("project_cycles.action.favorite.loading"),
      success: {
        title: t("project_cycles.action.favorite.success.title"),
        message: () => t("project_cycles.action.favorite.success.description"),
      },
      error: {
        title: t("project_cycles.action.favorite.failed.title"),
        message: () => t("project_cycles.action.favorite.failed.description"),
      },
    });
  };

  const handleRemoveFromFavorites = (e: MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    if (!workspaceSlug || !projectId) return;

    const removeFromFavoritePromise = toggleFavorite("cycle", cycleId, projectId.toString());

    setPromiseToast(removeFromFavoritePromise, {
      loading: t("project_cycles.action.unfavorite.loading"),
      success: {
        title: t("project_cycles.action.unfavorite.success.title"),
        message: () => t("project_cycles.action.unfavorite.success.description"),
      },
      error: {
        title: t("project_cycles.action.unfavorite.failed.title"),
        message: () => t("project_cycles.action.unfavorite.failed.description"),
      },
    });
  };

  const createdByDetails = cycleDetails.created_by ? getUserDetails(cycleDetails.created_by) : undefined;
  const assignees = (cycleDetails.assignee_ids ?? [])
    .map((id) => getUserDetails(id))
    .filter((member): member is IUserLite => member !== undefined);

  // handlers
  const openCycleOverview = (e: MouseEvent<HTMLButtonElement | HTMLAnchorElement>) => {
    e.preventDefault();
    e.stopPropagation();

    void router.navigate({ href: getCycleOverviewHref(pathname, searchParams, cycleId) });
  };

  return (
    <>
      <TransferIssuesModal
        handleClose={() => setTransferIssuesModal(false)}
        isOpen={transferIssuesModal}
        cycleId={cycleId.toString()}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      />
      <button
        onClick={openCycleOverview}
        className={`z-[1] flex flex-shrink-0 gap-1 text-11 text-accent-secondary ${isMobile || (isActive && !searchParams.has("peekCycle")) ? "flex" : "hidden group-hover:flex"}`}
      >
        <Eye className="my-auto h-4 w-4 text-accent-secondary" />
        <span>{t("project_cycles.more_details")}</span>
      </button>
      {showIssueCount && (
        <div className="flex items-center gap-1">
          <WorkItemsIcon className="h-4 w-4 text-tertiary" />
          <span className="text-11 text-tertiary">{cycleDetails.total_issues}</span>
        </div>
      )}
      {showTransferIssues && (
        // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
        <div
          className="flex h-6 cursor-pointer items-center gap-1 px-2 text-accent-secondary"
          onClick={() => {
            setTransferIssuesModal(true);
          }}
        >
          <TransferIcon className="w-4 fill-accent-primary" />
          <span>{t("project_cycles.transfer_work_items", { count: transferableIssuesCount })}</span>
        </div>
      )}
      <CycleDateSummary
        cycle={cycleDetails}
        isActive={isActive}
        projectUTCOffset={projectUTCOffset}
        timezone={timezone}
        t={t}
      />
      <CycleMembersSummary
        cycle={cycleDetails}
        isActive={isActive}
        workspaceSlug={workspaceSlug}
        isMobile={isMobile}
        createdBy={createdByDetails}
        assignees={assignees}
      />
      <AuthorizedFavoriteStar
        canFavorite={isEditingAllowed}
        isReady={favorites !== undefined}
        disabled={!!cycleDetails.archived_at}
        onAdd={handleAddToFavorites}
        onRemove={handleRemoveFromFavorites}
        selected={isCycleFavorite}
        preventDefault
        stopPropagation
      />
      <div className="hidden md:block">
        <CycleQuickActions
          cycle={cycleDetails}
          parentRef={parentRef}
          cycleId={cycleId}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
        />
      </div>
    </>
  );
};
