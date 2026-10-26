/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { useState, useEffect } from "react";
import { LockKeyhole, LockKeyholeOpen } from "lucide-react";
// plane imports
import { Tooltip } from "@plane/propel/tooltip";
// hooks
import { usePageOperations } from "@/hooks/use-page-operations";
import { usePagePermissions } from "@/hooks/use-page-permissions";
// store

type Props = {
  page: TPage;
  workspaceSlug: string;
};

export function PageLockControl({ page, workspaceSlug }: Props) {
  const { canCurrentUserLockPage } = usePagePermissions(page, workspaceSlug);
  const [transition, setTransition] = useState({ pageId: page.id, locked: page.is_locked, showUnlocked: false });
  if (transition.pageId !== page.id || transition.locked !== page.is_locked) {
    setTransition({
      pageId: page.id,
      locked: page.is_locked,
      showUnlocked: transition.pageId === page.id && transition.locked && !page.is_locked,
    });
  }
  const displayState = page.is_locked ? "locked" : transition.showUnlocked ? "unlocked" : "neutral";
  // page operations
  const {
    pageOperations: { toggleLock },
  } = usePageOperations({
    page,
    workspaceSlug,
  });

  useEffect(() => {
    if (!transition.showUnlocked) return undefined;
    const timer = setTimeout(() => setTransition((current) => ({ ...current, showUnlocked: false })), 600);
    return () => clearTimeout(timer);
  }, [transition]);

  if (!canCurrentUserLockPage) return null;

  // Render different UI based on the current display state
  return (
    <>
      {displayState === "neutral" && (
        <Tooltip tooltipContent="Lock" position="bottom">
          <button
            type="button"
            onClick={() => void toggleLock()}
            className="grid size-6 flex-shrink-0 place-items-center rounded-sm text-secondary transition-colors hover:bg-layer-1 hover:text-primary"
            aria-label="Lock"
          >
            <LockKeyhole className="size-3.5" />
          </button>
        </Tooltip>
      )}

      {displayState === "locked" && (
        <button
          type="button"
          onClick={() => void toggleLock()}
          className="flex h-6 items-center gap-1 rounded-sm bg-accent-primary/20 px-2 text-accent-primary transition-colors hover:bg-accent-primary/30"
          aria-label="Locked"
        >
          <LockKeyhole className="animate-lock-icon size-3.5 flex-shrink-0" />
          <span className="animate-text-slide-in overflow-hidden text-11 font-medium whitespace-nowrap transition-all duration-500 ease-out">
            Locked
          </span>
        </button>
      )}

      {displayState === "unlocked" && (
        <div
          className="flex h-6 animate-fade-out items-center gap-1 rounded-sm px-2 text-secondary"
          aria-label="Unlocked"
        >
          <LockKeyholeOpen className="animate-unlock-icon size-3.5 flex-shrink-0" />
          <span className="animate-text-slide-in animate-text-fade-out overflow-hidden text-11 font-medium whitespace-nowrap transition-all duration-500 ease-out">
            Unlocked
          </span>
        </div>
      )}
    </>
  );
}
