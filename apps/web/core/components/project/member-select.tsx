/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Ban } from "lucide-react";
// plane ui
import { Avatar, CustomSearchSelect } from "@plane/ui";
// helpers
import { getFileURL } from "@plane/utils/file";
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds } from "@/helpers/member-rosters";

type Props = {
  value: string | undefined;
  onChange: (val: string) => void;
  workspaceSlug: string;
  projectId: string;
  isDisabled?: boolean;
};

export function MemberSelect(props: Props) {
  const { value, onChange, workspaceSlug, projectId, isDisabled = false } = props;
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId);
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);
  if (projectRoster.isError || workspaceRoster.isError) {
    return (
      <div role="alert" className="text-13 text-danger-primary">
        Unable to load project members. Please retry.
        <button
          type="button"
          className="ml-2 underline"
          onClick={() => {
            void projectRoster.retry();
            void workspaceRoster.retry();
          }}
        >
          Retry
        </button>
      </div>
    );
  }
  const memberIds = projectRosterMemberIds(projectRoster.memberships, workspaceRoster.membersById, undefined, false);
  const options = memberIds?.flatMap((userId) => {
    const member = workspaceRoster.membersById?.get(userId)?.member;
    if (!member) return [];
    return [
      {
        value: member.id,
        query: member.display_name,
        content: (
          <div className="flex items-center gap-2">
            <Avatar name={member.display_name} src={getFileURL(member.avatar_url)} />
            {member.display_name}
          </div>
        ),
      },
    ];
  });
  const selectedId = value !== undefined && projectRoster.membershipsByMemberId?.has(value) ? value : undefined;
  const selectedOption = selectedId ? workspaceRoster.membersById?.get(selectedId)?.member : undefined;

  return (
    <CustomSearchSelect
      value={value}
      label={
        <div className="flex h-3.5 items-center gap-2">
          {selectedOption && <Avatar name={selectedOption.display_name} src={getFileURL(selectedOption.avatar_url)} />}
          {selectedOption ? (
            selectedOption.display_name
          ) : (
            <div className="flex items-center gap-2">
              <Ban className="h-3.5 w-3.5 rotate-90 text-placeholder" />
              <span className="text-13 text-placeholder">None</span>
            </div>
          )}
        </div>
      }
      buttonClassName="!px-3 !py-2 bg-surface-1"
      options={
        options && [
          ...options,
          {
            value: "none",
            query: "none",
            content: (
              <div className="flex items-center gap-2">
                <Ban className="h-3.5 w-3.5 rotate-90 text-placeholder" />
                <span className="py-0.5 text-13 text-placeholder">None</span>
              </div>
            ),
          },
        ]
      }
      maxHeight="md"
      onChange={onChange}
      disabled={isDisabled}
    />
  );
}
