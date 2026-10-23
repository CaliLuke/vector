/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { Link } from "@tanstack/react-router";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";

type TIssueUser = {
  activityId: string;
  customUserName?: string;
};

export function IssueCreatorDisplay(props: TIssueUser) {
  const { activityId, customUserName } = props;
  // hooks
  const {
    activity: { getActivityById },
  } = useIssueDetail();

  const activity = getActivityById(activityId);

  if (!activity) return <></>;

  return (
    <>
      {customUserName ? (
        <span className="font-medium text-primary">{customUserName || "Vector"}</span>
      ) : (
        <Link
          to="/$workspaceSlug/profile/$userId/"
          params={{ workspaceSlug: activity?.workspace_detail?.slug, userId: activity?.actor_detail?.id }}
          className="font-medium text-primary hover:underline"
        >
          {activity.actor_detail?.display_name}
        </Link>
      )}
    </>
  );
}
