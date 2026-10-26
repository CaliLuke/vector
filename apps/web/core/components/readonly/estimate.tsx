/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { EstimatePropertyIcon } from "@plane/propel/icons";
import { EEstimateSystem } from "@plane/types";
import { cn } from "@plane/utils/common";
import { convertMinutesToHoursMinutesString } from "@plane/utils/datetime";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";

export type TReadonlyEstimateProps = {
  className?: string;
  hideIcon?: boolean;
  value: string | undefined | null;
  placeholder?: string;
  projectId: string | undefined;
  workspaceSlug: string;
};

export const ReadonlyEstimate = function ReadonlyEstimate(props: TReadonlyEstimateProps) {
  const { className, hideIcon = false, value, placeholder, projectId, workspaceSlug } = props;

  const { t } = useTranslation();
  const {
    activeEstimate: currentActiveEstimate,
    points,
    isPending,
    isError,
    isAccessDenied,
    retry,
  } = useProjectEstimates(workspaceSlug, projectId);
  const estimatePoint = points?.find((point) => point.id === value);

  const displayValue = estimatePoint
    ? currentActiveEstimate?.type === EEstimateSystem.TIME
      ? convertMinutesToHoursMinutesString(Number(estimatePoint.value))
      : estimatePoint.value
    : null;

  return (
    <div className={cn("flex items-center gap-1 text-body-xs-regular", className)}>
      {!hideIcon && <EstimatePropertyIcon className="size-4 flex-shrink-0" />}
      <span className="flex-grow truncate">
        {isError || isAccessDenied ? (
          <span role="alert">
            Estimate unavailable.{" "}
            <button
              type="button"
              onClick={() => {
                void retry();
              }}
            >
              Retry
            </button>
          </span>
        ) : isPending ? (
          t("common.loading")
        ) : (
          (displayValue ?? placeholder ?? t("common.none"))
        )}
      </span>
    </div>
  );
};
