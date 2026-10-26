/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
// components
import { EUserPermissions } from "@plane/constants";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { EIssuesStoreType } from "@plane/types";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useProjectPermissions } from "@/hooks/use-permissions";

export function ProjectViewEmptyState() {
  // store hooks
  const { toggleCreateIssueModal } = useCommandPalette();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  // auth
  const isCreatingIssueAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    // TODO: Add translation
    <EmptyStateDetailed
      assetKey="work-item"
      title="View work items will appear here"
      description="Work items help you track individual pieces of work. With work items, keep track of what's going on, who is working on it, and what's done."
      actions={[
        {
          label: "New work item",
          onClick: () => {
            toggleCreateIssueModal(true, EIssuesStoreType.PROJECT_VIEW);
          },
          disabled: !isCreatingIssueAllowed,
          variant: "primary",
        },
      ]}
    />
  );
}
