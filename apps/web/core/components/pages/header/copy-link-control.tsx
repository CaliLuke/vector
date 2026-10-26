/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { useState, useRef, useEffect, useCallback } from "react";

import { LinkIcon, CheckIcon } from "@plane/propel/icons";
// plane imports
import { Tooltip } from "@plane/propel/tooltip";
import { IconButton } from "@plane/propel/icon-button";
import { cn } from "@plane/utils/common";
// hooks
import { usePageOperations } from "@/hooks/use-page-operations";
// store

type Props = {
  page: TPage;
  workspaceSlug: string;
};

export function PageCopyLinkControl({ page, workspaceSlug }: Props) {
  const [isCopied, setIsCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // page operations
  const { pageOperations } = usePageOperations({
    page,
    workspaceSlug,
  });

  // Cleanup timer on unmount
  useEffect(
    () => () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    },
    []
  );

  const handleCopy = useCallback(async () => {
    if (!(await pageOperations.copyLink())) return;
    setIsCopied(true);

    // Clear previous timer if it exists
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    // Set new timer
    timerRef.current = setTimeout(() => {
      setIsCopied(false);
      timerRef.current = null;
    }, 1000);
  }, [pageOperations]);

  return (
    <Tooltip tooltipContent={isCopied ? "Copied!" : "Copy link"} position="bottom">
      <IconButton
        variant="ghost"
        size="lg"
        icon={isCopied ? CheckIcon : LinkIcon}
        onClick={() => void handleCopy()}
        aria-label={isCopied ? "Copied link" : "Copy link"}
        className={cn(isCopied && "text-success-primary")}
      />
    </Tooltip>
  );
}
