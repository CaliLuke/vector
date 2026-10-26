/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ISSUE_PRIORITY_FILTERS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { PriorityIcon } from "@plane/propel/icons";
import { FilterSection } from "./helpers/filter-section";
import type { FilterSelectionProps } from "./helpers/filter-section";
export function FilterPriority(props: FilterSelectionProps) {
  const { t } = useTranslation();
  const options = ISSUE_PRIORITY_FILTERS.map((priority) => ({
    id: priority.key,
    name: t(priority.titleTranslationKey),
    icon: <PriorityIcon priority={priority.key} className="size-3.5" />,
  }));
  return <FilterSection {...props} title="Priority" options={options} />;
}
