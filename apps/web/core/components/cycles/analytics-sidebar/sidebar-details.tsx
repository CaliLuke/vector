/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import isEmpty from "lodash-es/isEmpty.js";

import { SquareUser } from "lucide-react";
// plane types
import { EEstimateSystem } from "@plane/types";
import { useTranslation } from "@plane/i18n";
import { MembersPropertyIcon, WorkItemsIcon } from "@plane/propel/icons";
import type { ICycle } from "@plane/types";
// plane ui
import { Avatar, AvatarGroup, TextArea } from "@plane/ui";
// helpers
import { getFileURL } from "@plane/utils/file";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
// plane web constants

type Props = {
  workspaceSlug: string;
  projectId: string;
  cycleDetails: ICycle;
};

function formatCycleCount(
  completed: number | undefined,
  total: number | undefined,
  translate: (key: string) => string
) {
  if (total === 0) return `0 ${translate("common.work_item")}`;
  return `${completed ?? 0}/${total ?? 0}`;
}

function getCycleIssueCount(cycle: ICycle, isCompleted: boolean, translate: (key: string) => string) {
  const snapshot = cycle.progress_snapshot;
  if (isCompleted && !isEmpty(snapshot))
    return formatCycleCount(
      snapshot.total_issues === 0 ? 0 : snapshot.completed_issues,
      snapshot.total_issues,
      translate
    );
  return formatCycleCount(cycle.completed_issues, cycle.total_issues, translate);
}

function hasCycleEstimatePoints(
  cycle: ICycle,
  estimateType: ReturnType<typeof useProjectEstimates>["activeEstimate"] | undefined
) {
  if (isEmpty(cycle.progress_snapshot || {})) return estimateType?.type === EEstimateSystem.POINTS;
  return !isEmpty(cycle.progress_snapshot?.estimate_distribution || {});
}

function getCycleEstimatePointCount(cycle: ICycle, isCompleted: boolean, translate: (key: string) => string) {
  const snapshot = cycle.progress_snapshot;
  if (isCompleted && !isEmpty(snapshot))
    return formatCycleCount(
      snapshot.total_issues === 0 ? 0 : snapshot.completed_estimate_points,
      snapshot.total_estimate_points,
      translate
    );
  return formatCycleCount(cycle.completed_estimate_points, cycle.total_estimate_points, translate);
}

function CycleEstimatePointsRow(props: { visible: boolean; count: string; translate: (key: string) => string }) {
  const { visible, count, translate } = props;
  if (!visible) return null;

  return (
    <div className="flex items-center justify-start gap-1">
      <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
        <WorkItemsIcon className="h-4 w-4" />
        <span className="text-14">{translate("points")}</span>
      </div>
      <div className="flex w-3/5 items-center">
        <span className="px-1.5 text-13 text-tertiary">{count}</span>
      </div>
    </div>
  );
}

export const CycleSidebarDetails = function CycleSidebarDetails(props: Props) {
  const { workspaceSlug, projectId, cycleDetails } = props;
  // hooks
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { isEnabled: areEstimateEnabled, activeEstimate: estimateDetails } = useProjectEstimates(
    workspaceSlug,
    projectId
  );
  const { t } = useTranslation();

  const cycleStatus = cycleDetails?.status?.toLocaleLowerCase();
  const isCompleted = cycleStatus === "completed";

  const estimateType = areEstimateEnabled ? estimateDetails : undefined;
  const cycleOwnerDetails = memberById(cycleDetails.owned_by_id)?.member;
  const issueCount = getCycleIssueCount(cycleDetails, isCompleted, t);
  const isEstimatePointValid = hasCycleEstimatePoints(cycleDetails, estimateType);
  const issueEstimatePointCount = getCycleEstimatePointCount(cycleDetails, isCompleted, t);
  return (
    <div className="flex w-full flex-col gap-5">
      {cycleDetails?.description && (
        <TextArea
          className="ring-none !m-0 max-h-max w-full resize-none !border-0 bg-transparent !p-0 text-13 leading-5 text-secondary outline-none"
          value={cycleDetails.description}
          disabled
        />
      )}

      <div className="flex flex-col gap-5 pt-2.5 pb-6">
        <div className="flex items-center justify-start gap-1">
          <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
            <SquareUser className="h-4 w-4" />
            <span className="text-14">{t("lead")}</span>
          </div>
          <div className="flex w-3/5 items-center rounded-xs">
            <div className="flex items-center gap-2.5">
              <Avatar name={cycleOwnerDetails?.display_name} src={getFileURL(cycleOwnerDetails?.avatar_url ?? "")} />
              <span className="text-13 text-secondary">{cycleOwnerDetails?.display_name}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-start gap-1">
          <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
            <MembersPropertyIcon className="h-4 w-4" />
            <span className="text-14">{t("members")}</span>
          </div>
          <div className="flex w-3/5 items-center rounded-xs">
            <div className="flex items-center gap-2.5">
              {cycleDetails?.assignee_ids && cycleDetails.assignee_ids.length > 0 ? (
                <>
                  <AvatarGroup showTooltip>
                    {cycleDetails.assignee_ids.map((member) => {
                      const memberDetails = memberById(member)?.member;
                      return (
                        <Avatar
                          key={member}
                          name={memberDetails?.display_name ?? ""}
                          src={getFileURL(memberDetails?.avatar_url ?? "")}
                          showTooltip={false}
                        />
                      );
                    })}
                  </AvatarGroup>
                </>
              ) : (
                <span className="px-1.5 text-13 text-tertiary">{t("no_assignee")}</span>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center justify-start gap-1">
          <div className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
            <WorkItemsIcon className="h-4 w-4" />
            <span className="text-14">{t("work_items")}</span>
          </div>
          <div className="flex w-3/5 items-center">
            <span className="px-1.5 text-13 text-tertiary">{issueCount}</span>
          </div>
        </div>

        <CycleEstimatePointsRow
          visible={isEstimatePointValid && !isCompleted}
          count={issueEstimatePointCount}
          translate={t}
        />
      </div>
    </div>
  );
};
