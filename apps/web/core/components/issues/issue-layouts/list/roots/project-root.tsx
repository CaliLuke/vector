/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { observer } from "mobx-react";

// plane imports
import { EUserPermissions, EUserPermissionsLevel } from "@plane/constants";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { ProjectIssueQuickActions } from "../../quick-action-dropdowns";
import { BaseListRoot } from "../base-list-root";

export const ListLayout = observer(function ListLayout() {
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // hooks
  const { allowPermissions } = useUserPermissions();

  if (!workspaceSlug) return null;

  const canEditPropertiesBasedOnProject = (projectId: string) =>
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug.toString(),
      projectId
    );

  return (
    <BaseListRoot
      QuickActions={ProjectIssueQuickActions}
      canEditPropertiesBasedOnProject={canEditPropertiesBasedOnProject}
    />
  );
});
