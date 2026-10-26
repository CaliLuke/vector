/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { getPagePermissions, NO_PAGE_PERMISSIONS } from "@/helpers/page-permissions";
import type { TPagePermissions } from "@/helpers/page-permissions";

export function usePagePermissions(
  page: TPage | null | undefined,
  workspaceSlug: string | undefined
): TPagePermissions {
  const currentUserId = useCurrentUserId();
  const { roles, workspaceRole } = useWorkspaceProjectRoles(workspaceSlug);
  return page ? getPagePermissions(page, currentUserId, roles, workspaceRole) : NO_PAGE_PERMISSIONS;
}
