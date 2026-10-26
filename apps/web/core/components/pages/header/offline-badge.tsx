/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

// plane imports
import { Tooltip } from "@plane/propel/tooltip";
// hooks
import useOnlineStatus from "@/hooks/use-online-status";
import { usePagePermissions } from "@/hooks/use-page-permissions";
// store

type Props = {
  page: TPage;
  workspaceSlug: string;
};

export function PageOfflineBadge({ page, workspaceSlug }: Props) {
  // use online status
  const { isOnline } = useOnlineStatus();
  const { isContentEditable } = usePagePermissions(page, workspaceSlug);

  if (!isContentEditable || isOnline) return null;

  return (
    <Tooltip
      tooltipHeading="You are offline."
      tooltipContent="You can continue making changes. They will be synced when you are back online."
    >
      <div className="flex h-7 flex-shrink-0 items-center gap-2 rounded-full bg-layer-1 px-3 py-0.5 text-11 font-medium text-tertiary">
        <span className="size-1.5 flex-shrink-0 rounded-full bg-layer-1" />
        <span>Offline</span>
      </div>
    </Tooltip>
  );
}
