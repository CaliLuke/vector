/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EPageAccess, EUserPermissions } from "@plane/types";
import type { EUserWorkspaceRoles, IUserProjectsRole, TPage, TUserPermissions } from "@plane/types";
import { effectiveProjectRole } from "@/helpers/permission-policy";

export type TPagePermissions = {
  canCurrentUserAccessPage: boolean;
  canCurrentUserEditPage: boolean;
  canCurrentUserDuplicatePage: boolean;
  canCurrentUserLockPage: boolean;
  canCurrentUserChangeAccess: boolean;
  canCurrentUserArchivePage: boolean;
  canCurrentUserDeletePage: boolean;
  canCurrentUserFavoritePage: boolean;
  canCurrentUserMovePage: boolean;
  isContentEditable: boolean;
};

export const NO_PAGE_PERMISSIONS: TPagePermissions = {
  canCurrentUserAccessPage: false,
  canCurrentUserEditPage: false,
  canCurrentUserDuplicatePage: false,
  canCurrentUserLockPage: false,
  canCurrentUserChangeAccess: false,
  canCurrentUserArchivePage: false,
  canCurrentUserDeletePage: false,
  canCurrentUserFavoritePage: false,
  canCurrentUserMovePage: false,
  isContentEditable: false,
};

function highestPageRole(
  page: TPage,
  roles: IUserProjectsRole,
  workspaceRole: EUserWorkspaceRoles
): TUserPermissions | undefined {
  let highest: TUserPermissions | undefined;
  for (const projectId of page.project_ids ?? []) {
    const role = effectiveProjectRole(workspaceRole, roles, projectId);
    if (role !== undefined && (highest === undefined || role > highest)) highest = role;
  }
  return highest;
}

export function getPagePermissions(
  page: TPage | undefined,
  currentUserId: string | undefined,
  roles: IUserProjectsRole | undefined,
  workspaceRole: EUserWorkspaceRoles | null | undefined
): TPagePermissions {
  if (!page || !currentUserId || !roles || workspaceRole === undefined || workspaceRole === null)
    return NO_PAGE_PERMISSIONS;
  const highestRole = highestPageRole(page, roles, workspaceRole);
  if (highestRole === undefined) return NO_PAGE_PERMISSIONS;

  const isOwned = page.owned_by === currentUserId;
  const isPagePublic = page.access === EPageAccess.PUBLIC;
  const isAdmin = highestRole === EUserPermissions.ADMIN;
  const isMemberOrHigher = highestRole >= EUserPermissions.MEMBER;
  const canManageOwnedPage = isOwned || isAdmin;
  const isContentEditable = !page.archived_at && !page.is_locked && (isOwned || (isPagePublic && isMemberOrHigher));

  return {
    canCurrentUserAccessPage: isPagePublic || isOwned,
    canCurrentUserEditPage: isPagePublic ? isMemberOrHigher : isOwned,
    canCurrentUserDuplicatePage: isMemberOrHigher,
    canCurrentUserLockPage: canManageOwnedPage,
    canCurrentUserChangeAccess: canManageOwnedPage,
    canCurrentUserArchivePage: canManageOwnedPage,
    canCurrentUserDeletePage: canManageOwnedPage,
    canCurrentUserFavoritePage: isMemberOrHigher,
    canCurrentUserMovePage: canManageOwnedPage,
    isContentEditable,
  };
}

export function pageRedirectLink(workspaceSlug: string, projectId: string, pageId?: string) {
  const pagesPath = `/${workspaceSlug}/projects/${projectId}/pages`;
  return pageId ? `${pagesPath}/${pageId}` : pagesPath;
}
