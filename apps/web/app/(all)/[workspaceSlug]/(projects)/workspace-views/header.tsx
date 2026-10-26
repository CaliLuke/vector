/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";

// plane imports
import {
  EIssueFilterType,
  ISSUE_DISPLAY_FILTERS_BY_PAGE,
  GLOBAL_VIEW_TRACKER_ELEMENTS,
  DEFAULT_GLOBAL_VIEWS_LIST,
} from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { ViewsIcon } from "@plane/propel/icons";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties, ICustomSearchSelectOption } from "@plane/types";
import { EIssueLayoutTypes } from "@plane/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { DisplayFiltersSelection, FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
import { DefaultWorkspaceViewQuickActions } from "@/components/workspace/views/default-view-quick-action";
import { CreateUpdateWorkspaceViewModal } from "@/components/workspace/views/modal";
import { WorkspaceViewQuickActions } from "@/components/workspace/views/quick-action";
// hooks
import { useWorkspaceIssuePreferences } from "@/providers/workspace-issue-preferences";
import { useWorkspaceViews } from "@/hooks/use-workspace-views";

export const GlobalIssuesHeader = function GlobalIssuesHeader() {
  // states
  const [createViewModal, setCreateViewModal] = useState(false);
  // router
  const router = useRouter();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { globalViewId } = useParams({ strict: false });
  // store hooks
  const preferences = useWorkspaceIssuePreferences();
  const { views: currentWorkspaceViews } = useWorkspaceViews(workspaceSlug);
  const viewDetails = currentWorkspaceViews?.find((view) => view.id === globalViewId);
  const { t } = useTranslation();

  const issueFilters = preferences?.issueFilters;

  const activeLayout = issueFilters?.displayFilters?.layout;
  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      if (!globalViewId) return;
      preferences?.updateFilters(EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter);
    },
    [preferences, globalViewId]
  );

  const handleDisplayProperties = useCallback(
    (property: Partial<IIssueDisplayProperties>) => {
      if (!globalViewId) return;
      preferences?.updateFilters(EIssueFilterType.DISPLAY_PROPERTIES, property);
    },
    [preferences, globalViewId]
  );

  const isLocked = viewDetails?.is_locked;

  const defaultViewDetails = DEFAULT_GLOBAL_VIEWS_LIST.find((view) => view.key === globalViewId);

  const defaultOptions = DEFAULT_GLOBAL_VIEWS_LIST.map((view) => ({
    value: view.key,
    query: view.key,
    content: <SwitcherLabel name={t(view.i18n_label)} LabelIcon={ViewsIcon} />,
  }));

  const workspaceOptions = (currentWorkspaceViews ?? []).map((view) => ({
    value: view.id,
    query: view.name,
    content: <SwitcherLabel name={view.name} LabelIcon={ViewsIcon} />,
  }));

  const switcherOptions: ICustomSearchSelectOption[] = [...defaultOptions, ...workspaceOptions];
  const currentLayoutFilters = useMemo(() => {
    const layout = activeLayout ?? EIssueLayoutTypes.SPREADSHEET;
    return ISSUE_DISPLAY_FILTERS_BY_PAGE.my_issues.layoutOptions[layout];
  }, [activeLayout]);

  return (
    <>
      <CreateUpdateWorkspaceViewModal isOpen={createViewModal} onClose={() => setCreateViewModal(false)} />
      <Header>
        <Header.LeftItem>
          <Breadcrumbs>
            <Breadcrumbs.Item
              component={<BreadcrumbLink label={t("views")} icon={<ViewsIcon className="h-4 w-4 text-tertiary" />} />}
            />
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={globalViewId ?? ""}
                  navigationItems={switcherOptions}
                  onChange={(value: string) => {
                    void router.navigate({ href: `/${workspaceSlug}/workspace-views/${value}` });
                  }}
                  title={viewDetails?.name ?? t(defaultViewDetails?.i18n_label ?? "")}
                  icon={
                    <Breadcrumbs.Icon>
                      <ViewsIcon className="size-4 flex-shrink-0 text-tertiary" />
                    </Breadcrumbs.Icon>
                  }
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
        </Header.LeftItem>

        <Header.RightItem className="items-center">
          {globalViewId && <WorkItemFiltersToggle />}
          {!isLocked && (
            <FiltersDropdown title={t("common.display")} placement="bottom-end">
              <DisplayFiltersSelection
                layoutDisplayFiltersOptions={currentLayoutFilters}
                displayFilters={issueFilters?.displayFilters ?? {}}
                handleDisplayFiltersUpdate={handleDisplayFilters}
                displayProperties={issueFilters?.displayProperties ?? {}}
                handleDisplayPropertiesUpdate={handleDisplayProperties}
              />
            </FiltersDropdown>
          )}
          <Button
            variant="primary"
            size="lg"
            data-ph-element={GLOBAL_VIEW_TRACKER_ELEMENTS.RIGHT_HEADER_ADD_BUTTON}
            onClick={() => setCreateViewModal(true)}
          >
            {t("workspace_views.add_view")}
          </Button>
          <div className="hidden md:block">
            {viewDetails && <WorkspaceViewQuickActions workspaceSlug={workspaceSlug} view={viewDetails} />}
            {defaultViewDetails && (
              <DefaultWorkspaceViewQuickActions workspaceSlug={workspaceSlug} view={defaultViewDetails} />
            )}
          </div>
        </Header.RightItem>
      </Header>
    </>
  );
};
