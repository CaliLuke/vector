/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { usePageRecord } from "@/components/pages/page-record-context";

// plane imports
import { ScrollArea } from "@plane/propel/scrollarea";
// store
// local imports
import { PageContentBrowser } from "../../editor/summary";
import { PageNavigationPaneOutlineTabEmptyState } from "./empty-state/outline";

export function PageNavigationPaneOutlineTabPanel() {
  // derived values
  const { editorRef } = usePageRecord();

  return (
    <ScrollArea
      orientation="vertical"
      size="sm"
      scrollType="hover"
      className="hide-scrollbar size-full overflow-y-auto"
      viewportClassName="px-4"
    >
      <PageContentBrowser
        className="mt-0"
        editorRef={editorRef}
        emptyState={<PageNavigationPaneOutlineTabEmptyState />}
      />
    </ScrollArea>
  );
}
