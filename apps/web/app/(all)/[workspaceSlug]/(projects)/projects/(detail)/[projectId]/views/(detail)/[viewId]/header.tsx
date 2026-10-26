/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { useCallback, useRef } from "react";

// plane imports
import {
  EIssueFilterType,
  ISSUE_DISPLAY_FILTERS_BY_PAGE,
  EUserPermissions,
  WORK_ITEM_TRACKER_ELEMENTS,
} from "@plane/constants";
import { Button } from "@plane/propel/button";
import { LockIcon, ViewsIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@plane/types";
import { EIssuesStoreType, EViewAccess, EIssueLayoutTypes } from "@plane/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherIcon, SwitcherLabel } from "@/components/common/switcher-label";
import { DisplayFiltersSelection, FiltersDropdown, LayoutSelection } from "@/components/issues/issue-layouts/filters";
import { ViewQuickActions } from "@/components/views/quick-actions";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectView, useProjectViews } from "@/hooks/use-project-views";
import { useProjectViewIssuePreferences } from "@/providers/project-view-issue-preferences";
import { useProjectPermissions } from "@/hooks/use-permissions";

// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

export const ProjectViewIssuesHeader = function ProjectViewIssuesHeader() {
  // refs
  const parentRef = useRef(null);
  // router
  const router = useRouter();
  const { workspaceSlug, projectId, viewId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/views/$viewId",
  });
  // store hooks
  const { toggleCreateIssueModal } = useCommandPalette();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const loader = routeProjectQuery.isPending;
  const { issueFilters, updateFilters } = useProjectViewIssuePreferences();
  const viewQuery = useProjectView(workspaceSlug, projectId, viewId);
  const viewsQuery = useProjectViews(workspaceSlug, projectId);

  const activeLayout = issueFilters.displayFilters?.layout;

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      void updateFilters(workspaceSlug, projectId, EIssueFilterType.DISPLAY_FILTERS, { layout: layout });
    },
    [workspaceSlug, projectId, updateFilters]
  );

  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      void updateFilters(workspaceSlug, projectId, EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter);
    },
    [workspaceSlug, projectId, updateFilters]
  );

  const handleDisplayProperties = useCallback(
    (property: Partial<IIssueDisplayProperties>) => {
      void updateFilters(workspaceSlug, projectId, EIssueFilterType.DISPLAY_PROPERTIES, property);
    },
    [workspaceSlug, projectId, updateFilters]
  );

  const viewDetails = viewQuery.view;

  const canUserCreateIssue = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  if (!viewDetails) return null;

  const switcherOptions = (viewsQuery.views ?? []).map((view) => ({
    value: view.id,
    query: view.name,
    content: <SwitcherLabel logo_props={view.logo_props} name={view.name} LabelIcon={ViewsIcon} />,
  }));

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs isLoading={loader}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label="Views"
                href={`/${workspaceSlug}/projects/${projectId}/views/`}
                icon={<ViewsIcon className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          <Breadcrumbs.Item
            component={
              <BreadcrumbNavigationSearchDropdown
                selectedItem={viewId}
                navigationItems={switcherOptions}
                onChange={(value: string) => {
                  void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/views/${value}` });
                }}
                title={viewDetails.name}
                icon={
                  <Breadcrumbs.Icon>
                    <SwitcherIcon logo_props={viewDetails.logo_props} LabelIcon={ViewsIcon} size={16} />
                  </Breadcrumbs.Icon>
                }
                isLast
              />
            }
          />
        </Breadcrumbs>

        {viewDetails.access === EViewAccess.PRIVATE ? (
          <div className="cursor-default text-tertiary">
            <Tooltip tooltipContent={"Private"}>
              <LockIcon className="h-4 w-4" />
            </Tooltip>
          </div>
        ) : (
          <></>
        )}
      </Header.LeftItem>
      <Header.RightItem className="items-center">
        <>
          {!viewDetails.is_locked && (
            <LayoutSelection
              layouts={[
                EIssueLayoutTypes.LIST,
                EIssueLayoutTypes.KANBAN,
                EIssueLayoutTypes.CALENDAR,
                EIssueLayoutTypes.SPREADSHEET,
                EIssueLayoutTypes.GANTT,
              ]}
              onChange={(layout) => handleLayoutChange(layout)}
              selectedLayout={activeLayout}
            />
          )}
          <WorkItemFiltersToggle />
          {!viewDetails.is_locked && (
            <FiltersDropdown title="Display" placement="bottom-end">
              <DisplayFiltersSelection
                layoutDisplayFiltersOptions={
                  activeLayout ? ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions[activeLayout] : undefined
                }
                displayFilters={issueFilters.displayFilters ?? {}}
                handleDisplayFiltersUpdate={handleDisplayFilters}
                displayProperties={issueFilters.displayProperties ?? {}}
                handleDisplayPropertiesUpdate={handleDisplayProperties}
                cycleViewDisabled={!currentProjectDetails?.cycle_view}
                moduleViewDisabled={!currentProjectDetails?.module_view}
              />
            </FiltersDropdown>
          )}
        </>
        {canUserCreateIssue && (
          <Button
            variant="primary"
            size="lg"
            onClick={() => {
              toggleCreateIssueModal(true, EIssuesStoreType.PROJECT_VIEW);
            }}
            data-ph-element={WORK_ITEM_TRACKER_ELEMENTS.HEADER_ADD_BUTTON.PROJECT_VIEW}
          >
            Add work item
          </Button>
        )}
        <div className="hidden md:block">
          <ViewQuickActions
            parentRef={parentRef}
            customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
            projectId={projectId}
            view={viewDetails}
            workspaceSlug={workspaceSlug}
          />
        </div>
      </Header.RightItem>
    </Header>
  );
};
