/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// i18n
import { useTranslation } from "@plane/i18n";
// hooks
import { useProjectListFilters } from "@/hooks/use-project-list-filters";
import { ControlledExpandableSearch } from "@/components/common/controlled-expandable-search";

export function ProjectSearch() {
  // i18n
  const { t } = useTranslation();
  // hooks
  const { searchQuery, updateSearchQuery } = useProjectListFilters();
  return (
    <ControlledExpandableSearch
      value={searchQuery}
      onChange={updateSearchQuery}
      placeholder={t("common.search.label")}
      className="flex items-center"
      triggerClassName="-mr-1"
      fieldClassName="ml-auto gap-1 bg-surface-1"
      expandedFieldClassName="w-30 border-subtle px-2.5 py-1.5 md:w-64"
      inputClassName=""
      searchIconClassName="h-3.5 w-3.5"
    />
  );
}
