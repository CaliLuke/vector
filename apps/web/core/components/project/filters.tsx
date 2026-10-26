/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { cn } from "@plane/utils/common";
// components
// hooks
import { useProjectListFilters } from "@/hooks/use-project-list-filters";
// local imports
import { ProjectFiltersDropdown } from "./dropdowns/filters/project-filters-dropdown";
import { ProjectOrderByDropdown } from "./dropdowns/order-by";

type Props = {
  filterMenuButton?: React.ReactNode;
  classname?: string;
  filterClassname?: string;
  isMobile?: boolean;
};

function HeaderFilters({ filterMenuButton, isMobile, classname = "", filterClassname = "" }: Props) {
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { displayFilters, updateDisplayFilters } = useProjectListFilters();

  return (
    <div className={cn("flex gap-3", classname)}>
      <ProjectOrderByDropdown
        value={displayFilters?.order_by}
        onChange={(val) => {
          if (!workspaceSlug || val === displayFilters?.order_by) return;
          updateDisplayFilters({
            order_by: val,
          });
        }}
        isMobile={isMobile}
      />
      <div className={cn(filterClassname)}>
        <ProjectFiltersDropdown menuButton={filterMenuButton || null} />
      </div>
    </div>
  );
}
export default HeaderFilters;
