/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { discussionOwnerKey } from "@/queries/issue-discussion";

// plane package imports
import type { TActivityFilters } from "@plane/constants";
import { E_SORT_ORDER, defaultActivityFilters } from "@plane/constants";
import { useLocalStorage } from "@plane/hooks";
// i18n
import { useTranslation } from "@plane/i18n";
//types
// components
import { CommentCreate } from "@/components/comments/comment-create";
// hooks
import { useProjectDetail } from "@/hooks/use-projects";
// local imports
import { IssueActivityCommentRoot } from "./activity-comment-root";
import { useIssueDiscussion } from "@/hooks/use-issue-discussion";
import type { TIssueServiceType } from "@plane/types";
import { useWorkItemCommentOperations } from "./helper";
import { ActivitySortRoot } from "./sort-root";
import { ActivityFilterRoot } from "./filter-root";

type TIssueActivity = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  serviceType?: TIssueServiceType;
  disabled?: boolean;
  isIntakeIssue?: boolean;
};

export const IssueActivity = function IssueActivity(props: TIssueActivity) {
  const { workspaceSlug, projectId, issueId, disabled = false, isIntakeIssue = false } = props;
  // i18n
  const { t } = useTranslation();
  // hooks
  const { setValue: setFilterValue, storedValue: selectedFilters } = useLocalStorage(
    "issue_activitynextFilters",
    defaultActivityFilters
  );
  const { setValue: setSortOrder, storedValue: sortOrder } = useLocalStorage("activity_sort_order", E_SORT_ORDER.ASC);

  const projectQuery = useProjectDetail(workspaceSlug, projectId);

  // toggle filter
  const toggleFilter = (filter: TActivityFilters) => {
    if (!selectedFilters) return;
    let nextFilters = [];
    if (selectedFilters.includes(filter)) {
      if (selectedFilters.length === 1) return; // Ensure at least one filter is applied
      nextFilters = selectedFilters.filter((f) => f !== filter);
    } else {
      nextFilters = [...selectedFilters, filter];
    }

    setFilterValue([...new Set(nextFilters)]);
  };

  const toggleSortOrder = () => {
    setSortOrder(sortOrder === E_SORT_ORDER.ASC ? E_SORT_ORDER.DESC : E_SORT_ORDER.ASC);
  };

  // helper hooks
  const discussion = useIssueDiscussion({ workspaceSlug, projectId, issueId, serviceType: props.serviceType });
  const activityOperations = useWorkItemCommentOperations(workspaceSlug, projectId, issueId, discussion);

  const project = projectQuery.data;
  const canCompose = !disabled && Boolean(discussion.scope.accountId);
  const renderCommentCreationBox = (
    <CommentCreate
      key={discussionOwnerKey(discussion.scope)}
      canSubmit={!discussion.isPending && !discussion.isError}
      workspaceSlug={workspaceSlug}
      entityId={issueId}
      activityOperations={activityOperations}
      showToolbarInitially
      projectId={projectId}
    />
  );
  return (
    <div className="space-y-4">
      {projectQuery.isError && (
        <button type="button" onClick={() => void projectQuery.retryAccess()}>
          {t("retry")}
        </button>
      )}
      {/* header */}
      <div className="flex items-center justify-between">
        <div className="text-h5-medium text-primary">{t("common.activity")}</div>
        <div className="flex items-center gap-2">
          <ActivitySortRoot sortOrder={sortOrder || E_SORT_ORDER.ASC} toggleSort={toggleSortOrder} />
          <ActivityFilterRoot
            selectedFilters={selectedFilters || defaultActivityFilters}
            toggleFilter={toggleFilter}
            isIntakeIssue={isIntakeIssue}
            projectId={projectId}
          />
        </div>
      </div>

      {/* rendering activity */}
      <div className="space-y-3">
        <div className="min-h-[200px]">
          <div className="space-y-3">
            {canCompose && sortOrder === E_SORT_ORDER.DESC && renderCommentCreationBox}
            <IssueActivityCommentRoot
              discussion={discussion}
              projectId={projectId}
              workspaceSlug={workspaceSlug}
              isIntakeIssue={isIntakeIssue}
              issueId={issueId}
              selectedFilters={selectedFilters || defaultActivityFilters}
              activityOperations={activityOperations}
              showAccessSpecifier={!!project?.anchor}
              disabled={disabled}
              sortOrder={sortOrder || E_SORT_ORDER.ASC}
            />
            {canCompose && sortOrder === E_SORT_ORDER.ASC && renderCommentCreationBox}
          </div>
        </div>
      </div>
    </div>
  );
};
