/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EUserWorkspaceRoles, TUserPermissions } from "@plane/types";
import { isWorkspaceRole } from "@/helpers/permission-policy";

export type ProjectCatalogueScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  workspaceRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectRole: TUserPermissions | undefined;
  projectMembershipRole: TUserPermissions | undefined;
  sessionGeneration: number;
};

export type WorkspaceCatalogueScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  workspaceRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectRoles: readonly (readonly [string, TUserPermissions])[] | undefined;
  sessionGeneration: number;
};

export function workspaceCatalogueScopeReady(scope: WorkspaceCatalogueScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    isWorkspaceRole(scope.workspaceRole) &&
    scope.membershipRevision &&
    scope.projectRoles !== undefined
  );
}
