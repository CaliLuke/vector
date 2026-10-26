/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouteProject } from "@/hooks/use-projects";
import { useParams, useRouter } from "@tanstack/react-router";
import { useCallback, useRef, useState } from "react";

// icons
import { ChartNoAxesColumn, PanelRight, SlidersHorizontal } from "lucide-react";
// plane imports
import { EUserPermissions, WORK_ITEM_TRACKER_ELEMENTS } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { ModuleIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { EIssuesStoreType, EIssueLayoutTypes } from "@plane/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { WorkItemsModal } from "@/components/analytics/work-items/modal";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import {
  FiltersDropdown,
  ParentIssueResponsiveLayoutSelection,
  persistParentIssueLayout,
} from "@/components/issues/issue-layouts/filters";
import { ModuleQuickActions } from "@/components/modules";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useProjectModules, useModuleDetails } from "@/hooks/use-modules";
import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { ParentIssueDisplaySelection } from "@/components/issues/issue-layouts/filters/header/display-filters/parent-issue-display-selection";
import { useIssueListCount } from "@/hooks/use-issue-list-count";

import { useProjectPermissions } from "@/hooks/use-permissions";

import { useParentChartPreferences } from "@/providers/cycle-module-preferences";
import { usePlatformOS } from "@/hooks/use-platform-os";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { IconButton } from "@plane/propel/icon-button";

export function ModuleIssuesHeader() {
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // states
  const [analyticsModal, setAnalyticsModal] = useState(false);
  // router
  const router = useRouter();
  const { workspaceSlug, projectId, moduleId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/modules/$moduleId",
  });
  // hooks
  const { isMobile } = usePlatformOS();
  // store hooks
  const { count: workItemsCount } = useIssueListCount(EIssuesStoreType.MODULE);
  const preferences = useParentIssuePreferences();
  const issueFilters = preferences.filters;
  const { module: moduleDetails } = useModuleDetails(workspaceSlug, projectId, moduleId);
  const catalogue = useProjectModules(workspaceSlug, projectId);
  const { toggleCreateIssueModal } = useCommandPalette();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const loader = routeProjectQuery.isPending;
  // local storage
  const chart = useParentChartPreferences("module", workspaceSlug, projectId, moduleId);
  // derived values
  const isSidebarCollapsed = chart.sidebarCollapsed;
  const activeLayout = issueFilters?.displayFilters?.layout;
  const canUserCreateIssue = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const toggleSidebar = () => {
    chart.update({ sidebarCollapsed: !isSidebarCollapsed });
  };

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      if (!projectId || !preferences) return;
      void persistParentIssueLayout(preferences.setDisplay, layout);
    },
    [projectId, preferences]
  );

  const switcherOptions = catalogue.modules?.map((item) => ({
    value: item.id,
    query: item.name,
    content: <SwitcherLabel name={item.name} LabelIcon={ModuleIcon} />,
  }));

  return (
    <>
      <WorkItemsModal
        workspaceSlug={workspaceSlug}
        isOpen={analyticsModal}
        onClose={() => setAnalyticsModal(false)}
        moduleDetails={moduleDetails ?? undefined}
        projectDetails={currentProjectDetails}
      />
      <Header>
        <Header.LeftItem>
          <div className="flex items-center gap-2">
            <Breadcrumbs onBack={router.history.back} isLoading={loader}>
              <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId?.toString()} />
              <Breadcrumbs.Item
                component={
                  <BreadcrumbLink
                    label="Modules"
                    href={`/${workspaceSlug}/projects/${projectId}/modules/`}
                    icon={<ModuleIcon className="h-4 w-4 text-tertiary" />}
                    isLast
                  />
                }
                isLast
              />
              <Breadcrumbs.Item
                component={
                  <BreadcrumbNavigationSearchDropdown
                    selectedItem={moduleId?.toString() ?? ""}
                    navigationItems={switcherOptions ?? []}
                    onChange={(value: string) => {
                      void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/modules/${value}` });
                    }}
                    title={moduleDetails?.name}
                    icon={<ModuleIcon className="size-3.5 flex-shrink-0 text-tertiary" />}
                    isLast
                  />
                }
              />
            </Breadcrumbs>
            {workItemsCount && workItemsCount > 0 ? (
              <Tooltip
                isMobile={isMobile}
                tooltipContent={`There are ${workItemsCount} ${
                  workItemsCount > 1 ? "work items" : "work item"
                } in this module`}
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
          <div className="hidden gap-2 md:flex">
            <ParentIssueResponsiveLayoutSelection activeLayout={activeLayout} onChange={handleLayoutChange} />
            {moduleId && <WorkItemFiltersToggle />}
            <FiltersDropdown
              title="Display"
              placement="bottom-end"
              miniIcon={<SlidersHorizontal className="size-3.5" />}
            >
              <ParentIssueDisplaySelection project={currentProjectDetails} />
            </FiltersDropdown>
          </div>

          {canUserCreateIssue ? (
            <>
              <Button className="hidden md:block" onClick={() => setAnalyticsModal(true)} variant="secondary" size="lg">
                <span className="hidden @4xl:flex">Analytics</span>
                <span className="@4xl:hidden">
                  <ChartNoAxesColumn className="size-3.5" />
                </span>
              </Button>
              <Button
                variant="primary"
                size="lg"
                className="hidden sm:flex"
                onClick={() => {
                  toggleCreateIssueModal(true, EIssuesStoreType.MODULE);
                }}
                data-ph-element={WORK_ITEM_TRACKER_ELEMENTS.HEADER_ADD_BUTTON.MODULE}
              >
                Add work item
              </Button>
            </>
          ) : (
            <></>
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
          {moduleId && moduleDetails && (
            <ModuleQuickActions
              module={moduleDetails}
              parentRef={parentRef}
              moduleId={moduleId}
              projectId={projectId.toString()}
              workspaceSlug={workspaceSlug.toString()}
              customClassName="flex-shrink-0 flex items-center justify-center bg-layer-1/70 rounded-sm size-[26px]"
            />
          )}
        </Header.RightItem>
      </Header>
    </>
  );
}
