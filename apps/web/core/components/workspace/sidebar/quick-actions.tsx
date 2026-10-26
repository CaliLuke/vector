/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { EUserPermissions, SIDEBAR_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { AddWorkItemIcon } from "@plane/propel/icons";
import { SidebarAddButton } from "@/components/sidebar/add-button";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
import { useWorkspacePermissions } from "@/hooks/use-permissions";

export function SidebarQuickActions() {
  const { t } = useTranslation();
  // router
  const { workspaceSlug } = useParams({ strict: false });
  // store hooks
  const { toggleCreateIssueModal } = useCommandPalette();
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug?.toString());
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  // derived values
  const canCreateIssue = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const disabled = joinedProjectIds.length === 0 || !canCreateIssue;

  return (
    <div className="flex cursor-pointer items-center justify-between gap-2">
      <SidebarAddButton
        label={
          <>
            <AddWorkItemIcon className="size-4" />
            <span className="max-w-[145px] truncate text-13 font-medium">{t("sidebar.new_work_item")}</span>
          </>
        }
        onClick={() => toggleCreateIssueModal(true)}
        disabled={disabled}
        data-ph-element={SIDEBAR_TRACKER_ELEMENTS.CREATE_WORK_ITEM_BUTTON}
      />
    </div>
  );
}
