/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CommentActions } from "@/components/comments/types";
// plane imports
import type { E_SORT_ORDER, TActivityFilters } from "@plane/constants";
import { filterActivityOnSelectedFilters } from "@plane/constants";

// components
import { CommentCard } from "@/components/comments/card/root";
// hooks
import type { useIssueDiscussion } from "@/hooks/use-issue-discussion";
import { EActivityFilterType as ActivityType } from "@plane/constants";
import orderBy from "lodash-es/orderBy.js";
// local imports
import { IssueActivityItem } from "./activity/activity-list";
import { IssueActivityLoader } from "./loader";

type TIssueActivityCommentRoot = {
  discussion: ReturnType<typeof useIssueDiscussion>;
  workspaceSlug: string;
  projectId: string;
  isIntakeIssue: boolean;
  issueId: string;
  selectedFilters: TActivityFilters[];
  activityOperations: CommentActions;
  showAccessSpecifier?: boolean;
  disabled?: boolean;
  sortOrder: E_SORT_ORDER;
};

export const IssueActivityCommentRoot = function IssueActivityCommentRoot(props: TIssueActivityCommentRoot) {
  const {
    workspaceSlug,
    isIntakeIssue,
    issueId,
    selectedFilters,
    activityOperations,
    showAccessSpecifier,
    projectId,
    disabled,
    sortOrder,
  } = props;
  const { discussion } = props;
  if (discussion.isError)
    return (
      <div role="alert">
        Could not load discussion.{" "}
        <button type="button" onClick={() => void discussion.retry().catch(() => {})}>
          Retry
        </button>
      </div>
    );
  if (discussion.isPending || !discussion.activity.isSuccess || !discussion.comments.isSuccess)
    return <IssueActivityLoader />;
  const activities = discussion.activity.data;
  const comments = discussion.comments.data;
  const activityAndComments = orderBy(
    [
      ...activities.map((activity) => ({
        id: activity.id,
        created_at: activity.created_at,
        activity_type:
          activity.field === "state"
            ? ActivityType.STATE
            : activity.field === "assignees"
              ? ActivityType.ASSIGNEE
              : activity.field === null
                ? ActivityType.DEFAULT
                : ActivityType.ACTIVITY,
      })),
      ...comments.map((comment) => ({
        id: comment.id,
        created_at: comment.created_at,
        activity_type: ActivityType.COMMENT,
      })),
    ],
    (item) => new Date(item.created_at || 0),
    sortOrder
  );

  if (activityAndComments.length <= 0) return null;

  const filteredActivityAndComments = filterActivityOnSelectedFilters(activityAndComments, selectedFilters);

  return (
    <div>
      {filteredActivityAndComments.map((activityComment, index) => {
        const comment = comments.find((item) => item.id === activityComment.id);
        const activity = activities.find((item) => item.id === activityComment.id);
        return comment ? (
          <CommentCard
            key={activityComment.id}
            workspaceSlug={workspaceSlug}
            entityId={issueId}
            comment={comment}
            activityOperations={activityOperations}
            ends={index === 0 ? "top" : index === filteredActivityAndComments.length - 1 ? "bottom" : undefined}
            showAccessSpecifier={!!showAccessSpecifier}
            showCopyLinkOption={!isIntakeIssue}
            disabled={disabled}
            projectId={projectId}
          />
        ) : activity ? (
          <IssueActivityItem
            key={activityComment.id}
            activity={activity}
            ends={index === 0 ? "top" : index === filteredActivityAndComments.length - 1 ? "bottom" : undefined}
          />
        ) : null;
      })}
    </div>
  );
};
