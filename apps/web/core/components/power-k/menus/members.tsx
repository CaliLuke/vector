/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Command } from "cmdk";
// plane imports
import { Avatar } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
// local imports
import { PowerKModalCommandItem } from "../ui/modal/command-item";

type Props = {
  workspaceSlug: string | undefined;
  handleSelect: (assigneeId: string) => void;
  heading?: string;
  userIds: string[] | undefined;
  value: string[];
};

export const PowerKMembersMenu = function PowerKMembersMenu(props: Props) {
  const { workspaceSlug, handleSelect, heading, userIds, value } = props;
  // store hooks
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);

  return (
    <Command.Group heading={heading}>
      {userIds?.map((memberId) => {
        const memberDetails = memberById(memberId)?.member;
        if (!memberDetails) return;

        return (
          <PowerKModalCommandItem
            key={memberId}
            iconNode={
              <Avatar
                name={memberDetails?.display_name}
                src={getFileURL(memberDetails?.avatar_url ?? "")}
                showTooltip={false}
                className="shrink-0"
              />
            }
            isSelected={value.includes(memberId)}
            label={memberDetails?.display_name}
            onSelect={() => handleSelect(memberId)}
          />
        );
      })}
    </Command.Group>
  );
};
