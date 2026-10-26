/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import sortBy from "lodash-es/sortBy.js";
import { EUserPermissions } from "@plane/types";
import type { IWorkspaceMember, TProjectMembership } from "@plane/types";

function orderMemberIds(
  ids: string[],
  currentUserId: string | undefined,
  getDisplayName: (memberId: string) => string | undefined
) {
  return sortBy(ids, [(id) => id !== currentUserId, (id) => getDisplayName(id)?.toLowerCase()]);
}

export function workspaceRosterMemberIds(
  members: readonly IWorkspaceMember[] | undefined,
  currentUserId?: string
): string[] | undefined {
  if (!members) return undefined;
  const displayNames = new Map(members.map((row) => [row.member.id, row.member.display_name]));
  return orderMemberIds(
    members.filter((row) => !row.member.is_bot).map((row) => row.member.id),
    currentUserId,
    (memberId) => displayNames.get(memberId)
  );
}

export function projectRosterMemberIds(
  memberships: readonly TProjectMembership[] | undefined,
  membersById: ReadonlyMap<string, IWorkspaceMember> | undefined,
  currentUserId?: string,
  includeGuestUsers = true
): string[] | undefined {
  if (!memberships) return undefined;
  return orderMemberIds(
    memberships
      .filter((membership) => includeGuestUsers || membership.role !== EUserPermissions.GUEST)
      .map((membership) => membership.member),
    currentUserId,
    (memberId) => membersById?.get(memberId)?.member.display_name
  );
}
