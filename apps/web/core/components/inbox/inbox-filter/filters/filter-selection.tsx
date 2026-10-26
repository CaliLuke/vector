/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useParams } from "@tanstack/react-router";
import { FilterSelectionSearch } from "@/components/common/filters/filter-selection-search";
// hooks
import { useProjectLabels } from "@/hooks/use-labels";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds } from "@/helpers/member-rosters";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { FilterDate } from "./date";
import { FilterLabels } from "./labels";
import { FilterMember } from "./members";
import { FilterPriority } from "./priority";
import { FilterStatus } from "./status";

export function InboxIssueFilterSelection() {
  // hooks
  const { isMobile } = usePlatformOS();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { data: currentUser } = useCurrentUser();
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId);
  const projectMemberIds = projectId
    ? projectRosterMemberIds(projectRoster.memberships, workspaceRoster.membersById, currentUser?.id, false)
    : undefined;
  const { labels: projectLabels } = useProjectLabels(workspaceSlug?.toString(), projectId?.toString());
  // states
  const [filtersSearchQuery, setFiltersSearchQuery] = useState("");

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      {/* oxlint-disable-next-line jsx_a11y/no-autofocus */}
      <FilterSelectionSearch value={filtersSearchQuery} onChange={setFiltersSearchQuery} autoFocus={!isMobile} />

      <div className="vertical-scrollbar scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-y-auto px-2.5">
        {/* status */}
        <div className="py-2">
          <FilterStatus searchQuery={filtersSearchQuery} />
        </div>
        {/* Priority */}
        <div className="py-2">
          <FilterPriority searchQuery={filtersSearchQuery} />
        </div>
        {/* assignees */}
        <div className="py-2">
          <FilterMember
            filterKey="assignees"
            label="Assignees"
            searchQuery={filtersSearchQuery}
            memberIds={projectMemberIds ?? []}
            workspaceSlug={workspaceSlug}
          />
        </div>
        {/* Created By */}
        <div className="py-2">
          <FilterMember
            filterKey="created_by"
            label="Created By"
            searchQuery={filtersSearchQuery}
            memberIds={projectMemberIds ?? []}
            workspaceSlug={workspaceSlug}
          />
        </div>
        {/* Labels */}
        <div className="py-2">
          <FilterLabels searchQuery={filtersSearchQuery} labels={projectLabels ?? []} />
        </div>
        {/* Created at */}
        <div className="py-2">
          <FilterDate filterKey="created_at" label="Created date" searchQuery={filtersSearchQuery} />
        </div>
        {/* Updated at */}
        <div className="py-2">
          <FilterDate filterKey="updated_at" label="Last updated date" searchQuery={filtersSearchQuery} />
        </div>
      </div>
    </div>
  );
}
