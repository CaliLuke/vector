/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter, useLocation } from "@tanstack/react-router";
import { useCallback } from "react";

// plane imports
import { cn } from "@plane/utils/common";
// hooks
import { useQueryParams } from "@/hooks/use-query-params";
// plane web imports
// local imports
import { PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM, PAGE_NAVIGATION_PANE_WIDTH } from "../navigation-pane";
import type { TVersionEditorProps } from "./editor";
import { PageVersionsMainContent } from "./main-content";

type Props = {
  editorComponent: React.FC<TVersionEditorProps>;
  handleRestore: (descriptionHTML: string) => Promise<void>;
  pageId: string;
  restoreEnabled: boolean;
};

export function PageVersionsOverlay(props: Props) {
  const { editorComponent, handleRestore, pageId, restoreEnabled } = props;
  // navigation
  const router = useRouter();
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  // query params
  const { updateQueryParams } = useQueryParams();
  // derived values
  const activeVersion = searchParams.get(PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM);
  const isOpen = !!activeVersion;

  const handleClose = useCallback(() => {
    const updatedRoute = updateQueryParams({
      paramsToRemove: [PAGE_NAVIGATION_PANE_VERSION_QUERY_PARAM],
    });
    void router.navigate({ href: updatedRoute });
  }, [router, updateQueryParams]);

  return (
    <div
      className={cn(
        "pointer-events-none absolute inset-0 z-[16] flex h-full overflow-hidden bg-surface-1 opacity-0 transition-opacity",
        {
          "pointer-events-auto opacity-100": isOpen,
        }
      )}
      style={{
        width: `calc(100% - ${PAGE_NAVIGATION_PANE_WIDTH}px)`,
      }}
    >
      <PageVersionsMainContent
        activeVersion={activeVersion}
        editorComponent={editorComponent}

        handleClose={handleClose}
        handleRestore={handleRestore}
        pageId={pageId}
        restoreEnabled={restoreEnabled}
      />
    </div>
  );
}
