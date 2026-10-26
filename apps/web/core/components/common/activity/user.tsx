/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";

// types
import type { TWorkspaceBaseActivity } from "@plane/types";
// store hooks
import { useWorkspaceById } from "@/hooks/use-workspaces";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

type TUser = {
  activity: TWorkspaceBaseActivity;
  customUserName?: string;
};

export const User = function User(props: TUser) {
  const { activity, customUserName } = props;
  const { data: workspaceDetail } = useWorkspaceById(activity.workspace);
  const { memberById } = useWorkspaceMemberRoster(workspaceDetail?.slug);
  // derived values
  const actorDetail = memberById(activity.actor)?.member;

  return (
    <>
      {customUserName || !actorDetail || !workspaceDetail || actorDetail.display_name?.includes("-intake") ? (
        <span className="font-medium text-primary">{customUserName || "Vector"}</span>
      ) : (
        <Link
          to="/$workspaceSlug/profile/$userId/"
          params={{ workspaceSlug: workspaceDetail?.slug, userId: actorDetail?.id }}
          className="font-medium text-primary hover:underline"
        >
          {actorDetail?.display_name}
        </Link>
      )}
    </>
  );
};
