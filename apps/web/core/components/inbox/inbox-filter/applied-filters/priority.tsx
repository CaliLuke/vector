/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ISSUE_PRIORITIES } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { PriorityIcon } from "@plane/propel/icons";
import type { TIssuePriorities } from "@plane/types";
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";
import { InboxAppliedFilterGroup, toggleAppliedFilterValue } from "./shared";

function getPriorityOptionDetail(priority: TIssuePriorities) {
  return ISSUE_PRIORITIES.find((option) => option.key === priority) || undefined;
}

export const InboxIssueAppliedFiltersPriority = function InboxIssueAppliedFiltersPriority() {
  // hooks
  const { t } = useTranslation();
  const { filters, setFilter } = useIntakePreferences();
  // derived values
  const filteredValues = filters?.priority || [];
  const clearFilter = () => setFilter("priority", undefined);
  const options = filteredValues.flatMap((value) => {
    const optionDetail = getPriorityOptionDetail(value);
    if (!optionDetail) return [];
    return [
      {
        key: value,
        leading: (
          <div className="relative flex h-3 w-3 flex-shrink-0 items-center justify-center overflow-hidden">
            <PriorityIcon priority={optionDetail.key} className="h-3 w-3" />
          </div>
        ),
        label: optionDetail.title,
        onRemove: () => setFilter("priority", toggleAppliedFilterValue(filteredValues, optionDetail.key)),
      },
    ];
  });

  return (
    <InboxAppliedFilterGroup
      label={t("common.priority")}
      options={options}
      hasFilters={filteredValues.length > 0}
      onClearAll={clearFilter}
    />
  );
};
