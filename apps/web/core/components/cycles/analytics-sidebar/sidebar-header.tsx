/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ArrowRight } from "lucide-react";
import { useState } from "react";
// Vector Imports
import { CYCLE_STATUS, EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { ChevronRightIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ICycle } from "@plane/types";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useQueryClient } from "@tanstack/react-query";
import { captureParentAccess } from "@/queries/cycle-module-scope";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useTimeZoneConverter } from "@/hooks/use-timezone-converter";
// services

type Props = {
  workspaceSlug: string;
  projectId: string;
  cycleDetails: ICycle;
  handleClose: () => void;
  isArchived?: boolean;
};

export const CycleSidebarHeader = function CycleSidebarHeader(props: Props) {
  const { workspaceSlug, projectId, cycleDetails, handleClose, isArchived = false } = props;
  const [minDate] = useState(() => new Date());
  // hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const catalogue = useProjectCycles(workspaceSlug, projectId),
    client = useQueryClient();
  const current = useCommittedScope(JSON.stringify([catalogue.scope, cycleDetails.id]));
  const access = captureParentAccess(client, catalogue.scope);
  const { t } = useTranslation();
  const { renderFormattedDateInUserTimezone, getProjectUTCOffset } = useTimeZoneConverter(projectId);

  // derived values
  const projectUTCOffset = getProjectUTCOffset();

  const cycleStatus = cycleDetails?.status?.toLocaleLowerCase();
  const isCompleted = cycleStatus === "completed";

  const currentCycle = CYCLE_STATUS.find((status) => status.value === cycleStatus);

  const datesMutation = catalogue.mutation;
  const updateDates = ({ from, to }: { from?: Date; to?: Date }) =>
    datesMutation.mutate(
      {
        type: "patch",
        id: cycleDetails.id,
        data: {
          start_date: renderFormattedPayloadDate(from) ?? null,
          end_date: renderFormattedPayloadDate(to) ?? null,
        },
      },
      {
        onSuccess: (result) => {
          if (current() && result.captured.isCurrent())
            setToast({
              type: TOAST_TYPE.SUCCESS,
              title: t("project_cycles.action.update.success.title"),
              message: t("project_cycles.action.update.success.description"),
            });
        },
        onError: (error) => {
          if (current() && access.isCurrent())
            setToast({
              type: TOAST_TYPE.ERROR,
              title: t("project_cycles.action.update.failed.title"),
              message: error instanceof Error ? error.message : t("project_cycles.action.update.failed.description"),
            });
        },
      }
    );

  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <>
      <div className="sticky top-0 z-10 flex items-center justify-between bg-surface-1 pt-2">
        <div className="flex size-5 items-center justify-center">
          <button
            className="flex size-6 flex-shrink-0 items-center justify-center rounded-full bg-layer-3 hover:bg-layer-3-hover"
            onClick={() => handleClose()}
          >
            <ChevronRightIcon className="size-4 stroke-2 text-secondary" />
          </button>
        </div>
      </div>
      <div className="flex w-full flex-col gap-2">
        <div className="flex items-start justify-between gap-3 pt-2">
          <h4 className="w-full text-18 font-semibold break-words text-primary">{cycleDetails.name}</h4>
          {currentCycle && (
            <span
              className="flex h-6 min-w-20 items-center justify-center truncate rounded-sm px-3 text-center text-11 font-medium whitespace-nowrap"
              style={{
                color: currentCycle.color,
                backgroundColor: `${currentCycle.color}20`,
              }}
            >
              {t(currentCycle.i18n_title)}
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          <DateRangeDropdown
            className="h-7"
            buttonVariant="border-with-text"
            minDate={minDate}
            value={{
              from: getDate(cycleDetails.start_date),
              to: getDate(cycleDetails.end_date),
            }}
            onSelect={(value) => updateDates({ from: value?.from, to: value?.to })}
            placeholder={{
              from: t("project_cycles.start_date"),
              to: t("project_cycles.end_date"),
            }}
            customTooltipHeading={t("project_cycles.in_your_timezone")}
            customTooltipContent={
              <span className="flex gap-1">
                {renderFormattedDateInUserTimezone(cycleDetails.start_date ?? "")}
                <ArrowRight className="my-auto h-3 w-3 flex-shrink-0" />
                {renderFormattedDateInUserTimezone(cycleDetails.end_date ?? "")}
              </span>
            }
            mergeDates
            showTooltip={!!cycleDetails.start_date && !!cycleDetails.end_date} // show tooltip only if both start and end date are present
            required={cycleDetails.status !== "draft"}
            disabled={!isEditingAllowed || isArchived || isCompleted || datesMutation.isPending}
          />

          {projectUTCOffset && (
            <span className="cursor-default rounded-md bg-layer-1 px-2 py-1 text-11 text-tertiary">
              {projectUTCOffset}
            </span>
          )}
        </div>
      </div>
    </>
  );
};
