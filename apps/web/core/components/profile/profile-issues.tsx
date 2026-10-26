/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import { EIssuesStoreType } from "@plane/types";
// components
import { ProfileIssuesKanBanLayout } from "@/components/issues/issue-layouts/kanban/roots/profile-issues-root";
import { ProfileIssuesListLayout } from "@/components/issues/issue-layouts/list/roots/profile-issues-root";
import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { WorkspaceLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/workspace-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useLocalIssuePreferences } from "@/providers/local-issue-preferences";

type Props = {
  type: "assigned" | "subscribed" | "created";
};

export function ProfileIssuesPage(props: Props) {
  void props.type;
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { userId } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/profile/$userId" });
  const localPreferences = useLocalIssuePreferences();
  if (!localPreferences || localPreferences.domain !== EIssuesStoreType.PROFILE)
    throw new Error("Profile issue preferences are not mounted for this route.");
  const { issueFilters, updateRichFilters } = localPreferences;
  // derived values
  const activeLayout = issueFilters?.displayFilters?.layout || undefined;

  return (
    <WorkspaceLevelWorkItemFiltersHOC
      entityId={userId}
      entityType={EIssuesStoreType.PROFILE}
      filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.profile_issues.filters}
      initialWorkItemFilters={issueFilters}
      updateFilters={updateRichFilters}
      workspaceSlug={workspaceSlug}
    >
      {({ editor: profileWorkItemsFilter }) => (
        <>
          <div className="flex h-full w-full flex-col">
            {profileWorkItemsFilter && <WorkItemFiltersRow editor={profileWorkItemsFilter} />}
            <div className="relative h-full w-full overflow-auto">
              {activeLayout === "list" ? (
                <ProfileIssuesListLayout />
              ) : activeLayout === "kanban" ? (
                <ProfileIssuesKanBanLayout />
              ) : null}
            </div>
          </div>
          {/* peek overview */}
          <IssuePeekOverview />
        </>
      )}
    </WorkspaceLevelWorkItemFiltersHOC>
  );
}
