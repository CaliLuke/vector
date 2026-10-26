/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import type { TProject } from "@plane/types";
import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { DisplayFiltersSelection } from "./display-filters-selection";

export function ParentIssueDisplaySelection({ project }: { project: TProject | undefined }) {
  const preferences = useParentIssuePreferences();
  const filters = preferences.filters;
  const layout = filters?.displayFilters.layout;
  return (
    <DisplayFiltersSelection
      layoutDisplayFiltersOptions={layout ? ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions[layout] : undefined}
      displayFilters={filters?.displayFilters}
      displayProperties={filters?.displayProperties ?? {}}
      handleDisplayFiltersUpdate={(patch) => {
        void preferences
          .setDisplay(patch)
          .catch((error) => console.warn("Issue preferences could not be saved.", error));
      }}
      handleDisplayPropertiesUpdate={(patch) => {
        void preferences
          .setProperties(patch)
          .catch((error) => console.warn("Issue preferences could not be saved.", error));
      }}
      ignoreGroupedFilters={[preferences.domain]}
      cycleViewDisabled={!project?.cycle_view}
      moduleViewDisabled={!project?.module_view}
    />
  );
}
