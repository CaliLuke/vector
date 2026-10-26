/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { CloseIcon } from "@plane/propel/icons";
// plane ui
import { Avatar } from "@plane/ui";
// helpers
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
// types
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

export type AppliedMembersFiltersProps = {
  className?: string;
  handleRemove: (val: string) => void;
  values: string[];
  editable: boolean | undefined;
  workspaceSlug: string | undefined;
};

export function AppliedMembersFilters(props: AppliedMembersFiltersProps) {
  const { handleRemove, values, editable, className, workspaceSlug } = props;
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);

  return (
    <>
      {values.map((memberId) => {
        const memberDetails = membersById?.get(memberId)?.member;

        if (!memberDetails) return null;

        return (
          <div
            key={memberId}
            className={cn("flex items-center gap-1 rounded-sm bg-layer-1 px-1.5 py-1 text-11", className)}
          >
            <Avatar
              name={memberDetails.display_name}
              src={getFileURL(memberDetails.avatar_url)}
              showTooltip={false}
              size={"sm"}
            />
            <span className="normal-case">{memberDetails.display_name}</span>
            {editable && (
              <button
                type="button"
                className="grid place-items-center text-tertiary hover:text-secondary"
                onClick={() => handleRemove(memberId)}
              >
                <CloseIcon height={10} width={10} strokeWidth={2} />
              </button>
            )}
          </div>
        );
      })}
    </>
  );
}
