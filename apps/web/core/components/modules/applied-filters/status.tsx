/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// ui
import { MODULE_STATUS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { ModuleStatusIcon } from "@plane/propel/icons";
import { AppliedStatusChips } from "@/components/core/filters/applied-status-chips";
import type { AppliedStatusFilterProps } from "@/components/core/filters/applied-status-chips";
// constants

export const AppliedStatusFilters = function AppliedStatusFilters(props: AppliedStatusFilterProps) {
  const { handleRemove, values, editable } = props;
  const { t } = useTranslation();

  const chips = values.flatMap((status) => {
    const statusDetails = MODULE_STATUS.find((item) => item.value === status);
    if (!statusDetails) return [];
    return [
      {
        value: status,
        label: t(statusDetails.i18n_label),
        className: "flex items-center gap-1 rounded-sm bg-layer-1 p-1 text-11",
        icon: <ModuleStatusIcon status={statusDetails.value} height="12px" width="12px" />,
      },
    ];
  });

  return <AppliedStatusChips chips={chips} editable={editable} onRemove={handleRemove} />;
};
