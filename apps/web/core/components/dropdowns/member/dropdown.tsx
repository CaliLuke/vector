/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";

import type { LucideIcon } from "lucide-react";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds, workspaceRosterMemberIds } from "@/helpers/member-rosters";
// local imports
import { MemberDropdownBase } from "./base";
import type { MemberDropdownProps } from "./types";

type TMemberDropdownProps = {
  icon?: LucideIcon;
  memberIds?: string[];
  onClose?: () => void;
  optionsClassName?: string;
  projectId?: string;
  renderByDefault?: boolean;
  workspaceSlug: string | undefined;
} & MemberDropdownProps;

export function MemberDropdown(props: TMemberDropdownProps) {
  const { memberIds: propsMemberIds, projectId, workspaceSlug } = props;
  const { data: currentUser } = useCurrentUser();
  const [shouldLoadProjectRoster, setShouldLoadProjectRoster] = useState(false);
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);
  const isProjectRosterEnabled = Boolean(projectId && !propsMemberIds && shouldLoadProjectRoster);
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId, {
    canLoad: isProjectRosterEnabled,
  });

  const memberIds =
    propsMemberIds ??
    (projectId
      ? projectRosterMemberIds(projectRoster.memberships, workspaceRoster.membersById, currentUser?.id, false)
      : workspaceRosterMemberIds(workspaceRoster.members, currentUser?.id));
  const getUserDetails = (userId: string) => workspaceRoster.membersById?.get(userId)?.member;
  const isUserSuspended = (userId: string) => workspaceRoster.membersById?.get(userId)?.is_active === false;
  const isRosterLoading = workspaceRoster.isLoading || (isProjectRosterEnabled && projectRoster.isLoading);
  const isRosterError = workspaceRoster.isError || (isProjectRosterEnabled && projectRoster.isError);
  const onDropdownOpen = useCallback(() => setShouldLoadProjectRoster(true), []);
  const onRosterRetry = () => {
    if (workspaceRoster.isError) void workspaceRoster.retry();
    else if (isProjectRosterEnabled && projectRoster.isError) void projectRoster.retry();
  };

  return (
    <MemberDropdownBase
      {...props}
      getIsUserSuspended={isUserSuspended}
      getUserDetails={getUserDetails}
      isRosterLoading={isRosterLoading}
      isRosterError={isRosterError}
      onRosterRetry={onRosterRetry}
      memberIds={isRosterLoading || isRosterError ? undefined : (memberIds ?? [])}
      onDropdownOpen={onDropdownOpen}
    />
  );
}
