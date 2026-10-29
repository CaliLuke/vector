/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useParams } from "react-router";

// icons
import type { LucideIcon } from "lucide-react";
import { MembersPropertyIcon } from "@plane/propel/icons";
// plane ui
import { Avatar, AvatarGroup } from "@plane/ui";
// plane utils
import { cn } from "@plane/utils/common";
// hooks
import { membersOptions } from "@/queries/lookups";
//
import type { TPublicMember } from "@plane/types";

type Props = {
  memberIds: string[];
  shouldShowBorder?: boolean;
};

type AvatarProps = {
  showTooltip: boolean;
  members: TPublicMember[];
  icon?: LucideIcon;
};

const ButtonAvatars = function ButtonAvatars(props: AvatarProps) {
  const { showTooltip, members, icon: Icon } = props;

  if (Array.isArray(members)) {
    if (members.length > 1) {
      return (
        <AvatarGroup size="md" showTooltip={!showTooltip}>
          {members.map((member) => {
            if (!member) return;
            return <Avatar key={member.id} src={member.member__avatar} name={member.member__display_name} />;
          })}
        </AvatarGroup>
      );
    } else if (members.length === 1) {
      return (
        <Avatar
          src={members[0].member__avatar}
          name={members[0].member__display_name}
          size="md"
          showTooltip={!showTooltip}
        />
      );
    }
  }

  return Icon ? (
    <Icon className="h-3 w-3 flex-shrink-0" />
  ) : (
    <MembersPropertyIcon className="mx-[4px] h-3 w-3 flex-shrink-0" />
  );
};

export const IssueBlockMembers = function IssueBlockMembers({ memberIds, shouldShowBorder = true }: Props) {
  const { data: members = [] } = useRecordCollection(membersOptions(useParams().anchor));

  const selectedMembers = members.filter((item) => memberIds.includes(item.member));

  return (
    <div className="relative flex h-full flex-wrap items-center gap-1">
      <div
        className={cn("flex flex-shrink-0 cursor-default items-center rounded-md text-11", {
          "border-[0.5px] border-strong px-2.5 py-1": shouldShowBorder && !selectedMembers?.length,
        })}
      >
        <div className="flex items-center gap-1.5 text-secondary">
          <ButtonAvatars members={selectedMembers} showTooltip={false} />
          {!shouldShowBorder && selectedMembers.length <= 1 && (
            <span>{selectedMembers?.[0]?.member__display_name ?? "No Assignees"}</span>
          )}
        </div>
      </div>
    </div>
  );
};
