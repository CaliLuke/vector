/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IEstimate } from "@plane/types";
// local imports
import { EstimateListItem } from "./estimate-list-item";

type TEstimateList = {
  estimates: readonly IEstimate[] | undefined;
  isAdmin: boolean;
  isEditable?: boolean;
  onDeleteClick?: (estimateId: string) => void;
};

export function EstimateList(props: TEstimateList) {
  const { estimates, isAdmin, isEditable = false, onDeleteClick } = props;

  if (!estimates || estimates?.length <= 0) return <></>;
  return (
    <div>
      {estimates &&
        estimates
          .filter((estimate) => estimate.id)
          .map((estimate) => (
            <EstimateListItem
              key={estimate.id}
              estimate={estimate}
              isAdmin={isAdmin}
              isEditable={isEditable}
              onDeleteClick={onDeleteClick}
            />
          ))}
    </div>
  );
}
