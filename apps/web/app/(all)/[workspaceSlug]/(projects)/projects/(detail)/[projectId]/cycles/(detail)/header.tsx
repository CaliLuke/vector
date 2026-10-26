/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { useCallback, useRef, useState } from "react";

// icons
import { ChartNoAxesColumn, PanelRight, SlidersHorizontal } from "lucide-react";
// plane imports
import { EUserPermissions, WORK_ITEM_TRACKER_ELEMENTS } from "@plane/constants";
import { usePlatformOS } from "@plane/hooks";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { IconButton } from "@plane/propel/icon-button";
import { CycleIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { EIssuesStoreType, EIssueLayoutTypes } from "@plane/types";
import { Breadcrumbs, BreadcrumbNavigationSearchDropdown, Header } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { WorkItemsModal } from "@/components/analytics/work-items/modal";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { CycleQuickActions } from "@/components/cycles/quick-actions";
import {
  FiltersDropdown,
  ParentIssueResponsiveLayoutSelection,
  persistParentIssueLayout,
} from "@/components/issues/issue-layouts/filters";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useProjectCycles, useCycleDetails } from "@/hooks/use-cycles";
import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { ParentIssueDisplaySelection } from "@/components/issues/issue-layouts/filters/header/display-filters/parent-issue-display-selection";
import { useIssueListCount } from "@/hooks/use-issue-list-count";
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";

import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

export function CycleIssuesHeader() {
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // states
  const [analyticsModal, setAnalyticsModal] = useState(false);
  // router
  const router = useRouter();
  const { workspaceSlug, projectId, cycleId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/cycles/$cycleId",
  });
  // i18n
  const { t } = useTranslation();
  // store hooks
  const { count: workItemsCount } = useIssueListCount(EIssuesStoreType.CYCLE);
  const preferences = useParentIssuePreferences();
  const issueFilters = preferences.filters;
  const { cycle: cycleDetails } = useCycleDetails(workspaceSlug, projectId, cycleId);
  const catalogue = useProjectCycles(workspaceSlug, projectId);
  const { toggleCreateIssueModal } = useCommandPalette();
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const loader = routeProjectQuery.isPending;
  const { isMobile } = usePlatformOS();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const activeLayout = issueFilters?.displayFilters?.layout;

  const chart = useParentChartPreferences("cycle", workspaceSlug, projectId, cycleId);

  const isSidebarCollapsed = chart.sidebarCollapsed;
  const toggleSidebar = () => {
    chart.update({ sidebarCollapsed: !isSidebarCollapsed });
  };

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      if (!workspaceSlug || !projectId) return;
      void persistParentIssueLayout(preferences.setDisplay, layout);
    },
    [workspaceSlug, projectId, preferences]
  );

  // derived values
  const isCompletedCycle = cycleDetails?.status?.toLocaleLowerCase() === "completed";
  const canUserCreateIssue = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const switcherOptions = catalogue.cycles?.map((item) => ({
    value: item.id,
    query: item.name,
    content: <SwitcherLabel name={item.name} LabelIcon={CycleIcon} />,
  }));

  return (
    <>
      <WorkItemsModal
        workspaceSlug={workspaceSlug}
        projectDetails={currentProjectDetails}
        isOpen={analyticsModal}
        onClose={() => setAnalyticsModal(false)}
        cycleDetails={cycleDetails ?? undefined}
      />
      <Header>
        <Header.LeftItem>
          <div className="flex items-center gap-2">
            <Breadcrumbs onBack={router.history.back} isLoading={loader}>
              <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
              <Breadcrumbs.Item
                component={
                  <BreadcrumbLink
                    label="Cycles"
                    href={`/${workspaceSlug}/projects/${projectId}/cycles/`}
                    icon={<CycleIcon className="h-4 w-4 text-tertiary" />}
                  />
                }
              />
              <Breadcrumbs.Item
                component={
                  <BreadcrumbNavigationSearchDropdown
                    selectedItem={cycleId}
                    navigationItems={switcherOptions ?? []}
                    onChange={(value: string) => {
                      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/cycles/${value}` });
                    }}
                    title={cycleDetails?.name}
                    icon={
                      <Breadcrumbs.Icon>
                        <CycleIcon className="size-4 flex-shrink-0 text-tertiary" />
                      </Breadcrumbs.Icon>
                    }
                    isLast
                  />
                }
                isLast
              />
            </Breadcrumbs>
            {workItemsCount && workItemsCount > 0 ? (
              <Tooltip
                isMobile={isMobile}
                tooltipContent={`There are ${workItemsCount} ${
                  workItemsCount > 1 ? "work items" : "work item"
                } in this cycle`}
                position="bottom"
              >
                <span className="flex flex-shrink-0 cursor-default items-center justify-center rounded-xl bg-accent-primary/20 px-2 text-center text-11 font-semibold text-accent-primary">
                  {workItemsCount}
                </span>
              </Tooltip>
            ) : null}
          </div>
        </Header.LeftItem>
        <Header.RightItem className="items-center">
          <div className="hidden items-center gap-2 md:flex">
            <ParentIssueResponsiveLayoutSelection activeLayout={activeLayout} onChange={handleLayoutChange} />
            <WorkItemFiltersToggle />
            <FiltersDropdown
              title={t("common.display")}
              placement="bottom-end"
              miniIcon={<SlidersHorizontal className="size-3.5" />}
            >
              <ParentIssueDisplaySelection project={currentProjectDetails} />
            </FiltersDropdown>

            {canUserCreateIssue && (
              <>
                <Button onClick={() => setAnalyticsModal(true)} variant="secondary" size="lg">
                  <span className="hidden @4xl:flex">Analytics</span>
                  <span className="@4xl:hidden">
                    <ChartNoAxesColumn className="size-3.5" />
                  </span>
                </Button>
                {!isCompletedCycle && (
                  <Button
                    variant="primary"
                    size="lg"
                    onClick={() => {
                      toggleCreateIssueModal(true, EIssuesStoreType.CYCLE);
                    }}
                    data-ph-element={WORK_ITEM_TRACKER_ELEMENTS.HEADER_ADD_BUTTON.CYCLE}
                  >
                    {t("issue.add.label")}
                  </Button>
                )}
              </>
            )}
            <IconButton
              variant="tertiary"
              size="lg"
              icon={PanelRight}
              onClick={toggleSidebar}
              className={cn({
                "bg-accent-subtle text-accent-primary": !isSidebarCollapsed,
              })}
            />
            {cycleDetails && (
              <CycleQuickActions
                cycle={cycleDetails}
                parentRef={parentRef}
                cycleId={cycleId}
                projectId={projectId}
                workspaceSlug={workspaceSlug}
                customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
              />
            )}
          </div>
        </Header.RightItem>
      </Header>
    </>
  );
}
