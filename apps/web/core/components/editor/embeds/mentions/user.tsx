/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, Link } from "@tanstack/react-router";

// plane imports
import { ROLE } from "@plane/constants";
import { Popover } from "@plane/propel/popover";
import { Avatar } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
// hooks
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useCurrentUser } from "@/hooks/use-current-user";

type Props = {
  id: string;
};

export const EditorUserMention = function EditorUserMention(props: Props) {
  const { id } = props;
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { membershipsByMemberId } = useProjectMemberRoster(workspaceSlug, projectId);
  // derived values
  const userDetails = memberById(id)?.member;
  const roleDetails = membershipsByMemberId?.get(id)?.role;
  const profileLink = `/${workspaceSlug}/profile/${id}`;

  if (!userDetails) {
    return <div className="not-prose inline rounded-sm bg-layer-1 px-1 py-0.5 text-tertiary no-underline">@user</div>;
  }

  return (
    <div
      className={cn(
        "not-prose inline rounded-sm bg-accent-subtle-active px-1 py-0.5 text-accent-primary no-underline",
        {
          "bg-label-yellow-bg text-label-yellow-text": id === currentUser?.id,
        }
      )}
    >
      <Popover>
        <Popover.Button delay={100} openOnHover>
          <Link to={profileLink}>@{userDetails?.display_name}</Link>
        </Popover.Button>
        <Popover.Panel side="bottom" align="start">
          <div className="w-60 rounded-lg border-[0.5px] border-strong bg-surface-1 p-3 shadow-raised-200">
            <div className="flex items-center gap-3">
              <div className="grid size-10 flex-shrink-0 place-items-center">
                <Avatar
                  src={getFileURL(userDetails?.avatar_url ?? "")}
                  name={userDetails?.display_name}
                  size={40}
                  className="text-18"
                  showTooltip={false}
                />
              </div>
              <div>
                <Link to={profileLink} className="not-prose text-13 font-medium text-primary hover:underline">
                  {userDetails?.first_name} {userDetails?.last_name}
                </Link>
                {roleDetails && <p className="text-11 text-secondary">{ROLE[roleDetails]}</p>}
              </div>
            </div>
          </div>
        </Popover.Panel>
      </Popover>
    </div>
  );
};
