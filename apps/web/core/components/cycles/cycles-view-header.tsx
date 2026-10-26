/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";

import { ListFilter } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { TCycleFilters } from "@plane/types";
import { calculateTotalFilters } from "@plane/utils/filter";
// components
import { ControlledExpandableSearch } from "@/components/common/controlled-expandable-search";
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
// hooks
import { useCycleCataloguePreferences } from "@/providers/cycle-module-preferences";
// local imports
import { CycleFiltersSelection } from "./dropdowns";
import { toggleCatalogueFilterValues } from "../cycles-modules/toggle-catalogue-filter-values";

type Props = {
  projectId: string;
};

export const CyclesViewHeader = function CyclesViewHeader(props: Props) {
  const { projectId } = props;
  // hooks
  const { workspaceSlug } = useParams({ strict: false });
  const preferences = useCycleCataloguePreferences(workspaceSlug, projectId);
  const { filters: currentProjectFilters, search: searchQuery } = preferences;
  const { update } = preferences;
  const { t } = useTranslation();
  const handleFilters = useCallback(
    (key: keyof TCycleFilters, value: string | string[]) => {
      if (!projectId) return;
      const newValues = toggleCatalogueFilterValues(currentProjectFilters?.[key], value);

      update({ filters: { ...currentProjectFilters, [key]: newValues } });
    },
    [currentProjectFilters, projectId, update]
  );

  const isFiltersApplied = calculateTotalFilters(currentProjectFilters ?? {}) !== 0;

  return (
    <div className="flex items-center gap-2">
      <ControlledExpandableSearch
        value={searchQuery}
        onChange={(search) => update({ search })}
        placeholder="Search"
        initiallyOpen={searchQuery !== ""}
        openWhenValueExists
        className="flex items-center"
        triggerClassName="-mr-1"
        fieldClassName="ml-auto gap-1 bg-surface-1"
        expandedFieldClassName="w-64 border-subtle px-2.5 py-1.5"
        searchIconClassName="h-3.5 w-3.5"
      />

      <FiltersDropdown
        icon={<ListFilter className="h-3 w-3" />}
        title={t("common.filters")}
        placement="bottom-end"
        isFiltersApplied={isFiltersApplied}
      >
        <CycleFiltersSelection filters={currentProjectFilters ?? {}} handleFiltersUpdate={handleFilters} />
      </FiltersDropdown>
    </div>
  );
};
