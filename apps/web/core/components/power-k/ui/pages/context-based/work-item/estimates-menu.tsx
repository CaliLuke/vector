/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { Command } from "cmdk";

import { Triangle } from "lucide-react";
// plane types
import { useTranslation } from "@plane/i18n";
import { EEstimateSystem } from "@plane/types";
import type { TIssue } from "@plane/types";
import { Spinner } from "@plane/ui";
import { convertMinutesToHoursMinutesString } from "@plane/utils/datetime";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
// local imports
import { PowerKModalCommandItem } from "../../../modal/command-item";

type Props = {
  handleSelect: (estimatePointId: string | null) => void;
  workItemDetails: TIssue;
};

export const PowerKWorkItemEstimatesMenu = function PowerKWorkItemEstimatesMenu(props: Props) {
  const { handleSelect, workItemDetails } = props;
  // store hooks
  const { workspaceSlug } = useParams({ strict: false });
  const {
    activeEstimate: currentActiveEstimate,
    points,
    isPending,
    isError,
    isAccessDenied,
    retry,
  } = useProjectEstimates(workspaceSlug, workItemDetails.project_id);
  const estimatePointIds = points?.flatMap((point) => (point.id ? [point.id] : []));
  const estimatePointById = (pointId: string) => points?.find((point) => point.id === pointId);
  // translation
  const { t } = useTranslation();

  if (isError || isAccessDenied)
    return (
      <div role="alert">
        Estimates are unavailable.{" "}
        <button
          type="button"
          onClick={() => {
            void retry();
          }}
        >
          Retry
        </button>
      </div>
    );
  if (isPending) return <Spinner />;

  return (
    <Command.Group>
      <PowerKModalCommandItem
        icon={Triangle}
        label={t("project_settings.estimates.no_estimate")}
        isSelected={workItemDetails.estimate_point === null}
        onSelect={() => handleSelect(null)}
      />
      {estimatePointIds && estimatePointIds.length > 0 ? (
        estimatePointIds.map((estimatePointId) => {
          const estimatePoint = estimatePointById(estimatePointId);
          if (!estimatePoint) return null;

          return (
            <PowerKModalCommandItem
              key={estimatePoint.id}
              icon={Triangle}
              label={
                currentActiveEstimate?.type === EEstimateSystem.TIME
                  ? convertMinutesToHoursMinutesString(Number(estimatePoint.value))
                  : estimatePoint.value
              }
              isSelected={workItemDetails.estimate_point === estimatePoint.id}
              onSelect={() => handleSelect(estimatePoint.id ?? null)}
            />
          );
        })
      ) : (
        <div className="text-center">No estimate found</div>
      )}
    </Command.Group>
  );
};
