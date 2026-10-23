/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DatePresetFilter } from "@/components/core/filters/date-preset-filter";

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (value: string | string[]) => void;
  searchQuery: string;
};

export function FilterEndDate(props: Props) {
  return <DatePresetFilter {...props} title="Due date" />;
}
