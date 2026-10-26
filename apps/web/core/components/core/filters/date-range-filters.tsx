/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DatePresetFilter } from "@/components/core/filters/date-preset-filter";

type Props = {
  startDate: string[] | null;
  dueDate: string[] | null;
  handleStartDateUpdate: (value: string | string[]) => void;
  handleDueDateUpdate: (value: string | string[]) => void;
  searchQuery: string;
};

export function DateRangeFilters({
  startDate,
  dueDate,
  handleStartDateUpdate,
  handleDueDateUpdate,
  searchQuery,
}: Props) {
  return (
    <>
      <div className="py-2">
        <DatePresetFilter
          title="Start date"
          appliedFilters={startDate}
          handleUpdate={handleStartDateUpdate}
          searchQuery={searchQuery}
        />
      </div>
      <div className="py-2">
        <DatePresetFilter
          title="Due date"
          appliedFilters={dueDate}
          handleUpdate={handleDueDateUpdate}
          searchQuery={searchQuery}
        />
      </div>
    </>
  );
}
