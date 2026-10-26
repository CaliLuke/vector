/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { CYCLE_STATUS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { cn } from "@plane/utils/common";
import { AppliedStatusChips } from "@/components/core/filters/applied-status-chips";
import type { AppliedStatusFilterProps } from "@/components/core/filters/applied-status-chips";

export const AppliedStatusFilters = function AppliedStatusFilters(props: AppliedStatusFilterProps) {
  const { handleRemove, values, editable } = props;
  const { t } = useTranslation();

  const chips = values.map((status) => {
    const statusDetails = CYCLE_STATUS.find((item) => item.value === status);
    return {
      value: status,
      label: statusDetails ? t(statusDetails.i18n_title) : "",
      className: cn(
        "flex items-center gap-1 rounded-sm px-1.5 py-1 text-11",
        statusDetails?.bgColor,
        statusDetails?.textColor
      ),
    };
  });

  return <AppliedStatusChips chips={chips} editable={editable} onRemove={handleRemove} />;
};
