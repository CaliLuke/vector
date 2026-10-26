/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { useParams } from "@tanstack/react-router";

import type { TCycleEstimateType } from "@plane/types";
import { EEstimateSystem } from "@plane/types";
import { CustomSelect } from "@plane/ui";
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useCycleDetails } from "@/hooks/use-cycles";
import { cycleHasPoints } from "@/queries/cycle-module-selectors";
// local imports
import { cycleEstimateOptions } from "../analytics-sidebar/issue-progress";

type TProps = {
  isArchived?: boolean;
  value: TCycleEstimateType;
  onChange: (value: TCycleEstimateType) => Promise<void>;
  showDefault?: boolean;
  projectId: string;
  cycleId: string;
};

export const EstimateTypeDropdown = function EstimateTypeDropdown(props: TProps) {
  const { value, onChange, projectId, cycleId, showDefault = false, isArchived } = props;
  const { workspaceSlug } = useParams({ strict: false });
  const { cycle } = useCycleDetails(workspaceSlug, projectId, cycleId, isArchived);
  const { isEnabled: isCurrentProjectEstimateEnabled, activeEstimate } = useProjectEstimates(workspaceSlug, projectId);
  const currentProjectEstimateType = activeEstimate?.type;
  return (cycleHasPoints(cycle ?? undefined) || isCurrentProjectEstimateEnabled) &&
    currentProjectEstimateType !== EEstimateSystem.CATEGORIES ? (
    <div className="relative flex items-center gap-2">
      <CustomSelect
        value={value}
        label={<span>{cycleEstimateOptions.find((v) => v.value === value)?.label ?? "None"}</span>}
        onChange={onChange}
        maxHeight="lg"
        buttonClassName="bg-surface-2 border-none rounded-sm text-13 font-medium "
      >
        {cycleEstimateOptions.map((item) => (
          <CustomSelect.Option key={item.value} value={item.value}>
            {item.label}
          </CustomSelect.Option>
        ))}
      </CustomSelect>
    </div>
  ) : showDefault ? (
    <span className="capitalize">{cycleEstimateOptions.find((v) => v.value === value)?.label ?? value}</span>
  ) : null;
};
