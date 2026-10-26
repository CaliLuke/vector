/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { INBOX_STATUS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { TInboxIssueStatus } from "@plane/types";
// constants
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";
import { InboxStatusIcon } from "../../inbox-status-icon";
import { InboxAppliedFilterGroup, toggleAppliedFilterValue } from "./shared";

function getStatusOptionDetail(status: TInboxIssueStatus) {
  return INBOX_STATUS.find((option) => option.status === status) || undefined;
}

export const InboxIssueAppliedFiltersStatus = function InboxIssueAppliedFiltersStatus() {
  // hooks
  const { filters, setFilter } = useIntakePreferences();
  const { t } = useTranslation();
  // derived values
  const filteredValues = filters?.status || [];
  const options = filteredValues.flatMap((value) => {
    const optionDetail = getStatusOptionDetail(value);
    if (!optionDetail) return [];
    const nextValues = toggleAppliedFilterValue(filteredValues, optionDetail.status);
    return [
      {
        key: String(value),
        leading: (
          <div className="relative flex h-3 w-3 flex-shrink-0 items-center justify-center overflow-hidden">
            <InboxStatusIcon type={optionDetail.status} />
          </div>
        ),
        label: t(optionDetail.i18n_title),
        onRemove: nextValues.length >= 1 ? () => setFilter("status", nextValues) : undefined,
      },
    ];
  });

  return <InboxAppliedFilterGroup label="Status" options={options} hasFilters={filteredValues.length > 0} />;
};
