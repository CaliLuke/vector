/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IEstimate } from "@plane/types";
import { orderedEstimatePoints } from "@/queries/estimates";
// plane imports
import { EEstimateSystem } from "@plane/constants";
import { convertMinutesToHoursMinutesString } from "@plane/utils/datetime";
// components
import { EstimateListItemButtons } from "./estimate-list-item-buttons";
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
// hooks

type TEstimateListItem = {
  estimate: IEstimate;
  isAdmin: boolean;
  isEditable: boolean;
  onDeleteClick?: (estimateId: string) => void;
};

export function EstimateListItem(props: TEstimateListItem) {
  const { estimate: currentEstimate } = props;
  const estimatePointValues = orderedEstimatePoints(currentEstimate)?.map((point) => point.value);
  if (!currentEstimate.id) return null;

  return (
    <SettingsBoxedControlItem
      title={currentEstimate.name}
      description={estimatePointValues
        ?.map((estimatePointValue) => {
          if (currentEstimate.type === EEstimateSystem.TIME) {
            return convertMinutesToHoursMinutesString(Number(estimatePointValue));
          }
          return estimatePointValue;
        })
        .join(", ")}
      control={
        <EstimateListItemButtons
          estimateId={currentEstimate.id}
          isAdmin={props.isAdmin}
          isEditable={props.isEditable}
          onDeleteClick={props.onDeleteClick}
        />
      }
    />
  );
}
