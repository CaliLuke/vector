/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useWorkspaceLabels } from "@/hooks/use-labels";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";

export function ActivityLabelPill({ labelId, workspaceSlug }: { labelId: string; workspaceSlug: string }) {
  // store hooks
  const { data: catalogueWorkspace } = useWorkspaceBySlug(workspaceSlug);
  const { getWorkspaceLabels } = useWorkspaceLabels(workspaceSlug);
  const workspaceLabels = getWorkspaceLabels(catalogueWorkspace?.id);

  return (
    <span
      className="h-1.5 w-1.5 flex-shrink-0 rounded-full"
      style={{
        backgroundColor: workspaceLabels?.find((l) => l.id === labelId)?.color ?? "#000000",
      }}
      aria-hidden="true"
    />
  );
}
