/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { Link } from "@tanstack/react-router";
import type { TIssueActivity } from "@plane/types";

type TIssueUser = {
  activity: TIssueActivity;
  customUserName?: string;
};

export function IssueUser(props: TIssueUser) {
  const { activity, customUserName } = props;
  // hooks

  if (!activity) return <></>;

  return (
    <>
      {customUserName ? (
        <span className="font-medium text-primary">{customUserName}</span>
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
