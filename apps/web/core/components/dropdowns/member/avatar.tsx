/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { LucideIcon } from "lucide-react";
import { MembersPropertyIcon } from "@plane/propel/icons";
// plane ui
import { Avatar, AvatarGroup } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
// plane utils
// helpers
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

type AvatarProps = {
  showTooltip: boolean;
  userIds: string | string[] | null;
  icon?: LucideIcon;
  size?: "sm" | "md" | "base" | "lg" | number;
  workspaceSlug: string | undefined;
};

export function ButtonAvatars(props: AvatarProps) {
  const { showTooltip, userIds, icon: Icon, size = "md", workspaceSlug } = props;
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const getUserDetails = (userId: string) => membersById?.get(userId)?.member;

  if (Array.isArray(userIds)) {
    if (userIds.length > 0)
      return (
        <AvatarGroup size={size} showTooltip={!showTooltip}>
          {userIds.map((userId) => {
            const userDetails = getUserDetails(userId);

            if (!userDetails) return;
            return <Avatar key={userId} src={getFileURL(userDetails.avatar_url)} name={userDetails.display_name} />;
          })}
        </AvatarGroup>
      );
  } else {
    if (userIds) {
      const userDetails = getUserDetails(userIds);
      return (
        <Avatar
          src={getFileURL(userDetails?.avatar_url ?? "")}
          name={userDetails?.display_name}
          size={size}
          showTooltip={!showTooltip}
        />
      );
    }
  }

  return Icon ? (
    <Icon className="h-3 w-3 flex-shrink-0" />
  ) : (
    <MembersPropertyIcon className={cn("mx-[4px] h-3 w-3 flex-shrink-0")} />
  );
}
