/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

import { ListFilter } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { ChevronDownIcon } from "@plane/propel/icons";
// components
import { ProjectFiltersDropdown } from "@/components/project/dropdowns/filters/project-filters-dropdown";
import { ProjectOrderByDropdown } from "@/components/project/dropdowns/order-by";
// hooks
import { useProjectListFilters } from "@/hooks/use-project-list-filters";

export function ProjectsListMobileHeader() {
  // i18n
  const { t } = useTranslation();
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { displayFilters, updateDisplayFilters } = useProjectListFilters();

  return (
    <div className="flex w-full border-b border-subtle bg-surface-1 py-2 md:hidden">
      <ProjectOrderByDropdown
        value={displayFilters?.order_by}
        onChange={(val) => {
          if (!workspaceSlug || val === displayFilters?.order_by) return;
          updateDisplayFilters({
            order_by: val,
          });
        }}
        isMobile
      />
      <div className="flex w-full justify-around border-l border-subtle">
        <ProjectFiltersDropdown
          menuButton={
            <div className="flex items-center gap-2 text-13 text-secondary">
              <ListFilter className="h-3 w-3" />
              {t("common.filters")}
              <ChevronDownIcon className="h-3 w-3" strokeWidth={2} />
            </div>
          }
        />
      </div>
    </div>
  );
}
