/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

import { ChartNoAxesColumn, SlidersHorizontal } from "lucide-react";
// plane imports
import { ISSUE_STORE_TO_FILTERS_MAP } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
// hooks
import { useProjectIssueDisplayPreferences } from "@/hooks/use-project-issue-display-preferences";
// plane web imports
import type { TProject } from "@plane/types";
// local imports
import { WorkItemsModal } from "../analytics/work-items/modal";
import { WorkItemFiltersToggle } from "../work-item-filters/filters-toggle";
import {
  DisplayFiltersSelection,
  FiltersDropdown,
  LayoutSelection,
  MobileLayoutSelection,
} from "./issue-layouts/filters";

type Props = {
  currentProjectDetails: TProject | undefined;
  projectId: string;
  workspaceSlug: string;
  canUserCreateIssue: boolean | undefined;
  storeType?: EIssuesStoreType.PROJECT | EIssuesStoreType.EPIC;
};
const LAYOUTS = [
  EIssueLayoutTypes.LIST,
  EIssueLayoutTypes.KANBAN,
  EIssueLayoutTypes.CALENDAR,
  EIssueLayoutTypes.SPREADSHEET,
  EIssueLayoutTypes.GANTT,
];

export const HeaderFilters = function HeaderFilters(props: Props) {
  const {
    currentProjectDetails,
    projectId,
    workspaceSlug,
    canUserCreateIssue,
    storeType = EIssuesStoreType.PROJECT,
  } = props;
  // i18n
  const { t } = useTranslation();
  // states
  const [analyticsModal, setAnalyticsModal] = useState(false);
  // store hooks
  const {
    issueFilters: displayPreferences,
    handleLayoutChange,
    handleDisplayFilters,
    handleDisplayProperties,
  } = useProjectIssueDisplayPreferences({ workspaceSlug, projectId });
  const displayFilters = displayPreferences?.displayFilters;
  const displayProperties = displayPreferences?.displayProperties;
  // derived values
  const activeLayout = displayFilters?.layout;
  const layoutDisplayFiltersOptions = ISSUE_STORE_TO_FILTERS_MAP[storeType]?.layoutOptions[activeLayout];

  return (
    <>
      <WorkItemsModal
        workspaceSlug={workspaceSlug}
        isOpen={analyticsModal}
        onClose={() => setAnalyticsModal(false)}
        projectDetails={currentProjectDetails ?? undefined}
        isEpic={storeType === EIssuesStoreType.EPIC}
      />
      <div className="hidden @4xl:flex">
        <LayoutSelection
          layouts={LAYOUTS}
          onChange={(layout) => handleLayoutChange(layout)}
          selectedLayout={activeLayout}
        />
      </div>
      <div className="flex @4xl:hidden">
        <MobileLayoutSelection
          layouts={LAYOUTS}
          onChange={(layout) => handleLayoutChange(layout)}
          activeLayout={activeLayout}
        />
      </div>
      <WorkItemFiltersToggle />
      <FiltersDropdown
        miniIcon={<SlidersHorizontal className="size-3.5" />}
        title={t("common.display")}
        placement="bottom-end"
      >
        <DisplayFiltersSelection
          layoutDisplayFiltersOptions={layoutDisplayFiltersOptions}
          displayFilters={displayFilters ?? {}}
          handleDisplayFiltersUpdate={handleDisplayFilters}
          displayProperties={displayProperties ?? {}}
          handleDisplayPropertiesUpdate={handleDisplayProperties}
          cycleViewDisabled={!currentProjectDetails?.cycle_view}
          moduleViewDisabled={!currentProjectDetails?.module_view}
          isEpic={storeType === EIssuesStoreType.EPIC}
        />
      </FiltersDropdown>
      {canUserCreateIssue ? (
        <Button className="hidden px-2 md:block" onClick={() => setAnalyticsModal(true)} variant="secondary" size="lg">
          <div className="hidden @4xl:flex">{t("common.analytics")}</div>
          <div className="flex @4xl:hidden">
            <ChartNoAxesColumn className="size-3.5" />
          </div>
        </Button>
      ) : (
        <></>
      )}
    </>
  );
};
